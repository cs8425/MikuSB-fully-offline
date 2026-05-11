package main

import (
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// global recycle buffer
var (
	copyBuf sync.Pool
)

var (
	targetDomains = []string{
		"kx486hcreg.execute-api.us-east-1.amazonaws.com", // ????
		"amazingseasuncdn.com",
		"amazingseasun.com",
		"seasungames.com",
		"snowbreak-game.com",
		"xoyo.games",
		"yo.games",
		"qcloud.com",
		"xgsdk.xoyo.games",
		"xqdata.xoyo.games",
		"tencentcs.com",
	}
)

type ProxyServer struct {
	tag string // for check inbound

	dialer *net.Dialer

	srv *ChanListener

	tlsTermination bool

	backend string
	targets []string

	shouldRedirect func(host string, port int, backend string) bool
}

// TODO: fix config
func NewProxyServer(tag string, dialer *net.Dialer, backend *ChanListener) *ProxyServer {
	if dialer == nil {
		dialer = &net.Dialer{
			Timeout: 5 * time.Second,
		}
	}
	return &ProxyServer{
		tag: tag,

		dialer: dialer,
		srv:    backend,

		tlsTermination: true,

		backend: "",
		targets: append([]string{}, targetDomains...),
		shouldRedirect: func(host string, port int, backend string) bool {
			host = strings.ToLower(host)
			for _, target := range targetDomains {
				if host == target || strings.HasSuffix(host, "."+target) {
					return true
				}
			}
			return false
		},
	}
}

func (px *ProxyServer) Start(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			Vln(4, px.tag, "Accept error:", err)
			continue
		}
		go px.handleConnSocks(conn, px.dialer)
	}
}

func (px *ProxyServer) handleConnSocks(p1 net.Conn, dialer *net.Dialer) {
	// defer p1.Close()
	needClose := true
	defer func() {
		if needClose {
			p1.Close()
		}
	}()
	var b [256]byte

	// read handshake header
	if _, err := io.ReadFull(p1, b[:2]); err != nil {
		Vln(3, px.tag, "read header err", p1, err)
		return
	}
	if b[0] != 0x05 { // only Socks5
		Vln(3, px.tag, "client not socks5", p1, b[:2])
		return
	}
	// max b[:255]
	if _, err := io.ReadFull(p1, b[:int(b[1])]); err != nil {
		Vln(3, px.tag, "read auth methods err", p1, err)
		return
	}
	// TODO: lookup 0x00

	// reply: NO AUTHENTICATION REQUIRED
	p1.Write([]byte{0x05, 0x00})

	// read req header
	if _, err := io.ReadFull(p1, b[:4]); err != nil {
		return
	}
	if b[1] != 0x01 { // 0x01: CONNECT
		px.replyAndClose(p1, 0x07) // X'07' Command not supported
		return
	}

	// handle ATYP (Address Type)
	var host string
	switch b[3] {
	case 0x01: // IP V4
		if _, err := io.ReadFull(p1, b[:4]); err != nil {
			return
		}
		host = net.IPv4(b[0], b[1], b[2], b[3]).String()
		// host = net.IP(b[0:4]).String()
	case 0x03: // DOMAINNAME
		// domain name length
		if _, err := io.ReadFull(p1, b[:1]); err != nil {
			return
		}
		n := int(b[0]) // max 255
		if _, err := io.ReadFull(p1, b[:n]); err != nil {
			return
		}
		host = string(b[0:n])
	case 0x04: // IP V6
		if _, err := io.ReadFull(p1, b[:16]); err != nil {
			return
		}
		host = net.IP(b[0:16]).String()
	default:
		px.replyAndClose(p1, 0x08) // X'08' Address type not supported
		return
	}

	// read port
	if _, err := io.ReadFull(p1, b[:2]); err != nil {
		return
	}
	port := int(b[0])<<8 | int(b[1])
	backend := net.JoinHostPort(host, strconv.Itoa(port))

	if px.shouldRedirect != nil && px.shouldRedirect(host, port, backend) {
		Vln(3, px.tag, "socks redirect:", backend, ">>", "sdk server")
		reply := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		// Vln(6, px.tag, "[dbg]conn", p2.LocalAddr(), "=>", p2.RemoteAddr())
		p1.Write(reply) // reply OK
		px.srv.Push(p1)
		needClose = false
		return
	}
	p2, err := dialer.Dial("tcp", backend)
	if err != nil {
		Vln(2, backend, err)
		px.replyAndClose(p1, 0x05) // X'05'
		return
	}

	reply := []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	Vln(6, px.tag, "[dbg]conn", p2.LocalAddr(), "=>", p2.RemoteAddr())
	p1.Write(reply) // reply OK

	handleCp(p1, p2)
}

func (px *ProxyServer) replyAndClose(p1 net.Conn, rpy int) {
	p1.Write([]byte{0x05, byte(rpy), 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	// p1.Close()
}

func handleCp(p1, p2 io.ReadWriteCloser) {
	defer p1.Close()
	defer p2.Close()

	// start tunnel
	p1die := make(chan struct{})
	go func() {
		buf := copyBuf.Get().([]byte)
		io.CopyBuffer(p1, p2, buf)
		close(p1die)
		copyBuf.Put(buf)
	}()

	p2die := make(chan struct{})
	go func() {
		buf := copyBuf.Get().([]byte)
		io.CopyBuffer(p2, p1, buf)
		close(p2die)
		copyBuf.Put(buf)
	}()

	// wait for tunnel termination
	select {
	case <-p1die:
	case <-p2die:
	}
}

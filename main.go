package main

import (
	"crypto/tls"
	"flag"
	"log"
	"net"
	"net/http"
	"time"
)

var (
	localAddr = flag.String("l", ":18888", "socks5 proxy bind address")

	sdkAddr13 = flag.String("l13", ":13443", "sdk server 13443")
	sdkAddr18 = flag.String("l18", ":18443", "sdk server 18443")
	sdkAddr31 = flag.String("l31", ":31443", "sdk server 31443")

	gameAddr = flag.String("game", "127.0.0.1:21000", "mock game server address")

	readTimeout  = flag.Int("rt", 5, "http ReadTimeout (Second), <= 0 disable")
	writeTimeout = flag.Int("wt", 0, "http WriteTimeout (Second), <= 0 disable")
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	flag.Parse()

	copyBuf.New = func() any {
		return make([]byte, 16384)
	}

	// sdk server api
	mux := http.NewServeMux()
	initRoute(mux, *gameAddr)
	hdr := reqlog(mux)

	tlsConfig := &tls.Config{
		NextProtos: []string{"http/1.1"}, // disable h2
		GetCertificate: func(chi *tls.ClientHelloInfo) (*tls.Certificate, error) {
			return GetOrCreateCert(chi, "*.xoyo.games")
		},
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}

	ln := NewChanListener()
	defer ln.Close()
	go internalSdkServer(ln, tlsConfig, hdr)

	go startSdkServer(tlsConfig, *sdkAddr13, hdr)
	go startSdkServer(tlsConfig, *sdkAddr18, hdr)
	go startSdkServer(tlsConfig, *sdkAddr31, hdr)

	// socks5
	listener, err := net.Listen("tcp", *localAddr)
	if err != nil {
		log.Fatal("listen error: ", err)
	}
	Vf(2, "[socks5]listening on %v\n", *localAddr)
	px := NewProxyServer("[socks5]", nil, ln)
	px.Start(listener)
}

func startSdkServer(tlsConfig *tls.Config, bind string, mux http.Handler) {
	srv := &http.Server{
		ReadTimeout:  time.Duration(*readTimeout) * time.Second,
		WriteTimeout: time.Duration(*writeTimeout) * time.Second,
		Handler:      mux,
	}
	listener, err := tls.Listen("tcp", bind, tlsConfig)
	if err != nil {
		log.Fatal("[tls]listen error: ", err, bind)
	}
	defer listener.Close()
	Vf(2, "[tls]listening %v \n", bind)

	if err := srv.Serve(listener); err != nil {
		log.Printf("[tls]Serve error: %v", err)
	}
}

func internalSdkServer(ln net.Listener, tlsConfig *tls.Config, mux http.Handler) {
	defer ln.Close()
	tlsLn := tls.NewListener(ln, tlsConfig)
	srv := &http.Server{
		ReadTimeout:  time.Duration(*readTimeout) * time.Second,
		WriteTimeout: time.Duration(*writeTimeout) * time.Second,
		Handler:      mux,
	}
	if err := srv.Serve(tlsLn); err != nil {
		log.Printf("[sdk]Serve error: %v", err)
	}
}

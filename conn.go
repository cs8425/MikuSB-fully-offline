package main

import (
	"io"
	"net"
	"sync"
)

type ChanListener struct {
	ch     chan net.Conn
	closeC chan struct{}

	mu     sync.Mutex
	closed bool
}

func NewChanListener() *ChanListener {
	return &ChanListener{
		ch:     make(chan net.Conn),
		closeC: make(chan struct{}),
	}
}

func (l *ChanListener) Push(conn net.Conn) error {
	l.mu.Lock()
	closed := l.closed
	l.mu.Unlock()

	if closed {
		return net.ErrClosed
	}

	select {
	case l.ch <- conn:
		return nil

	case <-l.closeC:
		return net.ErrClosed
	}
}

// net.Listener interface
func (l *ChanListener) Accept() (net.Conn, error) {
	select {
	case conn := <-l.ch:
		if conn == nil {
			return nil, io.EOF
		}
		return conn, nil

	case <-l.closeC:
		return nil, net.ErrClosed
	}
}

func (l *ChanListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	l.closed = true
	close(l.closeC)

	return nil
}

func (l *ChanListener) Addr() net.Addr {
	return dummyAddr("chan-listener")
}

type dummyAddr string

func (d dummyAddr) Network() string { return string(d) }

func (d dummyAddr) String() string { return string(d) }

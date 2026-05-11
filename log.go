package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	Verbosity = flag.Int("v", 3, "verbosity")
)

// log
func Vf(level int, format string, v ...interface{}) {
	if level <= *Verbosity {
		log.Printf(format, v...)
	}
}
func V(level int, v ...interface{}) {
	if level <= *Verbosity {
		log.Print(v...)
	}
}
func Vln(level int, v ...interface{}) {
	if level <= *Verbosity {
		log.Println(v...)
	}
}

type Event struct {
	Event     string `json:"ev"`
	ConnID    string `json:"cid"` // snowflake id
	Direction string `json:"dir,omitempty"`

	Timestamp int64 `json:"ts"`

	ClientAddr   string `json:"saddr,omitempty"`
	UpstreamAddr string `json:"daddr,omitempty"`  // ip & port after connect
	TargetAddr   string `json:"target,omitempty"` // the backend should connect to

	Context string `json:"ctx,omitempty"`
	Error   string `json:"err,omitempty"`

	HTTP   *HTTPEvent   `json:"http,omitempty"`
	Binary *BinaryEvent `json:"bin,omitempty"`
}

type HTTPEvent struct {
	Method  string              `json:"method,omitempty"`
	URL     string              `json:"url,omitempty"`
	Headers map[string][]string `json:"headers,omitempty"`

	Body   string `json:"body,omitempty"`
	Status int    `json:"status,omitempty"`
}

type BinaryEvent struct {
	Seq     uint64 `json:"s"`
	Type    uint16 `json:"t"`
	Payload []byte `json:"p"`
}

func (ev *Event) SetConnID(cid string) *Event {
	ev.ConnID = cid
	return ev
}

func (ev *Event) SetClientAddr(addr string) *Event {
	ev.ClientAddr = addr
	return ev
}

func (ev *Event) SetUpstreamAddr(addr string) *Event {
	ev.UpstreamAddr = addr
	return ev
}

func (ev *Event) SetTargetAddr(addr string) *Event {
	ev.TargetAddr = addr
	return ev
}

func (ev *Event) SetError(err string) *Event {
	ev.Error = err
	return ev
}

func (ev *Event) SetContext(ctx string) *Event {
	ev.Context = ctx
	return ev
}

func NewEvent(ev string) *Event {
	return &Event{
		Event:     ev,
		Timestamp: time.Now().UnixMilli(),
	}
}

type Logger struct {
	ch chan Event
	wg sync.WaitGroup
	f  *os.File
}

func NewLogger(path string, appendTime bool) *Logger {
	if appendTime {
		now := time.Now()
		// yyyyMMDD-HHMMSS
		formatted := now.Format("20060102-150405")
		ext := filepath.Ext(path)
		path, _ = strings.CutSuffix(path, ext)
		path = fmt.Sprintf("%v-%v%v", path, formatted, ext)
	}

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}

	l := &Logger{
		ch: make(chan Event, 4096),
		f:  f,
	}

	l.wg.Go(func() {
		enc := json.NewEncoder(f)
		for ev := range l.ch {
			if err := enc.Encode(ev); err != nil {
				log.Println("encode:", err)
			}
		}
	})

	return l
}

func (l *Logger) Log(ev *Event) {
	select {
	case l.ch <- *ev:
	default:
		log.Println("logger queue full")
	}
}

func (l *Logger) LogNow(ev Event) {
	ev.Timestamp = time.Now().UnixMilli()
	select {
	case l.ch <- ev:
	default:
		log.Println("logger queue full")
	}
}

func (l *Logger) Close() {
	close(l.ch)
	l.wg.Wait()
	_ = l.f.Close()
}

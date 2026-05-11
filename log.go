package main

import (
	"flag"
	"log"
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

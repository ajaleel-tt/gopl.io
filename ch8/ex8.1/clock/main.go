// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Clock is the clock2 server (GOPL §8.2) modified for Exercise 8.1:
//   - it accepts a -port flag so several instances can run side by side, and
//   - it honors the TZ environment variable, letting each instance advertise
//     a different (possibly fake) time zone.
//
// Example:
//
//	TZ=America/New_York go run . -port 8010
//	TZ=Asia/Tokyo       go run . -port 8020
package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	port := flag.String("port", "8000", "TCP port to listen on")
	flag.Parse()

	// LoadLocation("") would return UTC, so fall back to Local explicitly.
	loc := time.Local
	if tz := os.Getenv("TZ"); tz != "" {
		l, err := time.LoadLocation(tz)
		if err != nil {
			log.Fatalf("clock: bad TZ %q: %v", tz, err)
		}
		loc = l
	}

	addr := "localhost:" + *port
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("clock serving %s (%s)", addr, loc)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err) // e.g., connection aborted
			continue
		}
		go handleConn(conn, loc) // handle connections concurrently
	}
}

func handleConn(c net.Conn, loc *time.Location) {
	defer c.Close()
	for {
		_, err := io.WriteString(c, time.Now().In(loc).Format("15:04:05\n"))
		if err != nil {
			return // e.g., client disconnected
		}
		time.Sleep(1 * time.Second)
	}
}

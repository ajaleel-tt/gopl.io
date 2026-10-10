// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Reverb (Exercise 8.4) is the concurrent echo server reverb2, modified so each
// connection tracks its active echo goroutines with a sync.WaitGroup. When the
// client closes its write half (EOF on the server's read side) AND every echo
// goroutine has finished, the server closes its OWN write half (CloseWrite, per
// Exercise 8.3) instead of slamming the whole connection shut.
//
// Why reverb2 needed this: reverb2 runs each shout's echo in its own goroutine,
// then calls c.Close() the instant input ends — tearing down the connection
// while those goroutines are still mid-echo, so the Exercise 8.3 client loses
// the trailing echoes. The WaitGroup gives the connection a single "all echoes
// done" moment at which the half-close is finally safe.
//
// A -port flag is added so several instances can run off the book's hard-coded
// :8000 (useful when that port is taken).
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

func echo(c net.Conn, shout string, delay time.Duration) {
	fmt.Fprintln(c, "\t", strings.ToUpper(shout))
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", shout)
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", strings.ToLower(shout))
}

func handleConn(c net.Conn) {
	defer c.Close() // fully release the fd once everything below is done

	var wg sync.WaitGroup // counts active echo goroutines for THIS connection
	input := bufio.NewScanner(c)
	for input.Scan() {
		wg.Add(1) // before launching, so Wait can never observe a false zero
		go func(shout string) {
			defer wg.Done()
			echo(c, shout, 1*time.Second)
		}(input.Text())
	}
	// input.Scan returned false: the client closed its write half (EOF).
	// Wait for every in-flight echo to finish writing...
	wg.Wait()

	// ...then close only our write half, so the client's read loop sees EOF
	// and stops — after it has received every echo. conn is a *net.TCPConn.
	if cw, ok := c.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
}

func main() {
	port := flag.String("port", "8000", "TCP port to listen on")
	flag.Parse()

	addr := "localhost:" + *port
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("reverb (ex8.4) listening on %s", addr)
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Print(err) // e.g., connection aborted
			continue
		}
		go handleConn(conn)
	}
}

// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Reverb (Exercise 8.8) is the §8.3 concurrent echo server with an idle
// timeout: a client that shouts nothing for -timeout (default 10s) is
// disconnected.
//
// The read loop uses a blocking bufio.Scanner, which can't be select-ed on
// directly. So a reader goroutine turns each scanned line into a channel send,
// and the main loop uses select to wait on either "a shout arrived" or the
// timeout. A new time.After is created each iteration, so every shout resets
// the clock — making it an *idle* timeout rather than a total one.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

func echo(c net.Conn, shout string, delay time.Duration) {
	fmt.Fprintln(c, "\t", strings.ToUpper(shout))
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", shout)
	time.Sleep(delay)
	fmt.Fprintln(c, "\t", strings.ToLower(shout))
}

func handleConn(c net.Conn, timeout time.Duration) {
	defer c.Close()

	// Reader goroutine: converts blocking Scan into channel sends so the main
	// loop can select on input arrival. Closing `input` signals EOF/disconnect.
	input := make(chan string)
	go func() {
		scan := bufio.NewScanner(c)
		for scan.Scan() {
			input <- scan.Text()
		}
		close(input)
	}()

	for {
		select {
		case shout, ok := <-input:
			if !ok {
				return // client hung up on its own
			}
			go echo(c, shout, 1*time.Second)
		case <-time.After(timeout):
			log.Printf("disconnecting idle client %s", c.RemoteAddr())
			return // defer c.Close() drops the connection
		}
	}
}

// serve accepts connections until the listener is closed, one goroutine each.
func serve(l net.Listener, timeout time.Duration) {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Print(err)
			continue
		}
		go handleConn(conn, timeout)
	}
}

func main() {
	port := flag.String("port", "8000", "TCP port to listen on")
	timeout := flag.Duration("timeout", 10*time.Second, "disconnect a client idle this long")
	flag.Parse()

	l, err := net.Listen("tcp", "localhost:"+*port)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("reverb (ex8.8) on %s, idle timeout %s", l.Addr(), *timeout)
	serve(l, *timeout)
}

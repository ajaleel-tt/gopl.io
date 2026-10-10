// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Netcat (Exercise 8.3) is the netcat3 read/write client, modified to close
// only the WRITE half of the TCP connection when standard input ends.
//
// The original netcat3 calls conn.Close() once stdin is exhausted, which tears
// down BOTH directions at once and truncates any echoes the server hasn't sent
// yet. Closing only the write half (CloseWrite) sends the server an EOF on the
// read side — so its input.Scan() loop ends — while leaving our read half open
// to receive the server's final, delayed echoes. The server closes its end
// when done, which gives us EOF and unblocks <-done.
//
// The book notes conn's concrete type is *net.TCPConn; we use an ad-hoc
// interface assertion instead, which is cleaner and also covers *net.UnixConn.
//
// An optional command-line argument overrides the server address (default
// localhost:8000) so the program can be pointed at a server on any port.
package main

import (
	"io"
	"log"
	"net"
	"os"
)

func main() {
	addr := "localhost:8000"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		io.Copy(os.Stdout, conn) // NOTE: ignoring errors
		log.Println("done")
		done <- struct{}{} // signal the main goroutine
	}()
	mustCopy(conn, os.Stdin)

	// Half-close: signal EOF to the server without abandoning the read half.
	// Equivalent to conn.(*net.TCPConn).CloseWrite() for a TCP connection.
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	} else {
		conn.Close() // fallback for connection types without a write half
	}

	<-done // wait for background goroutine to finish
}

func mustCopy(dst io.Writer, src io.Reader) {
	if _, err := io.Copy(dst, src); err != nil {
		log.Fatal(err)
	}
}

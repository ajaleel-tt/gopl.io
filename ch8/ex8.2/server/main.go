// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Server (Exercise 8.2) is a concurrent FTP server. It implements enough of
// RFC 959, plus the RFC 2428 extensions (EPSV/EPRT), to interoperate with
// standard clients: the ftp command, curl (ftp://), and Python's ftplib.
//
// Each control connection is served by its own goroutine holding its own
// session state — current directory, transfer type, and data-connection
// setup — so concurrent clients never interfere with one another. That
// per-goroutine confinement is the lesson of GOPL chapter 8: the only thing
// the sessions share is the read-only filesystem, so no locks are needed.
//
// All paths are sandboxed to the directory named by -root; clients cannot
// escape it via ".." or absolute paths, and active-mode (PORT/EPRT) transfers
// may only target the client's own address (no FTP-bounce). Authentication is
// accept-any (anonymous-style) — this is a teaching server, not a hardened one.
//
// Usage:
//
//	go run ./ch8/ex8.2/server -root /some/dir -port 8021 -v
//	curl ftp://127.0.0.1:8021/
//	python3 -c 'from ftplib import FTP; ...'
package main

import (
	"errors"
	"flag"
	"log"
	"net"
	"os"
	"path/filepath"
)

func main() {
	host := flag.String("host", "127.0.0.1", "host/IP to listen on")
	port := flag.String("port", "8021", "control port to listen on")
	rootFlag := flag.String("root", ".", "directory to serve")
	verbose := flag.Bool("v", false, "log each command and reply")
	flag.Parse()

	root, err := filepath.Abs(*rootFlag)
	if err != nil {
		log.Fatal(err)
	}
	// Canonicalize (e.g. /tmp -> /private/tmp on macOS) so the containment
	// checks compare like with like.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		log.Fatalf("root %q is not a directory", root)
	}

	addr := net.JoinHostPort(*host, *port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("FTP server listening on %s, serving %s", addr, root)
	acceptLoop(listener, root, *verbose)
}

// acceptLoop serves connections until the listener is closed, spawning one
// goroutine per client. It is the shared entry point for main and the tests.
func acceptLoop(listener net.Listener, root string, verbose bool) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return // listener closed (e.g., test teardown)
			}
			log.Print(err) // transient accept error
			continue
		}
		s := &session{
			conn:    conn,
			root:    root,
			cwd:     root,
			verbose: verbose,
		}
		go s.serve() // one goroutine per client
	}
}

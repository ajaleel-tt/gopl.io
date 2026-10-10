// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

// startServer runs the echo server on a random loopback port with the given
// idle timeout and returns its address.
func startServer(t *testing.T, timeout time.Duration) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go serve(l, timeout)
	t.Cleanup(func() { l.Close() })
	return l.Addr().String()
}

// TestIdleClientDisconnected: a client that shouts nothing is dropped after
// roughly the timeout.
func TestIdleClientDisconnected(t *testing.T) {
	const timeout = 150 * time.Millisecond
	conn, err := net.Dial("tcp", startServer(t, timeout))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Send nothing. The read blocks until the server closes the connection.
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	start := time.Now()
	n, err := conn.Read(make([]byte, 16))
	elapsed := time.Since(start)

	if err != io.EOF {
		t.Fatalf("expected EOF from server closing idle client, got n=%d err=%v", n, err)
	}
	if elapsed < timeout {
		t.Errorf("closed too early: %v < timeout %v", elapsed, timeout)
	}
	if elapsed > timeout+time.Second {
		t.Errorf("closed too late: %v", elapsed)
	}
}

// TestActiveClientStaysConnected: shouting within the timeout keeps resetting
// it, so an active client is not dropped; once it goes silent, it is.
func TestActiveClientStaysConnected(t *testing.T) {
	const timeout = 200 * time.Millisecond
	conn, err := net.Dial("tcp", startServer(t, timeout))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Reader goroutine reports when the server closes the connection.
	closed := make(chan time.Time, 1)
	go func() {
		io.Copy(io.Discard, conn) // drains echoes until EOF
		closed <- time.Now()
	}()

	// Phase 1: shout every 50ms for 600ms (3x the timeout). If activity resets
	// the timer, the server must NOT close during this window.
	for i := 0; i < 12; i++ {
		fmt.Fprintln(conn, "hello")
		time.Sleep(50 * time.Millisecond)
	}
	select {
	case <-closed:
		t.Fatal("server dropped an ACTIVE client — the idle timer isn't resetting")
	default: // still connected after 600ms of shouting: good
	}

	// Phase 2: go silent. The server should disconnect within ~timeout.
	start := time.Now()
	select {
	case tc := <-closed:
		if idle := tc.Sub(start); idle > timeout+time.Second {
			t.Errorf("disconnected too late after going idle: %v", idle)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server never disconnected the now-idle client")
	}
}

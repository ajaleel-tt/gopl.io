package main

import (
	"bufio"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The broadcaster uses package-level channels, as in the book, so there is
// one chat room per test binary. TestIdleDisconnect drives it through a
// single scripted scenario and expects to start with an empty room, so it
// can't be repeated within one process (go test -count=N with N > 1).

type testClient struct {
	t    *testing.T
	conn net.Conn
	in   *bufio.Scanner
	name string
}

func join(t *testing.T, addr string) *testClient {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	c := &testClient{t: t, conn: conn, in: bufio.NewScanner(conn)}
	greeting := c.read()
	c.name = strings.TrimPrefix(greeting, "You are ")
	c.read() // roster
	return c
}

func (c *testClient) read() string {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if !c.in.Scan() {
		c.t.Fatalf("%s read: %v", c.name, c.in.Err())
	}
	return c.in.Text()
}

func (c *testClient) expect(want string) {
	c.t.Helper()
	if got := c.read(); got != want {
		c.t.Fatalf("%s got %q, want %q", c.name, got, want)
	}
}

func (c *testClient) say(line string) {
	c.t.Helper()
	if _, err := c.conn.Write([]byte(line + "\n")); err != nil {
		c.t.Fatal(err)
	}
}

// waitDisconnected reads until the server closes the connection and returns
// how long that took. A read timeout (rather than EOF) is a failure.
func (c *testClient) waitDisconnected(start time.Time) time.Duration {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for c.in.Scan() {
		// discard chat traffic that arrives before the disconnect
	}
	if err := c.in.Err(); err != nil {
		c.t.Fatalf("%s: expected server to close connection, got %v", c.name, err)
	}
	return time.Since(start)
}

func TestIdleDisconnect(t *testing.T) {
	idleTimeout = 300 * time.Millisecond
	baseline := runtime.NumGoroutine()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go broadcaster()
	go acceptLoop(ln)
	addr := ln.Addr().String()

	a := join(t, addr)
	b := join(t, addr)
	a.expect(b.name + " has arrived")
	start := time.Now()

	// b keeps talking more often than the timeout; a says nothing. Partway
	// through, b should see a get dropped, while b itself stays connected.
	aLeft := false
	var bLastSaid time.Time
	for i := 0; i < 6; i++ {
		b.say("ping")
		bLastSaid = time.Now()
		line := b.read()
		if line == a.name+" has left" {
			aLeft = true
			line = b.read()
		}
		if want := b.name + ": ping"; line != want {
			t.Fatalf("b got %q, want %q", line, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !aLeft {
		b.expect(a.name + " has left")
	}
	if d := a.waitDisconnected(start); d < idleTimeout {
		t.Errorf("a disconnected after %v, before the %v timeout", d, idleTimeout)
	}

	// Now b goes quiet and should be dropped too, timed from its last message.
	if d := b.waitDisconnected(bLastSaid); d < idleTimeout {
		t.Errorf("b disconnected after %v, before the %v timeout", d, idleTimeout)
	}

	// Every per-client goroutine (handleConn, clientWriter, idleWatchdog)
	// should have exited. Only broadcaster and acceptLoop remain.
	ln.Close() // stops acceptLoop
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline+1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline+1 {
		t.Errorf("goroutines: %d, want %d (baseline + broadcaster)", n, baseline+1)
	}
}

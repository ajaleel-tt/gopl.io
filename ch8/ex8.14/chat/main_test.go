package main

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// The broadcaster uses package-level channels, as in the book, so there is
// one chat room per test binary. TestNames drives it through a single
// scripted scenario and expects to start with an empty room, so it can't be
// repeated within one process (go test -count=N with N > 1).

const prompt = "Enter your name:"

type testClient struct {
	t    *testing.T
	conn net.Conn
	in   *bufio.Scanner
	name string
}

func dial(t *testing.T, addr string) *testClient {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	return &testClient{t: t, conn: conn, in: bufio.NewScanner(conn), name: conn.LocalAddr().String()}
}

// join dials, answers the name prompt and checks the welcome lines.
func join(t *testing.T, addr, name, wantRoster string) *testClient {
	t.Helper()
	c := dial(t, addr)
	c.expect(prompt)
	c.say(name)
	c.name = name
	c.expect("You are " + name)
	c.expect(wantRoster)
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

func (c *testClient) expectClosed() {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	for c.in.Scan() {
	}
	if err := c.in.Err(); err != nil {
		c.t.Fatalf("%s: expected server to close connection, got %v", c.name, err)
	}
}

// syncBuffer is a bytes.Buffer safe to share between the server goroutines
// writing log lines and the test reading them.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestNames(t *testing.T) {
	idleTimeout = 400 * time.Millisecond
	var logs syncBuffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	baseline := runtime.NumGoroutine()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go broadcaster()
	go acceptLoop(ln)
	addr := ln.Addr().String()

	alice := join(t, addr, "alice", "You are the only one here.")

	// A blank name is rejected with another prompt; whitespace is trimmed.
	bob := dial(t, addr)
	bob.expect(prompt)
	bob.say("   ")
	bob.expect(prompt)
	bob.say("  bob  ")
	bob.name = "bob"
	bob.expect("You are bob")
	bob.expect("Current clients: alice")
	alice.expect("bob has arrived")

	// Messages are prefixed with the chosen name, not the address.
	alice.say("hi bob")
	alice.expect("alice: hi bob")
	bob.expect("alice: hi bob")

	// A client that hangs up at the prompt never enters the room: nobody
	// hears it arrive or leave, so alice's next line is bob's message.
	quitter := dial(t, addr)
	quitter.expect(prompt)
	quitter.conn.Close()
	time.Sleep(50 * time.Millisecond)
	bob.say("hello")
	alice.expect("bob: hello")
	bob.expect("bob: hello")

	// A client that never answers the prompt is dropped by the idle timer.
	silent := dial(t, addr)
	silent.expect(prompt)
	start := time.Now()
	silent.expectClosed()
	if d := time.Since(start); d < idleTimeout/2 {
		t.Errorf("silent client dropped after %v; want about %v", d, idleTimeout)
	}
	if want := silent.name + ": idle for"; !strings.Contains(logs.String(), want) {
		t.Errorf("log missing %q; got:\n%s", want, logs.String())
	}

	// alice and bob go idle and are dropped too. Afterwards every
	// per-client goroutine must be gone, including the quitter's
	// clientWriter, which only handleConn could close.
	alice.expectClosed()
	bob.expectClosed()
	ln.Close()
	deadline := time.Now().Add(2 * time.Second)
	for runtime.NumGoroutine() > baseline+1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline+1 {
		t.Errorf("goroutines: %d, want %d (baseline + broadcaster)", n, baseline+1)
	}
}

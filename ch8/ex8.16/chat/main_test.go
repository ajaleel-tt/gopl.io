package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// The server uses package-level channels, as in the book, and done can only
// be closed once, so there is one server per test binary. TestServer drives
// it through a single scripted scenario that ends in shutdown, so it can't
// be repeated within one process (go test -count=N with N > 1).

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

func join(t *testing.T, addr, name, wantRoster string) *testClient {
	t.Helper()
	c := dial(t, addr)
	c.expect(prompt)
	c.enter(name, wantRoster)
	return c
}

func (c *testClient) enter(name, wantRoster string) {
	c.t.Helper()
	c.say(name)
	c.name = name
	c.expect("You are " + name)
	c.expect(wantRoster)
}

func (c *testClient) read(timeout time.Duration) (string, bool) {
	c.conn.SetReadDeadline(time.Now().Add(timeout))
	if !c.in.Scan() {
		return "", false
	}
	return c.in.Text(), true
}

func (c *testClient) expect(want string) {
	c.t.Helper()
	got, ok := c.read(2 * time.Second)
	if !ok {
		c.t.Fatalf("%s read: %v", c.name, c.in.Err())
	}
	if got != want {
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
	for {
		line, ok := c.read(2 * time.Second)
		if !ok {
			break
		}
		c.t.Errorf("%s: unexpected %q before close", c.name, line)
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

// waitFor polls cond every 10ms until it holds or timeout passes.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(10 * time.Millisecond)
	}
	return true
}

func TestServer(t *testing.T) {
	idleTimeout = time.Minute // not under test here
	maxClients = 3
	statsInterval = 50 * time.Millisecond
	writeGrace = 300 * time.Millisecond
	var logs syncBuffer
	log.SetOutput(&logs)
	defer log.SetOutput(os.Stderr)
	baseline := runtime.NumGoroutine()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	serveDone := make(chan struct{})
	go func() {
		serve(ln)
		close(serveDone)
	}()

	// --- Semaphore: at most maxClients at once. ---
	alice := join(t, addr, "alice", "You are the only one here.")
	bob := join(t, addr, "bob", "Current clients: alice")
	alice.expect("bob has arrived")
	stuck := join(t, addr, "stuck", "Current clients: alice, bob") // never reads again
	alice.expect("stuck has arrived")
	bob.expect("stuck has arrived")

	full := dial(t, addr)
	full.expect(fullNotice)
	full.expectClosed()

	// bob leaving frees a token. It's released when bob's handleConn
	// returns, just after "bob has left", so retry until it's free.
	bob.conn.Close()
	alice.expect("bob has left")
	var late *testClient
	if !waitFor(2*time.Second, func() bool {
		c := dial(t, addr)
		if line, _ := c.read(2 * time.Second); line != prompt {
			c.conn.Close() // still full
			return false
		}
		late = c
		return true
	}) {
		t.Fatal("token never released after bob left")
	}
	late.enter("late", "Current clients: alice, stuck")
	alice.expect("late has arrived")

	// --- Flood with a client that never reads (8.15), counted in stats. ---
	// alice floods ~20MB and stops reading too; late keeps reading.
	const n = 20000
	payload := strings.Repeat("x", 1000)
	sendErr := make(chan error, 1)
	go func() {
		w := bufio.NewWriter(alice.conn)
		for i := 0; i < n; i++ {
			fmt.Fprintf(w, "msg %d %s\n", i, payload)
		}
		if err := w.Flush(); err != nil {
			sendErr <- err
			return
		}
		time.Sleep(300 * time.Millisecond) // let late catch up
		_, err := fmt.Fprintln(alice.conn, "done")
		sendErr <- err
	}()
	for {
		line, ok := late.read(5 * time.Second)
		if !ok {
			t.Fatalf("late stalled during flood: %v", late.in.Err())
		}
		if line == "alice: done" {
			break
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatal(err)
	}

	// --- Stats: the ticker reports clients and drops. ---
	nonzero := regexp.MustCompile(`stats: 3 clients connected, [1-9][0-9]* messages dropped`)
	if !waitFor(time.Second, func() bool { return nonzero.MatchString(logs.String()) }) {
		t.Errorf("no stats line with 3 clients and drops; logs:\n%s", logs.String())
	}

	// --- Graceful shutdown (what main does on Ctrl+C). ---
	start := time.Now()
	close(done)

	// late is reading, so it gets the goodbye, then the connection closes.
	late.expect(shutdownNotice)
	late.expectClosed()

	// serve must finish even though stuck and alice aren't reading: their
	// writers get writeGrace, then are cut off.
	select {
	case <-serveDone:
		t.Logf("serve returned %v after shutdown began", time.Since(start).Round(time.Millisecond))
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after shutdown")
	}

	// The listener is closed, so new clients are refused outright.
	if c, err := net.Dial("tcp", addr); err == nil {
		c.Close()
		t.Error("dial after shutdown succeeded")
	}

	// Every goroutine the server started has exited, broadcaster included.
	for _, c := range []*testClient{alice, stuck, late, full} {
		c.conn.Close()
	}
	if !waitFor(5*time.Second, func() bool { return runtime.NumGoroutine() <= baseline }) {
		t.Errorf("goroutines: %d, want %d", runtime.NumGoroutine(), baseline)
	}
}

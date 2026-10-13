package main

import (
	"bufio"
	"fmt"
	"net"
	"runtime"
	"strings"
	"testing"
	"time"
)

// The broadcaster uses package-level channels, as in the book, so there is
// one chat room per test binary. TestSlowClient drives it through a single
// scripted scenario and expects to start with an empty room, so it can't be
// repeated within one process (go test -count=N with N > 1).

const prompt = "Enter your name:"

type testClient struct {
	t    *testing.T
	conn net.Conn
	in   *bufio.Scanner
	name string
}

func join(t *testing.T, addr, name, wantRoster string) *testClient {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	c := &testClient{t: t, conn: conn, in: bufio.NewScanner(conn), name: name}
	c.expect(prompt)
	c.say(name)
	c.expect("You are " + name)
	c.expect(wantRoster)
	return c
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

func TestSlowClient(t *testing.T) {
	idleTimeout = time.Minute // not under test here
	baseline := runtime.NumGoroutine()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go broadcaster()
	go acceptLoop(ln)
	addr := ln.Addr().String()

	// stuck joins and then never reads another byte.
	stuck := join(t, addr, "stuck", "You are the only one here.")
	reader := join(t, addr, "reader", "Current clients: stuck")
	talker := join(t, addr, "talker", "Current clients: reader, stuck")
	reader.expect("talker has arrived")

	// talker floods the room with ~20MB, far more than the kernel will
	// buffer for stuck's connection, then pauses so reader can catch up and
	// sends a final line. (talker never reads either, which is fine: it's
	// just a second slow client.)
	const n = 20000
	payload := strings.Repeat("x", 1000)
	sendErr := make(chan error, 1)
	go func() {
		w := bufio.NewWriter(talker.conn)
		for i := 0; i < n; i++ {
			if _, err := fmt.Fprintf(w, "msg %d %s\n", i, payload); err != nil {
				sendErr <- err
				return
			}
		}
		if err := w.Flush(); err != nil {
			sendErr <- err
			return
		}
		time.Sleep(300 * time.Millisecond)
		_, err := fmt.Fprintln(talker.conn, "done")
		sendErr <- err
	}()

	// With a broadcaster that waits on every client, this stalls once
	// stuck's buffers fill: no read returns within the timeout.
	got := 0
	for {
		line, ok := reader.read(5 * time.Second)
		if !ok {
			t.Fatalf("reader stalled after %d of %d messages: %v", got, n, reader.in.Err())
		}
		if line == "talker: done" {
			break
		}
		if strings.HasPrefix(line, "talker: msg ") {
			got++
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatal(err)
	}

	// stuck finally reads whatever reached it. Most of the flood must have
	// been skipped for it, or the test didn't exercise a slow client.
	stuckGot := 0
	for {
		line, ok := stuck.read(500 * time.Millisecond)
		if !ok {
			break
		}
		if strings.HasPrefix(line, "talker: msg ") {
			stuckGot++
		}
	}
	t.Logf("of %d messages: reader got %d, stuck got %d", n, got, stuckGot)
	if stuckGot >= n {
		t.Errorf("stuck received all %d messages; nothing was skipped", n)
	}

	// Everyone leaves; every per-client goroutine must exit, including
	// writers that were blocked writing to a client that wasn't reading.
	for _, c := range []*testClient{stuck, reader, talker} {
		c.conn.Close()
	}
	ln.Close()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline+1 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if n := runtime.NumGoroutine(); n > baseline+1 {
		t.Errorf("goroutines: %d, want %d (baseline + broadcaster)", n, baseline+1)
	}
}

package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// Each test gets its own server: there is no package-level state, so tests
// are independent and can be repeated with -count.

type testServer struct {
	addr     string
	logs     *syncBuffer
	shutdown context.CancelFunc
	done     chan struct{} // closed when serve returns
}

// startServer runs a server on a random port. Shutdown and a goroutine-leak
// check happen at cleanup if the test hasn't shut it down itself.
func startServer(t *testing.T, cfg Config) *testServer {
	t.Helper()
	baseline := runtime.NumGoroutine()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	logs := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	ts := &testServer{addr: ln.Addr().String(), logs: logs, shutdown: cancel, done: make(chan struct{})}
	srv := newServer(cfg, slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	go func() {
		srv.serve(ctx, ln)
		close(ts.done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-ts.done:
		case <-time.After(5 * time.Second):
			t.Fatal("serve did not return after shutdown")
		}
		if !waitFor(5*time.Second, func() bool { return runtime.NumGoroutine() <= baseline }) {
			t.Errorf("goroutines: %d, want %d", runtime.NumGoroutine(), baseline)
		}
	})
	return ts
}

// testConfig is defaultConfig with a short write grace, so shutting down
// with a client that isn't reading doesn't slow tests down.
func testConfig() Config {
	cfg := defaultConfig
	cfg.WriteGrace = 300 * time.Millisecond
	return cfg
}

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
	t.Cleanup(func() { conn.Close() })
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

func TestNamesAndRoster(t *testing.T) {
	ts := startServer(t, testConfig())

	alice := join(t, ts.addr, "alice", "You are the only one here.")

	bob := dial(t, ts.addr)
	bob.expect(prompt)
	bob.say("   ") // blank names are re-prompted
	bob.expect(prompt)
	bob.say("  bob  ") // whitespace is trimmed
	bob.name = "bob"
	bob.expect("You are bob")
	bob.expect("Current clients: alice")
	alice.expect("bob has arrived")

	alice.say("hi")
	alice.expect("alice: hi")
	bob.expect("alice: hi")

	bob.conn.Close()
	alice.expect("bob has left")
}

func TestIdleDisconnect(t *testing.T) {
	cfg := testConfig()
	cfg.IdleTimeout = 300 * time.Millisecond
	ts := startServer(t, cfg)

	// A client that never answers the name prompt is dropped too.
	silent := dial(t, ts.addr)
	silent.expect(prompt)

	alice := join(t, ts.addr, "alice", "You are the only one here.")
	bob := join(t, ts.addr, "bob", "Current clients: alice")
	alice.expect("bob has arrived")

	// bob keeps talking; alice goes quiet and is dropped.
	aliceLeft := false
	for range 6 {
		bob.say("ping")
		line, _ := bob.read(2 * time.Second)
		if line == "alice has left" {
			aliceLeft = true
			line, _ = bob.read(2 * time.Second)
		}
		if line != "bob: ping" {
			t.Fatalf("bob got %q, want %q", line, "bob: ping")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !aliceLeft {
		bob.expect("alice has left")
	}
	silent.expectClosed()

	// The deadline-based timeout knows the client's name, so the log can
	// say who was dropped (the book-era watchdog only had the address).
	logs := ts.logs.String()
	if !strings.Contains(logs, `msg="idle disconnect"`) || !strings.Contains(logs, "name=alice") {
		t.Errorf("no idle disconnect logged for alice; logs:\n%s", logs)
	}
}

func TestCapacity(t *testing.T) {
	cfg := testConfig()
	cfg.MaxClients = 2
	ts := startServer(t, cfg)

	alice := join(t, ts.addr, "alice", "You are the only one here.")
	bob := join(t, ts.addr, "bob", "Current clients: alice")
	alice.expect("bob has arrived")

	full := dial(t, ts.addr)
	full.expect(fullNotice)
	full.expectClosed()

	// bob leaving frees a slot, released when bob's handleConn returns.
	bob.conn.Close()
	alice.expect("bob has left")
	var carol *testClient
	if !waitFor(2*time.Second, func() bool {
		c := dial(t, ts.addr)
		if line, _ := c.read(2 * time.Second); line != prompt {
			return false // still full
		}
		carol = c
		return true
	}) {
		t.Fatal("slot never released after bob left")
	}
	carol.enter("carol", "Current clients: alice")
}

func TestSlowClient(t *testing.T) {
	cfg := testConfig()
	cfg.StatsInterval = 50 * time.Millisecond
	ts := startServer(t, cfg)

	join(t, ts.addr, "stuck", "You are the only one here.") // never reads again
	reader := join(t, ts.addr, "reader", "Current clients: stuck")
	talker := join(t, ts.addr, "talker", "Current clients: reader, stuck")
	reader.expect("talker has arrived")

	// talker floods ~20MB (and never reads either); reader must keep up.
	const n = 20000
	payload := strings.Repeat("x", 1000)
	sendErr := make(chan error, 1)
	go func() {
		w := bufio.NewWriter(talker.conn)
		for i := range n {
			fmt.Fprintf(w, "msg %d %s\n", i, payload)
		}
		if err := w.Flush(); err != nil {
			sendErr <- err
			return
		}
		time.Sleep(300 * time.Millisecond) // let reader catch up
		_, err := fmt.Fprintln(talker.conn, "done")
		sendErr <- err
	}()
	for {
		line, ok := reader.read(5 * time.Second)
		if !ok {
			t.Fatalf("reader stalled: %v", reader.in.Err())
		}
		if line == "talker: done" {
			break
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatal(err)
	}

	nonzero := regexp.MustCompile(`msg=stats clients=3 dropped=[1-9]`)
	if !waitFor(time.Second, func() bool { return nonzero.MatchString(ts.logs.String()) }) {
		t.Errorf("no stats line with 3 clients and drops; logs:\n%s", ts.logs.String())
	}
}

func TestShutdown(t *testing.T) {
	ts := startServer(t, testConfig())

	alice := join(t, ts.addr, "alice", "You are the only one here.")
	bob := join(t, ts.addr, "bob", "Current clients: alice")
	alice.expect("bob has arrived")
	naming := dial(t, ts.addr) // still at the name prompt
	naming.expect(prompt)

	// stuck stops reading and has a flood queued up, so its writer is
	// blocked when shutdown begins.
	stuck := join(t, ts.addr, "stuck", "Current clients: alice, bob")
	alice.expect("stuck has arrived")
	bob.expect("stuck has arrived")
	sendErr := make(chan error, 1)
	go func() {
		w := bufio.NewWriter(bob.conn)
		payload := strings.Repeat("x", 1000)
		for i := range 20000 {
			fmt.Fprintf(w, "msg %d %s\n", i, payload)
		}
		if err := w.Flush(); err != nil {
			sendErr <- err
			return
		}
		// Some of the flood may be skipped even for alice, so finish
		// with a line sent once she's had time to catch up.
		time.Sleep(300 * time.Millisecond)
		_, err := fmt.Fprintln(bob.conn, "done")
		sendErr <- err
	}()
	for { // alice reads throughout, so her own queue ends up empty
		line, ok := alice.read(5 * time.Second)
		if !ok {
			t.Fatalf("alice stalled: %v", alice.in.Err())
		}
		if line == "bob: done" {
			break
		}
	}
	if err := <-sendErr; err != nil {
		t.Fatal(err)
	}
	_ = stuck // its writer is blocked: it never read the flood

	start := time.Now()
	ts.shutdown()

	// Clients that are reading get the goodbye, and no "has left" spam,
	// whether they'd joined or were still choosing a name.
	alice.expect(shutdownNotice)
	alice.expectClosed()
	naming.expect(shutdownNotice)
	naming.expectClosed()

	// serve finishes even though stuck isn't reading: its writer gets
	// WriteGrace, then is cut off.
	select {
	case <-ts.done:
		t.Logf("serve returned %v after shutdown began", time.Since(start).Round(time.Millisecond))
	case <-time.After(3 * time.Second):
		t.Fatal("serve did not return after shutdown")
	}

	if c, err := net.Dial("tcp", ts.addr); err == nil {
		c.Close()
		t.Error("dial after shutdown succeeded")
	}
}

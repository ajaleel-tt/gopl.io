package main

import (
	"bufio"
	"net"
	"sort"
	"strings"
	"testing"
	"time"
)

// The broadcaster uses package-level channels, as in the book, so there is
// one chat room per test binary. TestRoster drives it through a single
// scripted scenario and expects to start with an empty room, so it can't be
// repeated within one process (go test -count=N with N > 1).

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
	if !strings.HasPrefix(greeting, "You are ") {
		t.Fatalf("greeting = %q", greeting)
	}
	c.name = strings.TrimPrefix(greeting, "You are ")
	return c
}

func (c *testClient) read() string {
	c.t.Helper()
	c.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if !c.in.Scan() {
		c.t.Fatalf("read: %v", c.in.Err())
	}
	return c.in.Text()
}

func (c *testClient) expect(want string) {
	c.t.Helper()
	if got := c.read(); got != want {
		c.t.Fatalf("%s got %q, want %q", c.name, got, want)
	}
}

func rosterOf(cs ...*testClient) string {
	var names []string
	for _, c := range cs {
		names = append(names, c.name)
	}
	sort.Strings(names)
	return "Current clients: " + strings.Join(names, ", ")
}

func TestRoster(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go broadcaster()
	go acceptLoop(ln)
	addr := ln.Addr().String()

	a := join(t, addr)
	a.expect("You are the only one here.")

	b := join(t, addr)
	b.expect(rosterOf(a))
	a.expect(b.name + " has arrived")

	c := join(t, addr)
	c.expect(rosterOf(a, b))
	a.expect(c.name + " has arrived")
	b.expect(c.name + " has arrived")

	// b leaves; the roster for the next arrival must no longer include it.
	b.conn.Close()
	a.expect(b.name + " has left")
	c.expect(b.name + " has left")

	d := join(t, addr)
	d.expect(rosterOf(a, c))
	a.expect(d.name + " has arrived")
	c.expect(d.name + " has arrived")

	// Ordinary chat still reaches everyone, including the sender.
	if _, err := a.conn.Write([]byte("hi\n")); err != nil {
		t.Fatal(err)
	}
	for _, x := range []*testClient{a, c, d} {
		x.expect(a.name + ": hi")
	}

	for _, x := range []*testClient{a, c, d} {
		x.conn.Close()
	}
}

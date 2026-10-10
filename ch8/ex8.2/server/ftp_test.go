// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"io"
	"net"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// newServer starts an in-process FTP server on a random loopback port, serving
// a small fixture tree from a temp directory. It returns the control address
// and the (canonical) served root.
func newServer(t *testing.T) (addr, root string) {
	t.Helper()
	root = t.TempDir()
	writeFile(t, filepath.Join(root, "hello.txt"), "hello world\n")
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "nested.txt"), "nested\n")

	// main canonicalizes the root (e.g. macOS /var -> /private/var, which
	// t.TempDir hits); match that so on-disk assertions line up.
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		root = resolved
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go acceptLoop(ln, root, false)
	t.Cleanup(func() { ln.Close() })
	return ln.Addr().String(), root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// testClient is a minimal FTP client for driving the server in tests.
type testClient struct {
	t    *testing.T
	ctrl *textproto.Conn
}

func dial(t *testing.T, addr string) *testClient {
	t.Helper()
	nc, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c := &testClient{t: t, ctrl: textproto.NewConn(nc)}
	c.expect(220) // server greeting
	t.Cleanup(func() { c.ctrl.Close() })
	return c
}

func (c *testClient) login() {
	c.t.Helper()
	c.cmd(331, "USER %s", "tester")
	c.cmd(230, "PASS %s", "secret")
}

// cmd sends a command and asserts the reply code matches want; it returns the
// reply message.
func (c *testClient) cmd(want int, format string, args ...interface{}) string {
	c.t.Helper()
	if err := c.ctrl.PrintfLine(format, args...); err != nil {
		c.t.Fatalf("send %q: %v", fmt.Sprintf(format, args...), err)
	}
	_, msg, err := c.ctrl.ReadResponse(want)
	if err != nil {
		c.t.Fatalf("cmd %q: %v", fmt.Sprintf(format, args...), err)
	}
	return msg
}

// expect reads a reply (sending nothing) and asserts its code.
func (c *testClient) expect(want int) string {
	c.t.Helper()
	_, msg, err := c.ctrl.ReadResponse(want)
	if err != nil {
		c.t.Fatalf("expect %d: %v", want, err)
	}
	return msg
}

// status sends a command and returns the reply code without asserting it, for
// commands expected to fail.
func (c *testClient) status(format string, args ...interface{}) int {
	c.t.Helper()
	if err := c.ctrl.PrintfLine(format, args...); err != nil {
		c.t.Fatalf("send: %v", err)
	}
	code, _, err := c.ctrl.ReadResponse(0) // 0 = don't check the code
	if err != nil {
		c.t.Fatalf("read: %v", err)
	}
	return code
}

// pasv issues PASV and dials the advertised data port.
func (c *testClient) pasv() net.Conn {
	c.t.Helper()
	msg := c.cmd(227, "PASV")
	lo, hi := strings.IndexByte(msg, '('), strings.IndexByte(msg, ')')
	if lo < 0 || hi < 0 {
		c.t.Fatalf("bad PASV reply %q", msg)
	}
	nums := strings.Split(msg[lo+1:hi], ",")
	if len(nums) != 6 {
		c.t.Fatalf("bad PASV reply %q", msg)
	}
	n := make([]int, 6)
	for i, s := range nums {
		n[i], _ = strconv.Atoi(strings.TrimSpace(s))
	}
	ip := fmt.Sprintf("%d.%d.%d.%d", n[0], n[1], n[2], n[3])
	port := n[4]<<8 | n[5]
	dc, err := net.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		c.t.Fatalf("data dial: %v", err)
	}
	return dc
}

// transfer runs a download-style command (LIST/RETR) over a passive data
// connection and returns the data received.
func (c *testClient) transfer(format string, args ...interface{}) []byte {
	c.t.Helper()
	dc := c.pasv()
	if err := c.ctrl.PrintfLine(format, args...); err != nil {
		c.t.Fatalf("send: %v", err)
	}
	c.expect(150)
	data, err := io.ReadAll(dc)
	dc.Close()
	if err != nil {
		c.t.Fatalf("read data: %v", err)
	}
	c.expect(226)
	return data
}

func TestLoginAndPwd(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()
	if msg := c.cmd(257, "PWD"); !strings.Contains(msg, `"/"`) {
		t.Errorf("PWD = %q, want it to contain \"/\"", msg)
	}
}

func TestRequiresLogin(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr) // greeting consumed, but not logged in
	if code := c.status("PWD"); code != 530 {
		t.Errorf("PWD before login: code = %d, want 530", code)
	}
}

func TestRetr(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()
	if got := string(c.transfer("RETR hello.txt")); got != "hello world\n" {
		t.Errorf("RETR hello.txt = %q, want %q", got, "hello world\n")
	}
}

func TestList(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()
	out := string(c.transfer("LIST"))
	for _, want := range []string{"hello.txt", "sub"} {
		if !strings.Contains(out, want) {
			t.Errorf("LIST missing %q in:\n%s", want, out)
		}
	}
}

func TestSize(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()
	c.cmd(200, "TYPE I")
	if msg := strings.TrimSpace(c.cmd(213, "SIZE hello.txt")); msg != "12" {
		t.Errorf("SIZE hello.txt = %q, want 12", msg)
	}
}

func TestStor(t *testing.T) {
	addr, root := newServer(t)
	c := dial(t, addr)
	c.login()

	dc := c.pasv()
	c.ctrl.PrintfLine("STOR up.txt")
	c.expect(150)
	io.WriteString(dc, "uploaded\n")
	dc.Close()
	c.expect(226)

	got, err := os.ReadFile(filepath.Join(root, "up.txt"))
	if err != nil {
		t.Fatalf("reading stored file: %v", err)
	}
	if string(got) != "uploaded\n" {
		t.Errorf("stored contents = %q, want %q", got, "uploaded\n")
	}
}

// TestConcurrentSessionsIsolated is the chapter's point: two simultaneous
// clients keep independent working directories.
func TestConcurrentSessionsIsolated(t *testing.T) {
	addr, _ := newServer(t)
	a := dial(t, addr)
	a.login()
	b := dial(t, addr)
	b.login()

	a.cmd(250, "CWD sub") // move A only; both connections stay open

	if msg := a.cmd(257, "PWD"); !strings.Contains(msg, "/sub") {
		t.Errorf("A PWD = %q, want /sub", msg)
	}
	if msg := b.cmd(257, "PWD"); !strings.Contains(msg, `"/"`) {
		t.Errorf("B PWD = %q, want / (A's cd must not leak)", msg)
	}
}

func TestSandbox(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()

	if code := c.status("CWD .."); code != 550 {
		t.Errorf("CWD .. from root: code = %d, want 550", code)
	}
	if code := c.status("RETR ../../../../../../etc/passwd"); code != 550 {
		t.Errorf("RETR escape attempt: code = %d, want 550", code)
	}
}

func TestRejectsFTPBounce(t *testing.T) {
	addr, _ := newServer(t)
	c := dial(t, addr)
	c.login()
	if code := c.status("PORT 8,8,8,8,0,21"); code != 501 {
		t.Errorf("PORT to foreign IP: code = %d, want 501", code)
	}
}

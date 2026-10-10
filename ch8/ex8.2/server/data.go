// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// dataTimeout bounds how long we wait for a data connection to be established.
const dataTimeout = 15 * time.Second

// --- data-connection negotiation ---

// handlePasv opens an IPv4 listening socket and reports it in the classic
// (h1,h2,h3,h4,p1,p2) form.
func (s *session) handlePasv() {
	l, err := s.listenData("tcp4")
	if err != nil {
		s.reply(425, "cannot open data connection: %v", err)
		return
	}
	s.setPassive(l)
	addr := l.Addr().(*net.TCPAddr)
	ip := addr.IP.To4()
	if ip == nil {
		ip = net.IPv4(127, 0, 0, 1).To4() // control socket was IPv6
	}
	p := addr.Port
	s.reply(227, "entering passive mode (%d,%d,%d,%d,%d,%d)",
		ip[0], ip[1], ip[2], ip[3], p>>8, p&0xFF)
}

// handleEpsv opens a listening socket (any address family) and reports just
// the port, per RFC 2428.
func (s *session) handleEpsv(arg string) {
	if strings.EqualFold(strings.TrimSpace(arg), "ALL") {
		// Client is declaring it will only use EPSV; nothing to open yet.
		s.reply(200, "EPSV ALL ok")
		return
	}
	l, err := s.listenData("tcp")
	if err != nil {
		s.reply(425, "cannot open data connection: %v", err)
		return
	}
	s.setPassive(l)
	port := l.Addr().(*net.TCPAddr).Port
	s.reply(229, "entering extended passive mode (|||%d|)", port)
}

// handlePort records the client-supplied address for an active-mode transfer.
func (s *session) handlePort(arg string) {
	nums := strings.Split(arg, ",")
	if len(nums) != 6 {
		s.reply(501, "bad PORT argument")
		return
	}
	octets := make([]int, 6)
	for i, n := range nums {
		v, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil || v < 0 || v > 255 {
			s.reply(501, "bad PORT argument")
			return
		}
		octets[i] = v
	}
	ip := fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], octets[3])
	if !sameHost(ip, s.controlRemoteIP()) {
		s.reply(501, "PORT address must match the control connection")
		return
	}
	port := octets[4]<<8 | octets[5]
	s.setActive(net.JoinHostPort(ip, strconv.Itoa(port)))
	s.reply(200, "PORT command successful")
}

// handleEprt records the client-supplied address from an EPRT command:
// |proto|addr|port|, where the delimiter is the first byte.
func (s *session) handleEprt(arg string) {
	if len(arg) < 2 {
		s.reply(501, "bad EPRT argument")
		return
	}
	delim := string(arg[0])
	parts := strings.Split(arg, delim) // ["", proto, addr, port, ""]
	if len(parts) < 5 {
		s.reply(501, "bad EPRT argument")
		return
	}
	addr, port := parts[2], parts[3]
	if !sameHost(addr, s.controlRemoteIP()) {
		s.reply(501, "EPRT address must match the control connection")
		return
	}
	s.setActive(net.JoinHostPort(addr, port))
	s.reply(200, "EPRT command successful")
}

func (s *session) listenData(network string) (net.Listener, error) {
	host, _, err := net.SplitHostPort(s.conn.LocalAddr().String())
	if err != nil {
		return nil, err
	}
	return net.Listen(network, net.JoinHostPort(host, "0"))
}

func (s *session) setPassive(l net.Listener) {
	s.clearData()
	s.dataListener = l
}

func (s *session) setActive(addr string) {
	s.clearData()
	s.dataAddr = addr
}

// clearData abandons any pending data-connection negotiation.
func (s *session) clearData() {
	if s.dataListener != nil {
		s.dataListener.Close()
		s.dataListener = nil
	}
	s.dataAddr = ""
}

// openDataConn establishes the negotiated data connection (passive: accept the
// client's connection; active: dial the client) and resets the negotiation so
// the next transfer must set it up afresh, as real clients expect.
func (s *session) openDataConn() (net.Conn, error) {
	switch {
	case s.dataListener != nil:
		l := s.dataListener
		s.dataListener = nil
		defer l.Close()
		if tl, ok := l.(*net.TCPListener); ok {
			tl.SetDeadline(time.Now().Add(dataTimeout))
		}
		return l.Accept()
	case s.dataAddr != "":
		addr := s.dataAddr
		s.dataAddr = ""
		return net.DialTimeout("tcp", addr, dataTimeout)
	default:
		return nil, fmt.Errorf("no data connection; issue PASV or PORT first")
	}
}

// --- data transfers ---

func (s *session) handleList(arg string, long bool) {
	dir, ok := s.listTarget(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		s.reply(550, "%v", err)
		return
	}
	s.reply(150, "here comes the directory listing")
	data, err := s.openDataConn()
	if err != nil {
		s.reply(425, "%v", err)
		return
	}
	for _, e := range entries {
		if long {
			fmt.Fprintf(data, "%s\r\n", longListLine(e))
		} else {
			fmt.Fprintf(data, "%s\r\n", e.Name())
		}
	}
	data.Close()
	s.reply(226, "directory send OK")
}

func (s *session) handleRetr(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	info, err := os.Stat(target)
	if err != nil {
		s.reply(550, "no such file: %s", arg)
		return
	}
	if info.IsDir() {
		s.reply(550, "not a regular file: %s", arg)
		return
	}
	f, err := os.Open(target)
	if err != nil {
		s.reply(550, "%v", err)
		return
	}
	defer f.Close()

	s.reply(150, "opening data connection for %s (%d bytes)",
		filepath.Base(target), info.Size())
	data, err := s.openDataConn()
	if err != nil {
		s.reply(425, "%v", err)
		return
	}
	n, err := io.Copy(data, f)
	data.Close()
	if err != nil {
		s.reply(426, "transfer failed: %v", err)
		return
	}
	s.reply(226, "transfer complete (%d bytes)", n)
}

func (s *session) handleStor(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		s.reply(550, "is a directory: %s", arg)
		return
	}
	f, err := os.Create(target)
	if err != nil {
		s.reply(550, "%v", err)
		return
	}
	defer f.Close()

	s.reply(150, "ready to receive %s", filepath.Base(target))
	data, err := s.openDataConn()
	if err != nil {
		s.reply(425, "%v", err)
		return
	}
	n, err := io.Copy(f, data)
	data.Close()
	if err != nil {
		s.reply(426, "transfer failed: %v", err)
		return
	}
	s.reply(226, "transfer complete (%d bytes)", n)
}

func (s *session) handleSize(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		s.reply(550, "no such file: %s", arg)
		return
	}
	s.reply(213, "%d", info.Size())
}

func (s *session) handleMdtm(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	info, err := os.Stat(target)
	if err != nil {
		s.reply(550, "no such file: %s", arg)
		return
	}
	s.reply(213, "%s", info.ModTime().UTC().Format("20060102150405"))
}

func (s *session) handleDele(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	if info, err := os.Stat(target); err != nil || info.IsDir() {
		s.reply(550, "no such file: %s", arg)
		return
	}
	if err := os.Remove(target); err != nil {
		s.reply(550, "%v", err)
		return
	}
	s.reply(250, "file deleted")
}

func (s *session) handleMkd(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	if err := os.Mkdir(target, 0o755); err != nil {
		s.reply(550, "%v", err)
		return
	}
	s.reply(257, "%q created", s.displayPath(target))
}

func (s *session) handleRmd(arg string) {
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	if err := os.Remove(target); err != nil { // Remove only deletes empty dirs
		s.reply(550, "%v", err)
		return
	}
	s.reply(250, "directory removed")
}

// listTarget resolves the optional path argument of LIST/NLST, ignoring any
// leading "-l"/"-a"-style flags that some clients prepend.
func (s *session) listTarget(arg string) (string, bool) {
	var path string
	for _, f := range strings.Fields(arg) {
		if !strings.HasPrefix(f, "-") {
			path = f
			break
		}
	}
	if path == "" {
		return s.cwd, true
	}
	return s.resolve(path)
}

// --- path sandboxing ---

// resolve interprets a client path (absolute paths are taken relative to the
// server root) and reports whether it stays within root. The check is lexical:
// symlinks inside root that point outside are not followed-checked.
func (s *session) resolve(arg string) (string, bool) {
	var target string
	if strings.HasPrefix(arg, "/") {
		target = filepath.Join(s.root, filepath.FromSlash(arg))
	} else {
		target = filepath.Join(s.cwd, filepath.FromSlash(arg))
	}
	rel, err := filepath.Rel(s.root, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return target, true
}

// displayPath renders an absolute path as the client sees it: rooted at "/".
func (s *session) displayPath(abs string) string {
	rel, err := filepath.Rel(s.root, abs)
	if err != nil || rel == "." {
		return "/"
	}
	return "/" + filepath.ToSlash(rel)
}

func (s *session) controlRemoteIP() string {
	host, _, _ := net.SplitHostPort(s.conn.RemoteAddr().String())
	return host
}

// sameHost reports whether two host strings denote the same IP address.
func sameHost(a, b string) bool {
	ipa, ipb := net.ParseIP(a), net.ParseIP(b)
	return ipa != nil && ipb != nil && ipa.Equal(ipb)
}

// longListLine formats one directory entry in ls -l style for LIST output.
func longListLine(e os.DirEntry) string {
	info, err := e.Info()
	if err != nil {
		return e.Name()
	}
	return fmt.Sprintf("%s 1 owner group %9d %s %s",
		info.Mode().String(), info.Size(),
		info.ModTime().Format("Jan _2 15:04"), e.Name())
}

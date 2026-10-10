// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

// session is the per-connection state, owned exclusively by the goroutine
// running serve. Nothing here is shared with other sessions.
type session struct {
	conn    net.Conn
	root    string // absolute, canonical server root
	cwd     string // absolute, always within root
	user    string
	authed  bool
	binary  bool // TYPE I vs A (we transfer raw either way)
	verbose bool

	// Data-connection negotiation; at most one is set at a time.
	dataListener net.Listener // passive (PASV/EPSV): we accept
	dataAddr     string       // active (PORT/EPRT): we dial
}

// serve runs the control-connection command loop until QUIT or disconnect.
func (s *session) serve() {
	defer s.conn.Close()
	defer s.clearData() // release a dangling passive listener, if any
	if s.verbose {
		s.log("connected")
		defer s.log("disconnected")
	}

	s.reply(220, "Go FTP server ready")

	scanner := bufio.NewScanner(s.conn)
	for scanner.Scan() {
		line := scanner.Text() // ScanLines strips the trailing \r\n
		if line == "" {
			continue
		}
		cmd, arg, _ := strings.Cut(line, " ")
		cmd = strings.ToUpper(strings.TrimSpace(cmd))
		arg = strings.TrimSpace(arg)
		if s.verbose {
			shown := arg
			if cmd == "PASS" {
				shown = "****"
			}
			s.log("> %s %s", cmd, shown)
		}
		if !s.dispatch(cmd, arg) {
			return
		}
	}
}

// dispatch handles one command and reports whether the session continues.
func (s *session) dispatch(cmd, arg string) bool {
	switch cmd {
	// --- session / no auth required ---
	case "USER":
		s.user = arg
		s.authed = false
		s.reply(331, "please send password for %s", arg)
	case "PASS":
		s.authed = true
		s.reply(230, "user %s logged in", s.user)
	case "QUIT":
		s.reply(221, "goodbye")
		return false
	case "SYST":
		s.reply(215, "UNIX Type: L8")
	case "FEAT":
		s.handleFeat()
	case "OPTS":
		s.reply(200, "%s ok", arg)
	case "NOOP":
		s.reply(200, "NOOP ok")
	case "TYPE":
		s.handleType(arg)
	case "MODE":
		if strings.EqualFold(arg, "S") {
			s.reply(200, "mode set to S")
		} else {
			s.reply(504, "only stream mode supported")
		}
	case "STRU":
		if strings.EqualFold(arg, "F") {
			s.reply(200, "structure set to F")
		} else {
			s.reply(504, "only file structure supported")
		}
	case "HELP":
		s.reply(214, "commands: USER PASS PWD CWD CDUP LIST NLST RETR STOR "+
			"SIZE MDTM DELE MKD RMD TYPE PASV EPSV PORT EPRT QUIT")

	// --- everything below requires login ---
	case "PWD", "XPWD":
		if s.needAuth() {
			s.reply(257, "%q is the current directory", s.displayPath(s.cwd))
		}
	case "CWD":
		if s.needAuth() {
			s.handleCwd(arg)
		}
	case "CDUP":
		if s.needAuth() {
			s.handleCwd("..")
		}
	case "PASV":
		if s.needAuth() {
			s.handlePasv()
		}
	case "EPSV":
		if s.needAuth() {
			s.handleEpsv(arg)
		}
	case "PORT":
		if s.needAuth() {
			s.handlePort(arg)
		}
	case "EPRT":
		if s.needAuth() {
			s.handleEprt(arg)
		}
	case "LIST":
		if s.needAuth() {
			s.handleList(arg, true)
		}
	case "NLST":
		if s.needAuth() {
			s.handleList(arg, false)
		}
	case "RETR":
		if s.needAuth() {
			s.handleRetr(arg)
		}
	case "STOR":
		if s.needAuth() {
			s.handleStor(arg)
		}
	case "SIZE":
		if s.needAuth() {
			s.handleSize(arg)
		}
	case "MDTM":
		if s.needAuth() {
			s.handleMdtm(arg)
		}
	case "DELE":
		if s.needAuth() {
			s.handleDele(arg)
		}
	case "MKD", "XMKD":
		if s.needAuth() {
			s.handleMkd(arg)
		}
	case "RMD", "XRMD":
		if s.needAuth() {
			s.handleRmd(arg)
		}
	case "ABOR":
		s.reply(226, "no transfer in progress")
	default:
		s.reply(502, "command %q not implemented", cmd)
	}
	return true
}

// reply sends a single-line status response, terminated with CRLF as the
// protocol requires.
func (s *session) reply(code int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	if _, err := fmt.Fprintf(s.conn, "%d %s\r\n", code, msg); err != nil {
		s.log("write error: %v", err)
	}
	if s.verbose {
		s.log("< %d %s", code, msg)
	}
}

func (s *session) needAuth() bool {
	if !s.authed {
		s.reply(530, "please login with USER and PASS")
		return false
	}
	return true
}

func (s *session) log(format string, args ...interface{}) {
	log.Printf("[%s] %s", s.conn.RemoteAddr(), fmt.Sprintf(format, args...))
}

func (s *session) handleFeat() {
	io.WriteString(s.conn, "211-Features:\r\n")
	for _, f := range []string{"EPSV", "PASV", "SIZE", "MDTM", "TYPE A;I", "UTF8"} {
		fmt.Fprintf(s.conn, " %s\r\n", f)
	}
	io.WriteString(s.conn, "211 End\r\n")
}

func (s *session) handleType(arg string) {
	t := strings.ToUpper(strings.TrimSpace(arg))
	if t == "" {
		s.reply(500, "TYPE requires an argument")
		return
	}
	switch t[0] {
	case 'A':
		s.binary = false
		s.reply(200, "type set to ASCII")
	case 'I', 'L':
		s.binary = true
		s.reply(200, "type set to binary")
	default:
		s.reply(504, "unsupported type %q", arg)
	}
}

func (s *session) handleCwd(arg string) {
	if arg == "" {
		arg = "/"
	}
	target, ok := s.resolve(arg)
	if !ok {
		s.reply(550, "access denied")
		return
	}
	info, err := os.Stat(target)
	switch {
	case err != nil:
		s.reply(550, "no such directory: %s", arg)
	case !info.IsDir():
		s.reply(550, "not a directory: %s", arg)
	default:
		s.cwd = target
		s.reply(250, "directory changed to %s", s.displayPath(target))
	}
}

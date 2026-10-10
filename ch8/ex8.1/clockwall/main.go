// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Clockwall (Exercise 8.1) is a client of several clock servers at once. It
// reads the time from each one and displays them together in a table, like
// the wall of clocks in some business offices.
//
// Each argument names a clock: NAME=host:port. Columns appear in the order
// given on the command line.
//
//	clockwall NewYork=localhost:8010 Tokyo=localhost:8020 London=localhost:8030
//
// Design note (the §8.4 lesson, one section early): the readers never share
// the times map. Each connection gets its own goroutine that streams readings
// onto a single channel; one aggregator goroutine owns the map and is the only
// thing that touches it. That's CSP — "share memory by communicating" — and it
// means no mutex is needed even though three goroutines feed the same table.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// reading is one timestamp received from one named clock.
type reading struct {
	name string
	time string
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: %s NAME=host:port [NAME=host:port ...]\n", os.Args[0])
		os.Exit(1)
	}

	var names []string // preserves command-line order = column order
	readings := make(chan reading)

	for _, arg := range os.Args[1:] {
		// strings.Cut (Go 1.18+) is the modern one-liner for splitting on the
		// first "="; the book-era equivalent is strings.SplitN(arg, "=", 2).
		name, addr, ok := strings.Cut(arg, "=")
		if !ok {
			log.Fatalf("bad argument %q: want NAME=host:port", arg)
		}
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatalf("dialing %s (%s): %v", name, addr, err)
		}
		names = append(names, name)
		go readClock(name, conn, readings)
	}

	runWall(names, readings)
}

// readClock streams times from one clock server onto the shared channel until
// the connection closes.
func readClock(name string, conn net.Conn, out chan<- reading) {
	defer conn.Close()
	input := bufio.NewScanner(conn)
	for input.Scan() {
		out <- reading{name: name, time: input.Text()}
	}
}

// runWall owns the latest-times map — no other goroutine reads or writes it —
// and reprints the table once per second.
func runWall(names []string, readings <-chan reading) {
	latest := make(map[string]string)
	tick := time.Tick(1 * time.Second)

	fmt.Println(format(names)) // header row
	for {
		select {
		case r := <-readings:
			latest[r.name] = r.time
		case <-tick:
			cells := make([]string, len(names))
			for i, n := range names {
				if cells[i] = latest[n]; cells[i] == "" {
					cells[i] = "--:--:--" // not heard from yet
				}
			}
			fmt.Println(format(cells))
		}
	}
}

// format lays out one row, one fixed-width column per clock.
func format(cells []string) string {
	var b strings.Builder
	for _, c := range cells {
		fmt.Fprintf(&b, "%-12s", c)
	}
	return strings.TrimRight(b.String(), " ")
}

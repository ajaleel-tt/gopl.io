// Exercise 8.15: Failure of any client program to read data in a timely
// manner ultimately causes all clients to get stuck. Modify the broadcaster
// to skip a message rather than wait if a client writer is not ready to
// accept it.
//
// Builds on Exercises 8.12 (roster on arrival), 8.13 (idle disconnect) and
// 8.14 (client names).
//
// Protocol: on connecting, the server sends "Enter your name:" and takes the
// first non-empty line as the client's name. Every later line is a message.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"time"
)

// idleTimeout is how long a client may go without sending a message before
// it is disconnected. It's a variable so tests can shorten it.
var idleTimeout = 5 * time.Minute

// clientBuffer is how many outgoing messages may queue for a client whose
// writer is busy. A client that falls further behind than this starts
// missing messages, instead of holding up everyone else.
const clientBuffer = 64

// A client is an outgoing message channel plus the client's name. Both
// fields are comparable, so a client is too and can still be a map key; two
// clients are equal when they share the same channel (and name).
type client struct {
	out  chan<- string
	name string
}

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming client messages
)

func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				trySend(cli, msg)
			}

		case cli := <-entering:
			// Tell the newcomer who is already here, before adding them,
			// so the list doesn't include themselves.
			trySend(cli, roster(clients))
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.out)
		}
	}
}

// trySend delivers msg to cli unless its queue is full, in which case the
// message is skipped for that client. The broadcaster never waits on a
// client, so one stuck reader can't stall the others.
func trySend(cli client, msg string) {
	select {
	case cli.out <- msg:
	default:
		// cli isn't keeping up; it misses this message.
	}
}

// roster describes the current clients. Map iteration order is random, so
// the names are sorted to give a stable announcement.
func roster(clients map[client]bool) string {
	if len(clients) == 0 {
		return "You are the only one here."
	}
	var names []string
	for cli := range clients {
		names = append(names, cli.name)
	}
	sort.Strings(names)
	return "Current clients: " + strings.Join(names, ", ")
}

func handleConn(conn net.Conn) {
	ch := make(chan string, clientBuffer) // outgoing client messages
	go clientWriter(conn, ch)

	// Start the watchdog before asking for a name, so a client that
	// connects and never answers is disconnected too.
	activity := make(chan struct{}, 1)
	go idleWatchdog(conn, activity)

	input := bufio.NewScanner(conn)
	who, ok := readName(input, ch, activity)
	if !ok {
		// Disconnected before giving a name. The broadcaster never saw
		// ch, so it's still ours to close.
		close(activity)
		close(ch)
		conn.Close()
		return
	}

	cli := client{out: ch, name: who}
	ch <- "You are " + who
	messages <- who + " has arrived"
	entering <- cli // from here on, the broadcaster owns ch

	for input.Scan() {
		messages <- who + ": " + input.Text()
		signal(activity)
	}
	// NOTE: ignoring potential errors from input.Err()
	close(activity) // lets the watchdog exit if it's still running

	leaving <- cli
	messages <- who + " has left"
	conn.Close()
}

// readName prompts until the client sends a non-empty line and returns it.
// It returns false if the connection ends first.
func readName(input *bufio.Scanner, ch chan<- string, activity chan<- struct{}) (string, bool) {
	for {
		ch <- "Enter your name:"
		if !input.Scan() {
			return "", false
		}
		signal(activity)
		if name := strings.TrimSpace(input.Text()); name != "" {
			return name, true
		}
	}
}

// signal records activity without blocking: if a signal is already pending,
// or the watchdog has already given up on us, there's nothing to add.
func signal(activity chan<- struct{}) {
	select {
	case activity <- struct{}{}:
	default:
	}
}

// idleWatchdog closes conn if no activity arrives within idleTimeout.
// Closing the connection makes the blocked input.Scan() in handleConn
// return false, so the client then leaves the normal way.
func idleWatchdog(conn net.Conn, activity <-chan struct{}) {
	for {
		select {
		case _, ok := <-activity:
			if !ok {
				return // client left on its own
			}
			// Activity: loop around, starting a fresh timeout.
		case <-time.After(idleTimeout):
			log.Printf("%s: idle for %v, disconnecting", conn.RemoteAddr(), idleTimeout)
			conn.Close()
			return
		}
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}
	go broadcaster()
	acceptLoop(listener)
}

// acceptLoop is split out of main so tests can serve on a random port.
func acceptLoop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

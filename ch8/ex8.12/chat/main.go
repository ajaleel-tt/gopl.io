// Exercise 8.12: Make the broadcaster announce the current set of clients to
// each new arrival. This requires that the clients set and the entering and
// leaving channels record the client name too.
//
// Based on gopl.io/ch8/chat (page 254).
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
)

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
				cli.out <- msg
			}

		case cli := <-entering:
			// Tell the newcomer who is already here, before adding them,
			// so the list doesn't include themselves.
			cli.out <- roster(clients)
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.out)
		}
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
	ch := make(chan string) // outgoing client messages
	go clientWriter(conn, ch)

	who := conn.RemoteAddr().String()
	cli := client{out: ch, name: who}
	ch <- "You are " + who
	messages <- who + " has arrived"
	entering <- cli

	input := bufio.NewScanner(conn)
	for input.Scan() {
		messages <- who + ": " + input.Text()
	}
	// NOTE: ignoring potential errors from input.Err()

	leaving <- cli
	messages <- who + " has left"
	conn.Close()
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

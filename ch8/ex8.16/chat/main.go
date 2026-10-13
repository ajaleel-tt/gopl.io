// "Exercise 8.16" (not in the book): finish the chat server with the
// Chapter 8 techniques the earlier exercises didn't need, so one program
// uses all of them:
//
//   - Graceful shutdown on Ctrl+C or SIGTERM by closing a done channel
//     (§8.9, as in du4). Clients get a goodbye message before being cut off.
//   - A sync.WaitGroup that tracks every connection, so shutdown waits for
//     all of them to finish (§8.5, as in thumbnail's makeThumbnails6).
//   - A counting semaphore that caps how many clients may connect at once
//     (§8.6, as in crawl2), acquired with a non-blocking select (§8.7) so a
//     full server turns clients away instead of leaving them hanging.
//   - A periodic stats log driven by a ticker in the broadcaster's select,
//     switched off with a nil channel when not wanted (§8.7/§8.8, as in du3).
//
// Builds on Exercises 8.12 (roster), 8.13 (idle disconnect), 8.14 (names)
// and 8.15 (slow clients).
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Tunables. They're variables so tests can shrink them.
var (
	// idleTimeout is how long a client may go without sending a message
	// before it is disconnected.
	idleTimeout = 5 * time.Minute

	// maxClients is how many connections may be open at once.
	maxClients = 100

	// statsInterval is how often the broadcaster logs its stats; 0 turns
	// the stats log off.
	statsInterval = time.Minute

	// writeGrace is how long a departing client's writer gets to flush
	// what's queued for it before the connection is closed anyway.
	writeGrace = 2 * time.Second
)

// clientBuffer is how many outgoing messages may queue for a client whose
// writer is busy. A client that falls further behind than this starts
// missing messages, instead of holding up everyone else.
const clientBuffer = 64

const (
	shutdownNotice = "Server is shutting down. Goodbye!"
	fullNotice     = "Server is full, try again later."
)

// done is closed to tell every goroutine the server is shutting down.
// Closing a channel is a broadcast: every receive on it proceeds at once.
var done = make(chan struct{})

// cancelled polls done without blocking.
func cancelled() bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

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

// broadcaster runs until messages is closed, which serve does only once
// every connection has finished, so nothing can send to it afterwards.
func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	dropped := 0                     // messages skipped for slow clients

	// A nil channel is never ready, so with stats off this case is
	// simply disabled.
	var tick <-chan time.Time
	if statsInterval > 0 {
		ticker := time.NewTicker(statsInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for {
		select {
		case msg, ok := <-messages:
			if !ok {
				return
			}
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				if !trySend(cli, msg) {
					dropped++
				}
			}

		case cli := <-entering:
			// Tell the newcomer who is already here, before adding them,
			// so the list doesn't include themselves.
			trySend(cli, roster(clients))
			clients[cli] = true

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.out)

		case <-tick:
			log.Printf("stats: %d clients connected, %d messages dropped", len(clients), dropped)
		}
	}
}

// trySend delivers msg to cli unless its queue is full, in which case the
// message is skipped for that client and trySend reports false. The
// broadcaster never waits on a client, so one stuck reader can't stall the
// others.
func trySend(cli client, msg string) bool {
	select {
	case cli.out <- msg:
		return true
	default:
		return false
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
	writerDone := make(chan struct{})
	go func() {
		clientWriter(conn, ch)
		close(writerDone)
	}()

	// Start the watchdog before asking for a name, so a client that
	// connects and never answers is disconnected too.
	activity := make(chan struct{}, 1)
	go watchdog(conn, activity)

	input := bufio.NewScanner(conn)
	who, ok := readName(input, ch, activity)
	if !ok || cancelled() {
		// Gone, or shutting down, before entering the room. The
		// broadcaster never saw ch, so it's still ours to close.
		if cancelled() {
			sayGoodbye(ch)
		}
		close(activity)
		close(ch)
		finish(conn, writerDone)
		return
	}

	cli := client{out: ch, name: who}
	ch <- "You are " + who
	messages <- who + " has arrived"
	entering <- cli // from here on, the broadcaster owns ch

	for input.Scan() {
		messages <- who + ": " + input.Text()
		noteActivity(activity)
	}
	// NOTE: ignoring potential errors from input.Err()
	close(activity) // lets the watchdog exit if it's still running

	if cancelled() {
		sayGoodbye(ch) // safe: the broadcaster only closes ch after leaving
	}
	leaving <- cli
	if !cancelled() {
		// During shutdown everyone is leaving; don't flood the ones
		// still here with a "has left" for each.
		messages <- who + " has left"
	}
	finish(conn, writerDone)
}

// sayGoodbye queues the shutdown notice without blocking: if the client's
// queue is full, it isn't reading and wouldn't see it anyway.
func sayGoodbye(ch chan<- string) {
	select {
	case ch <- shutdownNotice:
	default:
	}
}

// finish lets the client's writer flush whatever is queued (its channel has
// been closed by now, so the writer ends once it's drained), then closes
// the connection. A writer blocked on a client that isn't reading gets
// writeGrace before the Close cuts it off.
func finish(conn net.Conn, writerDone <-chan struct{}) {
	select {
	case <-writerDone:
	case <-time.After(writeGrace):
	}
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
		noteActivity(activity)
		if name := strings.TrimSpace(input.Text()); name != "" {
			return name, true
		}
	}
}

// noteActivity records activity without blocking: if a signal is already
// pending, or the watchdog has already given up on us, there's nothing to
// add.
func noteActivity(activity chan<- struct{}) {
	select {
	case activity <- struct{}{}:
	default:
	}
}

// watchdog ends the client's input, unblocking input.Scan() in handleConn,
// either when no activity arrives within idleTimeout or when the server
// shuts down.
func watchdog(conn net.Conn, activity <-chan struct{}) {
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
		case <-done:
			// Stop reading but keep the connection open for writing, so
			// the goodbye can still be delivered. A deadline in the past
			// makes the pending Read fail at once.
			conn.SetReadDeadline(time.Now())
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

	// Signals arrive on a channel too. The first one starts a graceful
	// shutdown; a second one gives up waiting.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Print("shutting down; interrupt again to force")
		close(done)
		<-sigs
		log.Fatal("forced shutdown")
	}()

	serve(listener)
	log.Print("all clients gone, bye")
}

// serve runs the chat on listener and returns once done has been closed and
// every connection has finished.
func serve(listener net.Listener) {
	broadcasterDone := make(chan struct{})
	go func() {
		broadcaster()
		close(broadcasterDone)
	}()

	// Closing the listener is the only way to unblock Accept.
	go func() {
		<-done
		listener.Close()
	}()

	var wg sync.WaitGroup
	acceptLoop(listener, &wg)

	wg.Wait()       // every handleConn has returned...
	close(messages) // ...so nothing sends any more; stop the broadcaster
	<-broadcasterDone
}

// acceptLoop starts a handleConn for each connection, up to maxClients at
// once, until the listener is closed.
func acceptLoop(listener net.Listener, wg *sync.WaitGroup) {
	// tokens is a counting semaphore: each connection holds one token,
	// and the channel's capacity is how many can be held at once.
	tokens := make(chan struct{}, maxClients)
	for {
		conn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Print(err)
			continue
		}

		select {
		case tokens <- struct{}{}: // acquire a token
		default:
			fmt.Fprintln(conn, fullNotice)
			conn.Close()
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-tokens }() // release the token
			handleConn(conn)
		}()
	}
}

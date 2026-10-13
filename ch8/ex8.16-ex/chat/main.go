// "Exercise 8.16-ex": the ex8.16 capstone chat server rewritten with modern
// Go idioms instead of the book's. Same behaviour; compare side by side.
//
//   - Cancellation: context.Context from signal.NotifyContext replaces the
//     done channel and cancelled(); context.AfterFunc replaces the goroutines
//     that waited on done to close things.
//   - Idle timeout: conn.SetReadDeadline before each read replaces the
//     watchdog goroutine and its activity channel.
//   - Write grace: conn.SetWriteDeadline replaces select with time.After.
//   - Waiting: sync.WaitGroup.Go (Go 1.25) replaces Add/go/defer Done.
//   - Capacity: golang.org/x/sync/semaphore replaces the token channel.
//   - Logging: log/slog with structured fields replaces log.Printf.
//   - State: a server struct replaces package-level channels and tunables,
//     so each server (and each test) is independent.
//
// The broadcaster, its non-blocking sends, and the shutdown ordering are
// unchanged: those were never book-era, just Go.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sync/semaphore"
)

// Config holds the server's tunables.
type Config struct {
	IdleTimeout   time.Duration // disconnect clients silent this long
	MaxClients    int64         // connections open at once
	StatsInterval time.Duration // how often to log stats; 0 disables
	WriteGrace    time.Duration // time a departing client's writer gets to flush
	ClientBuffer  int           // outgoing messages queued per client
}

var defaultConfig = Config{
	IdleTimeout:   5 * time.Minute,
	MaxClients:    100,
	StatsInterval: time.Minute,
	WriteGrace:    2 * time.Second,
	ClientBuffer:  64,
}

const (
	prompt         = "Enter your name:"
	shutdownNotice = "Server is shutting down. Goodbye!"
	fullNotice     = "Server is full, try again later."
)

// A client is an outgoing message channel plus the client's name. Both
// fields are comparable, so a client is too and can be a map key; two
// clients are equal when they share the same channel (and name).
type client struct {
	out  chan<- string
	name string
}

type server struct {
	cfg      Config
	log      *slog.Logger
	entering chan client
	leaving  chan client
	messages chan string // all incoming client messages
}

func newServer(cfg Config, log *slog.Logger) *server {
	return &server{
		cfg:      cfg,
		log:      log,
		entering: make(chan client),
		leaving:  make(chan client),
		messages: make(chan string),
	}
}

func main() {
	ln, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		slog.Error("listen", "err", err)
		os.Exit(1)
	}

	// ctx is cancelled on the first Ctrl+C or SIGTERM. Calling stop then
	// restores default signal handling, so a second Ctrl+C kills the
	// process immediately.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	context.AfterFunc(ctx, func() {
		slog.Info("shutting down; interrupt again to force")
		stop()
	})

	newServer(defaultConfig, slog.Default()).serve(ctx, ln)
	slog.Info("all clients gone, bye")
}

// serve runs the chat on ln and returns once ctx is cancelled and every
// connection has finished.
func (s *server) serve(ctx context.Context, ln net.Listener) {
	broadcasterDone := make(chan struct{})
	go func() {
		s.broadcaster()
		close(broadcasterDone)
	}()

	// Closing the listener is the only way to unblock Accept.
	stopClosing := context.AfterFunc(ctx, func() { ln.Close() })
	defer stopClosing()

	var wg sync.WaitGroup
	sem := semaphore.NewWeighted(s.cfg.MaxClients)
	for {
		conn, err := ln.Accept()
		if errors.Is(err, net.ErrClosed) {
			break
		}
		if err != nil {
			s.log.Warn("accept", "err", err)
			continue
		}
		if !sem.TryAcquire(1) {
			fmt.Fprintln(conn, fullNotice)
			conn.Close()
			continue
		}
		wg.Go(func() {
			defer sem.Release(1)
			s.handleConn(ctx, conn)
		})
	}

	wg.Wait()         // every handleConn has returned...
	close(s.messages) // ...so nothing sends any more; stop the broadcaster
	<-broadcasterDone
}

// broadcaster runs until s.messages is closed, which serve does only once
// every connection has finished.
func (s *server) broadcaster() {
	clients := make(map[client]struct{}) // all connected clients
	dropped := 0                         // messages skipped for slow clients

	// A nil channel is never ready, so with stats off this case is
	// simply disabled.
	var tick <-chan time.Time
	if s.cfg.StatsInterval > 0 {
		ticker := time.NewTicker(s.cfg.StatsInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for {
		select {
		case msg, ok := <-s.messages:
			if !ok {
				return
			}
			for cli := range clients {
				if !trySend(cli, msg) {
					dropped++
				}
			}

		case cli := <-s.entering:
			// Tell the newcomer who is already here, before adding them,
			// so the list doesn't include themselves.
			trySend(cli, roster(clients))
			clients[cli] = struct{}{}

		case cli := <-s.leaving:
			delete(clients, cli)
			close(cli.out)

		case <-tick:
			s.log.Info("stats", "clients", len(clients), "dropped", dropped)
		}
	}
}

// trySend delivers msg to cli unless its queue is full, in which case the
// message is skipped for that client and trySend reports false.
func trySend(cli client, msg string) bool {
	select {
	case cli.out <- msg:
		return true
	default:
		return false
	}
}

// roster describes the current clients, sorted because map iteration order
// is random.
func roster(clients map[client]struct{}) string {
	if len(clients) == 0 {
		return "You are the only one here."
	}
	var names []string
	for cli := range clients {
		names = append(names, cli.name)
	}
	slices.Sort(names)
	return "Current clients: " + strings.Join(names, ", ")
}

func (s *server) handleConn(ctx context.Context, conn net.Conn) {
	ch := make(chan string, s.cfg.ClientBuffer) // outgoing client messages
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		clientWriter(conn, ch)
	}()
	// By the time this runs, ch has been closed (by us, or by the
	// broadcaster on leaving), so the writer ends once it has flushed;
	// the write deadline stops it waiting forever on a client that isn't
	// reading.
	defer func() {
		conn.SetWriteDeadline(time.Now().Add(s.cfg.WriteGrace))
		<-writerDone
		conn.Close()
	}()

	// On shutdown, fail the pending read at once but leave the connection
	// writable, so the goodbye can still be delivered.
	stopUnblock := context.AfterFunc(ctx, func() { conn.SetReadDeadline(time.Now()) })
	defer stopUnblock()

	// scan reads the next line, allowing IdleTimeout for it. The deadline
	// is set before checking ctx: if shutdown came first we see it here,
	// and if it comes later its past deadline overrides this one.
	input := bufio.NewScanner(conn)
	scan := func() bool {
		conn.SetReadDeadline(time.Now().Add(s.cfg.IdleTimeout))
		if ctx.Err() != nil {
			return false
		}
		return input.Scan()
	}

	who, ok := readName(scan, input, ch)
	if !ok || ctx.Err() != nil {
		// Gone, or shutting down, before entering the room. The
		// broadcaster never saw ch, so it's still ours to close.
		s.logEnd(ctx, conn, "", input.Err())
		if ctx.Err() != nil {
			sayGoodbye(ch)
		}
		close(ch)
		return
	}

	cli := client{out: ch, name: who}
	ch <- "You are " + who
	s.messages <- who + " has arrived"
	s.entering <- cli // from here on, the broadcaster owns ch

	for scan() {
		s.messages <- who + ": " + input.Text()
	}
	s.logEnd(ctx, conn, who, input.Err())

	if ctx.Err() != nil {
		sayGoodbye(ch) // safe: the broadcaster only closes ch after leaving
	}
	s.leaving <- cli
	if ctx.Err() == nil {
		// During shutdown everyone is leaving; don't flood the ones
		// still here with a "has left" for each.
		s.messages <- who + " has left"
	}
}

// logEnd records why a client's input ended, if it's worth recording.
func (s *server) logEnd(ctx context.Context, conn net.Conn, who string, err error) {
	switch {
	case ctx.Err() != nil, err == nil:
		// Shutdown, or the client hung up: nothing unusual.
	case errors.Is(err, os.ErrDeadlineExceeded):
		s.log.Info("idle disconnect", "addr", conn.RemoteAddr(), "name", who, "timeout", s.cfg.IdleTimeout)
	default:
		s.log.Debug("read error", "addr", conn.RemoteAddr(), "name", who, "err", err)
	}
}

// readName prompts until the client sends a non-empty line and returns it.
// It returns false if the input ends first.
func readName(scan func() bool, input *bufio.Scanner, ch chan<- string) (string, bool) {
	for {
		ch <- prompt
		if !scan() {
			return "", false
		}
		if name := strings.TrimSpace(input.Text()); name != "" {
			return name, true
		}
	}
}

// sayGoodbye queues the shutdown notice without blocking: if the client's
// queue is full, it isn't reading and wouldn't see it anyway.
func sayGoodbye(ch chan<- string) {
	select {
	case ch <- shutdownNotice:
	default:
	}
}

// clientWriter writes each message to conn. After a write error it keeps
// draining ch without writing, so nothing sending to ch ever blocks on it.
func clientWriter(conn net.Conn, ch <-chan string) {
	var err error
	for msg := range ch {
		if err == nil {
			_, err = fmt.Fprintln(conn, msg)
		}
	}
}

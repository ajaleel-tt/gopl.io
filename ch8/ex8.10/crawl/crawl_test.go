// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestRequestCancelled is the crux of Exercise 8.10: cancelling the context
// aborts an HTTP request that is already in flight, rather than waiting for the
// (never-responding) server.
func TestRequestCancelled(t *testing.T) {
	// The handler never responds; it just waits until the client hangs up.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done() // unblocks (server-side) when the client cancels
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() {
		_, err := extract(ctx, srv.URL)
		errc <- err
	}()

	time.Sleep(50 * time.Millisecond) // let the request get in flight
	cancel()

	select {
	case err := <-errc:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("extract returned %v, want an error wrapping context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("extract did not return after cancel — the in-flight request was not cancelled")
	}
}

// TestCrawlCancellation verifies cancellation terminates the crawler even on an
// effectively infinite site: every page links to two brand-new pages, so the
// frontier never empties. Without cancellation, crawlAll would run forever.
func TestCrawlCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond) // bound the fetch rate (and output)
		p := r.URL.Path
		fmt.Fprintf(w, `<a href="%sa">x</a><a href="%sb">y</a>`, p, p)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		crawlAll(ctx, []string{srv.URL + "/"})
		close(done)
	}()

	time.Sleep(200 * time.Millisecond) // let it crawl a while
	cancel()

	select {
	case <-done:
		// crawlAll returned → cancellation terminated the infinite crawl.
	case <-time.After(3 * time.Second):
		t.Fatal("crawlAll did not stop within 3s of cancellation")
	}
}

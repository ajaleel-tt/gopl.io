// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestFetchReturnsFirstAndCancelsRest: the fast server wins, and both slow
// servers observe their requests being cancelled.
func TestFetchReturnsFirstAndCancelsRest(t *testing.T) {
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "fast wins")
	}))
	defer fast.Close()

	// Slow handlers block; each records whether its request got cancelled.
	var cancelled int32
	slowHandler := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done(): // client aborted us (we lost the race)
			atomic.AddInt32(&cancelled, 1)
		case <-time.After(3 * time.Second): // the slow "response" that never wins
			fmt.Fprint(w, "slow")
		}
	}
	slow1 := httptest.NewServer(http.HandlerFunc(slowHandler))
	defer slow1.Close()
	slow2 := httptest.NewServer(http.HandlerFunc(slowHandler))
	defer slow2.Close()

	r, err := fetch(fast.URL, slow1.URL, slow2.URL)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(r.body); got != "fast wins" {
		t.Errorf("body = %q, want %q", got, "fast wins")
	}
	if r.url != fast.URL {
		t.Errorf("winner = %s, want the fast server %s", r.url, fast.URL)
	}

	// Cancellation propagates to the slow handlers asynchronously; give it a moment.
	time.Sleep(300 * time.Millisecond)
	if got := atomic.LoadInt32(&cancelled); got != 2 {
		t.Errorf("cancelled %d slow requests, want 2 (losers should be aborted)", got)
	}
}

// TestFetchAllFail: when every request fails, fetch returns an error.
func TestFetchAllFail(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer bad.Close()

	if _, err := fetch(bad.URL, bad.URL); err == nil {
		t.Error("expected an error when all requests fail, got nil")
	}
}

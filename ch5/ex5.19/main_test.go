package main

import (
	"testing"
)

func TestWackyAnswerReturnsNonZero(t *testing.T) {
	got := wackyAnswer()
	if got == 0 {
		t.Errorf("wackyAnswer() = 0, want non-zero")
	}
	if got != 42 {
		t.Errorf("wackyAnswer() = %d, want 42", got)
	}
}

// TestWackyAnswerDoesNotPanic confirms that the panic is fully contained
// inside wackyAnswer — no panic propagates to the caller.
func TestWackyAnswerDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic escaped wackyAnswer: %v", r)
		}
	}()
	_ = wackyAnswer()
}

// TestWackyAnswerRepeatable confirms the behavior is stable across calls
// (no global state, no flaky control flow).
func TestWackyAnswerRepeatable(t *testing.T) {
	for i := 0; i < 100; i++ {
		if got := wackyAnswer(); got != 42 {
			t.Fatalf("call %d: wackyAnswer() = %d, want 42", i, got)
		}
	}
}

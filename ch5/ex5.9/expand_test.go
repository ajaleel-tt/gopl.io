package main

import (
	"strings"
	"testing"
)

func TestExpandSimple(t *testing.T) {
	got := expand("hello $name", func(s string) string {
		return "world"
	})
	want := "hello world"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandMultiple(t *testing.T) {
	values := map[string]string{
		"a": "1",
		"b": "2",
	}
	got := expand("$a + $b = $c", func(s string) string {
		if v, ok := values[s]; ok {
			return v
		}
		return "?"
	})
	want := "1 + 2 = ?"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandNoVars(t *testing.T) {
	got := expand("plain text", strings.ToUpper)
	want := "plain text"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandPassesNameWithoutDollar(t *testing.T) {
	var seen []string
	expand("$foo and $bar", func(s string) string {
		seen = append(seen, s)
		return ""
	})
	if len(seen) != 2 || seen[0] != "foo" || seen[1] != "bar" {
		t.Errorf("expected callback to receive [foo bar], got %v", seen)
	}
}

func TestExpandAdjacent(t *testing.T) {
	got := expand("$a$b", func(s string) string { return s })
	want := "ab"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandUnderscoresAndDigits(t *testing.T) {
	got := expand("$foo_1 done", func(s string) string {
		if s == "foo_1" {
			return "OK"
		}
		return "FAIL"
	})
	want := "OK done"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestExpandLoneDollar(t *testing.T) {
	got := expand("price: $", func(s string) string { return "X" })
	want := "price: $"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

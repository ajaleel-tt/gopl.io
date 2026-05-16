package main

import (
	"strings"
	"testing"
)

func TestJoinMatchesStrings(t *testing.T) {
	cases := []struct {
		sep   string
		elems []string
	}{
		{", ", nil},
		{", ", []string{}},
		{", ", []string{"only"}},
		{", ", []string{"a", "b"}},
		{", ", []string{"a", "b", "c"}},
		{"", []string{"x", "y", "z"}},
		{"---", []string{"foo", "bar"}},
		{",", []string{"", "", ""}},
	}
	for _, tc := range cases {
		want := strings.Join(tc.elems, tc.sep)
		got := Join(tc.sep, tc.elems...)
		if got != want {
			t.Errorf("Join(%q, %q) = %q, want %q (matching strings.Join)",
				tc.sep, tc.elems, got, want)
		}
	}
}

func TestJoinNoArgs(t *testing.T) {
	if got := Join(", "); got != "" {
		t.Errorf("Join(\", \") = %q, want \"\"", got)
	}
}

func TestJoinSingleArg(t *testing.T) {
	// With one element, the separator never appears.
	if got := Join("XYZ", "lone"); got != "lone" {
		t.Errorf("Join(\"XYZ\", \"lone\") = %q, want \"lone\"", got)
	}
}

func TestJoinSpread(t *testing.T) {
	parts := []string{"alpha", "beta", "gamma"}
	if got := Join("-", parts...); got != "alpha-beta-gamma" {
		t.Errorf("Join(\"-\", parts...) = %q, want \"alpha-beta-gamma\"", got)
	}
}

func TestJoinEmptySep(t *testing.T) {
	if got := Join("", "a", "b", "c"); got != "abc" {
		t.Errorf("Join(\"\", \"a\", \"b\", \"c\") = %q, want \"abc\"", got)
	}
}

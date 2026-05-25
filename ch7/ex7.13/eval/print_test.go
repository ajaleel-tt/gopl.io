package eval

import (
	"reflect"
	"testing"
)

// TestStringRoundTrip parses each input, prints the resulting tree, and
// parses the printed form again. The two trees must be DeepEqual: the
// String method is only useful if it preserves the syntax tree.
func TestStringRoundTrip(t *testing.T) {
	inputs := []string{
		"3.14",
		"x",
		"-x",
		"+-x",
		"x + y",
		"1 + 2 + 3",        // left-associative
		"1 + 2 * 3",        // mixed precedence
		"(1 + 2) * 3",      // explicit grouping
		"5 / 9 * (F - 32)", // from eval_test
		"sqrt(A / pi)",
		"pow(x, 3) + pow(y, 3)",
		"sin(-x) + pow(x, 2)",
	}

	for _, in := range inputs {
		e1, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q) failed: %v", in, err)
			continue
		}
		printed := e1.String()
		e2, err := Parse(printed)
		if err != nil {
			t.Errorf("Parse(%q) [round-trip of %q] failed: %v", printed, in, err)
			continue
		}
		if !reflect.DeepEqual(e1, e2) {
			t.Errorf("round-trip mismatch\n  input:   %s\n  printed: %s\n  e1: %#v\n  e2: %#v",
				in, printed, e1, e2)
		}
	}
}

// TestStringOutput pins the printed form for a few inputs so changes to
// the printer are visible in test failures.
func TestStringOutput(t *testing.T) {
	cases := []struct{ in, want string }{
		{"x", "x"},
		{"3.14", "3.14"},
		{"-x", "(-x)"},
		{"x + y", "(x + y)"},
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"sqrt(A / pi)", "sqrt((A / pi))"},
	}
	for _, c := range cases {
		e, err := Parse(c.in)
		if err != nil {
			t.Errorf("Parse(%q): %v", c.in, err)
			continue
		}
		if got := e.String(); got != c.want {
			t.Errorf("Parse(%q).String() = %q, want %q", c.in, got, c.want)
		}
	}
}

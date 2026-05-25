package eval

import (
	"math"
	"reflect"
	"testing"
)

// TestVariadicEvalHandBuilt evaluates trees constructed directly,
// without going through the parser — the path the book hints at.
func TestVariadicEvalHandBuilt(t *testing.T) {
	// min(x, y, 3)
	minExpr := variadic{op: "min", args: []Expr{
		Var("x"), Var("y"), literal(3),
	}}
	// max(x, y, 3)
	maxExpr := variadic{op: "max", args: []Expr{
		Var("x"), Var("y"), literal(3),
	}}

	cases := []struct {
		e            Expr
		env          Env
		wantMinOrMax float64
	}{
		{minExpr, Env{"x": 5, "y": 2}, 2},
		{minExpr, Env{"x": -1, "y": 10}, -1},
		{minExpr, Env{"x": 100, "y": 50}, 3}, // literal wins
		{maxExpr, Env{"x": 5, "y": 2}, 5},
		{maxExpr, Env{"x": -1, "y": 10}, 10},
		{maxExpr, Env{"x": 1, "y": 2}, 3}, // literal wins
	}
	for _, c := range cases {
		got := c.e.Eval(c.env)
		if got != c.wantMinOrMax {
			t.Errorf("%s in %v = %g, want %g", c.e, c.env, got, c.wantMinOrMax)
		}
	}
}

// TestVariadicViaParser confirms the parser extension produces variadic
// nodes (not call nodes) for min/max, and that they evaluate correctly.
func TestVariadicViaParser(t *testing.T) {
	e, err := Parse("min(x, y, 3) + max(1, 2)")
	if err != nil {
		t.Fatal(err)
	}
	// Sanity-check that the tree contains a variadic, not a call, by
	// inspecting via reflect.
	if !containsVariadic(e) {
		t.Errorf("expected tree to contain variadic node, got %#v", e)
	}
	got := e.Eval(Env{"x": 7, "y": 4})
	want := 3.0 + 2.0 // min(7,4,3) + max(1,2)
	if got != want {
		t.Errorf("Eval = %g, want %g", got, want)
	}
}

func containsVariadic(e Expr) bool {
	switch e := e.(type) {
	case variadic:
		return true
	case unary:
		return containsVariadic(e.x)
	case binary:
		return containsVariadic(e.x) || containsVariadic(e.y)
	case call:
		for _, a := range e.args {
			if containsVariadic(a) {
				return true
			}
		}
	}
	return false
}

// TestVariadicCheck covers the validation rules: known op, non-empty args,
// and recursion into operands.
func TestVariadicCheck(t *testing.T) {
	cases := []struct {
		name    string
		e       Expr
		wantErr string
	}{
		{"empty args", variadic{op: "min"}, "min requires at least one argument"},
		{"unknown op", variadic{op: "avg", args: []Expr{literal(1)}}, `unknown variadic operator "avg"`},
		{"bad inner", variadic{op: "min", args: []Expr{call{fn: "log", args: []Expr{literal(1)}}}}, `unknown function "log"`},
	}
	for _, c := range cases {
		vars := make(map[Var]bool)
		err := c.e.Check(vars)
		if err == nil || err.Error() != c.wantErr {
			t.Errorf("%s: Check() error = %v, want %q", c.name, err, c.wantErr)
		}
	}
}

// TestVariadicRoundTrip extends ex7.13's round-trip property to the new
// node type.
func TestVariadicRoundTrip(t *testing.T) {
	inputs := []string{
		"min(x, y)",
		"max(1, 2, 3, 4)",
		"min(x, y) + max(a, b)",
		"max(-x, sqrt(x))",
	}
	for _, in := range inputs {
		e1, err := Parse(in)
		if err != nil {
			t.Errorf("Parse(%q): %v", in, err)
			continue
		}
		e2, err := Parse(e1.String())
		if err != nil {
			t.Errorf("re-parse of %q (=%q): %v", in, e1.String(), err)
			continue
		}
		if !reflect.DeepEqual(e1, e2) {
			t.Errorf("round-trip mismatch for %q:\n  printed: %s\n  e1: %#v\n  e2: %#v",
				in, e1.String(), e1, e2)
		}
	}
}

// Guard against NaN comparison quirks: math.NaN() is not less than
// anything, so min/max with NaN should propagate the first operand.
// (Not a stated requirement; documents current behavior so a change
// here is intentional.)
func TestVariadicNaNBehavior(t *testing.T) {
	e := variadic{op: "min", args: []Expr{
		literal(math.NaN()), literal(1), literal(2),
	}}
	got := e.Eval(nil)
	if !math.IsNaN(got) {
		t.Errorf("min(NaN, 1, 2) = %g, want NaN (first-operand wins when no later value is strictly less)", got)
	}
}

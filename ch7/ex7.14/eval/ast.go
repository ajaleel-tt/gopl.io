// Exercise 7.14: adds a new concrete Expr type, variadic, that takes a
// variable number of operands. It currently implements "min" and "max",
// which the existing unary/binary forms can't express naturally. The
// parser is extended so min(a, b, ...) and max(a, b, ...) produce a
// variadic node instead of a call node.
//
// The String round-trip from ex7.13 still holds for the new type.

package eval

// An Expr is an arithmetic expression.
type Expr interface {
	// Eval returns the value of this Expr in the environment env.
	Eval(env Env) float64
	// Check reports errors in this Expr and adds its Vars to the set.
	Check(vars map[Var]bool) error
	// String returns the expression in source form. The output is
	// fully parenthesized: re-parsing it yields a tree equivalent to
	// the original under reflect.DeepEqual.
	String() string
}

// A Var identifies a variable, e.g., x.
type Var string

// A literal is a numeric constant, e.g., 3.141.
type literal float64

// A unary represents a unary operator expression, e.g., -x.
type unary struct {
	op rune // one of '+', '-'
	x  Expr
}

// A binary represents a binary operator expression, e.g., x+y.
type binary struct {
	op   rune // one of '+', '-', '*', '/'
	x, y Expr
}

// A call represents a function call expression, e.g., sin(x).
type call struct {
	fn   string // one of "pow", "sin", "sqrt"
	args []Expr
}

// A variadic represents an n-ary operator, e.g., min(x, y, z).
// Unlike call, the number of operands is not fixed.
type variadic struct {
	op   string // "min" or "max"
	args []Expr // must be non-empty
}

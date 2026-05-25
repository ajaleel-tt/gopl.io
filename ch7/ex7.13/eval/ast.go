// Exercise 7.13: extends the eval package so that every concrete Expr
// type implements fmt.Stringer, allowing an Expr to pretty-print itself
// back to source form. String is also added to the Expr interface so
// the syntax tree as a whole satisfies fmt.Stringer.

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

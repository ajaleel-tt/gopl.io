// Exercise 7.14: add a new concrete type (variadic, supporting min/max)
// that satisfies the Expr interface. Demonstrates both ways of using it:
// constructing the tree directly, and via the extended parser.
package main

import (
	"fmt"

	"gopl.io/ch7/ex7.14/eval"
)

func main() {
	// Via the extended parser. min and max accept any arity >= 1.
	exprs := []struct {
		src string
		env eval.Env
	}{
		{"min(x, y, 3)", eval.Env{"x": 7, "y": 5}},
		{"max(x, y, 3)", eval.Env{"x": 7, "y": 5}},
		{"min(x, y) + max(x, y)", eval.Env{"x": 4, "y": 9}},
		{"max(-x, sqrt(x))", eval.Env{"x": 9}},
	}

	for _, c := range exprs {
		e, err := eval.Parse(c.src)
		if err != nil {
			fmt.Printf("%-30s parse error: %v\n", c.src, err)
			continue
		}
		vars := map[eval.Var]bool{}
		if err := e.Check(vars); err != nil {
			fmt.Printf("%-30s check error: %v\n", c.src, err)
			continue
		}
		fmt.Printf("%-25s in %v = %g\n  printed: %s\n\n",
			c.src, c.env, e.Eval(c.env), e)
	}
}

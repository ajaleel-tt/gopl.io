package eval

import (
	"fmt"
	"strings"
)

// Every concrete Expr type satisfies fmt.Stringer. The output is fully
// parenthesized rather than minimal: emitting the structure verbatim
// guarantees the round-trip Parse(e.String()) yields an equivalent
// tree without having to model operator precedence in the printer.

func (v Var) String() string { return string(v) }

func (l literal) String() string { return fmt.Sprintf("%g", float64(l)) }

func (u unary) String() string { return fmt.Sprintf("(%c%s)", u.op, u.x) }

func (b binary) String() string {
	return fmt.Sprintf("(%s %c %s)", b.x, b.op, b.y)
}

func (c call) String() string {
	args := make([]string, len(c.args))
	for i, a := range c.args {
		args[i] = a.String()
	}
	return fmt.Sprintf("%s(%s)", c.fn, strings.Join(args, ", "))
}

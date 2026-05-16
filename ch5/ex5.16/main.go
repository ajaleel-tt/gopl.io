// Exercise 5.16: variadic strings.Join.
package main

import (
	"fmt"
	"strings"
)

// Join concatenates elems separated by sep. Variadic counterpart to
// strings.Join, where the separator is required (and therefore comes first,
// since the variadic parameter must be last).
func Join(sep string, elems ...string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	// Pre-size the buffer to avoid reallocation: total = sum(elems) + sep*(n-1).
	n := len(sep) * (len(elems) - 1)
	for _, e := range elems {
		n += len(e)
	}
	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, e := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(e)
	}
	return b.String()
}

func main() {
	fmt.Printf("%q\n", Join(", ", "a", "b", "c"))
	fmt.Printf("%q\n", Join(" - ", "only"))
	fmt.Printf("%q\n", Join(", "))
	fmt.Printf("%q\n", Join("", "a", "b", "c"))

	parts := []string{"alpha", "beta", "gamma"}
	fmt.Printf("%q\n", Join(" / ", parts...))
}

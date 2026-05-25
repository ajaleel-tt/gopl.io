// Exercise 7.13: add a String method to Expr that pretty-prints the
// syntax tree, and verify that re-parsing the printed form yields an
// equivalent tree.
//
// The real work lives in ./eval (String methods + round-trip test).
// This program is a small demo: parse, print, re-parse, compare.
package main

import (
	"fmt"
	"reflect"

	"gopl.io/ch7/ex7.13/eval"
)

func main() {
	inputs := []string{
		"5 / 9 * (F - 32)",
		"sqrt(A / pi)",
		"pow(x, 3) + pow(y, 3)",
		"-1 + -x",
		"1 + 2 * 3",
	}

	for _, in := range inputs {
		e1, err := eval.Parse(in)
		if err != nil {
			fmt.Printf("%-25s parse error: %v\n", in, err)
			continue
		}
		printed := e1.String()
		e2, err := eval.Parse(printed)
		if err != nil {
			fmt.Printf("%-25s re-parse error: %v\n", in, err)
			continue
		}
		fmt.Printf("input:    %s\nprinted:  %s\nequal?    %v\n\n",
			in, printed, reflect.DeepEqual(e1, e2))
	}
}

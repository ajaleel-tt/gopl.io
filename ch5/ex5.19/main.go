// Exercise 5.19: a function with no return statement that nevertheless
// returns a non-zero value, via panic + recover and a named return value.
//
// Mechanism:
//  1. The function declares a named return value (`answer`).
//  2. It immediately defers a closure that calls recover() to swallow any
//     panic, then assigns answer = 42.
//  3. The function body panics. Control transfers to the deferred closure,
//     which sets answer.
//  4. With the panic recovered and answer set, the function returns
//     normally — but with no `return` keyword anywhere in the source.
package main

import "fmt"

func wackyAnswer() (answer int) {
	defer func() {
		recover()
		answer = 42
	}()
	panic("never propagates")
}

func main() {
	fmt.Printf("wackyAnswer() = %d\n", wackyAnswer())
}

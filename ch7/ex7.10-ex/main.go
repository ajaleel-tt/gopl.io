// Exercise 7.10 — modernized variant.
//
// The book's version makes IsPalindrome take sort.Interface, illustrating
// that a "sorting" contract can be reused for any algorithm that needs to
// compare elements of an indexed sequence. That's a clever pedagogical move
// for a pre-generics language.
//
// Modern Go (1.18+) has type parameters. The natural way to write a function
// "I can run on any indexed sequence of comparable values" is to just take
// []T directly, constrained by `comparable`. We no longer need sort.Interface
// as a stand-in for "give me Len + equality" — we have []T and ==.
//
// Side-by-side with ch7/ex7.10 (the book-era version), this file shows what
// changes: signature, body (== instead of !Less&&!Less), and call sites
// (raw slices, no sort.IntSlice / sort.StringSlice adapter casts).
package main

import "fmt"

// IsPalindrome reports whether s reads the same forward and backward.
// Two-pointer walk from both ends; O(n/2) time, O(1) space.
func IsPalindrome[T comparable](s []T) bool {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		if s[i] != s[j] {
			return false
		}
	}
	return true
}

func main() {
	fmt.Printf("%-30v → palindrome=%v\n", []string{"a", "b", "a"}, IsPalindrome([]string{"a", "b", "a"}))
	fmt.Printf("%-30v → palindrome=%v\n", []string{"r", "a", "c", "e", "c", "a", "r"}, IsPalindrome([]string{"r", "a", "c", "e", "c", "a", "r"}))
	fmt.Printf("%-30v → palindrome=%v\n", []string{"a", "b", "c"}, IsPalindrome([]string{"a", "b", "c"}))
	fmt.Printf("%-30v → palindrome=%v\n", []int{1, 2, 3, 2, 1}, IsPalindrome([]int{1, 2, 3, 2, 1}))
	fmt.Printf("%-30v → palindrome=%v\n", []int{1, 2, 3}, IsPalindrome([]int{1, 2, 3}))

	// Bonus: works on types comparable but not orderable.
	// The book version (sort.Interface) couldn't handle this — Less needs
	// an ordering. Equality (==) is enough for palindrome detection.
	fmt.Printf("%-30v → palindrome=%v\n", []bool{true, false, true}, IsPalindrome([]bool{true, false, true}))
}

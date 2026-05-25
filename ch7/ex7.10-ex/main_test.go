package main

import (
	"testing"
)

func TestIsPalindromeStrings(t *testing.T) {
	cases := []struct {
		s    []string
		want bool
	}{
		{nil, true},
		{[]string{"a"}, true},
		{[]string{"a", "a"}, true},
		{[]string{"a", "b"}, false},
		{[]string{"a", "b", "a"}, true},
		{[]string{"a", "b", "b", "a"}, true},
		{[]string{"a", "b", "c"}, false},
		{[]string{"r", "a", "c", "e", "c", "a", "r"}, true},
		{[]string{"h", "e", "l", "l", "o"}, false},
	}
	for _, tc := range cases {
		// No sort.StringSlice cast — pass the slice directly.
		if got := IsPalindrome(tc.s); got != tc.want {
			t.Errorf("IsPalindrome(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestIsPalindromeInts(t *testing.T) {
	cases := []struct {
		s    []int
		want bool
	}{
		{nil, true},
		{[]int{42}, true},
		{[]int{1, 1}, true},
		{[]int{1, 2}, false},
		{[]int{1, 2, 1}, true},
		{[]int{1, 2, 3, 2, 1}, true},
		{[]int{1, 2, 3, 4, 5}, false},
		{[]int{-1, 0, -1}, true},
	}
	for _, tc := range cases {
		// No sort.IntSlice cast.
		if got := IsPalindrome(tc.s); got != tc.want {
			t.Errorf("IsPalindrome(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

func TestIsPalindromeRunes(t *testing.T) {
	cases := []struct {
		s    string
		want bool
	}{
		{"", true},
		{"a", true},
		{"aa", true},
		{"ab", false},
		{"aba", true},
		{"abba", true},
		{"racecar", true},
		{"hello", false},
		{"日本日", true},
		{"日本本日", true},
		{"日本語", false},
	}
	for _, tc := range cases {
		// No custom runes adapter type needed — []rune(s) is already []T comparable.
		if got := IsPalindrome([]rune(tc.s)); got != tc.want {
			t.Errorf("IsPalindrome(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// TestIsPalindromeOnNonOrderableType demonstrates a capability the book
// version doesn't have: works on any `comparable` type, not just `Ordered`.
// bools have == but no <, so the sort.Interface version can't express this.
func TestIsPalindromeOnNonOrderableType(t *testing.T) {
	cases := []struct {
		s    []bool
		want bool
	}{
		{[]bool{}, true},
		{[]bool{true}, true},
		{[]bool{true, true}, true},
		{[]bool{true, false, true}, true},
		{[]bool{true, false}, false},
		{[]bool{true, false, false, true}, true},
	}
	for _, tc := range cases {
		if got := IsPalindrome(tc.s); got != tc.want {
			t.Errorf("IsPalindrome(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// TestIsPalindromeOnStructs — another capability that's natural here.
// Comparable structs (no slice/map/func fields) work directly.
type Point struct{ X, Y int }

func TestIsPalindromeOnStructs(t *testing.T) {
	a := Point{1, 2}
	b := Point{3, 4}
	c := Point{1, 2}

	if !IsPalindrome([]Point{a, b, c}) { // a == c, b in the middle → palindrome
		t.Error("[a b c] where a==c should be a palindrome")
	}
	if IsPalindrome([]Point{a, b}) {
		t.Error("[a b] should not be a palindrome")
	}
}

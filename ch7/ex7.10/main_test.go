package main

import (
	"sort"
	"testing"
)

func TestIsPalindromeStrings(t *testing.T) {
	cases := []struct {
		s    []string
		want bool
	}{
		{nil, true},                         // empty
		{[]string{"a"}, true},               // single
		{[]string{"a", "a"}, true},          // pair, equal
		{[]string{"a", "b"}, false},         // pair, not equal
		{[]string{"a", "b", "a"}, true},     // odd, palindrome
		{[]string{"a", "b", "b", "a"}, true}, // even, palindrome
		{[]string{"a", "b", "c"}, false},    // odd, not palindrome
		{[]string{"r", "a", "c", "e", "c", "a", "r"}, true},
		{[]string{"h", "e", "l", "l", "o"}, false},
	}
	for _, tc := range cases {
		if got := IsPalindrome(sort.StringSlice(tc.s)); got != tc.want {
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
		if got := IsPalindrome(sort.IntSlice(tc.s)); got != tc.want {
			t.Errorf("IsPalindrome(%v) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// runes adapts a []rune to sort.Interface so we can palindrome-check strings
// at the rune level. This is the right granularity for Unicode: a multi-byte
// rune isn't equal to its constituent bytes.
type runes []rune

func (r runes) Len() int           { return len(r) }
func (r runes) Less(i, j int) bool { return r[i] < r[j] }
func (r runes) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

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
		{"日本日", true},   // Unicode palindrome
		{"日本本日", true},  // Unicode palindrome (even-length)
		{"日本語", false},  // Unicode non-palindrome
	}
	for _, tc := range cases {
		if got := IsPalindrome(runes(tc.s)); got != tc.want {
			t.Errorf("IsPalindrome(%q) = %v, want %v", tc.s, got, tc.want)
		}
	}
}

// TestIsPalindromeNeverCallsSwap verifies that IsPalindrome only uses Len
// and Less. It's a real interface contract — IsPalindrome should be safe
// to call on data you don't want to mutate.
func TestIsPalindromeNeverCallsSwap(t *testing.T) {
	calls := &swapTracker{IntSlice: sort.IntSlice{1, 2, 3, 2, 1}}
	got := IsPalindrome(calls)
	if !got {
		t.Errorf("IsPalindrome([1 2 3 2 1]) = false, want true")
	}
	if calls.swaps != 0 {
		t.Errorf("IsPalindrome called Swap %d times, want 0", calls.swaps)
	}
}

type swapTracker struct {
	sort.IntSlice
	swaps int
}

func (s *swapTracker) Swap(i, j int) {
	s.swaps++
	s.IntSlice.Swap(i, j)
}

// TestIsPalindromeStableTies confirms our equality definition handles the
// tricky case where Less returns false in both directions (i.e., elements
// are "equal" under the ordering).
func TestIsPalindromeStableTies(t *testing.T) {
	// All elements equal — should be a palindrome.
	all5 := sort.IntSlice{5, 5, 5, 5, 5}
	if !IsPalindrome(all5) {
		t.Error("all-equal sequence should be a palindrome")
	}
}

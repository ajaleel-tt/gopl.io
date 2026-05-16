// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 165.

// Package intset provides a set of integers based on a bit vector.
package intset

import (
	"bytes"
	"fmt"
)

// uintSize is the size in bits of a uint on this platform: 32 or 64.
//
// How the expression works:
//   - ^uint(0)   is all-ones in a uint: 0xFFFFFFFF (32b) or 0xFFFFFFFFFFFFFFFF (64b).
//   - >> 63      shifts right by 63. On a 64-bit uint the high bit becomes bit 0,
//     yielding 1. On a 32-bit uint, shifting by ≥ width yields 0.
//   - 32 << k    is therefore 32<<1=64 on 64-bit and 32<<0=32 on 32-bit.
const uintSize = 32 << (^uint(0) >> 63)

//!+intset

// An IntSet is a set of small non-negative integers.
// Its zero value represents the empty set.
type IntSet struct {
	words []uint
}

// Has reports whether the set contains the non-negative value x.
func (s *IntSet) Has(x int) bool {
	word, bit := x/uintSize, uint(x%uintSize)
	return word < len(s.words) && s.words[word]&(1<<bit) != 0
}

// Add adds the non-negative value x to the set.
func (s *IntSet) Add(x int) {
	word, bit := x/uintSize, uint(x%uintSize)
	for word >= len(s.words) {
		s.words = append(s.words, 0)
	}
	s.words[word] |= 1 << bit
}

// UnionWith sets s to the union of s and t.
func (s *IntSet) UnionWith(t *IntSet) {
	for i, tword := range t.words {
		if i < len(s.words) {
			s.words[i] |= tword
		} else {
			s.words = append(s.words, tword)
		}
	}
}

//!-intset

//!+string

// String returns the set as a string of the form "{1 2 3}".
func (s *IntSet) String() string {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < uintSize; j++ {
			if word&(1<<uint(j)) != 0 {
				if buf.Len() > len("{") {
					buf.WriteByte(' ')
				}
				fmt.Fprintf(&buf, "%d", uintSize*i+j)
			}
		}
	}
	buf.WriteByte('}')
	return buf.String()
}

//!-string

// Len returns the number of elements in the set.
func (s *IntSet) Len() int {
	n := 0
	for _, word := range s.words {
		for j := 0; j < uintSize; j++ {
			if word&(1<<uint(j)) != 0 {
				n++
			}
		}
	}
	return n
}

// Remove removes x from the set. Removing a value not in the set is a no-op.
func (s *IntSet) Remove(x int) {
	word, bit := x/uintSize, uint(x%uintSize)
	if word < len(s.words) {
		s.words[word] &^= 1 << bit
	}
}

// Clear removes all elements from the set.
func (s *IntSet) Clear() {
	s.words = nil
}

// Copy returns a copy of the set, independent of the receiver.
func (s *IntSet) Copy() *IntSet {
	c := &IntSet{words: make([]uint, len(s.words))}
	copy(c.words, s.words)
	return c
}

// AddAll adds each of the given non-negative values to the set.
func (s *IntSet) AddAll(xs ...int) {
	for _, x := range xs {
		s.Add(x)
	}
}

// IntersectWith sets s to the intersection of s and t (elements in both).
func (s *IntSet) IntersectWith(t *IntSet) {
	for i := range s.words {
		if i < len(t.words) {
			s.words[i] &= t.words[i]
		} else {
			// t has no bits set in this word, so the intersection has none.
			s.words[i] = 0
		}
	}
}

// DifferenceWith sets s to the difference s - t (elements in s but not in t).
func (s *IntSet) DifferenceWith(t *IntSet) {
	for i, tword := range t.words {
		if i < len(s.words) {
			s.words[i] &^= tword
		}
		// Words past len(s.words) contribute nothing to s, so we can stop
		// implicitly — there's nothing to clear.
	}
}

// SymmetricDifference sets s to the symmetric difference of s and t
// (elements in exactly one of s, t but not both).
func (s *IntSet) SymmetricDifference(t *IntSet) {
	for i, tword := range t.words {
		if i < len(s.words) {
			s.words[i] ^= tword
		} else {
			s.words = append(s.words, tword)
		}
	}
}

// Elems returns the elements of the set in ascending order. The returned
// slice is independent of the set's internal storage.
func (s *IntSet) Elems() []int {
	elems := make([]int, 0, s.Len())
	for i, word := range s.words {
		if word == 0 {
			continue
		}
		for j := 0; j < uintSize; j++ {
			if word&(1<<uint(j)) != 0 {
				elems = append(elems, uintSize*i+j)
			}
		}
	}
	return elems
}

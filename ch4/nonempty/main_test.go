package main

import (
	"testing"
)

func TestRemoveAdjacentDups(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"all same", []string{"a", "a", "a"}, []string{"a"}},
		{"adjacent pair", []string{"a", "a", "b"}, []string{"a", "b"}},
		{"trailing dups", []string{"a", "b", "b"}, []string{"a", "b"}},
		{"middle dups", []string{"a", "b", "b", "c"}, []string{"a", "b", "c"}},
		{"multiple groups", []string{"a", "a", "b", "b", "c", "c"}, []string{"a", "b", "c"}},
		{"non-adjacent dups preserved", []string{"a", "b", "a"}, []string{"a", "b", "a"}},
		{"single element", []string{"a"}, []string{"a"}},
		{"empty slice", []string{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// make a copy so we don't affect other tests
			input := make([]string, len(tt.input))
			copy(input, tt.input)

			got := removeAdjacentDups(input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %q, want %q", got, tt.want)
				}
			}
		})
	}
}

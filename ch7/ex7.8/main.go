// Exercise 7.8: a stateful multi-tier sort, plus the repeated sort.Stable
// alternative for comparison.
//
// Approach A — MultiSorter (one sort, comparator stack):
//   The Less method tries each comparator in priority order; the first one
//   that yields a strict ordering decides. Only one pass over the data, so
//   the cost is O(n log n * k) in the worst case (each comparison may walk
//   the whole stack), but k is small (≤ number of columns) and most
//   comparisons short-circuit on the primary key.
//
// Approach B — repeated sort.Stable:
//   Sort by the LOWEST-priority key first, then the next, ..., then the
//   HIGHEST priority. Because stable sort preserves the relative order of
//   equal elements, earlier sorts' results are preserved as tiebreakers.
//   Cost is O(k * n log n) — k full sort passes. Conceptually simpler but
//   does k times more comparisons in practice.
package main

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"
)

type Track struct {
	Title  string
	Artist string
	Album  string
	Year   int
	Length time.Duration
}

func length(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		panic(s)
	}
	return d
}

// lessFunc is a comparator for Tracks.
type lessFunc func(x, y *Track) bool

// MultiSorter implements sort.Interface with a stack of comparators.
// The first element of less is the primary sort key; the last is the lowest
// priority tiebreaker.
type MultiSorter struct {
	tracks []*Track
	less   []lessFunc
}

// NewMultiSorter constructs a MultiSorter with the given comparators in
// priority order (highest priority first).
func NewMultiSorter(less ...lessFunc) *MultiSorter {
	return &MultiSorter{less: less}
}

func (m *MultiSorter) Sort(tracks []*Track) {
	m.tracks = tracks
	sort.Sort(m)
}

func (m *MultiSorter) Len() int      { return len(m.tracks) }
func (m *MultiSorter) Swap(i, j int) { m.tracks[i], m.tracks[j] = m.tracks[j], m.tracks[i] }

func (m *MultiSorter) Less(i, j int) bool {
	p, q := m.tracks[i], m.tracks[j]
	// Try each comparator in priority order; the first that breaks the tie wins.
	for k := 0; k < len(m.less)-1; k++ {
		less := m.less[k]
		switch {
		case less(p, q):
			return true
		case less(q, p):
			return false
		}
	}
	// Last comparator settles any remaining tie.
	if len(m.less) == 0 {
		return false
	}
	return m.less[len(m.less)-1](p, q)
}

// ClickColumn simulates clicking a column header: the given comparator
// becomes the highest-priority sort key. Any prior occurrence of the same
// key is removed so the same column clicked twice doesn't duplicate keys.
//
// Comparator identity is by function pointer; identical-purpose but
// separately defined lambdas won't dedupe (a known limitation of using
// function values as identity).
func (m *MultiSorter) ClickColumn(less lessFunc) {
	// Remove existing identical comparator (function pointer match).
	lessPtr := fmt.Sprintf("%p", less)
	for i, f := range m.less {
		if fmt.Sprintf("%p", f) == lessPtr {
			m.less = append(m.less[:i], m.less[i+1:]...)
			break
		}
	}
	m.less = append([]lessFunc{less}, m.less...)
}

// StableSortMulti sorts tracks by each comparator in turn, going from the
// LOWEST priority to the HIGHEST. Because sort.SliceStable preserves the
// relative order of equal elements, the earlier sorts act as tiebreakers
// for the later, higher-priority sorts.
func StableSortMulti(tracks []*Track, less []lessFunc) {
	for k := len(less) - 1; k >= 0; k-- {
		f := less[k]
		sort.SliceStable(tracks, func(i, j int) bool {
			return f(tracks[i], tracks[j])
		})
	}
}

// Standard column comparators.
var (
	byTitle  lessFunc = func(x, y *Track) bool { return x.Title < y.Title }
	byArtist lessFunc = func(x, y *Track) bool { return x.Artist < y.Artist }
	byAlbum  lessFunc = func(x, y *Track) bool { return x.Album < y.Album }
	byYear   lessFunc = func(x, y *Track) bool { return x.Year < y.Year }
	byLength lessFunc = func(x, y *Track) bool { return x.Length < y.Length }
)

func printTracks(tracks []*Track) {
	const format = "%v\t%v\t%v\t%v\t%v\t\n"
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintf(tw, format, "Title", "Artist", "Album", "Year", "Length")
	fmt.Fprintf(tw, format, "-----", "------", "-----", "----", "------")
	for _, t := range tracks {
		fmt.Fprintf(tw, format, t.Title, t.Artist, t.Album, t.Year, t.Length)
	}
	tw.Flush()
}

func sample() []*Track {
	return []*Track{
		{"Go", "Delilah", "From the Roots Up", 2012, length("3m38s")},
		{"Go", "Moby", "Moby", 1992, length("3m37s")},
		{"Go Ahead", "Alicia Keys", "As I Am", 2007, length("4m36s")},
		{"Ready 2 Go", "Martin Solveig", "Smash", 2011, length("4m24s")},
	}
}

func main() {
	// Approach A: stateful clicks.
	fmt.Println("--- MultiSorter: click Year, then Title ---")
	tracks := sample()
	ms := NewMultiSorter()
	ms.ClickColumn(byYear)
	ms.ClickColumn(byTitle) // most recent → primary
	ms.Sort(tracks)
	printTracks(tracks)

	// Approach B: repeated sort.Stable, same effective key order.
	fmt.Println("\n--- StableSortMulti: primary Title, then Year ---")
	tracks = sample()
	StableSortMulti(tracks, []lessFunc{byTitle, byYear})
	printTracks(tracks)
}

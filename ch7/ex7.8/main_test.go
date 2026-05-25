package main

import (
	"reflect"
	"testing"
)

func titles(tracks []*Track) []string {
	out := make([]string, len(tracks))
	for i, t := range tracks {
		out[i] = t.Title
	}
	return out
}

func TestMultiSorterEmpty(t *testing.T) {
	ms := NewMultiSorter()
	tracks := sample()
	ms.Sort(tracks)
	// With no comparators, Less always returns false → sort.Sort is a no-op.
	if titles(tracks)[0] != "Go" {
		t.Errorf("expected unchanged order with no comparators")
	}
}

func TestMultiSorterSinglePrimary(t *testing.T) {
	tracks := sample()
	ms := NewMultiSorter(byTitle)
	ms.Sort(tracks)
	want := []string{"Go", "Go", "Go Ahead", "Ready 2 Go"}
	if got := titles(tracks); !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestMultiSorterTitleThenYear(t *testing.T) {
	tracks := sample()
	ms := NewMultiSorter(byTitle, byYear) // primary Title, secondary Year
	ms.Sort(tracks)

	// "Go" titles tie, so secondary key Year orders them: 1992 before 2012.
	want := []struct {
		title string
		year  int
	}{
		{"Go", 1992},
		{"Go", 2012},
		{"Go Ahead", 2007},
		{"Ready 2 Go", 2011},
	}
	for i, w := range want {
		if tracks[i].Title != w.title || tracks[i].Year != w.year {
			t.Errorf("pos %d: got (%q, %d), want (%q, %d)",
				i, tracks[i].Title, tracks[i].Year, w.title, w.year)
		}
	}
}

// TestMultiSorterClickColumn verifies the "stateful click" semantic:
// click promotes a comparator to primary; the previous primary becomes
// the secondary.
func TestMultiSorterClickColumn(t *testing.T) {
	tracks := sample()
	ms := NewMultiSorter()

	// Click Year first → primary Year
	ms.ClickColumn(byYear)
	// Click Title → primary becomes Title, secondary Year
	ms.ClickColumn(byTitle)
	ms.Sort(tracks)

	want := []struct {
		title string
		year  int
	}{
		{"Go", 1992},
		{"Go", 2012},
		{"Go Ahead", 2007},
		{"Ready 2 Go", 2011},
	}
	for i, w := range want {
		if tracks[i].Title != w.title || tracks[i].Year != w.year {
			t.Errorf("pos %d: got (%q, %d), want (%q, %d)",
				i, tracks[i].Title, tracks[i].Year, w.title, w.year)
		}
	}
}

func TestMultiSorterClickRemovesDuplicate(t *testing.T) {
	ms := NewMultiSorter()
	ms.ClickColumn(byYear)
	ms.ClickColumn(byTitle)
	ms.ClickColumn(byYear) // re-click Year — should promote, not duplicate

	if len(ms.less) != 2 {
		t.Errorf("expected 2 comparators after re-click, got %d", len(ms.less))
	}
	// Year should be primary now.
	// We can't compare function values directly, so check by behavior:
	tracks := sample()
	ms.Sort(tracks)
	// Primary is Year (ascending), so 1992 comes first.
	if tracks[0].Year != 1992 {
		t.Errorf("after re-click of byYear, primary not Year (got year %d first)",
			tracks[0].Year)
	}
}

// TestStableSortMultiAgreesWithMultiSorter is the comparison point:
// given the same priority order of keys, both approaches must produce
// identical output.
func TestStableSortMultiAgreesWithMultiSorter(t *testing.T) {
	priorities := []lessFunc{byTitle, byYear, byLength}

	a := sample()
	NewMultiSorter(priorities...).Sort(a)

	b := sample()
	StableSortMulti(b, priorities)

	for i := range a {
		if a[i].Title != b[i].Title || a[i].Year != b[i].Year ||
			a[i].Artist != b[i].Artist {
			t.Errorf("position %d differs: MultiSorter=%+v StableSortMulti=%+v",
				i, *a[i], *b[i])
		}
	}
}

func TestStableSortMultiOrdering(t *testing.T) {
	tracks := sample()
	StableSortMulti(tracks, []lessFunc{byTitle, byYear})

	want := []struct {
		title string
		year  int
	}{
		{"Go", 1992},
		{"Go", 2012},
		{"Go Ahead", 2007},
		{"Ready 2 Go", 2011},
	}
	for i, w := range want {
		if tracks[i].Title != w.title || tracks[i].Year != w.year {
			t.Errorf("pos %d: got (%q, %d), want (%q, %d)",
				i, tracks[i].Title, tracks[i].Year, w.title, w.year)
		}
	}
}

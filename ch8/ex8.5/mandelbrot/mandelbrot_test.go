// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

package main

import (
	"bytes"
	"fmt"
	"image"
	"runtime"
	"sort"
	"testing"
)

func renderImage(workers int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	render(img, workers)
	return img
}

// TestParallelMatchesSequential proves that parallelizing changed nothing about
// the output: the image is byte-identical regardless of worker count. Run with
// -race to also confirm the concurrent writes to the shared image never race.
func TestParallelMatchesSequential(t *testing.T) {
	seq := renderImage(1)
	for _, w := range []int{2, 4, runtime.NumCPU()} {
		if par := renderImage(w); !bytes.Equal(seq.Pix, par.Pix) {
			t.Errorf("workers=%d produced a different image than workers=1", w)
		}
	}
}

// workerCounts is an ascending, de-duplicated ladder of worker counts to sweep.
func workerCounts() []int {
	n := runtime.NumCPU()
	seen, out := map[int]bool{}, []int{}
	for _, c := range []int{1, 2, 4, 8, n, 2 * n, 4 * n} {
		if c > 0 && !seen[c] {
			seen[c] = true
			out = append(out, c)
		}
	}
	sort.Ints(out)
	return out
}

func BenchmarkRender(b *testing.B) {
	for _, w := range workerCounts() {
		b.Run(fmt.Sprintf("workers=%03d", w), func(b *testing.B) {
			img := image.NewRGBA(image.Rect(0, 0, width, height))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				render(img, w)
			}
		})
	}
}

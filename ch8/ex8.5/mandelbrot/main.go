// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Mandelbrot (Exercise 8.5) emits a PNG of the Mandelbrot fractal, computing
// its rows in parallel. The per-pixel math is identical to the §3.3 program;
// only the main loop is parallelized.
//
// Rows are handed to a pool of worker goroutines through a channel. The channel
// is both the fan-out mechanism and a dynamic load balancer: Mandelbrot rows
// cost very different amounts (points inside the set run the full iteration
// count; points outside escape almost at once), so a static row split would
// leave some workers idle while others grind. Pulling from a shared channel
// lets faster workers grab more rows.
//
// Usage:
//
//	go run . -workers 8 > out.png
package main

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"log"
	"math/cmplx"
	"os"
	"runtime"
	"sync"
)

const (
	xmin, ymin, xmax, ymax = -2, -2, +2, +2
	width, height          = 1024, 1024
)

// render fills img by computing each row in one of `workers` goroutines.
func render(img *image.RGBA, workers int) {
	rows := make(chan int, height) // work queue: one entry per row
	for py := 0; py < height; py++ {
		rows <- py
	}
	close(rows)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for py := range rows { // pull rows until the queue drains
				renderRow(img, py)
			}
		}()
	}
	wg.Wait()
}

// renderRow computes one horizontal line of pixels. Each row occupies a
// disjoint span of img.Pix (offset py*Stride), so workers handling different
// rows never write overlapping bytes — no locking is required.
func renderRow(img *image.RGBA, py int) {
	y := float64(py)/height*(ymax-ymin) + ymin
	for px := 0; px < width; px++ {
		x := float64(px)/width*(xmax-xmin) + xmin
		img.Set(px, py, mandelbrot(complex(x, y)))
	}
}

func mandelbrot(z complex128) color.Color {
	const iterations = 200
	const contrast = 15

	var v complex128
	for n := uint8(0); n < iterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return color.Gray{Y: 255 - contrast*n}
		}
	}
	return color.Black
}

func main() {
	workers := flag.Int("workers", runtime.NumCPU(), "number of worker goroutines")
	flag.Parse()

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	render(img, *workers)
	if err := png.Encode(os.Stdout, img); err != nil {
		log.Fatal(err)
	}
}

// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 16.
//!+

// Fetch prints the content found at each specified URL.
package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	for _, url := range os.Args[1:] {
		resp, err := http.Get(url)
		const buf_size = 512
		var buf [buf_size]byte
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
			os.Exit(1)
		}
		var num_bytes_read int
		var reader_err error
		for {
			num_bytes_read, reader_err = resp.Body.Read(buf[:])
			if num_bytes_read > 0 {
				os.Stdout.Write(buf[0:num_bytes_read])
			}
			if reader_err == io.EOF {
				break
			}
			if reader_err != nil {
				fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
				os.Exit(1)
			}
		}
		// b, err := ioutil.ReadAll(resp.Body)
		/*
			resp.Body.Close()
			if err != nil {
				fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
				os.Exit(1)
			}
			fmt.Printf("%s", b)
		*/
	}
}

//!-

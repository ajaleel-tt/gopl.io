// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 16.
//!+

// Fetch prints the content found at each specified URL.
package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	for _, url := range os.Args[1:] {
		if !strings.HasPrefix(url, "http://") {
			url = "http://" + url
		}
		resp, err := http.Get(url)
		const bufSize = 512
		var buf [bufSize]byte
		if err != nil {
			_, err := fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
			if err != nil {
				return
			}
			os.Exit(1)
		}
		fmt.Printf("HTTP Status: %s\n", resp.Status)
		var numBytesRead int
		var readerErr error
		for {
			numBytesRead, readerErr = resp.Body.Read(buf[:])
			if numBytesRead > 0 {
				_, err := io.Copy(os.Stdout, bytes.NewReader(buf[0:numBytesRead]))
				if err != nil {
					return
				}
			}
			if readerErr == io.EOF {
				break
			}
			if readerErr != nil {
				_, err := fmt.Fprintf(os.Stderr, "fetch: reading %s: %v\n", url, err)
				if err != nil {
					return
				}
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

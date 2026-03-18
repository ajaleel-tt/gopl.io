// Exercise 4.13: Search the Open Movie Database (OMDB) by movie name
// and download the poster image.
// Usage: poster [-apikey key] <movie title>
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

type Movie struct {
	Title    string `json:"Title"`
	Year     string `json:"Year"`
	Poster   string `json:"Poster"`
	Response string `json:"Response"`
	Error    string `json:"Error"`
}

func main() {
	apikey := flag.String("apikey", "", "OMDB API key (or set OMDB_API_KEY env var)")
	flag.Parse()

	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "Usage: poster [-apikey key] <movie title>")
		os.Exit(1)
	}
	title := strings.Join(flag.Args(), " ")

	key := *apikey
	if key == "" {
		key = os.Getenv("OMDB_API_KEY")
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "No API key provided. Use -apikey flag or set OMDB_API_KEY env var.")
		os.Exit(1)
	}

	// Search OMDB for the movie.
	searchURL := "https://omdbapi.com/?apikey=" + key + "&t=" + url.QueryEscape(title)
	resp, err := http.Get(searchURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "OMDB request failed: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		fmt.Fprintf(os.Stderr, "OMDB request failed: %s\n", resp.Status)
		os.Exit(1)
	}
	var movie Movie
	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		resp.Body.Close()
		fmt.Fprintf(os.Stderr, "decoding OMDB response: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()

	if movie.Response == "False" {
		fmt.Fprintf(os.Stderr, "OMDB error: %s\n", movie.Error)
		os.Exit(1)
	}
	if movie.Poster == "N/A" {
		fmt.Fprintf(os.Stderr, "No poster available for %q\n", movie.Title)
		os.Exit(1)
	}

	// Download the poster image.
	resp, err = http.Get(movie.Poster)
	if err != nil {
		fmt.Fprintf(os.Stderr, "downloading poster: %v\n", err)
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		fmt.Fprintf(os.Stderr, "downloading poster: %s\n", resp.Status)
		os.Exit(1)
	}

	// Sanitize filename: replace path separators and null bytes.
	filename := strings.NewReplacer("/", "_", "\\", "_", "\x00", "_").Replace(movie.Title) + ".jpg"
	f, err := os.Create(filename)
	if err != nil {
		resp.Body.Close()
		fmt.Fprintf(os.Stderr, "creating file: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		resp.Body.Close()
		f.Close()
		fmt.Fprintf(os.Stderr, "writing poster: %v\n", err)
		os.Exit(1)
	}
	resp.Body.Close()
	f.Close()

	fmt.Printf("Saved poster for %q (%s) to %s\n", movie.Title, movie.Year, filename)
}

// Exercise 4.12: Build a tool that lets you search xkcd comics offline.
// Usage:
//   xkcd index           — download all comics to ~/.xkcd-index.json
//   xkcd search <terms>  — search comics by keyword
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Comic struct {
	Num        int    `json:"num"`
	Title      string `json:"title"`
	Transcript string `json:"transcript"`
	Alt        string `json:"alt"`
	Img        string `json:"img"`
}

func indexPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	return filepath.Join(home, ".xkcd-index.json")
}

func fetchComic(n int) (*Comic, error) {
	url := fmt.Sprintf("https://xkcd.com/%d/info.0.json", n)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %s for comic %d", resp.Status, n)
	}
	var c Comic
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return &c, nil
}

func fetchLatest() (*Comic, error) {
	resp, err := http.Get("https://xkcd.com/info.0.json")
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("HTTP %s fetching latest comic", resp.Status)
	}
	var c Comic
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return &c, nil
}

func loadIndex(path string) (map[int]Comic, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[int]Comic), nil
		}
		return nil, err
	}
	var comics []Comic
	if err := json.Unmarshal(data, &comics); err != nil {
		return nil, err
	}
	index := make(map[int]Comic, len(comics))
	for _, c := range comics {
		index[c.Num] = c
	}
	return index, nil
}

func saveIndex(path string, index map[int]Comic) error {
	comics := make([]Comic, 0, len(index))
	for _, c := range index {
		comics = append(comics, c)
	}
	data, err := json.MarshalIndent(comics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func buildIndex() {
	path := indexPath()
	index, err := loadIndex(path)
	if err != nil {
		log.Fatalf("loading index: %v", err)
	}
	fmt.Printf("Existing index has %d comics.\n", len(index))

	latest, err := fetchLatest()
	if err != nil {
		log.Fatalf("fetching latest comic: %v", err)
	}
	fmt.Printf("Latest comic is #%d.\n", latest.Num)

	fetched := 0
	for i := 1; i <= latest.Num; i++ {
		if i == 404 {
			continue // comic 404 intentionally returns HTTP 404
		}
		if _, ok := index[i]; ok {
			continue // already indexed
		}
		c, err := fetchComic(i)
		if err != nil {
			log.Printf("warning: skipping comic %d: %v", i, err)
			continue
		}
		index[c.Num] = *c
		fetched++
		if fetched%100 == 0 {
			fmt.Printf("Fetched %d new comics (up to #%d)...\n", fetched, i)
		}
		time.Sleep(10 * time.Millisecond) // rate limit
	}

	if err := saveIndex(path, index); err != nil {
		log.Fatalf("saving index: %v", err)
	}
	fmt.Printf("Done. Index now has %d comics (fetched %d new).\n", len(index), fetched)
}

func search(terms []string) {
	path := indexPath()
	index, err := loadIndex(path)
	if err != nil {
		log.Fatalf("loading index: %v", err)
	}
	if len(index) == 0 {
		fmt.Fprintln(os.Stderr, "Index is empty. Run 'xkcd index' first.")
		os.Exit(1)
	}

	query := strings.ToLower(strings.Join(terms, " "))
	matches := 0
	for _, c := range index {
		if strings.Contains(strings.ToLower(c.Title), query) ||
			strings.Contains(strings.ToLower(c.Transcript), query) ||
			strings.Contains(strings.ToLower(c.Alt), query) {
			matches++
			fmt.Printf("https://xkcd.com/%d/\n", c.Num)
			fmt.Printf("Title: %s\n", c.Title)
			transcript := c.Transcript
			if len(transcript) > 500 {
				transcript = transcript[:500] + "..."
			}
			if transcript != "" {
				fmt.Printf("Transcript: %s\n", transcript)
			}
			fmt.Println()
		}
	}
	fmt.Printf("%d results.\n", matches)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: xkcd index | xkcd search <terms...>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "index":
		buildIndex()
	case "search":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: xkcd search <terms...>")
			os.Exit(1)
		}
		search(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		fmt.Fprintln(os.Stderr, "Usage: xkcd index | xkcd search <terms...>")
		os.Exit(1)
	}
}

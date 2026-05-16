// Exercise 5.13: crawl that mirrors pages locally, restricted to the seed
// URLs' domains. Off-domain URLs are skipped entirely.
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/net/html"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: ex5.13 <url>...")
		os.Exit(1)
	}
	seeds := os.Args[1:]

	allowed := make(map[string]bool)
	for _, s := range seeds {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			allowed[u.Host] = true
		}
	}

	crawler := makeCrawler(allowed, "mirror", 250*time.Millisecond)
	breadthFirst(crawler, seeds)
}

// breadthFirst calls f for each item in the worklist. Any items returned by f
// are added to the worklist. f is called at most once for each item.
func breadthFirst(f func(item string) []string, worklist []string) {
	seen := make(map[string]bool)
	for len(worklist) > 0 {
		items := worklist
		worklist = nil
		for _, item := range items {
			if !seen[item] {
				seen[item] = true
				worklist = append(worklist, f(item)...)
			}
		}
	}
}

// makeCrawler returns a crawl function bound to the given allowed-host set
// and output directory. delay is the pause inserted before each fetch to
// avoid hammering the target server; pass 0 to disable.
func makeCrawler(allowed map[string]bool, mirrorDir string, delay time.Duration) func(string) []string {
	return func(rawurl string) []string {
		u, err := url.Parse(rawurl)
		if err != nil {
			log.Printf("parse %s: %v", rawurl, err)
			return nil
		}
		if !allowed[u.Host] {
			return nil
		}

		if delay > 0 {
			time.Sleep(delay)
		}

		fmt.Println(rawurl)
		body, links, err := fetchAndExtract(rawurl)
		if err != nil {
			log.Print(err)
			return nil
		}

		dest := localPath(mirrorDir, u)
		if err := writeFile(dest, body); err != nil {
			log.Printf("save %s: %v", rawurl, err)
		}
		return links
	}
}

// fetchAndExtract performs the HTTP GET, returns the response body, and (for
// HTML responses) returns absolute links found in the document.
func fetchAndExtract(rawurl string) (body []byte, links []string, err error) {
	resp, err := http.Get(rawurl)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("getting %s: %s", rawurl, resp.Status)
	}

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %v", rawurl, err)
	}

	if !strings.Contains(resp.Header.Get("Content-Type"), "text/html") {
		return body, nil, nil
	}

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return body, nil, fmt.Errorf("parsing %s: %v", rawurl, err)
	}

	base := resp.Request.URL
	var visit func(n *html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					if linkURL, err := base.Parse(a.Val); err == nil {
						links = append(links, linkURL.String())
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return body, links, nil
}

// localPath maps a URL to a local filesystem path under mirrorDir. URLs whose
// path is empty or ends with "/" get an "index.html" suffix.
func localPath(mirrorDir string, u *url.URL) string {
	p := u.Path
	if p == "" || strings.HasSuffix(p, "/") {
		p = path.Join(p, "index.html")
	}
	return filepath.Join(mirrorDir, u.Host, filepath.FromSlash(p))
}

func writeFile(p string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

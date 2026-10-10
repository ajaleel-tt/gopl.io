// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// Mirror (Exercise 8.7) makes a local, browsable copy of a web site. Starting
// from a seed URL, it concurrently fetches every reachable page *within the
// seed's domain*, writes each one to a directory on disk at a path derived from
// its URL, and rewrites the same-domain links inside each page to relative
// paths pointing at the local copies (so the mirror works offline). Links to
// other domains are left untouched.
//
// It reuses the bounded, self-terminating crawl of Exercise 8.6: a counting
// semaphore caps concurrent fetches, and an n-counter of pending worklist sends
// gives clean termination.
//
// Scope notes: it follows and rewrites <a href> links (as the book's crawler
// does), handles HTML pages, and matches domains by exact host. A fuller mirror
// would also rewrite <img>/<link>/<script> and fetch those assets, and
// normalize query strings and trailing slashes.
//
// Usage:
//
//	go run . -o ./out https://example.com
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

func main() {
	outputDir := flag.String("o", "mirror", "output directory")
	parallelism := flag.Int("parallelism", 20, "maximum concurrent fetches")
	flag.Parse()
	if len(flag.Args()) != 1 {
		log.Fatal("usage: mirror [-o dir] [-parallelism n] <url>")
	}
	mirrored, err := mirror(flag.Args()[0], *outputDir, *parallelism)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("mirrored %d pages into %s\n", len(mirrored), *outputDir)
}

// mirror crawls the seed's site, saving link-rewritten copies of every
// reachable same-domain page under outputDir. It returns the set of page URLs
// mirrored and terminates once no work remains.
func mirror(seedURL, outputDir string, parallelism int) ([]string, error) {
	seed, err := url.Parse(seedURL)
	if err != nil {
		return nil, err
	}
	domain := seed.Host
	tokens := make(chan struct{}, parallelism) // bound concurrent fetches

	// fetchAndSave downloads one page, writes a link-rewritten copy to disk,
	// and returns the same-domain URLs it links to (for further crawling).
	fetchAndSave := func(page *url.URL) []*url.URL {
		tokens <- struct{}{}
		defer func() { <-tokens }()

		resp, err := http.Get(page.String())
		if err != nil {
			log.Print(err)
			return nil
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Printf("%s: %s", page, resp.Status)
			return nil
		}
		doc, err := html.Parse(resp.Body)
		if err != nil {
			log.Printf("parsing %s: %v", page, err)
			return nil
		}

		// One walk does both jobs: collect same-domain links to crawl, and
		// rewrite each one in place to a relative path into the local mirror.
		found := rewriteLinks(doc, page, domain)

		if err := save(outputDir, page, doc); err != nil {
			log.Printf("saving %s: %v", page, err)
		}
		return found
	}

	worklist := make(chan []*url.URL)
	n := 1 // pending sends to worklist (starts with the seed)
	go func() { worklist <- []*url.URL{seed} }()

	// main owns seen/mirrored/n — confined to one goroutine, so no locks.
	var mirrored []string
	seen := make(map[string]bool)
	for ; n > 0; n-- {
		for _, u := range <-worklist {
			if u.Host != domain || seen[u.String()] {
				continue // off-domain, or already mirrored
			}
			seen[u.String()] = true
			mirrored = append(mirrored, u.String())
			n++
			go func(u *url.URL) { worklist <- fetchAndSave(u) }(u)
		}
	}
	return mirrored, nil
}

// rewriteLinks walks doc, and for every same-domain <a href> it (1) records the
// absolute target for crawling and (2) rewrites the attribute to a relative
// path pointing at that target's local copy. Off-domain links are left alone.
func rewriteLinks(doc *html.Node, page *url.URL, domain string) []*url.URL {
	var found []*url.URL
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for i := range n.Attr {
				if n.Attr[i].Key != "href" {
					continue
				}
				target, err := page.Parse(n.Attr[i].Val) // resolve against page URL
				if err != nil {
					continue
				}
				target.Fragment = "" // ignore #anchors for identity and paths
				if target.Host == domain && (target.Scheme == "http" || target.Scheme == "https") {
					found = append(found, target)
					n.Attr[i].Val = relativeLocalHref(page, target)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(doc)
	return found
}

// siteRel is u's path within its host directory:
//
//	http://h/           -> index.html
//	http://h/a.html     -> a.html
//	http://h/sub/       -> sub/index.html
//	http://h/sub/c.html -> sub/c.html
func siteRel(u *url.URL) string {
	p := u.Path
	if p == "" {
		p = "/"
	}
	if strings.HasSuffix(p, "/") {
		p += "index.html"
	}
	return strings.TrimPrefix(p, "/")
}

// localPath is the on-disk file for u: outputDir/host/<siteRel>.
func localPath(outputDir string, u *url.URL) string {
	return filepath.Join(outputDir, u.Host, filepath.FromSlash(siteRel(u)))
}

// relativeLocalHref is the href to embed in the page at `from` so it points at
// the local copy of `to`: a relative path between their site-relative paths.
func relativeLocalHref(from, to *url.URL) string {
	rel, err := filepath.Rel(filepath.Dir(siteRel(from)), siteRel(to))
	if err != nil {
		return siteRel(to)
	}
	return filepath.ToSlash(rel)
}

// save renders doc to u's local path, creating parent directories as needed.
func save(outputDir string, u *url.URL, doc *html.Node) error {
	path := localPath(outputDir, u)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return html.Render(f, doc)
}

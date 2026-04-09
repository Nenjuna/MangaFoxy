package scraper

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/html"
)

const userAgent = "Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36"

// client is a shared HTTP client with timeouts.
// http.DefaultClient has no timeout — a hanging remote can block a goroutine forever.
var client = &http.Client{
	Timeout: 15 * time.Second,
}

// ChapterEntry holds a scraped chapter URL and its title.
type ChapterEntry struct {
	URL   string
	Title string
}

// ---------------------------------------------------------------------------
// Chapter list scraping
// ---------------------------------------------------------------------------

// ScrapeChapters POSTs to the MangaOwl chapter AJAX endpoint for the given
// slug and returns a deduplicated list of chapter entries.
// Mirrors Django: requests.post(url, headers=header) → BeautifulSoup → ul > a
func ScrapeChapters(slug string) ([]ChapterEntry, error) {
	targetURL := "https://mangaowl.io/read-1/" + slug + "/ajax/chapters/"

	req, err := http.NewRequest(http.MethodPost, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", "https://mangaowl.io/read-1/"+slug+"/")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", targetURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("POST %s: HTTP %d", targetURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	entries, err := parseChapterList(string(body))
	if err != nil {
		return nil, fmt.Errorf("parse chapters: %w", err)
	}
	return entries, nil
}

// parseChapterList finds the first <ul> in htmlStr and extracts all <a> tags
// within it as chapter entries.
// Mirrors Django: lik_html.find("ul") → chap.find_all("a")
func parseChapterList(htmlStr string) ([]ChapterEntry, error) {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil, err
	}

	ulNode := findFirstUL(doc)
	if ulNode == nil {
		return nil, nil
	}

	var entries []ChapterEntry
	seen := map[string]bool{}

	walkElements(ulNode, func(n *html.Node) {
		if n.Data != "a" {
			return
		}
		href := attrVal(n, "href")
		title := strings.TrimSpace(textContent(n))
		if href != "" && title != "" && !seen[href] {
			entries = append(entries, ChapterEntry{URL: href, Title: title})
			seen[href] = true
		}
	})

	return entries, nil
}

// ---------------------------------------------------------------------------
// Chapter image scraping
// ---------------------------------------------------------------------------

// ScrapeImages fetches a MangaOwl chapter page and returns all image URLs
// found in div.page-break.no-gaps elements.
// Mirrors Django: requests.get(chapter_url) → soup.select("div.page-break.no-gaps") → img[data-src|src]
func ScrapeImages(chapterURL string) ([]string, error) {
	// Build Referer from the chapter URL's origin
	referer := originOf(chapterURL)

	req, err := http.NewRequest(http.MethodGet, chapterURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", chapterURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: HTTP %d", chapterURL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	imgs := parseImages(string(body))
	if len(imgs) == 0 {
		return nil, fmt.Errorf("no images found at %s", chapterURL)
	}
	return imgs, nil
}

// parseImages walks the HTML tree and collects image src/data-src values from
// every div with classes "page-break" and "no-gaps".
// Mirrors Django: chapter_soup.select("div.page-break.no-gaps") → i.img → data-src or src
func parseImages(htmlStr string) []string {
	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}

	var images []string
	walkElements(doc, func(n *html.Node) {
		if n.Data != "div" {
			return
		}
		cls := attrVal(n, "class")
		if !strings.Contains(cls, "page-break") || !strings.Contains(cls, "no-gaps") {
			return
		}
		// First direct <img> child — mirrors Django's i.img
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == html.ElementNode && c.Data == "img" {
				src := attrVal(c, "data-src")
				if src == "" {
					src = attrVal(c, "src")
				}
				if src = strings.TrimSpace(src); src != "" {
					images = append(images, src)
				}
				break
			}
		}
	})

	return images
}

// ---------------------------------------------------------------------------
// HTML tree helpers
// ---------------------------------------------------------------------------

// findFirstUL does a depth-first search and returns the first <ul> element.
func findFirstUL(n *html.Node) *html.Node {
	if n.Type == html.ElementNode && n.Data == "ul" {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := findFirstUL(c); found != nil {
			return found
		}
	}
	return nil
}

// walkElements calls fn for every element node in the subtree rooted at n.
func walkElements(n *html.Node, fn func(*html.Node)) {
	if n.Type == html.ElementNode {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkElements(c, fn)
	}
}

// attrVal returns the value of the named attribute, or "".
func attrVal(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

// textContent concatenates all text nodes within n.
func textContent(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textContent(c))
	}
	return sb.String()
}

// originOf returns the scheme+host of a URL, e.g. "https://mangaowl.io".
func originOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

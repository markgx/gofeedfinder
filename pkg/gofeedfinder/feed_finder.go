package gofeedfinder

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/markgx/gofeedfinder/pkg/gofeedfinder/internal"
)

// MIME type constants for feed detection
const (
	MimeTypeRSS      = "application/rss+xml"
	MimeTypeAtom     = "application/atom+xml"
	MimeTypeJSON     = "application/json"
	MimeTypeFeedJSON = "application/feed+json"
)

// Feed represents a discovered feed with its URL, title, and type.
type Feed struct {
	URL   string // The absolute URL of the feed
	Title string // Optional title of the feed
	Type  string // Feed type: "rss", "atom", or "json"
}

// FindFeeds discovers feed links on the provided web page URL.
// It returns a slice of discovered Feed objects or an error if the page
// cannot be accessed or no feeds are found.
func FindFeeds(url string) ([]Feed, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP request failed with status %d", resp.StatusCode)
	}

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	feeds := ExtractFeedLinks(string(html), url)
	if len(feeds) > 0 {
		return feeds, nil
	}
	return nil, errors.New("no feeds found")
}

// ExtractFeedLinks extracts feed links from an HTML string.
// It searches for <link> elements with appropriate rel and type attributes
// that indicate RSS, Atom, or JSON feeds.
// The url is used to resolve relative URLs to absolute ones.
func ExtractFeedLinks(html string, url string) []Feed {
	feeds := []Feed{}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []Feed{}
	}

	doc.Find("link").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		title, _ := s.Attr("title")
		rel, _ := s.Attr("rel")
		rel = strings.ToLower(rel)
		linkType, _ := s.Attr("type")
		linkType = strings.ToLower(linkType)

		if rel == "alternate" && href != "" {
			var feedType string
			switch linkType {
			case MimeTypeRSS:
				feedType = "rss"
			case MimeTypeAtom:
				feedType = "atom"
			case MimeTypeJSON, MimeTypeFeedJSON:
				feedType = "json"
			}

			if feedType != "" {
				resolvedURL := internal.ResolveFeedURL(href, url)
				feeds = append(feeds, Feed{
					URL:   resolvedURL,
					Title: title,
					Type:  feedType,
				})
			}
		}
	})

	return feeds
}

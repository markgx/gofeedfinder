package gofeedfinder

import (
	"bufio"
	"bytes"
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

// MaxHeadSize limits how much of the HTML head section we'll read (1MB default)
const MaxHeadSize = 1024 * 1024

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

	feeds, err := ExtractFeedLinksFromStream(resp.Body, url)
	if err != nil {
		return nil, err
	}
	
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

// ExtractFeedLinksFromStream extracts feed links from an HTML stream.
// It only reads the HTML head section to optimize memory usage and performance.
// The stream reading stops when </head> is encountered or MaxHeadSize is reached.
func ExtractFeedLinksFromStream(reader io.Reader, baseURL string) ([]Feed, error) {
	limitedReader := io.LimitReader(reader, MaxHeadSize)
	
	headHTML, err := extractHeadSection(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("failed to extract head section: %w", err)
	}
	
	if len(headHTML) == 0 {
		return []Feed{}, nil
	}
	
	return ExtractFeedLinks(headHTML, baseURL), nil
}

// extractHeadSection reads from the input stream and extracts only the HTML head section.
// It stops reading when it encounters </head> or reaches the size limit.
func extractHeadSection(reader io.Reader) (string, error) {
	var headBuffer bytes.Buffer
	scanner := bufio.NewScanner(reader)
	
	inHead := false
	headStartFound := false
	
	for scanner.Scan() {
		line := scanner.Text()
		lineLower := strings.ToLower(line)
		
		// Look for opening <head> tag
		if !headStartFound && strings.Contains(lineLower, "<head") {
			inHead = true
			headStartFound = true
		}
		
		// If we haven't found head yet but found body, give up
		if !headStartFound && strings.Contains(lineLower, "<body") {
			break
		}
		
		// If we're in the head section, write the line to buffer
		if inHead {
			headBuffer.WriteString(line)
			headBuffer.WriteString("\n")
		}
		
		// Look for closing </head> tag
		if inHead && strings.Contains(lineLower, "</head>") {
			break
		}
		
		// If we're in the head section but encounter body without proper </head>, abort
		if inHead && strings.Contains(lineLower, "<body") && !strings.Contains(lineLower, "</head>") {
			return "", nil
		}
	}
	
	if err := scanner.Err(); err != nil {
		return "", err
	}
	
	return headBuffer.String(), nil
}

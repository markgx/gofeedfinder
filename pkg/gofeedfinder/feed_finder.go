package gofeedfinder

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

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

// Options configures feed discovery behavior
type Options struct {
	ScanCommonPaths bool // Whether to scan common feed paths when no feeds found in HTML
	MaxConcurrency  int  // Maximum concurrent requests for path scanning (default: 3)
}

// FindFeeds discovers feed links on the provided web page URL.
// It returns a slice of discovered Feed objects or an error if the page
// cannot be accessed or no feeds are found.
func FindFeeds(url string) ([]Feed, error) {
	return FindFeedsWithOptions(url, Options{})
}

// FindFeedsWithOptions discovers feed links on the provided web page URL with configurable options.
// It returns a slice of discovered Feed objects or an error if the page
// cannot be accessed or no feeds are found.
func FindFeedsWithOptions(url string, opts Options) ([]Feed, error) {
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
	
	// If we found feeds via HTML parsing, return them
	if len(feeds) > 0 {
		return feeds, nil
	}
	
	// If no feeds found and scanning is enabled, try common paths
	if opts.ScanCommonPaths {
		commonFeeds, err := ScanCommonFeedPaths(url, opts.MaxConcurrency)
		if err != nil {
			return nil, err
		}
		if len(commonFeeds) > 0 {
			return commonFeeds, nil
		}
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

// Common feed paths to check, ordered by likelihood
var commonFeedPaths = []string{
	"/feed",
	"/rss",
	"/atom.xml",
	"/index.xml",
	"/rss.xml",
	"/feed.xml",
	"/feeds/all.atom.xml",
	"/feeds/posts/default",
	"/api/rss",
	"/feed.rss",
}

// ScanCommonFeedPaths scans common feed paths on a domain when no feeds are found via HTML parsing.
// It uses controlled concurrency to check multiple paths simultaneously.
func ScanCommonFeedPaths(baseURL string, maxConcurrency int) ([]Feed, error) {
	if maxConcurrency <= 0 {
		maxConcurrency = 3
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base URL: %w", err)
	}

	// Channel to control concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	results := make(chan Feed, len(commonFeedPaths))
	var wg sync.WaitGroup

	// Launch goroutines for each path
	for _, path := range commonFeedPaths {
		wg.Add(1)
		go func(feedPath string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			fullURL := parsedURL.Scheme + "://" + parsedURL.Host + feedPath
			if feed, err := checkFeedURL(fullURL); err == nil && feed != nil {
				results <- *feed
			}
		}(path)
	}

	// Close results channel when all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var feeds []Feed
	for feed := range results {
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// checkFeedURL checks if a URL contains a valid feed by first making a HEAD request,
// then validating the content if it looks promising
func checkFeedURL(url string) (*Feed, error) {
	// First, make a HEAD request to check if the URL exists and get content type
	headResp, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	defer headResp.Body.Close()

	if headResp.StatusCode < 200 || headResp.StatusCode >= 300 {
		return nil, fmt.Errorf("HEAD request failed with status %d", headResp.StatusCode)
	}

	contentType := strings.ToLower(headResp.Header.Get("Content-Type"))
	
	// Check if content type suggests it's a feed
	var feedType string
	if strings.Contains(contentType, "application/rss+xml") || strings.Contains(contentType, "text/xml") {
		feedType = "rss"
	} else if strings.Contains(contentType, "application/atom+xml") {
		feedType = "atom"
	} else if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "application/feed+json") {
		feedType = "json"
	} else {
		// If content type is not clearly a feed type, make a GET request to validate content
		return validateFeedContent(url)
	}

	return &Feed{
		URL:   url,
		Title: "", // We don't extract title from common path scanning
		Type:  feedType,
	}, nil
}

// validateFeedContent makes a GET request and validates that the content is actually a feed
func validateFeedContent(url string) (*Feed, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET request failed with status %d", resp.StatusCode)
	}

	// Read first 1KB to check for feed indicators
	buffer := make([]byte, 1024)
	n, err := resp.Body.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	content := strings.ToLower(string(buffer[:n]))
	
	// Check for feed format indicators in content
	if strings.Contains(content, "<rss") || strings.Contains(content, "<rdf:rdf") {
		return &Feed{URL: url, Title: "", Type: "rss"}, nil
	}
	if strings.Contains(content, "<feed") && strings.Contains(content, "xmlns") {
		return &Feed{URL: url, Title: "", Type: "atom"}, nil
	}
	if strings.Contains(content, `"version"`) && (strings.Contains(content, `"title"`) || strings.Contains(content, `"items"`)) {
		return &Feed{URL: url, Title: "", Type: "json"}, nil
	}

	return nil, errors.New("content does not appear to be a valid feed")
}

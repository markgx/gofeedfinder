package gofeedfinder

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindFeeds_Success(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	mockHTML := `<html><head>
		<link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml" title="Example RSS Feed">
		</head><body></body></html>`

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(mockHTML)),
			Header:     make(http.Header),
		}, nil
	})

	feeds, err := FindFeeds("https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []Feed{
		{
			URL:   "https://example.com/feed.xml",
			Title: "Example RSS Feed",
			Type:  "rss",
		},
	}
	if !cmp.Equal(feeds, expected) {
		t.Errorf("FindFeeds() = %+v, want %+v", feeds, expected)
	}
}

func TestFindFeeds_NoFeeds(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	mockHTML := `<html><head><title>No feeds here</title></head><body></body></html>`

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(mockHTML)),
			Header:     make(http.Header),
		}, nil
	})

	feeds, err := FindFeeds("https://example.com")
	if err == nil || feeds != nil {
		t.Errorf("expected error for no feeds, got feeds=%+v, err=%v", feeds, err)
	}
}

func TestFindFeeds_HTTPError(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return nil, errors.New("mock network error")
	})

	feeds, err := FindFeeds("https://example.com")
	if err == nil || feeds != nil {
		t.Errorf("expected error for HTTP error, got feeds=%+v, err=%v", feeds, err)
	}
}

func TestFindFeeds_Non200Status(t *testing.T) {
	origTransport := http.DefaultTransport
	defer func() { http.DefaultTransport = origTransport }()

	http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
		}, nil
	})

	feeds, err := FindFeeds("https://example.com")
	// io.ReadAll will still succeed, but the HTML will not contain feeds
	if err == nil || feeds != nil {
		t.Errorf("expected error for non-200 status, got feeds=%+v, err=%v", feeds, err)
	}
}

// roundTripperFunc allows us to mock http.RoundTripper inline
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestExtractFeedLinks(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected []Feed
	}{
		{
			name: "RSS feed link",
			html: `<html><head>
				<link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml" title="Example RSS Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/feed.xml",
					Title: "Example RSS Feed",
					Type:  "rss",
				},
			},
		},
		{
			name: "Atom feed link",
			html: `<html><head>
				<link rel="alternate" type="application/atom+xml" href="/atom.xml" title="Example Atom Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/atom.xml",
					Title: "Example Atom Feed",
					Type:  "atom",
				},
			},
		},
		{
			name: "JSON feed link",
			html: `<html><head>
				<link rel="alternate" type="application/feed+json" href="feed.json" title="Example JSON Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/feed.json",
					Title: "Example JSON Feed",
					Type:  "json",
				},
			},
		},
		{
			name: "Multiple feed links",
			html: `<html><head>
				<link rel="alternate" type="application/rss+xml" href="/rss.xml" title="RSS Feed">
				<link rel="alternate" type="application/atom+xml" href="/atom.xml" title="Atom Feed">
				<link rel="alternate" type="application/feed+json" href="/feed.json" title="JSON Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/rss.xml",
					Title: "RSS Feed",
					Type:  "rss",
				},
				{
					URL:   "https://example.com/atom.xml",
					Title: "Atom Feed",
					Type:  "atom",
				},
				{
					URL:   "https://example.com/feed.json",
					Title: "JSON Feed",
					Type:  "json",
				},
			},
		},
		{
			name: "Case insensitive rel and type attributes",
			html: `<html><head>
				<link REL="ALTERNATE" TYPE="APPLICATION/RSS+XML" href="/rss.xml" title="RSS Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/rss.xml",
					Title: "RSS Feed",
					Type:  "rss",
				},
			},
		},
		{
			name:     "No feeds in HTML",
			html:     `<html><head><title>No feeds here</title></head><body></body></html>`,
			baseURL:  "https://example.com",
			expected: []Feed{},
		},
		{
			name:     "Empty HTML",
			html:     "",
			baseURL:  "https://example.com",
			expected: []Feed{},
		},
		{
			name: "Legacy JSON feed type",
			html: `<html><head>
				<link rel="alternate" type="application/json" href="/feed.json" title="JSON Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/feed.json",
					Title: "JSON Feed",
					Type:  "json",
				},
			},
		},
		{
			name: "Feed link without title",
			html: `<html><head>
				<link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/feed.xml",
					Title: "",
					Type:  "rss",
				},
			},
		},
		{
			name: "Non-feed link elements",
			html: `<html><head>
				<link rel="stylesheet" href="/style.css">
				<link rel="canonical" href="https://example.com">
				<link rel="alternate" type="application/rss+xml" href="/feed.xml" title="RSS Feed">
				</head><body></body></html>`,
			baseURL: "https://example.com",
			expected: []Feed{
				{
					URL:   "https://example.com/feed.xml",
					Title: "RSS Feed",
					Type:  "rss",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFeedLinks(tt.html, tt.baseURL)

			if !cmp.Equal(result, tt.expected) {
				t.Errorf("ExtractFeedLinks() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

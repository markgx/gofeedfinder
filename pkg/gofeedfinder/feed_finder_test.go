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
	tests := []struct {
		name       string
		statusCode int
		wantError  string
	}{
		{
			name:       "404 Not Found",
			statusCode: 404,
			wantError:  "HTTP request failed with status 404",
		},
		{
			name:       "500 Internal Server Error",
			statusCode: 500,
			wantError:  "HTTP request failed with status 500",
		},
		{
			name:       "403 Forbidden",
			statusCode: 403,
			wantError:  "HTTP request failed with status 403",
		},
		{
			name:       "301 Moved Permanently",
			statusCode: 301,
			wantError:  "HTTP request failed with status 301",
		},
		{
			name:       "100 Continue",
			statusCode: 100,
			wantError:  "HTTP request failed with status 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origTransport := http.DefaultTransport
			defer func() { http.DefaultTransport = origTransport }()

			http.DefaultTransport = roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(strings.NewReader("response body")),
					Header:     make(http.Header),
				}, nil
			})

			feeds, err := FindFeeds("https://example.com")
			if err == nil {
				t.Errorf("expected error for status %d, got nil", tt.statusCode)
			}
			if feeds != nil {
				t.Errorf("expected nil feeds for status %d, got %+v", tt.statusCode, feeds)
			}
			if err != nil && err.Error() != tt.wantError {
				t.Errorf("expected error %q, got %q", tt.wantError, err.Error())
			}
		})
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

func TestExtractFeedLinksFromStream(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected []Feed
		wantErr  bool
	}{
		{
			name: "RSS feed in head section",
			html: `<html><head>
				<link rel="alternate" type="application/rss+xml" href="https://example.com/feed.xml" title="Example RSS Feed">
				</head><body>lots of body content here</body></html>`,
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
			name: "Multiple feeds in head, stops at body",
			html: `<html><head>
				<link rel="alternate" type="application/rss+xml" href="/rss.xml" title="RSS Feed">
				<link rel="alternate" type="application/atom+xml" href="/atom.xml" title="Atom Feed">
				</head><body>
				<link rel="alternate" type="application/feed+json" href="/feed.json" title="Should not be found">
				</body></html>`,
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
			},
		},
		{
			name:     "No head section",
			html:     `<html><body>No head here</body></html>`,
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
			name: "Head section with no feeds",
			html: `<html><head>
				<title>No feeds here</title>
				<meta charset="utf-8">
				</head><body></body></html>`,
			baseURL:  "https://example.com",
			expected: []Feed{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			result, err := ExtractFeedLinksFromStream(reader, tt.baseURL)

			if tt.wantErr && err == nil {
				t.Errorf("ExtractFeedLinksFromStream() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ExtractFeedLinksFromStream() unexpected error: %v", err)
			}

			if !cmp.Equal(result, tt.expected) {
				t.Errorf("ExtractFeedLinksFromStream() = %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestExtractHeadSection(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name: "Basic head section",
			html: `<html>
<head>
<title>Test</title>
<link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head>
<body>Body content</body>
</html>`,
			expected: `<head>
<title>Test</title>
<link rel="alternate" type="application/rss+xml" href="/feed.xml">
</head>
`,
		},
		{
			name: "Head with attributes",
			html: `<html>
<head lang="en">
<meta charset="utf-8">
</head>
<body>Body</body>`,
			expected: `<head lang="en">
<meta charset="utf-8">
</head>
`,
		},
		{
			name:     "No head section",
			html:     `<html><body>No head</body></html>`,
			expected: "",
		},
		{
			name: "Head section without closing tag (stops at body)",
			html: `<html>
<head>
<title>Test</title>
<body>Body starts here</body>`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.html)
			result, err := extractHeadSection(reader)

			if err != nil {
				t.Errorf("extractHeadSection() unexpected error: %v", err)
			}

			if result != tt.expected {
				t.Errorf("extractHeadSection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

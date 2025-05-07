package internal

import (
	"testing"
)

func TestResolveFeedURL(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		baseURL  string
		expected string
	}{
		{
			name:     "absolute URL with http",
			href:     "http://example.com/feed.xml",
			baseURL:  "https://base.com",
			expected: "http://example.com/feed.xml",
		},
		{
			name:     "absolute URL with https",
			href:     "https://example.com/feed.xml",
			baseURL:  "https://base.com",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "relative URL with leading slash",
			href:     "/feed.xml",
			baseURL:  "https://example.com",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "relative URL without leading slash",
			href:     "feed.xml",
			baseURL:  "https://example.com",
			expected: "https://example.com/feed.xml",
		},
		{
			name:     "relative URL with path",
			href:     "blog/feed.xml",
			baseURL:  "https://example.com",
			expected: "https://example.com/blog/feed.xml",
		},
		{
			name:     "relative URL with base URL having path",
			href:     "feed.xml",
			baseURL:  "https://example.com/blog/",
			expected: "https://example.com/blog/feed.xml",
		},
		{
			name:     "invalid base URL",
			href:     "/feed.xml",
			baseURL:  "://invalid-url",
			expected: "/feed.xml", // Should return original href on error
		},
		{
			name:     "invalid href URL",
			href:     "://invalid-url",
			baseURL:  "https://example.com",
			expected: "://invalid-url", // Should return original href on error
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ResolveFeedURL(tc.href, tc.baseURL)
			if result != tc.expected {
				t.Errorf("ResolveFeedURL(%q, %q) = %q, want %q",
					tc.href, tc.baseURL, result, tc.expected)
			}
		})
	}
}

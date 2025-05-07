package internal

import (
	"net/url"
	"strings"
)

// resolveFeedURL resolves a possibly relative feed URL (href) to an absolute URL using the given baseURL.
// If href is already absolute, it is returned as-is. If resolution fails, the original href is returned.
func ResolveFeedURL(href, baseURL string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	u, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(u).String()
}

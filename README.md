# gofeedfinder

[![Go Reference](https://pkg.go.dev/badge/github.com/markgx/gofeedfinder.svg)](https://pkg.go.dev/github.com/markgx/gofeedfinder)

A command-line utility and Go library designed to detect a website's RSS, Atom, or JSON feed(s) if available.

## CLI

### Installation

```
go install github.com/markgx/gofeedfinder/cmd/gofeedfinder@latest
```

### Usage

```
gofeedfinder [--with-attributes] [--scan-common-paths] <url>
```

### Arguments

- `<url>`: The URL of the website to check for feeds

### Options

- `--with-attributes`: Display additional feed attributes (title and type) along with the URL
- `--scan-common-paths`: Scan common feed paths when no feeds found in HTML (e.g., /feed, /rss, /atom.xml)

### Examples

Basic usage:
```
$ gofeedfinder https://example.com
https://example.com/feed.xml
https://example.com/atom.xml
```

With attributes:
```
$ gofeedfinder --with-attributes https://example.com
https://example.com/feed.xml title=Example Site Feed type=rss
https://example.com/atom.xml title=Example Site type=atom
```

With common path scanning (when no feeds found in HTML):
```
$ gofeedfinder --scan-common-paths https://example.com
https://example.com/feed
https://example.com/rss.xml
```

## Library

### Installation

```
go get github.com/markgx/gofeedfinder
```

### Usage

```go
import "github.com/markgx/gofeedfinder/pkg/gofeedfinder"

// Find feeds from a website URL
feeds, err := gofeedfinder.FindFeeds("https://example.com")
if err != nil {
    // Handle error
}

// Find feeds with additional options
opts := gofeedfinder.Options{
    ScanCommonPaths: true, // Scan common paths when no feeds found in HTML
    MaxConcurrency:  3,    // Maximum concurrent requests for path scanning
}
feeds, err := gofeedfinder.FindFeedsWithOptions("https://example.com", opts)
if err != nil {
    // Handle error
}

// Process the discovered feeds
for _, feed := range feeds {
    fmt.Printf("URL: %s\n", feed.URL)
    fmt.Printf("Title: %s\n", feed.Title)
    fmt.Printf("Type: %s\n", feed.Type) // "rss", "atom", or "json"
}

// Extract feed links from HTML with a base URL
html := `<html>...</html>`
url := "https://example.com"
feeds := gofeedfinder.ExtractFeedLinks(html, url)
```

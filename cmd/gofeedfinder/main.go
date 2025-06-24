package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markgx/gofeedfinder/pkg/gofeedfinder"
)

var version = "dev"

func main() {
	withAttributes := flag.Bool("with-attributes", false, "Display additional feed attributes")
	scanCommonPaths := flag.Bool("scan-common-paths", false, "Scan common feed paths when no feeds found in HTML")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: gofeedfinder [--with-attributes] [--scan-common-paths] [--version] <url>")
		os.Exit(1)
	}

	url := flag.Args()[0]

	opts := gofeedfinder.Options{
		ScanCommonPaths: *scanCommonPaths,
		MaxConcurrency:  3,
	}
	feeds, err := gofeedfinder.FindFeedsWithOptions(url, opts)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	for _, feed := range feeds {
		if *withAttributes {
			fmt.Printf("%s", feed.URL)
			if feed.Title != "" {
				fmt.Printf(" title=%s", feed.Title)
			}
			fmt.Printf(" type=%s\n", feed.Type)
		} else {
			fmt.Println(feed.URL)
		}
	}
}

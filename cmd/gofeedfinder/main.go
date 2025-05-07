package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/markgx/gofeedfinder/pkg/gofeedfinder"
)

func main() {
	withAttributes := flag.Bool("with-attributes", false, "Display additional feed attributes")
	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Println("Usage: gofeedfinder [--with-attributes] <url>")
		os.Exit(1)
	}

	url := flag.Args()[0]

	feeds, err := gofeedfinder.FindFeeds(url)
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

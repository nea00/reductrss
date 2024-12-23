package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gocolly/colly"
	"github.com/mmcdole/gofeed"
)

/*
Parse RSS feed
Read title from post array
Read post urls from post array
Scrape image from post url
Create bluesky post from template

Init post feed by posting all the current posts from the RSS feed
Read latest post date to memory
Scan feed once a day, collect all latter posts into array, create posts from array
Store new date into memory
Rinse and repeat
*/

const FEED string = "http://reductress.com/feed"

type Image struct {
	alt      string
	ref      string
	mimeType string
	size     string
}

type Post struct {
	postType  string
	text      string
	createdAt string
	embed     []Image
}

func main() {
	InitializeFeed()
}

func readFeed() *gofeed.Feed {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL(FEED)
	return feed
}

func createPost() Post {
	return Post{}
}

func createImage(url string, title string) (Image, error) {

	resp, _ := http.Head(url)
	fmt.Println(resp.Header.Get("Content-Type"))
	return Image{
		alt: title,
	}, nil
}

func scrapePostImage(url string, title string) Image {
	collector := colly.NewCollector()
	var img Image
	var err error
	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		if e.Attr("class") == "attachment-default-post size-default-post wp-post-image" || e.Attr("class") == "headshot" {
			imgurl := e.Attr("src")
			img, err = createImage(imgurl, title)
			if err != nil {
				log.Printf("Failed to visit given url: %s", imgurl)
				log.Fatal(err)
			}
		}
	})

	err = collector.Visit(url)
	if err != nil {
		log.Printf("Failed to visit given url: %s", url)
		log.Fatal(err)
	}

	return img

}

func InitializeFeed() {
	newFeed := readFeed()
	for _, item := range newFeed.Items {
		scrapePostImage(item.Link, item.Title)
	}
}

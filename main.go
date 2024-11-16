package main

import (
	"fmt"
	"log"
	"strings"

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

type Post struct {
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

func scrapePostImage(url string) {
	collector := colly.NewCollector()

	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		if e.Attr("class") == "attachment-default-post size-default-post wp-post-image" || e.Attr("class") == "headshot" {
			fmt.Println("found")
			imgurl := e.Attr("src")
			err := collector.Visit(imgurl)
			if err != nil {
				log.Printf("Failed to visit given url: %s", imgurl)
				log.Fatal(err)
			}
		}
	})

	collector.OnResponse(func(r *colly.Response) {
		if strings.Contains(r.Headers.Get("Content-Type"), "image/jpeg") {
			path := "./image.jpeg"
			err := r.Save(path)
			if err != nil {
				log.Printf("Failed to save image")
				log.Fatal(err)
			}
		}
	})

	err := collector.Visit(url)
	if err != nil {
		log.Printf("Failed to visit given url: %s", url)
		log.Fatal(err)
	}
}

func InitializeFeed() {
	newFeed := readFeed()
	fmt.Println(newFeed.Items[0].Published)
	for _, item := range newFeed.Items {

	}
}

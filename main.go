package main

import (
	"fmt"

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

func main() {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("http://reductress.com/feed")
	for _, item := range feed.Items {
		fmt.Println(item.Title)
	}
	fmt.Println(feed)
}

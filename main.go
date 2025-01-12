package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gocolly/colly"
	"github.com/joho/godotenv"
	"github.com/mmcdole/gofeed"

	_ "image/jpeg"
)

type Image struct {
	Alt         string      `json:"alt"`
	ImageBlob   ImageBlob   `json:"image"`
	AspectRatio AspectRatio `json:"aspectRatio"`
}

type AspectRatio struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type ImageBlob struct {
	BlobType string `json:"$type"`
	Ref      Ref    `json:"ref"`
	MimeType string `json:"mimeType"`
	Size     int    `json:"size"`
}

type Ref struct {
	Link string `json:"$link"`
}

type Post struct {
	PostType  string  `json:"$type"`
	Text      string  `json:"text"`
	CreatedAt string  `json:"createdAt"`
	Embed     []Image `json:"embed"`
	Facets    []Facet `json:"facets"`
	Link      string
}

type Facet struct {
	Index    Index     `json:"index"`
	Features []Feature `json:"features"`
}

type Index struct {
	ByteStart int `json:"byteStart"`
	ByteEnd   int `json:"byteEnd"`
}

type Feature struct {
	Type string `json:"$type"`
	Uri  string `json:"uri"`
}

type AuthData struct {
	accessJWT  string
	refreshJWT string
	did        string
}

type responseBlob struct {
	Blob struct {
		Ref  Ref `json:"ref"`
		Size int `json:"size"`
	} `json:"blob"`
}

func main() {
	godotenv.Load()

	auth := authenticate()
	postArray := createPostArray(auth)

	sendPosts(postArray, auth)
}

// Iterates over the array of generated posts posting them to bsky
func sendPosts(postArray []Post, auth AuthData) {

	for _, post := range postArray {
		imageData := map[string]interface{}{
			"$type":  "app.bsky.embed.images",
			"images": post.Embed,
		}

		postData := map[string]interface{}{
			"$type":     post.PostType,
			"text":      post.Text,
			"createdAt": post.CreatedAt,
			"embed":     imageData,
			"facets":    post.Facets,
		}

		data := map[string]interface{}{
			"repo":       auth.did,
			"collection": "app.bsky.feed.post",
			"record":     postData,
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			log.Fatal(err)
		}

		client := &http.Client{}

		req, err := http.NewRequest("POST", "https://bsky.social/xrpc/com.atproto.repo.createRecord", bytes.NewBuffer(jsonData))
		req.Header.Add("Authorization", "Bearer "+auth.accessJWT)
		req.Header.Add("Content-Type", "application/json")
		_, err = client.Do(req)
		if err != nil {
			log.Fatal(err)
		}
	}

}

// Authenticates to bsky and stores relevant auth values to AuthData for later use
func authenticate() AuthData {
	auth := AuthData{}

	url := "https://bsky.social/xrpc/com.atproto.server.createSession"
	handle := os.Getenv("HANDLE")
	password := os.Getenv("PASSWORD")
	contentType := "application/json"

	data := map[string]interface{}{
		"identifier": handle,
		"password":   password,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

	resp, err := http.Post(url, contentType, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Fatal(err)
	}

	var bodyData map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&bodyData)
	if err != nil {
		log.Fatalln(err)
	}

	for key, value := range bodyData {
		if key == "accessJwt" {
			auth.accessJWT = value.(string)
		}

		if key == "refreshJwt" {
			auth.refreshJWT = value.(string)
		}

		if key == "did" {
			auth.did = value.(string)
		}
	}
	return auth
}

// Fetches the latest RSS feed
func readFeed() *gofeed.Feed {
	fp := gofeed.NewParser()
	feed, _ := fp.ParseURL("http://reductress.com/feed")
	return feed
}

// Creates proper image data for the API request
func createImage(url string, title string, auth AuthData) Image {
	headers, err := http.Head(url)
	if err != nil {
		log.Fatal(err)
	}
	mimetype := headers.Header.Get("Content-Type")
	ref, size := getImageBlob(url, mimetype, auth)
	width, height := getImageAspectRatio(url)

	blob := ImageBlob{
		BlobType: "blob",
		Ref:      ref,
		MimeType: mimetype,
		Size:     size,
	}

	aspectRatio := AspectRatio{
		Width:  width,
		Height: height,
	}

	image := Image{
		Alt:         title,
		ImageBlob:   blob,
		AspectRatio: aspectRatio,
	}

	return image
}

// Communicates with bsky to get the appropriate blob data for the image
func getImageBlob(imgUrl, mimetype string, auth AuthData) (Ref, int) {
	resp, err := http.Get(imgUrl)
	if err != nil {
		log.Fatal(err)
	}

	img := resp.Body
	pds := "https://bsky.social/xrpc/com.atproto.repo.uploadBlob"
	client := &http.Client{}

	req, err := http.NewRequest("POST", pds, img)
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Add("Content-Type", mimetype)
	req.Header.Add("Authorization", "Bearer "+auth.accessJWT)
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var bodyData responseBlob
	json.Unmarshal(body, &bodyData)
	if err != nil {
		log.Fatal(err)
	}

	ref := Ref{
		Link: bodyData.Blob.Ref.Link,
	}

	return ref, bodyData.Blob.Size

}

// Returns width and height of the image used for the post
func getImageAspectRatio(imgUrl string) (int, int) {
	resp, err := http.Get(imgUrl)
	if err != nil {
		log.Fatal(err)
	}

	image, _, err := image.Decode(resp.Body)
	if err != nil {
		fmt.Println(imgUrl)
		log.Fatal(err)
	}

	bounds := image.Bounds()

	width := bounds.Dx()
	height := bounds.Dy()

	return width, height
}

// Fetches and returns the image tied to the post using the given url from the RSS feed
func scrapePostImage(url string, title string, auth AuthData) Image {
	collector := colly.NewCollector()
	var img Image
	var err error
	collector.OnHTML("img[src]", func(e *colly.HTMLElement) {
		if e.Attr("class") == "attachment-default-post size-default-post wp-post-image" || e.Attr("class") == "headshot" {
			imgurl := e.Attr("src")
			img = createImage(imgurl, title, auth)
		}
	})

	err = collector.Visit(url)
	if err != nil {
		log.Printf("Failed to visit given url: %s", url)
		log.Fatal(err)
	}

	return img

}

// Returns all the posts that are newer than the last saved timestamp, if a timestamp doesnt exist then return all posts and generate the initial timestamp
func createPostArray(auth AuthData) []Post {
	init := false
	currentTime := time.Now()

	if _, err := os.Stat("date"); err != nil {
		os.WriteFile("date", []byte(currentTime.Format("Mon, 02 Jan 2006 15:04:05 -0700")), 0644)
		init = true
	}

	post := Post{}
	postArray := []Post{}
	newFeed := readFeed()

	for _, item := range newFeed.Items {
		if !init && compareTime(item.Published) {
			break
		}

		img := scrapePostImage(item.Link, item.Title, auth)

		var imgSlice []Image
		imgSlice = append(imgSlice, img)

		byteStart := len(item.Title + ": ")
		byteEnd := byteStart + len(item.Link)

		currentTime = time.Now()
		timestamp := currentTime.Format(time.RFC3339Nano)
		post = Post{
			PostType:  "app.bsky.feed.post",
			Text:      item.Title + ": " + item.Link,
			CreatedAt: timestamp,
			Embed:     imgSlice,
			Link:      item.Link,
			Facets: []Facet{
				{
					Index:    Index{ByteStart: byteStart, ByteEnd: byteEnd},
					Features: []Feature{{Type: "app.bsky.richtext.facet#link", Uri: item.Link}},
				},
			},
		}
		postArray = append(postArray, post)
	}
	return postArray
}

// Compares the saved timestamp and the timestamp of the post
func compareTime(timestamp string) bool {

	layout := "Mon, 02 Jan 2006 15:04:05 -0700"

	file, err := os.Open("date")
	if err != nil {
		log.Fatal(err)
	}

	fileDate := ""
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		fileDate = scanner.Text()
	}

	postTime, err := time.Parse(layout, timestamp)
	if err != nil {
		log.Fatal(err)
	}

	fileTime, err := time.Parse(layout, fileDate)
	if err != nil {
		log.Fatal(err)
	}

	return postTime.Before(fileTime)
}

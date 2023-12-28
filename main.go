package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/gocolly/colly"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync" // Import sync package for safe concurrent access to imageData
)

var (
	imageDataMutex sync.Mutex // Mutex for safe concurrent access to imageData
	imageData      string     // Global variable for storing image data
)

type pageInfo struct {
	StatusCode int
	Links      map[string]int
	Image      string
}

func handler(w http.ResponseWriter, r *http.Request) {
	baseURL := "http://localhost:7171/?url="
	ImageURL := "http://localhost:7171/image"
	URL := r.URL.Query().Get("url")
	if URL == "" {
		log.Println("missing URL argument")
		return
	}
	log.Println("visiting", URL)

	c := colly.NewCollector()

	p := &pageInfo{Links: make(map[string]int)}

	// count links
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link != "" {
			p.Links[baseURL+link]++
		}
	})
	// extract image
	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		imageLink := e.Request.AbsoluteURL(e.Attr("src"))
		response, err := http.Get(imageLink)
		if err != nil {
			log.Println("error:", err)
			return
		}
		defer response.Body.Close()

		// Check for JPEG images and encode:
		if strings.Contains(imageLink, ".jpg") {
			imgBytes, err := ioutil.ReadAll(response.Body) // Read the image content
			if err != nil {
				log.Println("error reading image:", err)
				return
			}
			// Encode the image bytes
			encodedImage := base64.StdEncoding.EncodeToString(imgBytes)
			// Lock the mutex while updating imageData to ensure safe concurrent access
			imageDataMutex.Lock()
			imageData = encodedImage
			imageDataMutex.Unlock()
		}
	})
	// extract status code
	c.OnResponse(func(r *colly.Response) {
		log.Println("response received", r.StatusCode)
		p.StatusCode = r.StatusCode
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("error:", r.StatusCode, err)
		p.StatusCode = r.StatusCode
	})

	c.Visit(URL)

	// Lock the mutex while accessing imageData to ensure safe concurrent access
	imageDataMutex.Lock()
	defer imageDataMutex.Unlock()

	// Include the base64-encoded image in the JSON response
	p.Image = ImageURL

	// dump results
	b, err := json.Marshal(p)
	if err != nil {
		log.Println("failed to serialize response:", err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	// Lock the mutex while accessing imageData to ensure safe concurrent access
	imageDataMutex.Lock()
	defer imageDataMutex.Unlock()

	if imageData != "" {
		// Decode the base64-encoded image data
		decodedImage, err := base64.StdEncoding.DecodeString(imageData)
		if err != nil {
			http.Error(w, "Error decoding image data", http.StatusInternalServerError)
			return
		}

		// Set the Content-Type header to indicate that you're sending an image
		w.Header().Set("Content-Type", "image/jpeg") // Change this to the appropriate content type if it's not JPEG

		// Write the decoded image data to the response writer
		_, err = w.Write(decodedImage)
		if err != nil {
			http.Error(w, "Error writing image data to response", http.StatusInternalServerError)
			return
		}
	} else {
		// Handle the case where no image data is available
		http.Error(w, "No image data available", http.StatusNotFound)
	}
}

func main() {
	addr := ":7171"

	http.HandleFunc("/", handler)
	http.HandleFunc("/image", imageHandler)

	log.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

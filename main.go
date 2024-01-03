package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gocolly/colly"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
)

var baseURL = "http://localhost:7171/?url="
var ImageURL = "http://localhost:7171/image"

var (
	imageDataMutex sync.Mutex // Mutex for safe concurrent access to imageData
	imageData      []byte     // Global variable for storing compressed image data
)

type pageInfo struct {
	StatusCode int
	Links      map[string]int
	Image      string
}

func handler(w http.ResponseWriter, r *http.Request) {
	var err error

	URL := r.URL.Query().Get("url")
	if URL == "" {
		log.Println("missing URL argument")
		return
	}

	log.Println("visiting", URL)

	c := colly.NewCollector()

	p := &pageInfo{Links: make(map[string]int)}

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Request.AbsoluteURL(e.Attr("href"))
		if link != "" {
			p.Links[baseURL+link]++
		}
	})

	c.OnHTML("img[src]", func(e *colly.HTMLElement) {
		imageLink := e.Request.AbsoluteURL(e.Attr("src"))
		response, err := http.Get(imageLink)
		if err != nil {
			log.Println("error:", err)
			return
		}

		defer response.Body.Close()

		if strings.Contains(imageLink, ".jpg") {
			imgBytes, err := ioutil.ReadAll(response.Body)
			if err != nil {
				log.Println("error reading image:", err)
				return
			}

			encodedImage := base64.StdEncoding.EncodeToString(imgBytes)

			// Compress the base64-encoded image data with gzip
			compressedImage, compressedSize, err := compressWithGzip([]byte(encodedImage))
			fmt.Printf("Compressed Data Length: %d character\n", compressedSize)
			if err != nil {
				log.Println("error compressing image:", err)
				return
			}

			imageDataMutex.Lock()
			imageData = compressedImage
			imageDataMutex.Unlock()
		}
	})

	c.OnResponse(func(r *colly.Response) {
		log.Println("response received", r.StatusCode)
		p.StatusCode = r.StatusCode
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("error:", r.StatusCode, err)
		p.StatusCode = r.StatusCode
	})

	c.Visit(URL)

	imageDataMutex.Lock()
	defer imageDataMutex.Unlock()

	p.Image = ImageURL

	b, err := json.Marshal(p)
	if err != nil {
		log.Println("failed to serialize response:", err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(b)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
	imageDataMutex.Lock()
	defer imageDataMutex.Unlock()

	if len(imageData) > 0 {
		// Decompress the compressed base64-encoded image data with gzip
		decompressedData, decompressedSize, err := decompressWithGzip(imageData)
		if err != nil {
			http.Error(w, "Error decompressing image data", http.StatusInternalServerError)
			return
		}
		// Print the size of decompressed data
		fmt.Printf("Decompressed Data Length: %d character\n", decompressedSize)

		// Decode the base64-encoded image data
		decodedImage, err := base64.StdEncoding.DecodeString(string(decompressedData))
		if err != nil {
			http.Error(w, "Error decoding image data", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/jpeg")
		_, err = w.Write(decodedImage)
		if err != nil {
			http.Error(w, "Error writing image data to response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "No image data available", http.StatusNotFound)
	}
}

func compressWithGzip(data []byte) ([]byte, int, error) {
	var compressedData bytes.Buffer
	writer, gzipErr := gzip.NewWriterLevel(&compressedData, gzip.BestCompression)
	if gzipErr != nil {
		return nil, 0, gzipErr
	}

	_, err := writer.Write(data)
	if err != nil {
		writer.Close()
		return nil, 0, err
	}

	err = writer.Close()
	if err != nil {
		return nil, 0, err
	}

	// Calculate the size of compressed data
	compressedSize := compressedData.Len()

	return compressedData.Bytes(), compressedSize, nil
}

func decompressWithGzip(data []byte) ([]byte, int, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	defer reader.Close()

	decompressedData, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, 0, err
	}

	decompressedSize := len(decompressedData)

	return decompressedData, decompressedSize, nil
}

func main() {
	addr := ":7171"

	http.HandleFunc("/", handler)
	http.HandleFunc("/image", imageHandler)

	log.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

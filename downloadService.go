package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gocolly/colly"
)

// Global Constants
const (
	downloadPath = "nDownloads"
	logFileName  = "nDownloaded.log"
)

func main() {
	useTorProxy := false // Set to true or false

	banner()
	fmt.Print(color.BlueString(" [i] Welcome to nDownloader! Press Enter to start... "))
	_, _ = fmt.Scanln()

	logFile, err := setupLogging()
	if err != nil {
		color.Set(color.FgRed, color.Bold)
		fmt.Println("Error setting up logging:", err)
		return
	}
	defer logFile.Close()

	c := colly.NewCollector(
		colly.Async(true),
	)

	if useTorProxy {
		c.WithTransport(&http.Transport{
			Proxy: http.ProxyURL(&url.URL{
				Scheme: "socks5",
				Host:   "127.0.0.1:9150",
			}),
		})
	}

	c.OnResponse(func(r *colly.Response) {
		nID, pageNumber := extractIDAndPage(r.Request.URL.String())
		fileName := filepath.Join(downloadPath, fmt.Sprintf("%s_%d.jpg", nID, pageNumber))
		if err := ioutil.WriteFile(fileName, r.Body, 0644); err != nil {
			fmt.Println("Error saving image:", err)
		} else {
			logDownload(logFile, fmt.Sprintf("%s page %d", nID, pageNumber))
		}
	})

	var errorOccurred bool
	c.OnError(func(r *colly.Response, err error) {
		fmt.Println("Error on page:", r.Request.URL.String(), "Error:", err)
		errorOccurred = true
	})

	for {
		nID := generateRandomID()
		pageNumber := 1
		errorOccurred = false

		for !errorOccurred {
			nURL := fmt.Sprintf("https://i.nhentai.net/galleries/%s/%d.jpg", nID, pageNumber)
			c.Visit(nURL)
			c.Wait()
			if errorOccurred {
				break
			}
			pageNumber++
		}
	}
}

func extractIDAndPage(rawURL string) (string, int) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		return "", 0
	}

	segments := strings.Split(path.Clean(parsedURL.Path), "/")
	if len(segments) >= 3 {
		nID := segments[len(segments)-2]
		pageNumberStr := segments[len(segments)-1]
		pageNumberStr = strings.TrimSuffix(pageNumberStr, path.Ext(pageNumberStr)) // Remove .jpg
		pageNumber, err := strconv.Atoi(pageNumberStr)
		if err != nil {
			fmt.Println("Error converting page number:", err)
			return nID, 0
		}
		return nID, pageNumber
	}
	return "", 0
}

func setupLogging() (*os.File, error) {
	if _, err := os.Stat(downloadPath); os.IsNotExist(err) {
		if err := os.Mkdir(downloadPath, os.ModePerm); err != nil {
			return nil, err
		}
	}
	return os.OpenFile(filepath.Join(downloadPath, logFileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func setupHttpClient(useTorProxy bool) *http.Client {
	// Simple HTTP client setup. Modify here to include Tor proxy if required.
	return &http.Client{}
}

func generateRandomID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func isValidImage(resp *http.Response) bool {
	return resp.StatusCode == 200 && resp.Header.Get("Content-Type") == "image/jpeg"
}

func saveImage(resp *http.Response, fileName, operativeSystem string) error {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(fileName, body, 0644)
}

func logDownload(logFile *os.File, nID string) {
	logEntry := fmt.Sprintf("[%s] Downloaded: %s\n", time.Now().Format("02 Jan 2006 - 15:04:05"), nID)
	_, _ = logFile.WriteString(logEntry)
}

func banner() {
	color.Set(color.FgWhite, color.Bold)
	fmt.Println("_________ .__  .__                ")
	fmt.Println("\\_   ___ \\|  | |  |   ____ ___.__.")
	fmt.Println("/    \\  \\/|  | |  | _/ __ <   |  |")
	fmt.Println("\\     \\___|  |_|  |_\\  ___/\\___  |")
	fmt.Println(" \\______  /____/____/\\___  > ____|")
	fmt.Println("        \\/               \\/\\/     ")
	color.Unset()
}

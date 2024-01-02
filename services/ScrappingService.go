package services

import "sync"

var (
	imageDataMutex sync.Mutex // Mutex for safe concurrent access to imageData
	imageData      []byte     // Global variable for storing compressed image data
)

// ScrappingService is a service that scrapes a website and returns the links and images found on that website
type ScrappingService struct {
}

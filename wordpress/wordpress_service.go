package wordpress

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// WordPressService manages the interaction with a WordPress site via the REST API
type WordPressService struct {
	siteURL      string
	username     string
	appPassword  string
	client       *http.Client
	isConnected  bool
	mutex        sync.Mutex
}

// Page represents a WordPress page
type Page struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Slug    string `json:"slug"`
	Link    string `json:"link"`
}

// PageList represents a list of WordPress pages
type PageList []Page

// NewWordPressService creates a new instance of WordPressService
func NewWordPressService() *WordPressService {
	return &WordPressService{
		client: &http.Client{},
	}
}

// Connect establishes a connection to the WordPress site
func (s *WordPressService) Connect(siteURL, username, appPassword string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Validate inputs
	if siteURL == "" {
		return fmt.Errorf("site URL cannot be empty")
	}
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if appPassword == "" {
		return fmt.Errorf("application password cannot be empty")
	}

	// Normalize site URL (ensure it ends with /)
	if !strings.HasSuffix(siteURL, "/") {
		siteURL = siteURL + "/"
	}

	// Validate URL format
	_, err := url.Parse(siteURL)
	if err != nil {
		return fmt.Errorf("invalid site URL: %w", err)
	}

	// Test connection by making a simple request to the WordPress REST API
	testURL := fmt.Sprintf("%swp-json/wp/v2/pages?per_page=1", siteURL)
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth header
	req.SetBasicAuth(username, appPassword)

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to WordPress site: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to authenticate with WordPress site: HTTP %d", resp.StatusCode)
	}

	// Store credentials
	s.siteURL = siteURL
	s.username = username
	s.appPassword = appPassword
	s.isConnected = true

	return nil
}

// IsConnected returns the connection status
func (s *WordPressService) IsConnected() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isConnected
}

// GetPages fetches a list of pages from the WordPress site
func (s *WordPressService) GetPages() (PageList, error) {
	s.mutex.Lock()
	if !s.isConnected {
		s.mutex.Unlock()
		return nil, fmt.Errorf("not connected to WordPress site")
	}
	siteURL := s.siteURL
	username := s.username
	appPassword := s.appPassword
	s.mutex.Unlock()

	// Create request URL
	requestURL := fmt.Sprintf("%swp-json/wp/v2/pages?per_page=100", siteURL)
	
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth header
	req.SetBasicAuth(username, appPassword)

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pages: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch pages: HTTP %d", resp.StatusCode)
	}

	// Parse response
	var pages []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return nil, fmt.Errorf("failed to parse pages response: %w", err)
	}

	// Convert to PageList
	var pageList PageList
	for _, page := range pages {
		id, _ := page["id"].(float64)
		title, _ := page["title"].(map[string]interface{})
		titleRendered, _ := title["rendered"].(string)
		content, _ := page["content"].(map[string]interface{})
		contentRendered, _ := content["rendered"].(string)
		slug, _ := page["slug"].(string)
		link, _ := page["link"].(string)

		pageList = append(pageList, Page{
			ID:      int(id),
			Title:   titleRendered,
			Content: contentRendered,
			Slug:    slug,
			Link:    link,
		})
	}

	return pageList, nil
}

// GetPageContent fetches the content of a specific page
func (s *WordPressService) GetPageContent(pageID int) (string, error) {
	s.mutex.Lock()
	if !s.isConnected {
		s.mutex.Unlock()
		return "", fmt.Errorf("not connected to WordPress site")
	}
	siteURL := s.siteURL
	username := s.username
	appPassword := s.appPassword
	s.mutex.Unlock()

	// Create request URL
	requestURL := fmt.Sprintf("%swp-json/wp/v2/pages/%d", siteURL, pageID)
	
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add basic auth header
	req.SetBasicAuth(username, appPassword)

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page content: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch page content: HTTP %d", resp.StatusCode)
	}

	// Parse response
	var page map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return "", fmt.Errorf("failed to parse page response: %w", err)
	}

	// Extract content
	content, ok := page["content"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid page content format")
	}

	contentRendered, ok := content["rendered"].(string)
	if !ok {
		return "", fmt.Errorf("invalid page content format")
	}

	return contentRendered, nil
}

// UpdatePageContent updates the content of a specific page
func (s *WordPressService) UpdatePageContent(pageID int, newContent string) error {
	s.mutex.Lock()
	if !s.isConnected {
		s.mutex.Unlock()
		return fmt.Errorf("not connected to WordPress site")
	}
	siteURL := s.siteURL
	username := s.username
	appPassword := s.appPassword
	s.mutex.Unlock()

	// Create request URL
	requestURL := fmt.Sprintf("%swp-json/wp/v2/pages/%d", siteURL, pageID)
	
	// Create request body
	requestBody := map[string]interface{}{
		"content": newContent,
	}
	
	bodyJSON, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to create request body: %w", err)
	}

	// Create request
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.SetBasicAuth(username, appPassword)
	req.Header.Set("Content-Type", "application/json")

	// Make the request
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update page content: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update page content: HTTP %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

// Disconnect closes the connection to the WordPress site
func (s *WordPressService) Disconnect() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.isConnected = false
	s.siteURL = ""
	s.username = ""
	s.appPassword = ""
}
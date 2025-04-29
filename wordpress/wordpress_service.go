package wordpress

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// WordPressService manages the interaction with a WordPress site via the REST API
type WordPressService struct {
	siteURL            string
	username           string
	appPassword        string
	client             *http.Client
	isConnected        bool
	mutex              sync.Mutex
	savedSites         []SavedSite
	currentSiteName    string
	siteChangeCallback func()
}

// Page represents a WordPress page
type Page struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Slug    string `json:"slug"`
	Link    string `json:"link"`
}

// SavedSite represents a saved WordPress site with credentials
type SavedSite struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	AppPassword string `json:"appPassword"` // This will be stored encrypted
}

// PageList represents a list of WordPress pages
type PageList []Page

// NewWordPressService creates a new instance of WordPressService
func NewWordPressService() *WordPressService {
	service := &WordPressService{
		client:           &http.Client{
			Timeout: 30 * time.Second, // <-- Add a reasonable timeout (e.g., 30 seconds)
		},
		savedSites:       []SavedSite{},
		currentSiteName:  "",
		siteChangeCallback: nil,
	}
	
	// Load saved sites
	service.LoadSavedSites()
	
	return service
}

// GetConfigDir returns the directory for storing configuration files
func (s *WordPressService) GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	configDir := filepath.Join(homeDir, ".wordpress-inference")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}
	
	return configDir, nil
}

func (s *WordPressService) GetCurrentSiteName() string {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    return s.currentSiteName
}

// SaveSite saves a site's credentials to the configuration file
func (s *WordPressService) SaveSite(name, siteURL, username, appPassword string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Check if site with this name already exists
	for i, site := range s.savedSites {
		if site.Name == name {
			// Update existing site
			s.savedSites[i].URL = siteURL
			s.savedSites[i].Username = username
			s.savedSites[i].AppPassword = encryptPassword(appPassword)
			s.currentSiteName = name
			return s.saveSitesToFile()
		}
	}
	
	// Add new site
	s.savedSites = append(s.savedSites, SavedSite{
		Name:        name,
		URL:         siteURL,
		Username:    username,
		AppPassword: encryptPassword(appPassword),
	})
	s.currentSiteName = name
	if s.siteChangeCallback != nil {
		s.siteChangeCallback()
	}
	
	return s.saveSitesToFile()
}

// saveSitesToFile saves the sites to a JSON file
func (s *WordPressService) saveSitesToFile() error {
	configDir, err := s.GetConfigDir()
	if err != nil {
		return err
	}
	
	sitesFile := filepath.Join(configDir, "saved_sites.json")
	
	data, err := json.MarshalIndent(s.savedSites, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal saved sites: %w", err)
	}
	
	if err := os.WriteFile(sitesFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write saved sites file: %w", err)
	}
	
	return nil
}

// LoadSavedSites loads saved sites from the configuration file
func (s *WordPressService) LoadSavedSites() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	configDir, err := s.GetConfigDir()
	if err != nil {
		return err
	}
	
	sitesFile := filepath.Join(configDir, "saved_sites.json")
	
	// Check if file exists
	if _, err := os.Stat(sitesFile); os.IsNotExist(err) {
		// File doesn't exist, initialize with empty list
		s.savedSites = []SavedSite{}
		return nil
	}
	
	data, err := os.ReadFile(sitesFile)
	if err != nil {
		return fmt.Errorf("failed to read saved sites file: %w", err)
	}
	
	if err := json.Unmarshal(data, &s.savedSites); err != nil {
		return fmt.Errorf("failed to unmarshal saved sites: %w", err)
	}
	
	return nil
}

// GetSavedSites returns the list of saved sites
func (s *WordPressService) GetSavedSites() []SavedSite {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Return a copy to avoid race conditions
	sites := make([]SavedSite, len(s.savedSites))
	copy(sites, s.savedSites)
	
	return sites
}

// GetSavedSite returns a saved site by name
func (s *WordPressService) GetSavedSite(name string) (SavedSite, bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	for _, site := range s.savedSites {
		if site.Name == name {
			// Return a copy with decrypted password
			return SavedSite{
				Name:        site.Name,
				URL:         site.URL,
				Username:    site.Username,
				AppPassword: decryptPassword(site.AppPassword),
			}, true
		}
	}
	
	return SavedSite{}, false
}

// DeleteSavedSite deletes a saved site by name
func (s *WordPressService) DeleteSavedSite(name string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	for i, site := range s.savedSites {
		if site.Name == name {
			// Remove site from slice
			s.savedSites = append(s.savedSites[:i], s.savedSites[i+1:]...)
			return s.saveSitesToFile()
		}
	}
	
	return fmt.Errorf("site with name '%s' not found", name)
}

// Simple encryption/decryption functions (for demonstration purposes)
// In a production environment, use a more secure encryption method

func encryptPassword(password string) string {
	// Simple base64 encoding for demonstration
	return base64.StdEncoding.EncodeToString([]byte(password))
}

func decryptPassword(encrypted string) string {
	// Simple base64 decoding for demonstration
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return ""
	}
	return string(data)
}

// Connect establishes a connection to the WordPress site
func (s *WordPressService) Connect(siteURL, username, appPassword string) error {
	s.mutex.Lock() // Lock at start
	log.Println("wpService.Connect: Lock acquired.")

	// Use flags and variables to manage state across the lock release
	var callbackToCall func() = nil
	siteNameFound := ""
	connectionSuccessful := false // Track success to ensure unlock on error paths

	// Defer unlock ensures it happens even on early error returns
	defer func() {
		// Only unlock if connection wasn't successful OR if we didn't need a callback
		// If connection was successful AND callback was needed, it was unlocked manually.
		if !connectionSuccessful || callbackToCall == nil {
			log.Println("wpService.Connect: Releasing lock via defer.")
			s.mutex.Unlock()
		} else {
			log.Println("wpService.Connect: Lock was released manually before callback, defer skipped unlock.")
		}
	}()

	// ... (Input validation) ...
	if siteURL == "" || username == "" || appPassword == "" {
		log.Println("wpService.Connect: Input validation failed.")
		// Return error (defer will unlock)
		return fmt.Errorf("site URL, username, and application password cannot be empty")
	}
	log.Println("wpService.Connect: Input validated.")

	// Normalize site URL (ensure it ends with /)
	if !strings.HasSuffix(siteURL, "/") {
		siteURL = siteURL + "/"
	}

	// Validate URL format
	_, err := url.Parse(siteURL)
	if err != nil {
		return fmt.Errorf("invalid site URL: %w", err)
	}
	log.Printf("wpService.Connect: Normalized URL: %s", siteURL)

	// Test connection by making a simple request to the WordPress REST API
	testURL := fmt.Sprintf("%swp-json/wp/v2/pages?per_page=1", siteURL)
	log.Printf("wpService.Connect: Creating request for test URL: %s", testURL)
	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		log.Printf("wpService.Connect: Error creating request: %v", err)
		return fmt.Errorf("failed to create request: %w", err)
	}
	log.Println("wpService.Connect: Request created.")

	// Add basic auth header
	req.SetBasicAuth(username, appPassword)
	log.Println("wpService.Connect: Basic auth set.")

	// Make the request
	log.Printf("wpService.Connect: Executing client.Do(req). Timeout: %v", s.client.Timeout)
	resp, err := s.client.Do(req)
	// Check for network errors first
	if err != nil {
		log.Printf("wpService.Connect: client.Do(req) failed. Error: %v", err)
		// Return error (defer will unlock)
		return fmt.Errorf("failed to connect to WordPress site: %w", err)
	}
	// Ensure body is closed even if status check fails
	defer resp.Body.Close()
	log.Printf("wpService.Connect: client.Do(req) finished. Response Status: %s", resp.Status)


	// Check response status code
	log.Printf("wpService.Connect: Response status code: %d", resp.StatusCode)
	if resp.StatusCode != http.StatusOK {
		// Return error (defer will unlock)
		return fmt.Errorf("failed to authenticate with WordPress site: HTTP %d", resp.StatusCode)
	}

	// --- If we reach here, connection is successful ---
	connectionSuccessful = true // Mark as successful for defer logic
	log.Println("wpService.Connect: Connection successful. Storing credentials.")
	s.siteURL = siteURL
	s.username = username
	s.appPassword = appPassword
	s.isConnected = true

	// Check for saved site and prepare for callback
	for _, site := range s.savedSites {
		if site.URL == siteURL && site.Username == username {
			s.currentSiteName = site.Name
			siteNameFound = site.Name
			if s.siteChangeCallback != nil {
				callbackToCall = s.siteChangeCallback // Get ref
			}
			break
		}
	}

	// If we need to call the callback, unlock manually FIRST
	if callbackToCall != nil {
		log.Println("wpService.Connect: Releasing lock manually before callback.")
		s.mutex.Unlock() // Manual unlock

		log.Printf("wpService.Connect: Calling siteChangeCallback for site: %s", siteNameFound)
		callbackToCall() // Call the callback (lock is released)
		log.Println("wpService.Connect: siteChangeCallback finished.")
	} else {
		log.Println("wpService.Connect: No callback needed or no matching site found.")
		// If no callback, the defer will handle the unlock
	}

	log.Println("wpService.Connect: Returning nil (success).")
	return nil // Success!
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
	s.currentSiteName = ""
	
	if s.siteChangeCallback != nil {
		s.siteChangeCallback()
	}
}

// SetSiteChangeCallback sets a function to be called when the current site changes
func (s *WordPressService) SetSiteChangeCallback(callback func()) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.siteChangeCallback = callback
}
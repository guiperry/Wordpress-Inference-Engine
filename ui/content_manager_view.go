package ui

import (
	"fmt"
	"log"

	"Inference_Engine/inference"
	"Inference_Engine/wordpress"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ContentManagerView represents the WordPress content manager view
type ContentManagerView struct {
	container        *container.Split
	wpService        *wordpress.WordPressService
	inferenceService *inference.InferenceService
	window           fyne.Window

	// Connection UI elements
	siteURLEntry    *widget.Entry
	usernameEntry   *widget.Entry
	passwordEntry   *widget.Entry
	connectButton   *widget.Button
	statusLabel     *widget.Label

	// Content UI elements
	pageList        *widget.List
	contentEditor   *widget.Entry
	saveButton      *widget.Button
	
	// Data
	pages           wordpress.PageList
	selectedPageID  int
}

// NewContentManagerView creates a new WordPress content manager view
func NewContentManagerView(inferenceService *inference.InferenceService, window fyne.Window) *ContentManagerView {
	view := &ContentManagerView{
		wpService:        wordpress.NewWordPressService(),
		inferenceService: inferenceService,
		window:           window,
		pages:            wordpress.PageList{},
		selectedPageID:   -1,
	}
	view.initialize()
	return view
}

// initialize initializes the content manager view
func (v *ContentManagerView) initialize() {
	// Create connection UI elements
	v.siteURLEntry = widget.NewEntry()
	v.siteURLEntry.SetPlaceHolder("WordPress Site URL (e.g., https://example.com/)")
	
	v.usernameEntry = widget.NewEntry()
	v.usernameEntry.SetPlaceHolder("Username")
	
	v.passwordEntry = widget.NewPasswordEntry()
	v.passwordEntry.SetPlaceHolder("Application Password")
	
	v.connectButton = widget.NewButton("Connect", func() {
		v.connectToWordPress()
	})
	
	v.statusLabel = widget.NewLabel("Status: Disconnected")
	
	// Create content UI elements
	v.pageList = widget.NewList(
		func() int {
			return len(v.pages)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Page Title")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(v.pages) {
				obj.(*widget.Label).SetText(v.pages[id].Title)
			}
		},
	)
	
	v.pageList.OnSelected = func(id widget.ListItemID) {
		if id < len(v.pages) {
			v.loadPageContent(v.pages[id].ID)
		}
	}
	
	v.contentEditor = widget.NewMultiLineEntry()
	v.contentEditor.SetPlaceHolder("Page content will appear here...")
	v.contentEditor.Wrapping = fyne.TextWrapWord
	
	v.saveButton = widget.NewButton("Save Content", func() {
		v.savePageContent()
	})
	v.saveButton.Disable() // Disable until a page is selected
	
	// Create layout
	connectionForm := container.NewVBox(
		widget.NewLabel("WordPress Connection"),
		widget.NewLabel("Site URL:"),
		v.siteURLEntry,
		widget.NewLabel("Username:"),
		v.usernameEntry,
		widget.NewLabel("Application Password:"),
		v.passwordEntry,
		v.connectButton,
		v.statusLabel,
	)
	
	contentContainer := container.NewVSplit(
		container.NewBorder(
			widget.NewLabel("Pages:"),
			nil, nil, nil,
			container.NewScroll(v.pageList),
		),
		container.NewBorder(
			widget.NewLabel("Content:"),
			container.NewHBox(v.saveButton),
			nil, nil,
			container.NewScroll(v.contentEditor),
		),
	)
	contentContainer.SetOffset(0.3) // 30% for page list, 70% for content editor
	
	// Main layout
	v.container = container.NewHSplit(
		connectionForm,
		contentContainer,
	)
	v.container.SetOffset(0.25) // 25% for connection form, 75% for content
}

// connectToWordPress connects to the WordPress site
func (v *ContentManagerView) connectToWordPress() {
	siteURL := v.siteURLEntry.Text
	username := v.usernameEntry.Text
	password := v.passwordEntry.Text
	
	if siteURL == "" || username == "" || password == "" {
		dialog.ShowError(fmt.Errorf("please fill in all connection fields"), v.window)
		return
	}
	
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Connecting", "Connecting to WordPress site...", v.window)
	progress.Show()
	
	// Connect in a goroutine to avoid blocking the UI
	go func() {
		defer progress.Hide()
		
		err := v.wpService.Connect(siteURL, username, password)
		if err != nil {
			log.Printf("Error connecting to WordPress: %v", err)
			dialog.ShowError(fmt.Errorf("failed to connect: %w", err), v.window)
			v.statusLabel.SetText("Status: Connection failed")
			return
		}
		
		// Update status
		v.statusLabel.SetText("Status: Connected")
		
		// Fetch pages
		v.fetchPages()
	}()
}

// fetchPages fetches the list of pages from the WordPress site
func (v *ContentManagerView) fetchPages() {
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Fetching", "Fetching pages...", v.window)
	progress.Show()
	
	// Fetch pages in a goroutine
	go func() {
		// Fetch data first
		pages, err := v.wpService.GetPages()

		// --- UI Updates Start Here ---
		// Hide the progress dialog *before* potentially showing another dialog or updating UI
		progress.Hide()

		// Now handle results and update UI
		if err != nil {
			log.Printf("Error fetching pages: %v", err)
			// Show error dialog *after* hiding progress
			dialog.ShowError(fmt.Errorf("failed to fetch pages: %w", err), v.window)
			return // Exit goroutine after showing error
		}

		// Update non-dialog UI elements (Ideally queue these)
		v.pages = pages
		v.pageList.Refresh() // Refresh the list data

		// Show success dialog *after* progress is hidden
		dialog.ShowInformation("Success", fmt.Sprintf("Fetched %d pages", len(pages)), v.window)

	}() // End of goroutine
}

// loadPageContent loads the content of the selected page
func (v *ContentManagerView) loadPageContent(pageID int) {
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Loading", "Loading page content...", v.window)
	progress.Show()
	
	// Load content in a goroutine
	go func() {
		// Perform the content loading logic
		content, err := v.wpService.GetPageContent(pageID)

		// --- UI Updates Start Here ---
		// Hide the progress dialog *before* potentially showing another dialog or updating UI
		progress.Hide()

		if err != nil {
			log.Printf("Error loading page content: %v", err)
			// Show error dialog *after* hiding progress
			dialog.ShowError(fmt.Errorf("failed to load page content: %w", err), v.window)
			return // Exit goroutine
		}

		// Update content editor and state (Ideally queue these)
		v.contentEditor.SetText(content)
		v.selectedPageID = pageID
		v.saveButton.Enable()

	}() // End of goroutine
}

// savePageContent saves the edited content back to the WordPress site
func (v *ContentManagerView) savePageContent() {
	if v.selectedPageID < 0 {
		dialog.ShowError(fmt.Errorf("no page selected"), v.window)
		return
	}
	
	content := v.contentEditor.Text
	
	// Confirm before saving
	dialog.ShowConfirm("Save Changes", "Are you sure you want to save these changes to the WordPress page?", func(confirmed bool) {
		if !confirmed {
			return
		}
		
		// Show progress dialog
		progress := dialog.NewProgressInfinite("Saving", "Saving page content...", v.window)
		progress.Show()
		
		// Save content in a goroutine
		go func() {
			// Perform the save operation
			err := v.wpService.UpdatePageContent(v.selectedPageID, content)

			// --- UI Updates Start Here ---
			// Hide the progress dialog *before* potentially showing another dialog
			progress.Hide()

			if err != nil {
				log.Printf("Error saving page content: %v", err)
				// Show error dialog *after* hiding progress
				dialog.ShowError(fmt.Errorf("failed to save page content: %w", err), v.window)
				return // Exit goroutine
			}

			// Show success dialog *after* hiding progress
			dialog.ShowInformation("Success", "Page content saved successfully", v.window)
		}() // End of goroutine
	}, v.window)
}

// Container returns the container for the content manager view
func (v *ContentManagerView) Container() fyne.CanvasObject {
	return v.container
}

// GetSelectedPageID returns the ID of the currently selected page
func (v *ContentManagerView) GetSelectedPageID() int {
	return v.selectedPageID
}

// GetSelectedPageTitle returns the title of the currently selected page
func (v *ContentManagerView) GetSelectedPageTitle() string {
	for _, page := range v.pages {
		if page.ID == v.selectedPageID {
			return page.Title
		}
	}
	return ""
}

// GetPageByID returns a page by its ID
func (v *ContentManagerView) GetPageByID(id int) *wordpress.Page {
	for i, page := range v.pages {
		if page.ID == id {
			return &v.pages[i]
		}
	}
	return nil
}

// SelectPageByID selects a page in the list by its ID
func (v *ContentManagerView) SelectPageByID(id int) {
	for i, page := range v.pages {
		if page.ID == id {
			v.pageList.Select(i)
			break
		}
	}
}

// SelectPageByIndex selects a page in the list by its index
func (v *ContentManagerView) SelectPageByIndex(index int) {
	if index >= 0 && index < len(v.pages) {
		v.pageList.Select(index)
	}
}

// GetPageCount returns the number of pages
func (v *ContentManagerView) GetPageCount() int {
	return len(v.pages)
}
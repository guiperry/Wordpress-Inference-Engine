package ui

import (
	"fmt"
	"log"

	"sync" // Import sync package
	"Inference_Engine/inference"
	"Inference_Engine/wordpress"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// ContentManagerView represents the WordPress content manager view
type ContentManagerView struct {
	container        fyne.CanvasObject
	wpService        *wordpress.WordPressService
	inferenceService *inference.InferenceService
	window           fyne.Window

	// Status UI element
	statusLabel *widget.Label

	// Content UI elements
	pageList          *widget.List
	contentEditor     *widget.Entry
	saveButton        *widget.Button
	loadContentButton *widget.Button
	previewImage      *canvas.Image // For displaying image previews

	// Data
	pages          wordpress.PageList
	selectedPageID int

	// Reference to content generator view (will be set after creation)
	contentGeneratorView *ContentGeneratorView
	dialogMutex          sync.Mutex // ADDED: Mutex for dialog operations
}

// RefreshStatus updates the status label based on the current service connection state.
func (v *ContentManagerView) RefreshStatus() {
	if v.wpService == nil {
		log.Println("ContentManagerView: WordPress service is nil, cannot refresh status.")
		v.statusLabel.SetText("Status: Error (Service unavailable)")
		return
	}

	if v.wpService.IsConnected() {
		siteName := v.wpService.GetCurrentSiteName() // Assuming you add a method like this to your service
		if siteName == "" {
			siteName = "Connected Site" // Fallback if name isn't stored/retrieved
		}
		v.statusLabel.SetText(fmt.Sprintf("Status: Connected to %s", siteName))
		// --- ADD THIS: Call fetchPages when connected ---
		// Only fetch if the list is currently empty to avoid redundant calls
		// every time the tab is selected.
		if len(v.pages) == 0 {
			log.Println("ContentManagerView: Connected and page list empty, fetching pages...")
			go v.fetchPages() // Fetch in the background
		} else {
			log.Println("ContentManagerView: Connected, pages already loaded.")
		}
		// --- END OF ADDED CODE ---
	} else {
		v.statusLabel.SetText("Status: Disconnected")
		// Clear page list if disconnected
		if len(v.pages) > 0 { // Only clear if not already empty
			log.Println("ContentManagerView: Disconnected, clearing page list.")
			v.pages = nil
			v.pageList.Refresh()
			v.contentEditor.SetText("")
			v.saveButton.Disable()
			v.loadContentButton.Disable()
			v.selectedPageID = -1 // Reset selected ID
		}
	}
	v.statusLabel.Refresh()
}

// NewContentManagerView creates a new WordPress content manager view
func NewContentManagerView(wpService *wordpress.WordPressService, inferenceService *inference.InferenceService, window fyne.Window) *ContentManagerView {
	view := &ContentManagerView{
		wpService:        wpService,
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
	// Create status label
	v.statusLabel = widget.NewLabel("Wordpress Connection Status: Initializing...")

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
			// Load preview if link is available
			if v.pages[id].Link != "" {
				v.loadPagePreview(v.pages[id].Link)
			}
		}
	}

	v.contentEditor = widget.NewMultiLineEntry()
	v.contentEditor.SetPlaceHolder("Page content will appear here...")
	v.contentEditor.Wrapping = fyne.TextWrapWord

	v.saveButton = widget.NewButton("Save Content", func() {
		v.savePageContent()
	})
	v.saveButton.Disable() // Disable until a page is selected

	v.loadContentButton = widget.NewButton("Load to Generator", func() {
		v.loadSelectedContentToGenerator()
	})
	v.loadContentButton.Disable() // Disable until a page is selected

	// Initialize preview image
	v.previewImage = &canvas.Image{
		FillMode:  canvas.ImageFillOriginal,
		ScaleMode: canvas.ImageScaleFastest,
	}

	v.previewImage.SetMinSize(fyne.NewSize(600, 350)) // Example: Set minimum width 200, height 150

	// Create layout
	editorAndPreview := container.NewVSplit(
		container.NewScroll(v.contentEditor),
		container.NewBorder(
			widget.NewLabel("Preview:"),
			nil, nil, nil,
			container.NewScroll(v.previewImage),
		),
	)
	editorAndPreview.Offset = 0.2 // 20% editor, 80% preview

	rightPanel := container.NewBorder(
		widget.NewLabel("Content:"),
		container.NewHBox(layout.NewSpacer(), v.saveButton, v.loadContentButton),
		nil,
		nil,
		editorAndPreview,
	)

	contentContainer := container.NewHSplit(
		container.NewBorder(
			widget.NewLabel("Pages:"),
			nil, nil, nil,
			container.NewScroll(v.pageList),
		),
		rightPanel,
	)
	contentContainer.SetOffset(0.2) // 20% for page list, 80% for content editor

	// Main layout with status label at top
	v.container = container.NewBorder(
		v.statusLabel,
		nil,
		nil,
		nil,
		contentContainer,
	)
	v.RefreshStatus()
}

// fetchPages fetches the list of pages from the WordPress site
func (v *ContentManagerView) fetchPages() {
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Fetching", "Fetching pages...", v.window)
	progress.Show()

	// Fetch pages in a goroutine
	go func() {
		// Fetch data first
		pages, err := v.wpService.GetPages(1, 10) // Get first batch with 10 pages

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

		const maxDisplayLength = 5000 // Adjust as needed, slightly less than capacity
		displayContent := content
		if len(content) > maxDisplayLength {
			log.Printf("Truncating content for display (original length: %d)", len(content))
			displayContent = content[:maxDisplayLength] + "\n... (Content Truncated)"
		}

		log.Printf("Loading content for page %d, display length: %d", pageID, len(displayContent))

		v.contentEditor.SetText(displayContent) // Use truncated content
		v.selectedPageID = pageID
		v.saveButton.Enable()
		v.loadContentButton.Enable()

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

// loadSelectedContentToGenerator fetches the *text* content for the selected page,
// sends it to the generator view, and then clears the manager view.
func (v *ContentManagerView) loadSelectedContentToGenerator() {
	if v.selectedPageID < 0 {
		dialog.ShowError(fmt.Errorf("no page selected"), v.window)
		return
	}
	if v.contentGeneratorView == nil {
		dialog.ShowError(fmt.Errorf("content generator view not available"), v.window)
		return
	}

	// Find the selected page details (needed for title)
	var selectedPage *wordpress.Page
	for i := range v.pages {
		if v.pages[i].ID == v.selectedPageID {
			selectedPage = &v.pages[i]
			break
		}
	}
	if selectedPage == nil {
		dialog.ShowError(fmt.Errorf("selected page details not found"), v.window)
		return
	}

	// Fetch the actual content (text) on demand
	progress := dialog.NewProgressInfinite("Loading Content", "Fetching page content for generator...", v.window)
	progress.Show()

	go func() {
		defer progress.Hide()
		content, err := v.wpService.GetPageContent(v.selectedPageID) // Still need this function!
		if err != nil {
			log.Printf("Error loading page content for generator: %v", err)
			dialog.ShowError(fmt.Errorf("failed to load content for '%s': %w", selectedPage.Title, err), v.window)
			return
		}

		// Add the fetched content to the generator
		v.contentGeneratorView.AddSourceContent(
			selectedPage.Title,
			content, // The actual text content
			"WordPress",
			selectedPage.ID,
			false,
		)

		// --- Add code to clear the UI elements ---
		v.contentEditor.SetText("")    // Clear the editor
		v.previewImage.Resource = nil  // Clear the preview image resource
		v.previewImage.Refresh()       // Refresh the image widget
		v.selectedPageID = -1          // Reset selected ID
		v.saveButton.Disable()         // Disable save button
		v.loadContentButton.Disable()  // Disable load button
		v.pageList.UnselectAll()       // Unselect item in the list
		log.Println("ContentManagerView: Cleared editor and preview after loading to generator.")
		// --- End of added code ---

		dialog.ShowInformation("Content Added", fmt.Sprintf("Added content of '%s' to content generator and cleared manager view.", selectedPage.Title), v.window)
	}()
}

// SetContentGeneratorView sets the reference to the content generator view
func (v *ContentManagerView) SetContentGeneratorView(generatorView *ContentGeneratorView) {
	v.contentGeneratorView = generatorView
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

// loadPagePreview triggers the screenshot capture and updates the image widget.
func (v *ContentManagerView) loadPagePreview(pageURL string) {
	if pageURL == "" {
		v.previewImage.Resource = nil // Clear image if no URL
		v.previewImage.Refresh()
		return
	}

	// Show progress indicator
	v.dialogMutex.Lock() // Lock before showing dialog
	progress := dialog.NewProgressInfinite("Loading Preview", "Capturing page screenshot...", v.window)
	progress.Show()
	v.dialogMutex.Unlock() // Unlock after showing

	v.previewImage.Resource = nil // Clear previous image while loading
	v.previewImage.Refresh()

	go func() {
		// Don't use defer for hiding; hide explicitly before showing other dialogs.
		// defer progress.Hide()

		imgBytes, err := v.wpService.GetPageScreenshot(pageURL)
		// Hide progress *before* potentially showing an error dialog.

		v.dialogMutex.Lock() // Lock before hiding/showing next dialog
		progress.Hide()
		if err != nil {
			log.Printf("Error getting page screenshot: %v", err)
			dialog.ShowError(fmt.Errorf("failed to load preview for %s: %w", pageURL, err), v.window)
			v.dialogMutex.Unlock() // Unlock after showing error
			v.previewImage.Resource = nil // Ensure image is cleared on error
			v.previewImage.Refresh()

			return
		}

		// Create Fyne resource from image bytes
		imgResource := fyne.NewStaticResource(fmt.Sprintf("preview_%d.png", v.selectedPageID), imgBytes) // Use PNG if GetPageScreenshot returns PNG

		// Update the image widget
		// Unlock here if no error occurred
		v.dialogMutex.Unlock()
		v.previewImage.Resource = imgResource
		v.previewImage.Refresh()
	}()
}

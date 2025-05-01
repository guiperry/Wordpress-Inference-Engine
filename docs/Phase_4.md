Let's outline how to implement Phase 4 (Screenshot Preview) in place of the editor:

1. Update Dependencies:

You'll need to add a headless browser library. chromedp is a popular choice.

bash
go get github.com/chromedp/chromedp
Important: This approach requires the user running your application to have Google Chrome or Chromium installed and accessible in their system's PATH. You should document this requirement clearly.

2. Modify ContentManagerView Struct:

Remove editor-related fields and add fields for the preview.

go
// In ui/content_manager_view.go

import (
	// ... other imports
	"fyne.io/fyne/v2/canvas" // Import canvas for Image
	// ... chromedp imports will be needed in the service layer
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
	// contentEditor     *widget.Entry // REMOVED
	// saveButton        *widget.Button // REMOVED
	loadContentButton *widget.Button // Keep this to send content to Generator
	refreshPreviewBtn *widget.Button // New button for preview refresh
	previewImage      *canvas.Image  // New image widget for preview

	// Data
	pages             wordpress.PageList
	selectedPageID    int
	selectedPageURL   string // Store the public URL of the selected page
	// fullPageContent   string // Still needed if 'Load to Generator' fetches it on demand
	// contentChunks     []string // REMOVED (No longer chunking for display)
	// currentChunkIndex int      // REMOVED
	// pageLinkContainer *fyne.Container // REMOVED

	// Reference to content generator view
	contentGeneratorView *ContentGeneratorView
}
3. Update initialize Function:

Reconfigure the layout to use the previewImage and refreshPreviewBtn instead of the editor and save button.

go
// In ui/content_manager_view.go initialize()

func (v *ContentManagerView) initialize() {
	// ... (statusLabel, pageList creation remains similar) ...

	v.pageList.OnSelected = func(id widget.ListItemID) {
		if id < len(v.pages) {
			v.selectedPageID = v.pages[id].ID
			v.selectedPageURL = v.pages[id].Link // Assuming Page struct has the public Link/URL
			v.loadPagePreview(v.selectedPageURL) // Trigger preview load
			v.loadContentButton.Enable()         // Enable button once a page is selected
		} else {
			// Clear selection state
			v.selectedPageID = -1
			v.selectedPageURL = ""
			v.previewImage.Image = nil // Clear image
			v.previewImage.Refresh()
			v.loadContentButton.Disable()
		}
	}

	// REMOVED: v.contentEditor = widget.NewMultiLineEntry() ...
	// REMOVED: v.saveButton = widget.NewButton("Save Content", ...)

	v.loadContentButton = widget.NewButton("Load Content to Generator", func() {
		// This button now needs to fetch the content on demand
		v.loadSelectedContentToGenerator()
	})
	v.loadContentButton.Disable() // Disable initially

	v.refreshPreviewBtn = widget.NewButton("Refresh Preview", func() {
		if v.selectedPageURL != "" {
			v.loadPagePreview(v.selectedPageURL)
		}
	})

	// Setup the preview image widget
	v.previewImage = &canvas.Image{
		FillMode: canvas.ImageFillContain, // Or ImageFillStretch, ImageFillOriginal
		ScaleMode: canvas.ImageScaleFastest, // Or ImageScaleSmooth
	}
	// Optional: Set a placeholder or minimum size
	// v.previewImage.SetMinSize(fyne.NewSize(300, 200))

	// Layout for the preview area (right side)
	previewArea := container.NewBorder(
		widget.NewLabel("Preview:"), // Top label
		container.NewHBox( // Bottom buttons
			layout.NewSpacer(),
			v.refreshPreviewBtn,
			v.loadContentButton,
		),
		nil, // Left
		nil, // Right
		container.NewScroll(v.previewImage), // Center: Scrollable preview image
	)

	// Split between page list and preview area
	contentContainer := container.NewHSplit( // Use HSplit for side-by-side
		container.NewBorder(
			widget.NewLabel("Pages:"),
			nil, nil, nil,
			container.NewScroll(v.pageList),
		),
		previewArea, // Use the new previewArea layout
	)
	contentContainer.SetOffset(0.3) // Adjust split ratio as needed

	// Main layout
	v.container = container.NewBorder(
		v.statusLabel,
		nil, nil, nil,
		contentContainer,
	)
	v.RefreshStatus()
}
4. Implement Screenshot Logic (in wordpress service):

You'll need a function in your WordPressService (or a dedicated PreviewService) to handle the screenshot capture using chromedp.

go
// In wordpress/wordpress_service.go (or a new preview service)

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/chromedp/cdproto/emulation"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

// GetPageScreenshot captures a screenshot of a given URL.
// Returns PNG image bytes or an error.
func (s *WordPressService) GetPageScreenshot(pageURL string) ([]byte, error) {
	if pageURL == "" {
		return nil, fmt.Errorf("page URL cannot be empty")
	}

	log.Printf("Attempting to capture screenshot for: %s", pageURL)

	// --- Chromedp Setup ---
	// Consider creating context options once, e.g., disabling headless for debugging
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		// chromedp.Flag("headless", false), // Uncomment for debugging
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true), // Often needed in containerized environments
		chromedp.Flag("disable-dev-shm-usage", true), // Often needed in containerized environments
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	defer cancelAlloc()

	// Create context
	ctx, cancelCtx := chromedp.NewContext(allocCtx, chromedp.WithLogf(log.Printf))
	defer cancelCtx()

	// Create a timeout context
	timeoutCtx, cancelTimeout := context.WithTimeout(ctx, 60*time.Second) // 60-second timeout
	defer cancelTimeout()
	// --- End Chromedp Setup ---


	var buf []byte
	// Capture screenshot
	err := chromedp.Run(timeoutCtx,
		// Set viewport size if desired (optional)
		// emulation.SetDeviceMetricsOverride(1280, 800, 1, false),
		chromedp.Navigate(pageURL),
		// Wait for page load (adjust selector or wait time as needed)
		// chromedp.WaitVisible(`body`, chromedp.ByQuery), // Wait for body tag
		chromedp.Sleep(3*time.Second), // Simple wait, adjust as needed
		// Capture screenshot
		chromedp.FullScreenshot(&buf, 90), // 90% quality JPEG, use 0 for PNG
		// Or capture specific element:
		// chromedp.Screenshot(`#main-content`, &buf, chromedp.NodeVisible, chromedp.ByID),
	)

	if err != nil {
		log.Printf("Chromedp error capturing screenshot for %s: %v", pageURL, err)
		return nil, fmt.Errorf("failed to capture screenshot: %w", err)
	}

	if len(buf) == 0 {
		log.Printf("Captured empty screenshot for %s", pageURL)
		return nil, fmt.Errorf("captured empty screenshot")
	}

	log.Printf("Successfully captured screenshot for %s (size: %d bytes)", pageURL, len(buf))
	return buf, nil
}

// Ensure your Page struct includes the public URL ('Link')
// You might need to adjust GetPages to fetch this field if it doesn't already.
// Example modification in GetPages:
// req.Param("fields", "id,title,status,link") // Add 'link' to fields

5. Implement UI Logic for Preview:

Create functions in ContentManagerView to trigger the screenshot and update the UI.

go
// In ui/content_manager_view.go

import (
	// ... other imports
	"bytes" // Needed to create reader for image resource
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	// ...
)


// loadPagePreview triggers the screenshot capture and updates the image widget.
func (v *ContentManagerView) loadPagePreview(pageURL string) {
	if pageURL == "" {
		v.previewImage.Image = nil // Clear image if no URL
		v.previewImage.Refresh()
		return
	}

	// Show progress indicator
	progress := dialog.NewProgressInfinite("Loading Preview", "Capturing page screenshot...", v.window)
	progress.Show()
	v.previewImage.Image = nil // Clear previous image while loading
	v.previewImage.Refresh()

	go func() {
		defer progress.Hide() // Ensure progress dialog is hidden

		imgBytes, err := v.wpService.GetPageScreenshot(pageURL)
		if err != nil {
			log.Printf("Error getting page screenshot: %v", err)
			dialog.ShowError(fmt.Errorf("failed to load preview for %s: %w", pageURL, err), v.window)
			v.previewImage.Image = nil // Ensure image is cleared on error
			v.previewImage.Refresh()
			return
		}

		// Create Fyne resource from image bytes
		imgResource := fyne.NewStaticResource(fmt.Sprintf("preview_%d.png", v.selectedPageID), imgBytes) // Use PNG if GetPageScreenshot returns PNG

		// Update the image widget
		v.previewImage.Image = imgResource
		v.previewImage.Refresh()
	}()
}

// loadSelectedContentToGenerator fetches the *text* content for the selected page
// and sends it to the generator view.
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
		dialog.ShowInformation("Content Added", fmt.Sprintf("Added content of '%s' to content generator", selectedPage.Title), v.window)
	}()
}

6. Adjust GetPages:

Make sure your GetPages function in the wordpress service fetches the link field for each page, as this is needed for the public URL to capture the screenshot.

go
// In wordpress/wordpress_service.go GetPages() function (example)
func (s *WordPressService) GetPages(page, perPage int) (PageList, int, error) {
	// ... existing setup ...
	req := s.client.Pages.List(context.Background(), &wordpress.PageListOptions{
		ListOptions: wordpress.ListOptions{
			Page:    page,
			PerPage: perPage,
			Fields:  "id,title,status,link", // Ensure 'link' is requested
		},
		// Add other options like Status: "publish", OrderBy: "title", Order: "asc" if needed
	})
	// ... rest of the function ...
}
Summary of Changes:

Removed the widget.MultiLineEntry and "Save" button.
Added widget.Image for preview and a "Refresh Preview" button.
Layout changed to show the page list on the left and the preview area on the right.
Added chromedp dependency and screenshot logic (GetPageScreenshot).
Selecting a page now triggers loadPagePreview.
"Load Content to Generator" button remains but fetches the text content via GetPageContent on demand before sending it to the generator tab.
Requires Chrome/Chromium installation on the user's machine.
This approach provides the visual preview you wanted, sacrificing the direct editing capability within this tab. Remember to clearly communicate the Chrome/Chromium dependency to your users.
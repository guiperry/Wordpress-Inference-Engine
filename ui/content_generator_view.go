package ui

import (
	"fmt"
	"io"
	"log"
	"path/filepath"
	"strings"

	"Inference_Engine/inference"
	"Inference_Engine/wordpress"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ContentGeneratorView represents the content generator view
type ContentGeneratorView struct {
	container        *container.Split
	wpService        *wordpress.WordPressService
	inferenceService *inference.InferenceService
	window           fyne.Window

	// Source content UI elements
	sourceList         *widget.List
	addSourceButton    *widget.Button
	removeSourceButton *widget.Button

	// Generation UI elements
	promptEntry      *widget.Entry
	generateButton   *widget.Button
	resultOutput     *widget.Entry
	saveToFileButton *widget.Button
	saveToWPButton   *widget.Button

	// Data
	sourceContents      []SourceContent
	selectedSourceIndex int
}

// SourceContent represents a source content item
type SourceContent struct {
	Title   string
	Content string
	Source  string // "WordPress", "File", etc.
	ID      int    // WordPress page ID or other identifier
	IsSample bool
}

// NewContentGeneratorView creates a new content generator view
func NewContentGeneratorView(wpService *wordpress.WordPressService, inferenceService *inference.InferenceService, window fyne.Window) *ContentGeneratorView {
	view := &ContentGeneratorView{
		wpService:           wpService,
		inferenceService:    inferenceService,
		window:              window,
		sourceContents:      []SourceContent{},
		selectedSourceIndex: -1,
	}
	view.initialize()
	return view
}

// Initializes the content generator view
func (v *ContentGeneratorView) initialize() {
	// Create source content UI elements
	v.sourceList = widget.NewList(
		func() int {
			return len(v.sourceContents)
		},
		func() fyne.CanvasObject {
			check := widget.NewCheck("Sample", nil) // Checkbox for "Is Sample?"
			label := widget.NewLabel("Template Source")
			// Use HBox for layout. Spacer pushes label left if needed, or just box them.
			// Add padding or adjust layout as needed for aesthetics.
			return container.NewHBox(check, label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(v.sourceContents) {
				hbox := obj.(*fyne.Container)
				check := hbox.Objects[0].(*widget.Check)
				label := hbox.Objects[1].(*widget.Label)
	
				label.SetText(v.sourceContents[id].Title)
				check.SetChecked(v.sourceContents[id].IsSample)
	
				// --- Handle Checkbox Changes ---
				// Use OnChanged within UpdateItem to capture the correct 'id'
				check.OnChanged = func(checked bool) {
					// Prevent index out of bounds if list refreshes during interaction
					if id < len(v.sourceContents) {
						v.sourceContents[id].IsSample = checked
						log.Printf("Source '%s' marked as sample: %t", v.sourceContents[id].Title, checked)
						// No list refresh needed here, just update the data model
					}
				}
			}
		},
	)

	v.sourceList.OnSelected = func(id widget.ListItemID) {
		v.selectedSourceIndex = id
		v.removeSourceButton.Enable()
	}

	v.addSourceButton = widget.NewButton("Add Source", func() {
		v.showAddSourceDialog()
	})

	v.removeSourceButton = widget.NewButton("Remove Source", func() {
		v.removeSourceContent()
	})
	v.removeSourceButton.Disable()

	// Create generation UI elements
	v.promptEntry = widget.NewMultiLineEntry()
	v.promptEntry.SetPlaceHolder("Enter a prompt or topic for the AI to generate content about...")
	v.promptEntry.Wrapping = fyne.TextWrapWord
	v.promptEntry.SetMinRowsVisible(10) // <--- Add this line

	v.generateButton = widget.NewButton("Generate Content", func() {
		v.generateContent()
	})

	v.resultOutput = widget.NewMultiLineEntry()
	v.resultOutput.SetPlaceHolder("Generated content will appear here...")
	v.resultOutput.Wrapping = fyne.TextWrapWord
	v.resultOutput.MultiLine = true

	// Create layout
	sourceContainer := container.NewBorder(
		widget.NewLabel("Content Source List:"),
		container.NewHBox(v.addSourceButton, v.removeSourceButton),
		nil, nil,
		container.NewScroll(v.sourceList),
	)

	promptContainer := container.NewBorder(
		widget.NewLabel("Prompt:"),         // Top
		v.generateButton,                   // Bottom
		nil,                                // Left
		nil,                                // Right
		container.NewScroll(v.promptEntry), // Center - Scroll expands
	)

	// Create save buttons
	v.saveToFileButton = widget.NewButton("Save to File", func() {
		v.saveGeneratedContentToFile()
	})
	v.saveToWPButton = widget.NewButton("Save to WordPress", func() {
		v.saveGeneratedContent()
	})

	// Initially disable save buttons until content is generated
	v.saveToFileButton.Disable()
	v.saveToWPButton.Disable()

	resultContainer := container.NewBorder(
		widget.NewLabel("Generated Content:"),                   // Top
		container.NewHBox(v.saveToFileButton, v.saveToWPButton), // Bottom
		nil,                                 // Left
		nil,                                 // Right
		container.NewScroll(v.resultOutput), // Center - Scroll expands
	)

	// Main layout
	leftPanel := container.NewVSplit(
		sourceContainer,
		promptContainer,
	)
	leftPanel.SetOffset(0.4) // 40% for source list, 60% for prompt

	v.container = container.NewHSplit(
		leftPanel,
		resultContainer,
	)
	v.container.SetOffset(0.4) // 40% for left panel, 60% for result
}

// AddSourceContent adds a source content item to the list
func (v *ContentGeneratorView) AddSourceContent(title, content, source string, id int, isSample bool) {
	v.sourceContents = append(v.sourceContents, SourceContent{
		Title:   title,
		Content: content,
		Source:  source,
		ID:      id,
		IsSample: isSample,
	})
	v.sourceList.Refresh()
}

// removeSourceContent removes the selected source content item
func (v *ContentGeneratorView) removeSourceContent() {
	if v.selectedSourceIndex < 0 || v.selectedSourceIndex >= len(v.sourceContents) {
		return
	}

	// Remove the item
	v.sourceContents = append(v.sourceContents[:v.selectedSourceIndex], v.sourceContents[v.selectedSourceIndex+1:]...)
	v.sourceList.Refresh()

	// Reset selection
	v.selectedSourceIndex = -1
	v.removeSourceButton.Disable()
}

// Container returns the container for the content generator view
func (v *ContentGeneratorView) Container() fyne.CanvasObject {
	return v.container
}

// GetSourceContents returns the list of source contents
func (v *ContentGeneratorView) GetSourceContents() []SourceContent {
	return v.sourceContents
}

// ClearSourceContents clears all source contents
func (v *ContentGeneratorView) ClearSourceContents() {
	v.sourceContents = []SourceContent{}
	v.sourceList.Refresh()
	v.selectedSourceIndex = -1
	v.removeSourceButton.Disable()
}

// showAddSourceDialog shows a dialog to add a source file
func (v *ContentGeneratorView) showAddSourceDialog() {
	// Create a file dialog
	dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if reader == nil {
			// User cancelled
			return
		}
		
		// Show progress dialog
		progress := dialog.NewProgressInfinite("Loading", "Loading file content...", v.window)
		progress.Show()
		
		// Process file in a goroutine
		go func() {
			defer reader.Close()
			
			// Read file content
			content, err := io.ReadAll(reader)
			
			// Hide progress dialog
			progress.Hide()
			
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to read file: %w", err), v.window)
				return
			}
			
			// Get file name from URI
			fileName := reader.URI().Name()
			
			// Add to source contents
			v.AddSourceContent(
				fileName,
				string(content),
				"File",
				-1, // No WordPress ID for files
				false, 
			)
			
			dialog.ShowInformation("Success", fmt.Sprintf("Added file '%s' to source content", fileName), v.window)
		}()
	}, v.window)
}

// generateContent generates content based on source content and prompt
func (v *ContentGeneratorView) generateContent() {
	// Validate inputs
	if len(v.sourceContents) == 0 {
		dialog.ShowError(fmt.Errorf("no source content available"), v.window)
		return
	}
	
	promptText := v.promptEntry.Text
	if promptText == "" {
		dialog.ShowError(fmt.Errorf("prompt cannot be empty"), v.window)
		return
	}
	
	// Show progress dialog
	progress := dialog.NewProgressInfinite("Generating", "Generating content with AI...", v.window)
	progress.Show()
	
	// Generate content in a goroutine
	go func() {
		// --- Separate True and Sample Sources ---
		var trueSourcesBuilder strings.Builder
		var sampleSourcesBuilder strings.Builder
		trueCount := 0
		sampleCount := 0

		for _, source := range v.sourceContents {
			var builder *strings.Builder
			var count *int

			if source.IsSample {
				builder = &sampleSourcesBuilder
				count = &sampleCount
			} else {
				builder = &trueSourcesBuilder
				count = &trueCount
			}

			if *count > 0 {
				builder.WriteString("\n\n--- Next Source ---\n\n")
			}
			builder.WriteString(fmt.Sprintf("Source Title: %s\n", source.Title))
			builder.WriteString(fmt.Sprintf("Source Type: %s\n", source.Source)) // e.g., WordPress, File
			builder.WriteString("Content:\n")
			builder.WriteString(source.Content)
			*count++
		}
		// --- End Separation ---

		// Check if there are any true sources if generation requires them
		if trueCount == 0 {
			progress.Hide()
			dialog.ShowError(fmt.Errorf("cannot generate content without at least one 'True Source' (uncheck 'Sample' for factual sources)"), v.window)
			return
		}


		// --- Use the new prompt ---
		finalPrompt := inference.GetWordPressContentGenerateWithSourcesPrompt(
			trueSourcesBuilder.String(),
			sampleSourcesBuilder.String(),
			promptText,
		)
		// --- End Use New Prompt ---

		
		// Call the inference service
		generatedContent, err := v.inferenceService.GenerateText(finalPrompt) // Use the combined prompt
		
		// Hide progress dialog
		progress.Hide()
		
		if err != nil {
			dialog.ShowError(fmt.Errorf("failed to generate content: %w", err), v.window)
			return
		}
		
		// Update the result output
		v.resultOutput.SetText(generatedContent)
		
		// Enable save buttons
		v.saveToFileButton.Enable()
		v.saveToWPButton.Enable()
		
		// Show success dialog
		dialog.ShowInformation("Success", "Content generated successfully", v.window)
	}()
}

// saveGeneratedContentToFile saves the generated content to a file
func (v *ContentGeneratorView) saveGeneratedContentToFile() {
	// Get the generated content
	generatedContent := v.resultOutput.Text
	if generatedContent == "" {
		dialog.ShowError(fmt.Errorf("no generated content to save"), v.window)
		return
	}
	
	// Show file save dialog
	dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}
		if writer == nil {
			// User cancelled
			return
		}
		
		// Show progress dialog
		progress := dialog.NewProgressInfinite("Saving", "Saving content to file...", v.window)
		progress.Show()
		
		// Save in a goroutine
		go func() {
			defer writer.Close()
			
			// Write content to file
			_, err := writer.Write([]byte(generatedContent))
			
			// Hide progress dialog
			progress.Hide()
			
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to save file: %w", err), v.window)
				return
			}
			
			// Get file name from URI
			fileName := filepath.Base(writer.URI().String())
			
			dialog.ShowInformation("Success", fmt.Sprintf("Content saved to file '%s'", fileName), v.window)
		}()
	}, v.window)
}

// saveGeneratedContent saves the generated content to a WordPress page
func (v *ContentGeneratorView) saveGeneratedContent() {
	// Check if WordPress service is connected
	if !v.wpService.IsConnected() {
		dialog.ShowError(fmt.Errorf("not connected to WordPress site"), v.window)
		return
	}
	
	// Get the generated content
	generatedContent := v.resultOutput.Text
	if generatedContent == "" {
		dialog.ShowError(fmt.Errorf("no generated content to save"), v.window)
		return
	}
	
	// Find WordPress pages from source content
	var wpPages []SourceContent
	for _, source := range v.sourceContents {
		if source.Source == "WordPress" && source.ID > 0 {
			wpPages = append(wpPages, source)
		}
	}
	
	// If no WordPress pages found, show error
	if len(wpPages) == 0 {
		dialog.ShowError(fmt.Errorf("no WordPress pages found in source content"), v.window)
		return
	}
	
	// If only one WordPress page, use that
	if len(wpPages) == 1 {
		v.confirmAndSaveToPage(wpPages[0].ID, wpPages[0].Title, generatedContent)
		return
	}
	
	// If multiple WordPress pages, show selection dialog
	var options []string
	for _, page := range wpPages {
		options = append(options, page.Title)
	}
	
	dialog.ShowCustom("Select Page", "Cancel", widget.NewSelect(options, func(selected string) {
		// Find the selected page
		for _, page := range wpPages {
			if page.Title == selected {
				v.confirmAndSaveToPage(page.ID, page.Title, generatedContent)
				break
			}
		}
	}), v.window)
}

// confirmAndSaveToPage confirms and saves content to a WordPress page
func (v *ContentGeneratorView) confirmAndSaveToPage(pageID int, pageTitle, content string) {
	// Confirm before saving
	dialog.ShowConfirm("Save to WordPress", fmt.Sprintf("Are you sure you want to save this content to the page '%s'?", pageTitle), func(confirmed bool) {
		if !confirmed {
			return
		}
		
		// Show progress dialog
		progress := dialog.NewProgressInfinite("Saving", "Saving content to WordPress...", v.window)
		progress.Show()
		
		// Save in a goroutine
		go func() {
			// Update the page content
			err := v.wpService.UpdatePageContent(pageID, content)
			
			// Hide progress dialog
			progress.Hide()
			
			if err != nil {
				dialog.ShowError(fmt.Errorf("failed to save content: %w", err), v.window)
				return
			}
			
			dialog.ShowInformation("Success", fmt.Sprintf("Content saved to page '%s'", pageTitle), v.window)
		}()
	}, v.window)
}
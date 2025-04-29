package ui

import (
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
	sourceList      *widget.List
	addSourceButton *widget.Button
	removeSourceButton *widget.Button
	
	// Generation UI elements
	promptEntry     *widget.Entry
	generateButton  *widget.Button
	resultOutput    *widget.Entry
	
	// Data
	sourceContents  []SourceContent
	selectedSourceIndex int
}

// SourceContent represents a source content item
type SourceContent struct {
	Title   string
	Content string
	Source  string // "WordPress", "File", etc.
	ID      int    // WordPress page ID or other identifier
}

// NewContentGeneratorView creates a new content generator view
func NewContentGeneratorView(wpService *wordpress.WordPressService, inferenceService *inference.InferenceService, window fyne.Window) *ContentGeneratorView {
	view := &ContentGeneratorView{
		wpService:        wpService,
		inferenceService: inferenceService,
		window:           window,
		sourceContents:   []SourceContent{},
		selectedSourceIndex: -1,
	}
	view.initialize()
	return view
}

// initialize initializes the content generator view
func (v *ContentGeneratorView) initialize() {
	// Create source content UI elements
	v.sourceList = widget.NewList(
		func() int {
			return len(v.sourceContents)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Source")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(v.sourceContents) {
				obj.(*widget.Label).SetText(v.sourceContents[id].Title)
			}
		},
	)
	
	v.sourceList.OnSelected = func(id widget.ListItemID) {
		v.selectedSourceIndex = id
		v.removeSourceButton.Enable()
	}
	
	v.addSourceButton = widget.NewButton("Add Source", func() {
		// This will be implemented in Phase 2
		dialog.ShowInformation("Not Implemented", "This feature will be implemented in Phase 2", v.window)
	})
	
	v.removeSourceButton = widget.NewButton("Remove Source", func() {
		v.removeSourceContent()
	})
	v.removeSourceButton.Disable()
	
	// Create generation UI elements
	v.promptEntry = widget.NewMultiLineEntry()
	v.promptEntry.SetPlaceHolder("Enter a prompt or topic for the AI to generate content about...")
	v.promptEntry.Wrapping = fyne.TextWrapWord
	
	v.generateButton = widget.NewButton("Generate Content", func() {
		// This will be implemented in Phase 2
		dialog.ShowInformation("Not Implemented", "This feature will be implemented in Phase 2", v.window)
	})
	
	v.resultOutput = widget.NewMultiLineEntry()
	v.resultOutput.SetPlaceHolder("Generated content will appear here...")
	v.resultOutput.Wrapping = fyne.TextWrapWord
	v.resultOutput.MultiLine = true
	
	// Create layout
	sourceContainer := container.NewBorder(
		widget.NewLabel("Source Content:"),
		container.NewHBox(v.addSourceButton, v.removeSourceButton),
		nil, nil,
		container.NewScroll(v.sourceList),
	)
	
	promptContainer := container.NewVBox(
		widget.NewLabel("Prompt:"),
		v.promptEntry,
		v.generateButton,
	)
	
	resultContainer := container.NewVBox(
		widget.NewLabel("Generated Content:"),
		container.NewScroll(v.resultOutput),
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
func (v *ContentGeneratorView) AddSourceContent(title, content, source string, id int) {
	v.sourceContents = append(v.sourceContents, SourceContent{
		Title:   title,
		Content: content,
		Source:  source,
		ID:      id,
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
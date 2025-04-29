package ui

import (
	"fmt"
	"log"

	"Inference_Engine/inference" // Assuming your inference package path

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// TestInferenceView represents the UI for the Test Inference tab
type TestInferenceView struct {
	container        fyne.CanvasObject
	inferenceService *inference.InferenceService
	window           fyne.Window

	promptInput    *widget.Entry
	responseOutput *widget.Entry
	testButton     *widget.Button
}

// NewTestInferenceView creates a new TestInferenceView
func NewTestInferenceView(service *inference.InferenceService, win fyne.Window) *TestInferenceView {
	view := &TestInferenceView{
		inferenceService: service,
		window:           win,
	}
	view.initialize()
	return view
}

// initialize sets up the UI elements for the view
func (v *TestInferenceView) initialize() {
	v.promptInput = widget.NewMultiLineEntry()
	v.promptInput.SetPlaceHolder("Enter a prompt to test the inference engine...")
	v.promptInput.Wrapping = fyne.TextWrapWord
	v.promptInput.SetMinRowsVisible(10)

	v.responseOutput = widget.NewMultiLineEntry()
	v.responseOutput.SetPlaceHolder("Response will appear here...")
	v.responseOutput.Wrapping = fyne.TextWrapWord
	v.responseOutput.MultiLine = true
	v.responseOutput.SetMinRowsVisible(10)

	v.testButton = widget.NewButton("Test Inference", v.handleTestInference)

	promptArea := container.NewBorder(
		widget.NewLabel("Test Prompt:"), // Top
		v.testButton,                    // Bottom
		nil,                             // Left
		nil,                             // Right
		container.NewScroll(v.promptInput), // Center - Scroll expands
	)

	responseArea := container.NewBorder(
		widget.NewLabel("Response:"),        // Top
		nil,                                 // Bottom
		nil,                                 // Left
		nil,                                 // Right
		container.NewScroll(v.responseOutput), // Center - Scroll expands
	)

	v.container = container.NewVSplit(
		promptArea,
		responseArea,
	)
	// You might need to explicitly cast v.container to *container.Split if SetOffset is needed immediately
	if split, ok := v.container.(*container.Split); ok {
		split.SetOffset(0.4) // Adjust split ratio if needed
	}
}

// handleTestInference contains the logic executed when the test button is pressed
func (v *TestInferenceView) handleTestInference() {
	prompt := v.promptInput.Text
	if prompt == "" {
		dialog.ShowInformation("Error", "Please enter a prompt", v.window)
		return
	}

	if !v.inferenceService.IsRunning() {
		dialog.ShowInformation("Error", "Inference service is not running. Check settings and logs.", v.window)
		return
	}

	// Show a loading indicator
	progress := dialog.NewProgressInfinite("Generating", "Sending prompt to "+v.inferenceService.GetActiveProviderName()+"..."+"\nPlease wait...", v.window)
	progress.Show()

	// Run in a goroutine to avoid blocking the UI
	go func() {
		// Defer hiding the progress indicator to ensure it closes even on error
		defer progress.Hide()

		response, err := v.inferenceService.GenerateText(prompt)

		if err != nil {
			log.Printf("UI Error: Test generation failed: %v", err)
			// Ensure dialogs are shown on the main thread if necessary,
			// but Fyne's dialogs are generally safe.
			dialog.ShowError(err, v.window)
			v.responseOutput.SetText(fmt.Sprintf("ERROR:\n%v", err)) // Show error in output
			return
		}

		v.responseOutput.SetText(response)
		log.Printf("UI: Test generation successful.") // Shorter log message
	}()
}

// Container returns the main container for this view
func (v *TestInferenceView) Container() fyne.CanvasObject {
	return v.container
}

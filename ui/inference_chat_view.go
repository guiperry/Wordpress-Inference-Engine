// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/ui/inference_chat_view.go
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

// InferenceChatView represents the UI for the Inference Chat tab
type InferenceChatView struct { // <-- Renamed struct
	container        fyne.CanvasObject
	inferenceService *inference.InferenceService
	window           fyne.Window
	

	promptInput    *widget.Entry
	responseOutput *widget.Entry
	sendButton     *widget.Button // Renamed button
}

// NewInferenceChatView creates a new InferenceChatView
func NewInferenceChatView(service *inference.InferenceService, win fyne.Window) *InferenceChatView { // <-- Renamed constructor
	view := &InferenceChatView{ // <-- Use new struct name
		inferenceService: service,
		window:           win,
	}
	view.initialize()
	return view
}

// initialize sets up the UI elements for the view
func (v *InferenceChatView) initialize() {
	v.promptInput = widget.NewMultiLineEntry()
	v.promptInput.SetPlaceHolder("Enter your message...")
	v.promptInput.Wrapping = fyne.TextWrapWord
	v.promptInput.SetMinRowsVisible(10)

	v.responseOutput = widget.NewMultiLineEntry()
	v.responseOutput.SetPlaceHolder("Response will appear here...")
	v.responseOutput.Wrapping = fyne.TextWrapWord
	v.responseOutput.MultiLine = true
	v.responseOutput.SetMinRowsVisible(10)
	//v.responseOutput.Disable() // Make response read-only
	//v.responseOutput.ReadOnly = true 

	// --- Removed Radio Group ---

	v.sendButton = widget.NewButton("Send Message", v.handleSendMessage) // Renamed button and handler

	promptArea := container.NewBorder(
		widget.NewLabel("Your Message:"), // Top
		v.sendButton,                    // Bottom (Only send button)
		nil,                             // Left
		nil,                             // Right
		container.NewScroll(v.promptInput), // Center - Scroll expands
	)

	responseArea := container.NewBorder(
		widget.NewLabel("AI Response:"),     // Top
		nil,                                 // Bottom
		nil,                                 // Left
		nil,                                 // Right
		container.NewScroll(v.responseOutput), // Center - Scroll expands
	)

	v.container = container.NewVSplit(
		promptArea,
		responseArea,
	)
	if split, ok := v.container.(*container.Split); ok {
		split.SetOffset(0.4) // Adjust split ratio if needed
	}
}

// handleSendMessage contains the logic executed when the send button is pressed
func (v *InferenceChatView) handleSendMessage() { // <-- Renamed handler
	prompt := v.promptInput.Text
	if prompt == "" {
		dialog.ShowInformation("Input Required", "Please enter a message", v.window)
		return
	}

	if !v.inferenceService.IsRunning() {
		dialog.ShowInformation("Service Error", "Inference service is not running. Check settings and logs.", v.window)
		return
	}

	// --- Simplified Logic: Always use proxy logic ---
	progressMsg := "Sending message via Proxy Logic..."
	log.Printf("UI: Initiating chat message via Proxy Logic")

	// Show a loading indicator
	progress := dialog.NewProgressInfinite("Generating", progressMsg, v.window)
	progress.Show()
	v.responseOutput.SetText("Generating...") // Indicate activity

	// Run in a goroutine to avoid blocking the UI
	go func() {
		defer progress.Hide()

		// Call GenerateText with empty modelName and instructionText
		// The DelegatorService will use its default primary model.
		response, err := v.inferenceService.GenerateText("", prompt, "")

		if err != nil {
			log.Printf("UI Error: Chat generation failed: %v", err)
			dialog.ShowError(fmt.Errorf("Generation failed:\n%w", err), v.window)
			v.responseOutput.SetText(fmt.Sprintf("ERROR:\n%v", err)) // Show error in output
			return
		}

		v.responseOutput.SetText(response)
		log.Printf("UI: Chat generation successful.")
	}()
}

// Container returns the main container for this view
func (v *InferenceChatView) Container() fyne.CanvasObject {
	return v.container
}

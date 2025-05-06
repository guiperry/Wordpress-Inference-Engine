// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/ui/test_inference_view.go
package ui

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"Inference_Engine/inference"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog" // Import layout
	"fyne.io/fyne/v2/widget"
)

// uiLogWriter struct and NewUILogWriter remain the same...
type uiLogWriter struct {
	logOutput    *widget.Entry
	originalOut  io.Writer
	mu           sync.Mutex
	buffer       []byte
	maxLogLength int
}

func NewUILogWriter(logWidget *widget.Entry, original io.Writer) *uiLogWriter {
	return &uiLogWriter{
		logOutput:    logWidget,
		originalOut:  original,
		maxLogLength: 10000,
	}
}

func (w *uiLogWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Write to original output if set
	if w.originalOut != nil {
		w.originalOut.Write(p)
	}

	// Append to buffer and process complete lines
	w.buffer = append(w.buffer, p...)
	for strings.Contains(string(w.buffer), "\n") {
		idx := strings.Index(string(w.buffer), "\n")
		line := string(w.buffer[:idx+1])
		w.buffer = w.buffer[idx+1:]

		// Update UI log widget
		w.logOutput.SetText(w.logOutput.Text + line)

		// Trim log if too long
		if len(w.logOutput.Text) > w.maxLogLength {
			w.logOutput.SetText(w.logOutput.Text[len(w.logOutput.Text)-w.maxLogLength:])
		}
	}

	return len(p), nil
}

// TestInferenceView represents the UI for the new Test Inference tab
type TestInferenceView struct {
	container        fyne.CanvasObject
	inferenceService *inference.InferenceService
	window           fyne.Window

	fallbackButton *widget.Button // Test oversized prompt fallback
	testMOAButton  *widget.Button // Test direct MOA call
	testGeminiButton *widget.Button // Test direct Gemini call
	logConsole     *widget.Entry
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
	v.fallbackButton = widget.NewButton("Trigger Fallback Test (Oversize Prompt)", v.handleFallbackTest)

	// --- ADDED: MOA Test Button ---
	v.testMOAButton = widget.NewButton("Test with MOA (Simple Prompt)", v.handleMOATest)

	// --- ADDED: Gemini Test Button ---
	v.testGeminiButton = widget.NewButton("Test Gemini Endpoint (Simple Prompt)", v.handleGeminiTest)
	// --- End Added ---

	v.logConsole = widget.NewMultiLineEntry()
	v.logConsole.SetPlaceHolder("Application logs will appear here...")
	v.logConsole.Wrapping = fyne.TextWrapOff // Keep lines intact
	v.logConsole.MultiLine = true
	

	// --- Update Layout ---
	topPanel := container.NewVBox(
		widget.NewLabel("Test Inference Mechanisms"),
		v.fallbackButton,
		v.testMOAButton, // Add MOA button
		v.testGeminiButton, // Add Gemini button
	)

	v.container = container.NewBorder(
		topPanel,                          // Top
		nil,                               // Bottom
		nil,                               // Left
		nil,                               // Right
		container.NewScroll(v.logConsole), // Center - Log console takes remaining space
	)
}

// handleFallbackTest sends an oversized prompt to trigger the fallback
func (v *TestInferenceView) handleFallbackTest() {
	if !v.inferenceService.IsRunning() { /* ... service not running dialog ... */
		return
	}

	// Create oversized prompt
	log.Println("UI: Preparing oversized prompt for fallback test...")
	oversizedPrompt := strings.Repeat("This is part of a very long test prompt designed to exceed the context window limit... ", 300)
	log.Printf("UI: Oversized prompt length: %d chars", len(oversizedPrompt))

	progressMsg := "Sending oversized prompt via Delegator..."
	log.Printf("UI: Initiating fallback test...")
	progress := dialog.NewProgressInfinite("Testing Fallback", progressMsg, v.window)
	progress.Show()

	go func() {
		defer progress.Hide()
		// Call GenerateText with empty modelName and instructionText
		// to trigger default primary/fallback logic in DelegatorService.
		response, err := v.inferenceService.GenerateText("", oversizedPrompt, "")

		if err != nil {
			log.Printf("UI Error: Fallback test failed: %v", err)
			dialog.ShowError(fmt.Errorf("Fallback test failed:\n%w\n\nCheck log console for details.", err), v.window)
			return
		}
		log.Printf("UI: Fallback test completed successfully (response length: %d). Check log console for trace.", len(response))
		dialog.ShowInformation("Fallback Test Complete", "Request finished. Check the log console below for the trace (Proxy failure -> Base success).", v.window)
	}()
}

// --- ADDED: handleMOATest ---
// handleMOATest sends a simple prompt directly to the MOA service
func (v *TestInferenceView) handleMOATest() {
	if !v.inferenceService.IsRunning() {
		dialog.ShowInformation("Service Error", "Inference service is not running. Check settings and logs.", v.window)
		return
	}

	// Use a simple, standard prompt for MOA testing
	testPrompt := "Explain the concept of a Mixture of Agents (MOA) in large language models in a concise paragraph."
	log.Println("UI: Preparing simple prompt for MOA test...")

	progressMsg := "Sending prompt directly to MOA..."
	log.Printf("UI: Initiating MOA test...")
	progress := dialog.NewProgressInfinite("Testing MOA", progressMsg, v.window)
	progress.Show()

	go func() {
		defer progress.Hide()
		// Call the specific MOA generation method
		response, err := v.inferenceService.GenerateTextWithMOA(testPrompt) // Use GenerateTextWithMOA

		if err != nil {
			log.Printf("UI Error: MOA test failed: %v", err)
			dialog.ShowError(fmt.Errorf("MOA test failed:\n%w\n\nCheck log console for details.", err), v.window)
			return
		}
		log.Printf("UI: MOA test completed successfully (response length: %d). Check log console for trace.", len(response))
		dialog.ShowInformation("MOA Test Complete", "Request finished via MOA. Check the log console below for the trace.", v.window)
		// Optionally, display the MOA response somewhere if needed,
		// but the primary goal here is observing the logs.
	}()
}

// --- ADDED: handleGeminiTest ---
// handleGeminiTest sends a simple prompt directly to the configured Gemini provider
func (v *TestInferenceView) handleGeminiTest() {
	if !v.inferenceService.IsRunning() {
		dialog.ShowInformation("Service Error", "Inference service is not running. Check settings and logs.", v.window)
		return
	}

	// Use a simple, standard prompt for Gemini testing
	testPrompt := "What is Google Gemini?"
	log.Println("UI: Preparing simple prompt for Gemini test...")

	progressMsg := "Sending prompt directly to Gemini..."
	log.Printf("UI: Initiating Gemini test...")
	progress := dialog.NewProgressInfinite("Testing Gemini", progressMsg, v.window)
	progress.Show()

	go func() {
		defer progress.Hide()
		// Call a new method in InferenceService to target a specific provider
		response, err := v.inferenceService.GenerateTextWithProvider("gemini", testPrompt)

		if err != nil {
			log.Printf("UI Error: Gemini test failed: %v", err)
			// Check specifically for the 404 error we saw earlier
			if strings.Contains(err.Error(), "status 404") {
				dialog.ShowError(fmt.Errorf("Gemini test failed with 404 Not Found.\nPlease check the API endpoint configuration in gemini_provider.go.\n\nError: %w", err), v.window)
			} else {
				dialog.ShowError(fmt.Errorf("Gemini test failed:\n%w\n\nCheck log console for details.", err), v.window)
			}
			return
		}
		log.Printf("UI: Gemini test completed successfully (response length: %d). Check log console for trace.", len(response))
		dialog.ShowInformation("Gemini Test Complete", "Request finished via Gemini. Check the log console below for the trace.", v.window)
	}()
}
// --- End Added ---

// Container returns the main container for this view
func (v *TestInferenceView) Container() fyne.CanvasObject {
	return v.container
}

// LogConsoleWidget returns the log console widget for log redirection
func (v *TestInferenceView) LogConsoleWidget() *widget.Entry {
	return v.logConsole
}

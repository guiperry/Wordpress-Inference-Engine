package main

import (
	"fmt" // Import fmt
	"log"

	"Inference_Engine/inference"
	"Inference_Engine/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/joho/godotenv"

	"Inference_Engine/wordpress"
)

func main() {
	// Load .env file contents into environment variables
	err := godotenv.Load() // Loads .env from the current directory
	if err != nil {
		// Log the error but continue; maybe the key is set directly in the environment
		log.Println("Warning: Error loading .env file:", err)
	}

	a := app.New()
	w := a.NewWindow("Wordpress Inference Engine")

	// Initialize the consolidated inference service
	inferenceService := inference.NewInferenceService()
	wpService := wordpress.NewWordPressService()

	// Try to start the service with the default provider (e.g., cerebras)
	if err := inferenceService.Start(); err != nil {
		// Log the error, but maybe allow the UI to load anyway
		log.Printf("ERROR: Failed to start default inference service: %v", err)
		dialog.ShowError(fmt.Errorf("Failed to start default inference provider '%s': %v\nPlease check API keys and configuration.", inferenceService.GetActiveProviderName(), err), w)
	} else {
		log.Printf("Inference service started with provider: %s", inferenceService.GetActiveProviderName())
	}

	// Create views
	contentManagerView := ui.NewContentManagerView(inferenceService, w)
	contentGeneratorView := ui.NewContentGeneratorView(wpService, inferenceService, w)
	inferenceSettingsView := ui.NewInferenceSettingsView(inferenceService, w)
	wordpressSettingsView := ui.NewWordPressSettingsView(wpService, w)

	// Combine settings views
	combinedSettings := container.NewVBox(
		inferenceSettingsView.Container(),
		widget.NewSeparator(),
		wordpressSettingsView.Container(),
	)

	// --- Test Tab ---
	promptInput := widget.NewMultiLineEntry()
	promptInput.SetPlaceHolder("Enter a prompt to test the inference engine...")
	promptInput.Wrapping = fyne.TextWrapWord

	responseOutput := widget.NewMultiLineEntry()
	responseOutput.SetPlaceHolder("Response will appear here...")
	responseOutput.Wrapping = fyne.TextWrapWord
	responseOutput.MultiLine = true
	// responseOutput.Disable() // Keep enabled for copy-paste

	testButton := widget.NewButton("Test Inference", func() {
		prompt := promptInput.Text
		if prompt == "" {
			dialog.ShowInformation("Error", "Please enter a prompt", w)
			return
		}

		if !inferenceService.IsRunning() {
			dialog.ShowInformation("Error", "Inference service is not running. Check settings and logs.", w)
			return
		}

		// Show a loading indicator
		progress := dialog.NewProgressInfinite("Generating", "Sending prompt to "+inferenceService.GetActiveProviderName()+"..."+"\nPlease wait...", w)
		progress.Show()

		// Run in a goroutine to avoid blocking the UI
		go func() {

			// Defer hiding the progress indicator to ensure it closes even on error
			defer progress.Hide()

			response, err := inferenceService.GenerateText(prompt)
			
			if err != nil {
				log.Printf("UI Error: Test generation failed: %v", err)
				dialog.ShowError(err, w)
				responseOutput.SetText(fmt.Sprintf("ERROR:\n%v", err)) // Show error in output
				return
				} else {

					responseOutput.SetText(response)
				}
			

			log.Printf("UI: Test generation successful. Response: %s", response)
		}()
	})

	testContainer := container.NewVSplit(
		container.NewVBox(
			widget.NewLabel("Test Prompt:"),
			promptInput,
			testButton,
		),
		container.NewVBox(
			widget.NewLabel("Response:"),
			container.NewScroll(responseOutput),
		),
	)
	testContainer.SetOffset(0.4) // Adjust split ratio

	// --- Main Tabs ---
	tabs := container.NewAppTabs(
		container.NewTabItem("Manager", contentManagerView.Container()),
		container.NewTabItem("Generator", contentGeneratorView.Container()),
		container.NewTabItem("Settings", combinedSettings),
		container.NewTabItem("Test Inference", testContainer),
	)

	// Ensure the service is stopped cleanly on exit
	w.SetCloseIntercept(func() {
		log.Println("Shutting down inference service...")
		if err := inferenceService.Stop(); err != nil {
			log.Printf("Error stopping inference service: %v", err)
		}
		w.Close()
	})

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(1024, 768))
	w.ShowAndRun()
}

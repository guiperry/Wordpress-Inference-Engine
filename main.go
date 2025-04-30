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

	"github.com/joho/godotenv"

	"Inference_Engine/wordpress"
	_ "Inference_Engine/inference"
)

func main() {
	// Load .env file contents into environment variables
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: Error loading .env file:", err)
	}
	// Ensure GEMINI_API_KEY is also loaded if present in .env

	a := app.NewWithID("com.inc-line.wordpressinferenceengine")
	a.Settings().SetTheme(&ui.HighContrastTheme{})
	w := a.NewWindow("Wordpress Inference Engine")

	// Initialize the consolidated inference service
	inferenceService := inference.NewInferenceService()
	wpService := wordpress.NewWordPressService()

	// ... (updateWindowTitle logic remains the same) ...
	updateWindowTitle := func() { /* ... */ }
	updateWindowTitle()
	if wpService != nil { wpService.SetSiteChangeCallback(updateWindowTitle) }


	// Try to start the inference service (which now configures both LLMs)
	if err := inferenceService.Start(); err != nil {
		log.Printf("ERROR: Failed to start inference service: %v", err)
		// Provide a more generic error message as specific provider might vary
		dialog.ShowError(fmt.Errorf("Failed to start inference service components: %v\nPlease check API keys (Cerebras, Gemini) and configuration.", err), w)
	} else {
		log.Println("Inference service started successfully.") // More generic success message
	}

	// Create views
	contentManagerView := ui.NewContentManagerView(wpService, inferenceService, w)
	contentGeneratorView := ui.NewContentGeneratorView(wpService, inferenceService, w)
	inferenceSettingsView := ui.NewInferenceSettingsView(inferenceService, w)
	wordpressSettingsView := ui.NewWordPressSettingsView(wpService, w)
	testInferenceView := ui.NewTestInferenceView(inferenceService, w) 

	// This needs to happen after both views are created.
	contentManagerView.SetContentGeneratorView(contentGeneratorView)

	// Combine settings views
	settingsContent := container.NewAdaptiveGrid(2, // <--- Changed from NewVBox
		inferenceSettingsView.Container(),
		wordpressSettingsView.Container(),
	)

	

	// --- Main Tabs ---
	tabs := container.NewAppTabs(
		container.NewTabItem("Manager", contentManagerView.Container()),
		container.NewTabItem("Generator", contentGeneratorView.Container()),
		container.NewTabItem("Settings", container.NewScroll(settingsContent)),
		container.NewTabItem("Test Inference", testInferenceView.Container()),
	)

	// --- Add OnSelected callback ---
	tabs.OnSelected = func(tab *container.TabItem) {
		if tab.Text == "Manager" {
			// When the Manager tab is selected, refresh its status
			contentManagerView.RefreshStatus()
		}
		// Add similar checks for other tabs if they need refreshing on select
	}
	// --- End of OnSelected callback ---

	// Set the initial selected tab (optional, defaults to first)
	tabs.SelectIndex(2) // Select Manager tab initially

	// Ensure the service is stopped cleanly on exit
	w.SetCloseIntercept(func() {
		log.Println("Shutting down inference service...")
		if err := inferenceService.Stop(); err != nil {
			log.Printf("Error stopping inference service: %v", err)
		}
		w.Close()
	})

	w.SetContent(tabs)
	w.Resize(fyne.NewSize(900, 700))
	w.ShowAndRun()
}

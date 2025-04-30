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
)

func main() {
	// Load .env file contents into environment variables
	err := godotenv.Load() // Loads .env from the current directory
	if err != nil {
		// Log the error but continue; maybe the key is set directly in the environment
		log.Println("Warning: Error loading .env file:", err)
	}

	// Use a unique reverse-domain style ID
	a := app.NewWithID("com.inc-line.wordpressinferenceengine") // Replace "com.example" with your domain or a unique identifier
	a.Settings().SetTheme(&ui.HighContrastTheme{})
	w := a.NewWindow("Wordpress Inference Engine")

	// Initialize the consolidated inference service
	inferenceService := inference.NewInferenceService()
	wpService := wordpress.NewWordPressService()

	// Update window title with current site if available
	updateWindowTitle := func() {
		if wpService != nil {
			siteName := wpService.GetCurrentSiteName()
			if siteName != "" {
				w.SetTitle(fmt.Sprintf("Wordpress Inference Engine - %s", siteName))
			} else {
				w.SetTitle("Wordpress Inference Engine")
			}
		}
	}
	updateWindowTitle()

	// Set up callback to update window title when site changes
	if wpService != nil {
		wpService.SetSiteChangeCallback(updateWindowTitle)
	}

	

	// Try to start the inference service with the default provider (e.g., cerebras)
	if err := inferenceService.Start(); err != nil {
		// Log the error, but maybe allow the UI to load anyway
		log.Printf("ERROR: Failed to start default inference service: %v", err)
		dialog.ShowError(fmt.Errorf("Failed to start default inference provider '%s': %v\nPlease check API keys and configuration.", inferenceService.GetActiveProviderName(), err), w)
	} else {
		log.Printf("Inference service started with provider: %s", inferenceService.GetActiveProviderName())
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

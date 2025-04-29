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

	a := app.New()
	a.Settings().SetTheme(&ui.HighContrastTheme{})
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
	testInferenceView := ui.NewTestInferenceView(inferenceService, w) 

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

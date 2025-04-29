package main

import (
	"fmt" // Import fmt
	"log"
	"os"

	"Inference_Engine/inference"
	"Inference_Engine/ui"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/joho/godotenv"
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

	// Try to start the service with the default provider (e.g., cerebras)
	if err := inferenceService.Start(); err != nil {
		// Log the error, but maybe allow the UI to load anyway
		log.Printf("ERROR: Failed to start default inference service: %v", err)
		dialog.ShowError(fmt.Errorf("Failed to start default inference provider '%s': %v\nPlease check API keys and configuration.", inferenceService.GetActiveProviderName(), err), w)
	} else {
		log.Printf("Inference service started with provider: %s", inferenceService.GetActiveProviderName())
	}

	// Create the Content Manager view
	contentManagerView := ui.NewContentManagerView(inferenceService, w)

	// --- Settings Tab ---
	providerOptions := []string{"cerebras", "openai"} // Add more registered providers here
	providerSelect := widget.NewSelect(providerOptions, func(selectedProvider string) {
		log.Printf("UI: Provider selection changed to: %s", selectedProvider)
		err := inferenceService.SwitchToProvider(selectedProvider)
		if err != nil {
			log.Printf("UI Error: Failed to switch provider: %v", err)
			dialog.ShowError(err, w)
			// Optionally reset dropdown to the actual active provider if switch failed
			// providerSelect.SetSelected(inferenceService.GetActiveProviderName())
		} else {
			log.Printf("UI: Switched provider successfully to %s", selectedProvider)
			// Update UI elements that depend on the provider if necessary
		}
	})
	// Set initial selection based on the service's state after Start()
	providerSelect.SetSelected(inferenceService.GetActiveProviderName())

	// API Key Inputs (remain similar)
	openaiKeyEntry := widget.NewPasswordEntry()
	openaiKeyEntry.SetPlaceHolder("OpenAI API Key (optional, loaded from env)")
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		openaiKeyEntry.SetText(key)
		openaiKeyEntry.Disable() // Indicate it's loaded from env
	}
	saveOpenAIButton := widget.NewButton("Set OpenAI Key Env Var", func() {
		key := openaiKeyEntry.Text
		if key != "" {
			os.Setenv("OPENAI_API_KEY", key)
			dialog.ShowInformation("Restart Required", "OpenAI API key environment variable set.\nPlease restart the application for the change to take full effect.", w)
			openaiKeyEntry.Disable()
		}
	})
	// Allow enabling if disabled
	openaiKeyEntry.OnChanged = func(_ string) {
		if openaiKeyEntry.Disabled() {
			openaiKeyEntry.Enable()
		}
	}

	cerebrasKeyEntry := widget.NewPasswordEntry()
	cerebrasKeyEntry.SetPlaceHolder("Cerebras API Key (optional, loaded from env)")
	if key := os.Getenv("CEREBRAS_API_KEY"); key != "" {
		cerebrasKeyEntry.SetText(key)
		cerebrasKeyEntry.Disable() // Indicate it's loaded from env
	}
	saveCerebrasButton := widget.NewButton("Set Cerebras Key Env Var", func() {
		key := cerebrasKeyEntry.Text
		if key != "" {
			os.Setenv("CEREBRAS_API_KEY", key)
			dialog.ShowInformation("Restart Required", "Cerebras API key environment variable set.\nPlease restart the application for the change to take full effect.", w)
			cerebrasKeyEntry.Disable()
		}
	})
	// Allow enabling if disabled
	cerebrasKeyEntry.OnChanged = func(_ string) {
		if cerebrasKeyEntry.Disabled() {
			cerebrasKeyEntry.Enable()
		}
	}

	// Model Selection (Example - could be dynamic based on provider)
	modelEntry := widget.NewEntry()
	modelEntry.SetPlaceHolder("Enter model name (e.g., gpt-4, llama-4-scout-17b-16e-instruct)")
	modelEntry.SetText(inferenceService.GetCurrentModel()) // Set initial model

	setModelButton := widget.NewButton("Set Model", func() {
		model := modelEntry.Text
		if model == "" {
			dialog.ShowInformation("Info", "Please enter a model name.", w)
			return
		}
		err := inferenceService.SetModel(model)
		if err != nil {
			dialog.ShowError(err, w)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Model set to '%s'", model), w)
		}
	})

	settingsContainer := container.NewVBox(
		widget.NewLabel("Inference Settings"),
		widget.NewLabel("Model Provider:"),
		providerSelect,
		widget.NewLabel("Model Name:"),
		modelEntry,
		setModelButton,
		widget.NewSeparator(),
		widget.NewLabel("API Keys (Set Environment Variable & Restart):"),
		openaiKeyEntry,
		saveOpenAIButton,
		cerebrasKeyEntry,
		saveCerebrasButton,
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
		container.NewTabItem("Content Manager", contentManagerView.Container()),
		container.NewTabItem("Settings", settingsContainer),
		container.NewTabItem("Test Inference", testContainer), // Added Test tab
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

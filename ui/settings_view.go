package ui

import (
	"fmt"
	"log"
	"net/url"
	"os"

	"Inference_Engine/inference"
	"Inference_Engine/wordpress"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// WordPressSettingsView represents the WordPress settings view
type WordPressSettingsView struct {
	container *fyne.Container
	wpService *wordpress.WordPressService
	window    fyne.Window

	// Connection UI elements
	siteNameEntry *widget.Entry
	siteURLEntry  *widget.Entry
	usernameEntry *widget.Entry
	passwordEntry *widget.Entry
	rememberCheck *widget.Check
	connectButton *widget.Button
	statusLabel   *widget.Label

	// Saved sites UI elements
	savedSitesList   *widget.List
	loadSiteButton   *widget.Button
	deleteSiteButton *widget.Button

	// Data
	savedSites        []wordpress.SavedSite
	selectedSiteIndex int

	// Callback for when connection status changes
	onConnectionChanged func(connected bool)
}

// NewWordPressSettingsView creates a new WordPress settings view
func NewWordPressSettingsView(wpService *wordpress.WordPressService, window fyne.Window) *WordPressSettingsView {
	view := &WordPressSettingsView{
		wpService:           wpService,
		window:              window,
		savedSites:          []wordpress.SavedSite{},
		selectedSiteIndex:   -1,
		onConnectionChanged: func(connected bool) {},
	}
	view.initialize()
	view.refreshSavedSites()
	return view
}

// initialize initializes the settings view
func (v *WordPressSettingsView) initialize() {
	// Create connection UI elements
	v.siteNameEntry = widget.NewEntry()
	v.siteNameEntry.SetPlaceHolder("Site Name (for saving)")

	v.siteURLEntry = widget.NewEntry()
	v.siteURLEntry.SetPlaceHolder("WordPress Site URL (e.g., https://example.com/)")

	v.usernameEntry = widget.NewEntry()
	v.usernameEntry.SetPlaceHolder("Username")

	v.passwordEntry = widget.NewPasswordEntry()
	v.passwordEntry.SetPlaceHolder("Application Password")

	v.rememberCheck = widget.NewCheck("Remember Me", nil)

	v.connectButton = widget.NewButton("Connect", func() {
		v.connectToWordPress()
	})

	v.statusLabel = widget.NewLabel("Status: Disconnected")

	// Create saved sites UI elements
	v.savedSitesList = widget.NewList(
		func() int {
			return len(v.savedSites)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Template Site Name")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(v.savedSites) {
				obj.(*widget.Label).SetText(v.savedSites[id].Name)
			}
		},
	)

	v.savedSitesList.OnSelected = func(id widget.ListItemID) {
		v.selectedSiteIndex = id
		v.loadSiteButton.Enable()
		v.deleteSiteButton.Enable()
	}

	v.loadSiteButton = widget.NewButton("Load Site", func() {
		v.loadSavedSite()
	})
	v.loadSiteButton.Disable()

	v.deleteSiteButton = widget.NewButton("Delete Site", func() {
		v.deleteSavedSite()
	})
	v.deleteSiteButton.Disable()

	// Create layout
	connectionForm := container.NewVBox(
		widget.NewLabel("WordPress Connection"),
		widget.NewLabel("Site Name:"),
		v.siteNameEntry,
		widget.NewLabel("Site URL:"),
		v.siteURLEntry,
		widget.NewLabel("Username:"),
		v.usernameEntry,
		widget.NewLabel("Application Password:"),
		v.passwordEntry,
		v.rememberCheck,
		v.connectButton,
		v.statusLabel,
	)

	savedSitesContent := container.NewBorder(
		nil, // Top
		// Buttons go at the bottom of this inner border layout
		container.NewHBox(layout.NewSpacer(), v.loadSiteButton, v.deleteSiteButton),
		nil,              // Left
		nil,              // Right
		v.savedSitesList, // List goes in the center
	)

	savedSitesContainer := container.NewBorder(
		widget.NewLabel("Saved Sites"),         // Top
		nil,                                    // Bottom
		nil,                                    // Left
		nil,                                    // Right
		container.NewScroll(savedSitesContent), // Center <-- The scrollable part now expands
	)

	// Main layout
	v.container = container.NewBorder(
		container.NewVBox(connectionForm, widget.NewSeparator()), // Top
		nil,                 // Bottom
		nil,                 // Left
		nil,                 // Right
		savedSitesContainer, // Center <-- This container now expands
	)
}

// InferenceSettingsView represents the inference engine settings view
type InferenceSettingsView struct {
	container        *fyne.Container // Keep this unexported
	inferenceService *inference.InferenceService
	window           fyne.Window

	// UI elements
	providerSelect   *widget.Select
	openaiKeyEntry   *widget.Entry
	cerebrasKeyEntry *widget.Entry
	modelEntry       *widget.Entry
}

// NewInferenceSettingsView creates a new inference settings view
func NewInferenceSettingsView(inferenceService *inference.InferenceService, window fyne.Window) *InferenceSettingsView {
	view := &InferenceSettingsView{
		inferenceService: inferenceService,
		window:           window,
	}
	view.initialize()
	return view
}

// Initializes the inference settings view
func (v *InferenceSettingsView) initialize() {
	// Provider selection
	providerOptions := []string{"cerebras", "openai"} // Add more registered providers here
	v.providerSelect = widget.NewSelect(providerOptions, func(selectedProvider string) {
		if selectedProvider == v.inferenceService.GetActiveProviderName() {
			// No actual change, likely triggered by SetSelected or refresh. Ignore.
			// log.Printf("UI: Provider selection callback triggered for current provider '%s', ignoring.", selectedProvider) // Optional: Add a log for debugging if needed
			return
		}

		log.Printf("UI: Provider selection changed to: %s", selectedProvider)
		err := v.inferenceService.SwitchToProvider(selectedProvider)
		if err != nil {
			log.Printf("UI Error: Failed to switch provider: %v", err)
			dialog.ShowError(err, v.window)
		} else {
			log.Printf("UI: Switched provider successfully to %s", selectedProvider)
		}
	})
	v.providerSelect.SetSelected(v.inferenceService.GetActiveProviderName())

	// API Key Inputs
	v.openaiKeyEntry = widget.NewPasswordEntry()
	v.openaiKeyEntry.SetPlaceHolder("OpenAI API Key (optional, loaded from env)")
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		v.openaiKeyEntry.SetText(key)
		v.openaiKeyEntry.Disable() // Indicate it's loaded from env
	}
	saveOpenAIButton := widget.NewButton("Set OpenAI Key Env Var", func() {
		key := v.openaiKeyEntry.Text
		if key != "" {
			os.Setenv("OPENAI_API_KEY", key)
			dialog.ShowInformation("Restart Required", "OpenAI API key environment variable set.\nPlease restart the application for the change to take full effect.", v.window)
			v.openaiKeyEntry.Disable()
		}
	})
	// Allow enabling if disabled
	v.openaiKeyEntry.OnChanged = func(_ string) {
		if v.openaiKeyEntry.Disabled() {
			v.openaiKeyEntry.Enable()
		} // <-- Brace was missing here
	}

	v.cerebrasKeyEntry = widget.NewPasswordEntry()
	v.cerebrasKeyEntry.SetPlaceHolder("Cerebras API Key (optional, loaded from env)")
	if key := os.Getenv("CEREBRAS_API_KEY"); key != "" {
		v.cerebrasKeyEntry.SetText(key)
		v.cerebrasKeyEntry.Disable() // Indicate it's loaded from env
	}
	saveCerebrasButton := widget.NewButton("Set Cerebras Key Env Var", func() {
		key := v.cerebrasKeyEntry.Text
		if key != "" {
			os.Setenv("CEREBRAS_API_KEY", key)
			dialog.ShowInformation("Restart Required", "Cerebras API key environment variable set.\nPlease restart the application for the change to take full effect.", v.window)
			v.cerebrasKeyEntry.Disable()
		}
	})
	// Allow enabling if disabled
	v.cerebrasKeyEntry.OnChanged = func(_ string) {
		if v.cerebrasKeyEntry.Disabled() {
			v.cerebrasKeyEntry.Enable()
		}
	} // <-- Brace was missing here

	// Model Selection
	v.modelEntry = widget.NewEntry()
	v.modelEntry.SetPlaceHolder("Enter model name (e.g., gpt-4, llama-4-scout-17b-16e-instruct)")
	v.modelEntry.SetText(v.inferenceService.GetCurrentModel()) // Set initial model

	setModelButton := widget.NewButton("Set Model", func() {
		model := v.modelEntry.Text
		if model == "" {
			dialog.ShowInformation("Info", "Please enter a model name.", v.window)
			return
		}
		err := v.inferenceService.SetModel(model)
		if err != nil {
			dialog.ShowError(err, v.window)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("Model set to '%s'", model), v.window)
		}
	})

	// Create layout
	v.container = container.NewVBox(
		widget.NewLabel("Inference Settings"),
		widget.NewLabel("Model Provider:"),
		v.providerSelect,
		widget.NewLabel("Model Name:"),
		v.modelEntry,
		setModelButton,
		widget.NewSeparator(),
		widget.NewLabel("API Keys (Set Environment Variable & Restart):"),
		v.openaiKeyEntry,
		saveOpenAIButton,
		v.cerebrasKeyEntry,
		saveCerebrasButton,
	)
}

// Container returns the container for the Inference Settings view
// This method was added to fix the error in main.go
func (v *InferenceSettingsView) Container() fyne.CanvasObject {
	return v.container
}

// Container returns the container for the WordPress Settings view
func (v *WordPressSettingsView) Container() fyne.CanvasObject {
	return v.container
}

// connectToWordPress connects to the WordPress site
func (v *WordPressSettingsView) connectToWordPress() {
	siteName := v.siteNameEntry.Text
	siteURL := v.siteURLEntry.Text
	username := v.usernameEntry.Text
	password := v.passwordEntry.Text
	remember := v.rememberCheck.Checked
	log.Printf("connectToWordPress: Initiated for URL: %s, User: %s", siteURL, username) // Log start

	if siteURL == "" || username == "" || password == "" {
		log.Println("connectToWordPress: Missing connection fields.")
		dialog.ShowError(fmt.Errorf("please fill in all connection fields"), v.window)
		return
	}

	// --- Update Status Immediately ---
	log.Println("connectToWordPress: Updating status to Connecting and disabling button.")
	v.statusLabel.SetText("Status: Connecting...")
	v.statusLabel.Refresh()   // Ensure UI updates
	v.connectButton.Disable() // Disable button during connection attempt

	// Show progress dialog
	log.Println("connectToWordPress: Showing progress dialog.")
	progress := dialog.NewProgressInfinite("Connecting", "Connecting to WordPress site...", v.window)
	progress.Show()

	// Use a channel to signal completion and pass the error back
	done := make(chan error)
	log.Println("connectToWordPress: Created 'done' channel.")

	// --- Connection Goroutine ---
	log.Println("connectToWordPress: Starting connection goroutine.")
	// This goroutine ONLY performs the network call.
	go func() {
		log.Println("connectToWordPress (goroutine): Started.")
		log.Printf("connectToWordPress (goroutine): Calling wpService.Connect for URL: %s", siteURL)
		// Perform the connection attempt. The service now has a timeout.
		err := v.wpService.Connect(siteURL, username, password)
		log.Printf("connectToWordPress (goroutine): wpService.Connect finished. Error: %v", err)
		// Check if channel is still open before sending
		// (Could be closed if main UI context is gone, though less likely here)
		log.Println("connectToWordPress (goroutine): Attempting to send result to 'done' channel.")
		select {
		case done <- err: // Send the result (nil or error) back
		log.Println("connectToWordPress (goroutine): Successfully sent result to 'done' channel.")
		default:
			// Channel closed or blocked, log if necessary
			log.Println("connectToWordPress (goroutine): 'done' channel blocked or closed before sending.")
		}
		log.Println("connectToWordPress (goroutine): Closing 'done' channel.")
		close(done) // Close channel once done
		log.Println("connectToWordPress (goroutine): Finished.")

	}()

	// --- UI Update Handling ---
	log.Println("connectToWordPress: Starting UI update handling goroutine.")
	go func() {
		log.Println("connectToWordPress (UI goroutine): Started. Waiting for result from 'done' channel.")
		err, ok := <-done // Receive the result from the connection goroutine
		log.Printf("connectToWordPress (UI goroutine): Received from 'done' channel. Error: %v, OK: %t", err, ok)

		if !ok {
			// Channel was closed without sending a value, unusual case
			log.Println("connectToWordPress (UI goroutine): 'done' channel closed unexpectedly.")
			// Attempt cleanup just in case
			log.Println("connectToWordPress (UI goroutine): Hiding progress (unexpected close).")
			progress.Hide()
			log.Println("connectToWordPress (UI goroutine): Enabling connect button (unexpected close).")
			v.connectButton.Enable()
			log.Println("connectToWordPress (UI goroutine): Setting status to Error (unexpected close).")
			v.statusLabel.SetText("Status: Error (Connection Aborted)")
			v.statusLabel.Refresh()
			log.Println("connectToWordPress (UI goroutine): Finished (unexpected close).")
			return
		}

		// --- All UI updates happen here, after the network call is done ---
		log.Println("connectToWordPress (UI goroutine): Hiding progress.")
		progress.Hide()          // Hide progress first
		log.Println("connectToWordPress (UI goroutine): Enabling connect button.")
		v.connectButton.Enable() // Re-enable button

		if err != nil {
			log.Printf("connectToWordPress (UI goroutine): Connection failed. Error: %v", err)
			v.statusLabel.SetText(fmt.Sprintf("Status: Connection failed (%s)", err.Error()))
			v.statusLabel.Refresh()
			log.Println("connectToWordPress (UI goroutine): Showing error dialog.")
			dialog.ShowError(fmt.Errorf("failed to connect: %w", err), v.window)
			if v.onConnectionChanged != nil {
				log.Println("connectToWordPress (UI goroutine): Calling onConnectionChanged(false).")
				v.onConnectionChanged(false)
			}
			log.Println("connectToWordPress (UI goroutine): Finished (error path).")
			return // Exit this UI update goroutine
		}

		// Success path
		log.Println("connectToWordPress (UI goroutine): Connection successful.")
		v.statusLabel.SetText("Status: Connected")
		v.statusLabel.Refresh()
		if v.onConnectionChanged != nil {
			log.Println("connectToWordPress (UI goroutine): Calling onConnectionChanged(true).")
			v.onConnectionChanged(true)
		}

		// Save site if remember is checked
		if remember {
			log.Println("connectToWordPress (UI goroutine): 'Remember Me' checked. Proceeding to save.")
			effectiveSiteName := siteName
			if effectiveSiteName == "" {
				u, parseErr := url.Parse(siteURL)
				if parseErr == nil && u != nil {
					effectiveSiteName = u.Host
				} else {
					effectiveSiteName = "WordPress Site" // Fallback
				}
				log.Printf("connectToWordPress (UI goroutine): Generated effective site name: %s", effectiveSiteName)
				v.siteNameEntry.SetText(effectiveSiteName)
				// v.siteNameEntry.Refresh() // Refresh might be needed
			}

			log.Printf("connectToWordPress (UI goroutine): Calling wpService.SaveSite for name: %s", effectiveSiteName)
			saveErr := v.wpService.SaveSite(effectiveSiteName, siteURL, username, password)
			if saveErr != nil {
				log.Printf("connectToWordPress (UI goroutine): Error saving site: %v", saveErr)
				dialog.ShowError(fmt.Errorf("connection successful, but failed to save site: %w", saveErr), v.window)
			} else {
				log.Println("connectToWordPress (UI goroutine): Site saved successfully. Refreshing saved sites list.")
				v.refreshSavedSites() // Refresh list after successful save
			}
		} else {
			log.Println("connectToWordPress (UI goroutine): 'Remember Me' not checked. Skipping save.")
		}
		log.Println("connectToWordPress (UI goroutine): Finished (success path).")
	}() // End of UI update handling goroutine
	log.Println("connectToWordPress: Exiting main function.")
} // End of connectToWordPress


// refreshSavedSites refreshes the list of saved sites
func (v *WordPressSettingsView) refreshSavedSites() {
	v.savedSites = v.wpService.GetSavedSites()
	v.savedSitesList.Refresh()

	// Reset selection
	v.selectedSiteIndex = -1
	v.loadSiteButton.Disable()
	v.deleteSiteButton.Disable()
}

// loadSavedSite loads a saved site's credentials into the form
func (v *WordPressSettingsView) loadSavedSite() {
	if v.selectedSiteIndex < 0 || v.selectedSiteIndex >= len(v.savedSites) {
		return
	}

	siteName := v.savedSites[v.selectedSiteIndex].Name
	site, found := v.wpService.GetSavedSite(siteName)
	if !found {
		dialog.ShowError(fmt.Errorf("site not found"), v.window)
		return
	}

	// Fill form with site details
	v.siteNameEntry.SetText(site.Name)
	v.siteURLEntry.SetText(site.URL)
	v.usernameEntry.SetText(site.Username)
	v.passwordEntry.SetText(site.AppPassword)
	v.rememberCheck.SetChecked(true)

	// Connect automatically
	//v.connectToWordPress()
}

// deleteSavedSite deletes a saved site
func (v *WordPressSettingsView) deleteSavedSite() {
	if v.selectedSiteIndex < 0 || v.selectedSiteIndex >= len(v.savedSites) {
		return
	}

	siteName := v.savedSites[v.selectedSiteIndex].Name

	// Confirm deletion
	dialog.ShowConfirm("Delete Site", fmt.Sprintf("Are you sure you want to delete the saved site '%s'?", siteName), func(confirmed bool) {
		if !confirmed {
			return
		}

		err := v.wpService.DeleteSavedSite(siteName)
		if err != nil {
			dialog.ShowError(err, v.window)
			return
		}

		v.refreshSavedSites()
	}, v.window)
}

// SetOnConnectionChanged sets the callback for when connection status changes
func (v *WordPressSettingsView) SetOnConnectionChanged(callback func(connected bool)) {
	v.onConnectionChanged = callback
}

// UpdateConnectionStatus updates the connection status label
func (v *WordPressSettingsView) UpdateConnectionStatus(connected bool) {
	if connected {
		v.statusLabel.SetText("Status: Connected")
	} else {
		v.statusLabel.SetText("Status: Disconnected")
	}
}



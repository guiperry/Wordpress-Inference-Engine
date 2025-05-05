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
	"fyne.io/fyne/v2/theme"
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
	view.updateConnectButtonState() // <-- Add initial state update
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

	v.connectButton = widget.NewButton("Connect", nil) // Action set later by updateConnectButtonState

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
	cerebrasKeyEntry *widget.Entry
	geminiKeyEntry   *widget.Entry // Added for Gemini key
	deepseekKeyEntry *widget.Entry // ADDED: Deepseek key
	// Removed modelEntry, replaced with display labels
	primaryModelsLabel   *widget.Label
	fallbackModelsLabel *widget.Label

	// --- ADDED: MOA Default Model Settings ---
	moaPrimaryModelSelect   *widget.Select // Changed from Entry to Select
	moaFallbackModelSelect *widget.Select // Changed from Entry to Select
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
	// --- Remove Provider Selection ---

	// API Key Inputs
	v.cerebrasKeyEntry = widget.NewPasswordEntry()
	v.cerebrasKeyEntry.SetPlaceHolder("Cerebras API Key (loaded from CEREBRAS_API_KEY)")
	if key := os.Getenv("CEREBRAS_API_KEY"); key != "" {
		v.cerebrasKeyEntry.SetText(key)
	}
	saveCerebrasButton := widget.NewButton("Set Cerebras Key Env Var", func() {
		key := v.cerebrasKeyEntry.Text
		if key != "" {
			os.Setenv("CEREBRAS_API_KEY", key)
			dialog.ShowInformation("Restart Required", "Cerebras API key environment variable set.\nPlease restart the application.", v.window)
			v.cerebrasKeyEntry.Disable()
		} else {
			dialog.ShowInformation("Input Required", "Please enter the Cerebras API Key.", v.window)
		}
	})
	v.cerebrasKeyEntry.OnChanged = func(_ string) {
		saveCerebrasButton.Enable() // Enable save button on change
	}

	// --- Add Gemini Key Input ---
	v.geminiKeyEntry = widget.NewPasswordEntry() // Use v.geminiKeyEntry
	v.geminiKeyEntry.SetPlaceHolder("Gemini API Key (loaded from GEMINI_API_KEY)")
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		v.geminiKeyEntry.SetText(key)
	}
	saveGeminiButton := widget.NewButton("Set Gemini Key Env Var", func() {
		key := v.geminiKeyEntry.Text
		if key != "" {
			os.Setenv("GEMINI_API_KEY", key)
			dialog.ShowInformation("Restart Required", "Gemini API key environment variable set.\nPlease restart the application.", v.window)
			v.geminiKeyEntry.Disable()
		} else {
			dialog.ShowInformation("Input Required", "Please enter the Gemini API Key.", v.window)
		}
	})
	v.geminiKeyEntry.OnChanged = func(_ string) {
		saveGeminiButton.Enable() // Enable save button on change
	}

	// --- ADDED: Deepseek Key Input ---
	v.deepseekKeyEntry = widget.NewPasswordEntry()
	v.deepseekKeyEntry.SetPlaceHolder("Deepseek API Key (loaded from DEEPSEEK_API_KEY)")
	if key := os.Getenv("DEEPSEEK_API_KEY"); key != "" {
		v.deepseekKeyEntry.SetText(key)
	}
	saveDeepseekButton := widget.NewButton("Set Deepseek Key Env Var", func() {
		key := v.deepseekKeyEntry.Text
		if key != "" {
			os.Setenv("DEEPSEEK_API_KEY", key)
			dialog.ShowInformation("Restart Required", "Deepseek API key environment variable set.\nPlease restart the application.", v.window)
			v.deepseekKeyEntry.Disable()
		} else {
			dialog.ShowInformation("Input Required", "Please enter the Deepseek API Key.", v.window)
		}
	})
	v.deepseekKeyEntry.OnChanged = func(_ string) {
		saveDeepseekButton.Enable() // Enable save button on change
	}
	// --- End ADDED ---
	// --- Display Configured Models ---
	v.primaryModelsLabel = widget.NewLabel("Primary Models: Loading...")
	v.fallbackModelsLabel = widget.NewLabel("Fallback Models: Loading...")

	// Refresh button to update displayed models (in case service restarts or config changes)
	refreshModelsButton := widget.NewButtonWithIcon("Refresh Models", theme.ViewRefreshIcon(), func() {
		v.refreshDisplayedModels()
	})

	// --- ADDED: MOA Default Model Settings ---
	moaSettingsLabel := widget.NewLabel("MOA Default Models (Affects Mixture-of-Agents):")

	// Create Select widgets, initially empty, will be populated by refreshDisplayedModels
	v.moaPrimaryModelSelect = widget.NewSelect([]string{}, func(selected string) {
		// Optional: Handle selection change directly if needed, otherwise button press is fine
		log.Printf("UI: MOA Primary dropdown selected: %s", selected)
	})

	setMOAPrimaryButton := widget.NewButton("Set MOA Primary", func() {
		model := v.moaPrimaryModelSelect.Selected // Get value from Select
		if model == "" {
			dialog.ShowInformation("Input Required", "Please enter a model name.", v.window)
			return
		}
		err := v.inferenceService.SetMOAPrimaryModel(model)
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to set MOA primary model: %w", err), v.window)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("MOA primary default set to '%s'. MOA reconfigured.", model), v.window)
		}
	})

	v.moaFallbackModelSelect = widget.NewSelect([]string{}, func(selected string) {
		// Optional: Handle selection change directly if needed
		log.Printf("UI: MOA Fallback dropdown selected: %s", selected)
	})

	setMOAFallbackButton := widget.NewButton("Set MOA Fallback", func() {
		// Similar logic to setMOAPrimaryButton, calling SetMOAFallbackModel
		model := v.moaFallbackModelSelect.Selected // Get value from Select
		// ... (validation) ...
		err := v.inferenceService.SetMOAFallbackModel(model)
		// ... (handle error/success dialog) ...
		if err != nil {
			dialog.ShowError(fmt.Errorf("Failed to set MOA fallback model: %w", err), v.window)
		} else {
			dialog.ShowInformation("Success", fmt.Sprintf("MOA fallback/aggregator default set to '%s'. MOA reconfigured.", model), v.window)
		}
	})
	// --- End ADDED ---
	// Create layout
	v.container = container.NewVBox(
		widget.NewLabel("Inference Settings"),
		widget.NewSeparator(),
		widget.NewLabel("Configured Models (Read-Only):"),
		v.primaryModelsLabel,
		v.fallbackModelsLabel,
		refreshModelsButton,
		widget.NewSeparator(),
		widget.NewLabel("API Keys (Set Environment Variable & Restart):"),
		v.cerebrasKeyEntry,
		saveCerebrasButton,
		v.geminiKeyEntry, // Add Gemini key entry
		saveGeminiButton, // Add Gemini save button
		v.deepseekKeyEntry, // ADDED: Deepseek key entry
		saveDeepseekButton, // ADDED: Deepseek save button
		widget.NewSeparator(),
		moaSettingsLabel,
		v.moaPrimaryModelSelect, // Use Select widget
		setMOAPrimaryButton,
		v.moaFallbackModelSelect, // Use Select widget
		setMOAFallbackButton,
	)

	// Initial refresh of displayed models
	v.refreshDisplayedModels()
}

// refreshDisplayedModels updates the labels showing the configured models.
func (v *InferenceSettingsView) refreshDisplayedModels() {
	primaryModels := v.inferenceService.GetPrimaryModels()
	currentPrimary := v.inferenceService.GetProxyModel() // Get current MOA primary

	fallbackModels := v.inferenceService.GetFallbackModels()
	currentFallback := v.inferenceService.GetBaseModel() // Get current MOA fallback

	v.primaryModelsLabel.SetText(fmt.Sprintf("Primary Models: %v", primaryModels))
	v.fallbackModelsLabel.SetText(fmt.Sprintf("Fallback Models: %v", fallbackModels))

	// Update options and selected value for MOA dropdowns
	v.moaPrimaryModelSelect.Options = primaryModels
	v.moaPrimaryModelSelect.SetSelected(currentPrimary) // Set current selection

	v.moaFallbackModelSelect.Options = fallbackModels
	v.moaFallbackModelSelect.SetSelected(currentFallback) // Set current selection
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

// updateConnectButtonState updates the connect button's text and action
func (v *WordPressSettingsView) updateConnectButtonState() {
	if v.wpService == nil {
		log.Println("WordPressSettingsView: Cannot update button state, wpService is nil")
		v.connectButton.SetText("Connect")
		v.connectButton.OnTapped = nil // Or set to a function showing an error
		v.connectButton.Disable()      // Disable if service is missing
		return
	}

	v.connectButton.Enable() // Ensure button is enabled unless explicitly disabled elsewhere

	if v.wpService.IsConnected() {
		v.connectButton.SetText("Disconnect")
		v.connectButton.OnTapped = func() {
			log.Println("Disconnect button tapped. Starting disconnect goroutine.")
			// Disable button immediately to prevent double clicks
			v.connectButton.Disable()
			v.connectButton.SetText("Disconnecting...")
			v.connectButton.Refresh()

			// Perform disconnect in a goroutine
			go func() {
				log.Println("Disconnect goroutine: Calling v.wpService.Disconnect()...") // <-- Add log BEFORE call
				v.wpService.Disconnect()
				log.Println("Disconnect goroutine: v.wpService.Disconnect() returned.") // <-- Add log AFTER call
				
				// --- Directly Update UI Elements After Disconnect ---
				log.Println("Disconnect UI update: Setting status and button directly.")
				v.statusLabel.SetText("Status: Disconnected")
				v.statusLabel.Refresh()

				v.connectButton.SetText("Connect")
				v.connectButton.OnTapped = v.connectToWordPress // Reset action to connect
				v.connectButton.Enable()                       // Ensure button is enabled
				v.connectButton.Refresh()                      // Refresh the button's appearance

				// Notify other parts of the application *after* this view's UI is updated
				if v.onConnectionChanged != nil {
						v.onConnectionChanged(false)
					}
					log.Println("Disconnect UI update: Complete.")
			
			}()
		}
	} else {
		v.connectButton.SetText("Connect")
		v.connectButton.OnTapped = func() {
			// Call the existing connect function
			v.connectToWordPress()
		}
	}
	v.connectButton.Refresh() // Refresh the button to show text change
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
	// v.connectButton.Disable() // Don't disable, let updateConnectButtonState handle it if needed
	v.connectButton.SetText("Connecting...") // Optionally change text during attempt
	v.connectButton.Refresh()

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

		// Ensure progress dialog is hidden in all cases
		defer progress.Hide()

		if !ok {
			// Channel was closed without sending a value, unusual case
			log.Println("connectToWordPress (UI goroutine): 'done' channel closed unexpectedly.")
			// Attempt cleanup just in case
			log.Println("connectToWordPress (UI goroutine): Unexpected close - updating UI state")
			v.updateConnectButtonState()
			v.connectButton.Refresh()
			log.Println("connectToWordPress (UI goroutine): Setting status to Error (unexpected close).")
			v.statusLabel.SetText("Status: Error (Connection Aborted)")
			v.statusLabel.Refresh()
			log.Println("connectToWordPress (UI goroutine): Finished (unexpected close).")
			return
		}

		// --- All UI updates happen here, after the network call is done ---
		log.Println("connectToWordPress (UI goroutine): Hiding progress.")
		progress.Hide() // Hide progress first
		log.Println("connectToWordPress (UI goroutine): Enabling connect button.")
		// v.connectButton.Enable() // Let updateConnectButtonState handle enabling

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
		
		// Update button state and force refresh
		v.updateConnectButtonState()
		v.connectButton.Refresh()
		v.window.Canvas().Refresh(v.connectButton)
		v.statusLabel.Refresh()
		
		// Update button state again to ensure consistency
		v.updateConnectButtonState()
		v.connectButton.Refresh()
		
		if v.onConnectionChanged != nil {
			log.Println("connectToWordPress (UI goroutine): Calling onConnectionChanged(true).")
			v.onConnectionChanged(true)
		}
		
		// Final refresh to ensure all UI updates are visible
		v.window.Canvas().Refresh(v.connectButton)
		v.window.Canvas().Refresh(v.statusLabel)

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
	v.onConnectionChanged = func(connected bool) {
		// Call the original callback first
		if callback != nil {
			callback(connected)
		}
		// Then update the button state
		v.updateConnectButtonState()
	}
}

// UpdateConnectionStatus updates the connection status label
func (v *WordPressSettingsView) UpdateConnectionStatus(connected bool) {
	if connected {
		v.statusLabel.SetText("Status: Connected")
	} else {
		v.statusLabel.SetText("Status: Disconnected")
	}
	v.statusLabel.Refresh()
	v.updateConnectButtonState() // Update button whenever status is explicitly updated
}

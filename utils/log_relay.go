package utils

import (
	"io"
	"log"
	"strings"
	"sync"
)

const maxLogLinesForDialog = 20 // Number of log lines to keep in the dialog display

// LogRelay captures log output and relays it to a UI callback.
type LogRelay struct {
	mu                sync.Mutex
	logMessageChannel chan string
	originalLogOutput io.Writer    // The log output active before this relay started
	uiUpdateCallback  func(string) // Callback to update the UI with new log text
	logBuffer         []string     // Stores the last N log lines
	active            bool
	wg                sync.WaitGroup // To wait for the processing goroutine to finish
}

// NewLogRelay creates a new LogRelay.
// uiUpdateCallback will be called with the aggregated last N log lines.
func NewLogRelay(uiUpdateCallback func(string)) *LogRelay {
	return &LogRelay{
		logMessageChannel: make(chan string, 200), // Buffered channel
		uiUpdateCallback:  uiUpdateCallback,
		logBuffer:         make([]string, 0, maxLogLinesForDialog),
	}
}

// Start begins capturing log output. It replaces the global log output.
func (lr *LogRelay) Start() {
	lr.mu.Lock()
	if lr.active {
		lr.mu.Unlock()
		log.Println("LogRelay: Start called when already active.")
		return
	}
	lr.originalLogOutput = log.Writer() // Capture current global log output
	log.SetOutput(lr)                    // Set this LogRelay as the new global log output
	lr.active = true
	lr.logBuffer = make([]string, 0, maxLogLinesForDialog) // Clear buffer on start
	lr.mu.Unlock()

	lr.wg.Add(1)
	go lr.processLogMessages()
	log.Println("LogRelay: Started and capturing global logs.") // This log will be captured by the relay
}

// Stop ceases log capture and restores the original log output.
func (lr *LogRelay) Stop() {
	lr.mu.Lock()
	if !lr.active {
		lr.mu.Unlock()
		return
	}
	lr.active = false
	// Restore original log output. Any logs after this point from other goroutines
	// or this function will go to the original output.
	if lr.originalLogOutput != nil {
		log.SetOutput(lr.originalLogOutput)
		log.Println("LogRelay: Stopped and restored original log output.") // This goes to the original output
	} else {
		log.Println("LogRelay: Stopped, but originalLogOutput was nil (this should not happen if Start was successful).")
	}
	lr.mu.Unlock()

	close(lr.logMessageChannel) // Signal the processing goroutine to stop
	lr.wg.Wait()                // Wait for the processing goroutine to finish
}

// Write implements io.Writer. This method is called by the log package when LogRelay is set as output.
func (lr *LogRelay) Write(p []byte) (n int, err error) {
	// Atomically get the original output and active state
	lr.mu.Lock()
	originalOutput := lr.originalLogOutput
	isActive := lr.active
	lr.mu.Unlock()

	// Write to the original output first
	if originalOutput != nil {
		originalOutput.Write(p) // Pass through to what was previously logging
	}

	if isActive {
		// Send to channel for async processing and UI update
		// Non-blocking send to prevent log calls from deadlocking if channel is full
		select {
		case lr.logMessageChannel <- string(p):
		default:
			// Log message dropped for UI if channel is full, but it was still written to originalLogOutput
		}
	}
	return len(p), nil
}

// processLogMessages reads from the channel, updates the buffer, and calls the UI callback.
func (lr *LogRelay) processLogMessages() {
	defer lr.wg.Done()
	for message := range lr.logMessageChannel {
		lr.mu.Lock()
		// Split message by newlines, as a single log.Print can contain multiple lines
		lines := strings.Split(strings.TrimSpace(message), "\n")
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				continue
			}
			if len(lr.logBuffer) >= maxLogLinesForDialog {
				lr.logBuffer = lr.logBuffer[1:] // Remove the oldest line
			}
			lr.logBuffer = append(lr.logBuffer, trimmedLine)
		}
		currentLogText := strings.Join(lr.logBuffer, "\n")
		lr.mu.Unlock()

		if lr.uiUpdateCallback != nil {
			lr.uiUpdateCallback(currentLogText) // UI update callback
		}
	}
}
// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/inference_service.go
package inference

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/teilomillet/gollm"
	"github.com/teilomillet/gollm/config"
	"github.com/teilomillet/gollm/llm"

	// Make sure provider packages are imported if needed for registration side effects
	// _ "path/to/your/gemini/provider/package" // If registration is there
	// _ "path/to/your/cerebras/provider/package" // If registration is there
)

// InferenceService manages the interaction with the gollm library and its providers.
type InferenceService struct {
	proxyLLM     llm.LLM          // The LLM instance for the proxy (Cerebras)
	baseLLM      llm.LLM          // The LLM instance for the base (Gemini)
	proxy        *OptimizingProxy // The layer for advanced techniques, now uses both LLMs
	proxyModel   string           // Store the currently configured proxy model (Cerebras)
	baseModel    string           // Store the currently configured base model (Gemini)
	proxyMaxTokens int            // Store the currently configured proxy max tokens
	baseMaxTokens  int            // Store the currently configured base max tokens
	isRunning    bool
	mutex        sync.Mutex
}

// NewInferenceService creates a new instance of InferenceService.
func NewInferenceService() *InferenceService {
	return &InferenceService{
		// Set sensible defaults
		proxyModel:   "llama-4-scout-17b-16e-instruct", // Default Cerebras model
		baseModel:    "gemini-1.5-pro-latest",         // Example default Gemini model
		proxyMaxTokens: 8000,                          // Default Cerebras max tokens (approx)
		baseMaxTokens:  1000000,                       // Default Gemini max tokens (approx)
	}
}

// Start configures the service with both proxy and base providers.
func (s *InferenceService) Start() error {
	log.Println("InferenceService: Starting...")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// --- Configure Proxy LLM (Cerebras) ---
	log.Println("InferenceService: Configuring Proxy LLM (Cerebras)...")
	proxyProviderName := "cerebras"
	proxyAPIKey := os.Getenv("CEREBRAS_API_KEY")
	if proxyAPIKey == "" {
		log.Println("Warning: CEREBRAS_API_KEY environment variable not set for Proxy LLM.")
		// Decide if this is a fatal error or just a warning
		// return errors.New("CEREBRAS_API_KEY environment variable not set")
	}
	proxyOpts := []config.ConfigOption{
		config.SetProvider(proxyProviderName),
		config.SetAPIKey(proxyAPIKey),
		config.SetModel(s.proxyModel),
		config.SetMaxTokens(s.proxyMaxTokens),
	}
	proxyLLMInstance, err := gollm.NewLLM(proxyOpts...)
	if err != nil {
		log.Printf("InferenceService: Failed to create Proxy LLM instance (Cerebras): %v", err)
		s.isRunning = false
		return fmt.Errorf("failed to create proxy LLM instance '%s': %w", proxyProviderName, err)
	}
	if pLLM, ok := proxyLLMInstance.(llm.LLM); ok {
		s.proxyLLM = pLLM
		log.Printf("InferenceService: Proxy LLM (Cerebras) configured successfully. Model: %s", s.proxyModel)
	} else {
		log.Printf("[ERROR] InferenceService: gollm.NewLLM result for proxy does not implement llm.LLM interface!")
		s.isRunning = false
		return fmt.Errorf("internal error: gollm.NewLLM for proxy did not return an instance implementing llm.LLM")
	}

	// --- Configure Base LLM (Gemini) ---
	log.Println("InferenceService: Configuring Base LLM (Gemini)...")
	baseProviderName := "gemini" // Assuming your Gemini provider is registered with this name
	baseAPIKey := os.Getenv("GEMINI_API_KEY") // Load Gemini key
	if baseAPIKey == "" {
		log.Println("Warning: GEMINI_API_KEY environment variable not set for Base LLM.")
		// Decide if fatal or warning
		// return errors.New("GEMINI_API_KEY environment variable not set")
	}
	baseOpts := []config.ConfigOption{
		config.SetProvider(baseProviderName),
		config.SetAPIKey(baseAPIKey),
		config.SetModel(s.baseModel),
		config.SetMaxTokens(s.baseMaxTokens),
	}
	baseLLMInstance, err := gollm.NewLLM(baseOpts...)
	if err != nil {
		log.Printf("InferenceService: Failed to create Base LLM instance (Gemini): %v", err)
		s.isRunning = false
		s.proxyLLM = nil // Clean up proxy if base fails
		return fmt.Errorf("failed to create base LLM instance '%s': %w", baseProviderName, err)
	}
	if bLLM, ok := baseLLMInstance.(llm.LLM); ok {
		s.baseLLM = bLLM
		log.Printf("InferenceService: Base LLM (Gemini) configured successfully. Model: %s", s.baseModel)
	} else {
		log.Printf("[ERROR] InferenceService: gollm.NewLLM result for base does not implement llm.LLM interface!")
		s.isRunning = false
		s.proxyLLM = nil
		s.baseLLM = nil
		return fmt.Errorf("internal error: gollm.NewLLM for base did not return an instance implementing llm.LLM")
	}

	// --- Create the Optimizing Proxy ---
	// Pass both LLM instances to the proxy
	s.proxy = NewOptimizingProxy(s.proxyLLM, s.baseLLM)
	log.Println("InferenceService: OptimizingProxy created.")

	s.isRunning = true
	log.Println("InferenceService: Started successfully with Proxy (Cerebras) and Base (Gemini).")
	return nil
}

// Stop cleans up the clients
func (s *InferenceService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isRunning {
		log.Println("InferenceService: Already stopped.")
		return nil
	}
	s.isRunning = false
	s.proxyLLM = nil // Clear LLMs
	s.baseLLM = nil
	s.proxy = nil // Clear proxy
	s.proxyModel = ""
	s.baseModel = ""
	log.Println("InferenceService stopped.")
	return nil
}

// GenerateText ALWAYS delegates to the proxy.
func (s *InferenceService) GenerateText(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.proxy == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or proxy not configured")
	}
	proxyInstance := s.proxy // Capture instance under lock
	s.mutex.Unlock()

	ctx := context.Background()

	log.Println("InferenceService: Delegating generation request to OptimizingProxy...")
	response, err := proxyInstance.GenerateSimple(ctx, promptText) // Proxy decides internally
	if err != nil {
		return "", err // Error message already includes context
	}
	log.Println("InferenceService: Generation successful via OptimizingProxy.")
	return response, nil
}

// --- Methods to remove/deprecate ---
// SwitchToProvider - No longer applicable
// GetActiveProviderName - Roles are fixed now

// --- Methods to potentially add/modify ---

// SetProxyModel allows changing the Cerebras model dynamically
func (s *InferenceService) SetProxyModel(model string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isRunning { return errors.New("service not running") }
	if model == "" { return errors.New("model name cannot be empty") }
	if model == s.proxyModel { return nil } // No change

	log.Printf("InferenceService: Setting Proxy (Cerebras) model to '%s'", model)
	s.proxyModel = model
	// Re-configure ONLY the proxy LLM part
	proxyAPIKey := os.Getenv("CEREBRAS_API_KEY")
	proxyOpts := []config.ConfigOption{
		config.SetProvider("cerebras"),
		config.SetAPIKey(proxyAPIKey),
		config.SetModel(s.proxyModel),
		config.SetMaxTokens(s.proxyMaxTokens),
	}
	proxyLLMInstance, err := gollm.NewLLM(proxyOpts...)
	if err != nil { /* handle error, maybe revert s.proxyModel */ return err }
	if pLLM, ok := proxyLLMInstance.(llm.LLM); ok {
		s.proxyLLM = pLLM
		// Update the proxy instance with the new proxyLLM
		s.proxy = NewOptimizingProxy(s.proxyLLM, s.baseLLM)
		log.Printf("InferenceService: Proxy model updated to '%s'", s.proxyModel)
		return nil
	}
	return fmt.Errorf("internal error: failed to reconfigure proxy LLM")
}

// SetBaseModel allows changing the Gemini model dynamically (similar logic)
func (s *InferenceService) SetBaseModel(model string) error {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    if !s.isRunning { return errors.New("service not running") }
    if model == "" { return errors.New("model name cannot be empty") }
    if model == s.baseModel { return nil } // No change

    log.Printf("InferenceService: Setting Base (Gemini) model to '%s'", model)
    s.baseModel = model
    // Re-configure ONLY the base LLM part
    baseAPIKey := os.Getenv("GEMINI_API_KEY")
    baseOpts := []config.ConfigOption{
        config.SetProvider("gemini"), // Use the correct provider name
        config.SetAPIKey(baseAPIKey),
        config.SetModel(s.baseModel),
        config.SetMaxTokens(s.baseMaxTokens),
    }
    baseLLMInstance, err := gollm.NewLLM(baseOpts...)
    if err != nil { /* handle error, maybe revert s.baseModel */ return err }
    if bLLM, ok := baseLLMInstance.(llm.LLM); ok {
        s.baseLLM = bLLM
        // Update the proxy instance with the new baseLLM
        s.proxy = NewOptimizingProxy(s.proxyLLM, s.baseLLM)
        log.Printf("InferenceService: Base model updated to '%s'", s.baseModel)
        return nil
    }
    return fmt.Errorf("internal error: failed to reconfigure base LLM")
}


// GetProxyModel returns the name of the proxy model.
func (s *InferenceService) GetProxyModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.proxyModel
}

// GetBaseModel returns the name of the base model.
func (s *InferenceService) GetBaseModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.baseModel
}

// IsRunning checks the client status
func (s *InferenceService) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// GetName identifies the service structure
func (s *InferenceService) GetName() string {
	return "InferenceService(Proxy:Cerebras, Base:Gemini)"
}

// Other generation methods (CoT, Reflection, Structured) should also just call the proxy
func (s *InferenceService) GenerateTextWithCoT(promptText string) (string, error) {
    s.mutex.Lock()
    if !s.isRunning || s.proxy == nil { s.mutex.Unlock(); return "", errors.New("service not running") }
    proxyInstance := s.proxy
    s.mutex.Unlock()
    ctx := context.Background()
    log.Println("InferenceService: Delegating CoT generation to OptimizingProxy...")
    return proxyInstance.GenerateWithCoT(ctx, promptText)
}

func (s *InferenceService) GenerateTextWithReflection(promptText string) (string, error) {
    s.mutex.Lock()
    if !s.isRunning || s.proxy == nil { s.mutex.Unlock(); return "", errors.New("service not running") }
    proxyInstance := s.proxy
    s.mutex.Unlock()
    ctx := context.Background()
    log.Println("InferenceService: Delegating Reflection generation to OptimizingProxy...")
    return proxyInstance.GenerateWithReflection(ctx, promptText)
}

func (s *InferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
    s.mutex.Lock()
    if !s.isRunning || s.proxy == nil { s.mutex.Unlock(); return "", errors.New("service not running") }
    proxyInstance := s.proxy
    s.mutex.Unlock()
    ctx := context.Background()
    log.Println("InferenceService: Delegating structured output generation to OptimizingProxy...")
    return proxyInstance.GenerateStructuredOutput(ctx, content, schema)
}

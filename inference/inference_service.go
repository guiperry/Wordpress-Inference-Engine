// /home/gperry/Documents/GitHub/cloud-equities/FIG_Inference/inference/inference_service.go
package inference

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	

	"github.com/teilomillet/gollm"
	"github.com/teilomillet/gollm/config"
	"github.com/teilomillet/gollm/llm" // Import the base llm package

	// Blank import should be in main.go
)

// InferenceService manages the interaction with the gollm library and its providers.
type InferenceService struct {
	baseLLM      llm.LLM          // The underlying LLM configured with a provider
	proxy        *OptimizingProxy // The layer for advanced techniques
	providerName string           // Store the name of the configured provider
	currentModel string           // Store the currently configured model
	maxTokens    int              // Store the currently configured max tokens
	isRunning    bool
	mutex        sync.Mutex
}

// NewInferenceService creates a new instance of InferenceService.
func NewInferenceService() *InferenceService {
	return &InferenceService{
		maxTokens: 1000, // Default max tokens
	}
}

// Start configures the service with a default provider (e.g., "cerebras").
func (s *InferenceService) Start() error {
	defaultProvider := "cerebras"
	log.Printf("InferenceService: Starting with default provider '%s'...", defaultProvider)
	s.mutex.Lock()
	err := s.configureProvider(defaultProvider) // This now configures baseLLM and proxy
	s.mutex.Unlock()
	if err != nil {
		log.Printf("InferenceService: Failed to start default provider '%s': %v Service not running.", defaultProvider, err)
		return fmt.Errorf("failed to start default inference provider '%s': %w", defaultProvider, err)
	}
	log.Printf("InferenceService: Started successfully with provider '%s'", s.providerName)
	return nil
}

// configureProvider sets up the base llm.LLM and the OptimizingProxy.
// NOTE: This method assumes the caller holds the mutex.
func (s *InferenceService) configureProvider(providerName string) error {
	// Ensure providerName is lowercase for consistent lookup and comparison
	providerName = strings.ToLower(providerName)

	if s.isRunning && s.providerName == providerName {
		log.Printf("InferenceService: Already running with provider %s. No change needed.", providerName)
		return nil
	}

	log.Printf("InferenceService: Configuring provider: %s", providerName)

	var options []config.ConfigOption
	var apiKey string
	model := ""
	maxTokens := s.maxTokens

	options = append(options, config.SetProvider(providerName))

	switch providerName {
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
		model = "gpt-3.5-turbo"
		if apiKey == "" {
			log.Println("Warning: OPENAI_API_KEY not set for OpenAI provider.")
		}
	case "cerebras":
		apiKey = os.Getenv("CEREBRAS_API_KEY")
		model = "llama-4-scout-17b-16e-instruct"
		if apiKey == "" {
			return errors.New("CEREBRAS_API_KEY environment variable not set")
		}
	default:
		log.Printf("Warning: Attempting to configure unknown provider '%s'.", providerName)
		envKey := fmt.Sprintf("%s_API_KEY", strings.ToUpper(providerName))
		apiKey = os.Getenv(envKey)
		if apiKey == "" {
			log.Printf("Warning: No API key found in env var '%s' for provider '%s'", envKey, providerName)
		}
		model = "default-model"
	}

	if s.currentModel != "" {
		model = s.currentModel
	} else {
		s.currentModel = model
	}

	if apiKey != "" {
		options = append(options, config.SetAPIKey(apiKey))
	}
	if model != "" {
		options = append(options, config.SetModel(model))
	}
	options = append(options, config.SetMaxTokens(maxTokens))

	// Create the gollm.LLM instance
	log.Printf("[DEBUG] configureProvider: Preparing to call gollm.NewLLM for provider '%s' with options...", providerName)
	llmInstance, err := gollm.NewLLM(options...)
	if err != nil {
		// Log the specific error from gollm.NewLLM
		log.Printf("[DEBUG] configureProvider: gollm.NewLLM failed: %v", err)
		// Reset state
		s.isRunning = false // Ensure state reflects failure
		s.baseLLM = nil
		s.proxy = nil
		s.providerName = ""
		s.currentModel = ""
		return fmt.Errorf("failed to create internal LLM instance for provider '%s': %w", providerName, err)
	}
	log.Printf("[DEBUG] configureProvider: gollm.NewLLM succeeded for provider '%s'.", providerName)

	// Store the base llm.LLM interface
	if baseLLM, ok := llmInstance.(llm.LLM); ok {
		s.baseLLM = baseLLM
		// Create or update the proxy with the new base LLM
		s.proxy = NewOptimizingProxy(s.baseLLM) // Re-creates proxy, which is fine for now
	} else {
		s.isRunning = false
		s.baseLLM = nil
		s.proxy = nil
		s.providerName = ""
		s.currentModel = ""
		log.Printf("[ERROR] configureProvider: gollm.NewLLM result does not implement llm.LLM interface!")
		return fmt.Errorf("internal error: gollm.NewLLM did not return an instance implementing llm.LLM")
	}

	s.providerName = providerName
	s.isRunning = true
	log.Printf("InferenceService: Configured successfully with provider '%s', model '%s'", providerName, s.currentModel)
	return nil
}

// Stop cleans up the client
func (s *InferenceService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isRunning {
		log.Println("InferenceService: Already stopped.")
		return nil
	}
	s.isRunning = false
	s.baseLLM = nil // Clear base LLM
	s.proxy = nil   // Clear proxy
	s.providerName = ""
	s.currentModel = ""
	log.Println("InferenceService stopped.")
	return nil
}

// GenerateText uses the OptimizingProxy for simple generation.
func (s *InferenceService) GenerateText(promptText string) (string, error) {
	s.mutex.Lock()
	// Check if proxy is available and running
	if !s.isRunning || s.proxy == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or proxy not configured")
	}
	proxyInstance := s.proxy // Capture instance under lock
	providerName := s.providerName
	s.mutex.Unlock()

	ctx := context.Background()

	// Delegate to the proxy's simple generation method
	log.Printf("InferenceService: Delegating simple generation to proxy (%s)...", providerName)
	response, err := proxyInstance.GenerateSimple(ctx, promptText)
	if err != nil {
		// Error message already includes context from proxy/base LLM
		return "", err
	}
	log.Printf("InferenceService: Simple generation successful via proxy (%s).", providerName)
	return response, nil
}

// GenerateTextWithCoT exposes Chain-of-Thought generation via the proxy.
func (s *InferenceService) GenerateTextWithCoT(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.proxy == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or proxy not configured")
	}
	proxyInstance := s.proxy
	providerName := s.providerName
	s.mutex.Unlock()

	ctx := context.Background()
	log.Printf("InferenceService: Delegating CoT generation to proxy (%s)...", providerName)
	response, err := proxyInstance.GenerateWithCoT(ctx, promptText)
	if err != nil {
		return "", err
	}
	log.Printf("InferenceService: CoT generation successful via proxy (%s).", providerName)
	return response, nil
}

// GenerateTextWithReflection exposes Reflection generation via the proxy.
func (s *InferenceService) GenerateTextWithReflection(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.proxy == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or proxy not configured")
	}
	proxyInstance := s.proxy
	providerName := s.providerName
	s.mutex.Unlock()

	ctx := context.Background()
	log.Printf("InferenceService: Delegating Reflection generation to proxy (%s)...", providerName)
	response, err := proxyInstance.GenerateWithReflection(ctx, promptText)
	if err != nil {
		return "", err
	}
	log.Printf("InferenceService: Reflection generation successful via proxy (%s).", providerName)
	return response, nil
}

// GenerateStructuredOutput exposes structured output generation via the proxy.
func (s *InferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.proxy == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or proxy not configured")
	}
	proxyInstance := s.proxy
	providerName := s.providerName
	s.mutex.Unlock()

	ctx := context.Background()
	log.Printf("InferenceService: Delegating structured output generation to proxy (%s)...", providerName)
	response, err := proxyInstance.GenerateStructuredOutput(ctx, content, schema)
	if err != nil {
		return "", err
	}
	log.Printf("InferenceService: Structured output generation successful via proxy (%s).", providerName)
	return response, nil
}


// SwitchToProvider reconfigures the base LLM and the proxy.
func (s *InferenceService) SwitchToProvider(providerName string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Ensure providerName is lowercase before passing to configureProvider
	providerName = strings.ToLower(providerName)

	log.Printf("InferenceService: Attempting to switch to provider '%s'", providerName)
	s.currentModel = "" // Reset model when switching provider
	err := s.configureProvider(providerName) // Reconfigures baseLLM and proxy
	if err != nil {
		log.Printf("InferenceService: Failed to switch to provider '%s': %v", providerName, err)
		return err
	}
	log.Printf("InferenceService: Successfully switched to provider '%s'", s.providerName)
	return nil
}

// SetModel reconfigures the base LLM and the proxy with a new model.
func (s *InferenceService) SetModel(model string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return errors.New("inference service is not running")
	}
	if model == "" {
		return errors.New("model name cannot be empty")
	}
	if model == s.currentModel {
		log.Printf("InferenceService: Model already set to '%s'. No change needed.", model)
		return nil
	}

	log.Printf("InferenceService: Setting model to '%s' for provider '%s'", model, s.providerName)
	s.currentModel = model // Store the new desired model

	// Re-configure the provider, which updates baseLLM and proxy
	// configureProvider uses s.providerName (which is already set)
	err := s.configureProvider(s.providerName)
	if err != nil {
		log.Printf("InferenceService: Failed to reconfigure provider '%s' with new model '%s': %v", s.providerName, model, err)
		// Consider reverting s.currentModel if needed
		// s.currentModel = previousModel // Need to store previous model if revert is desired
		return fmt.Errorf("failed to set model: %w", err)
	}

	log.Printf("InferenceService: Model successfully updated to '%s'", s.currentModel)
	return nil
}

// IsRunning checks the client status
func (s *InferenceService) IsRunning() bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.isRunning
}

// GetName identifies the service and its active provider
func (s *InferenceService) GetName() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.providerName != "" {
		return "InferenceService(gollm:" + s.providerName + ")"
	}
	return "InferenceService(unconfigured)"
}

// GetActiveProviderName returns the name of the provider currently configured.
func (s *InferenceService) GetActiveProviderName() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.providerName
}

// GetCurrentModel returns the name of the model currently configured.
func (s *InferenceService) GetCurrentModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.currentModel
}

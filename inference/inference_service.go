// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/inference_service.go
package inference

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/teilomillet/gollm"
	"github.com/teilomillet/gollm/config"
	"github.com/teilomillet/gollm/llm"
)

// InferenceService manages the interaction with the gollm library and its providers.
type InferenceService struct {
	proxyLLM llm.LLM // The LLM instance for the proxy (Cerebras)
	baseLLM  llm.LLM // The LLM instance for the base (Gemini)
	// proxy        *OptimizingProxy // REMOVED
	delegator      *DelegatorService // ADDED: Handles delegation and fallback
	proxyModel     string
	baseModel      string
	proxyMaxTokens int
	baseMaxTokens  int
	isRunning      bool
	mutex          sync.Mutex
	moa            *gollm.MOA
}

// NewInferenceService creates a new instance of InferenceService.
func NewInferenceService() *InferenceService {
	return &InferenceService{
		proxyModel:     "llama-4-scout-17b-16e-instruct",
		baseModel:      "gemini-2.0-flash",
		proxyMaxTokens: 5000,    // Default Cerebras max tokens (approx) - Delegator uses this too
		baseMaxTokens:  150000, // Default Gemini max tokens (approx)
	}
}

// Start configures the service with both proxy and base providers and the delegator.
func (s *InferenceService) Start() error {
	log.Println("InferenceService: Starting...")
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// --- Configure Proxy LLM (Cerebras) ---
	log.Println("InferenceService: Configuring Proxy LLM (Cerebras)...")
	proxyProviderName := "cerebras"
	proxyAPIKey := os.Getenv("CEREBRAS_API_KEY")
	// ... (error handling for key) ...
	proxyOpts := []config.ConfigOption{
		config.SetProvider(proxyProviderName),
		config.SetAPIKey(proxyAPIKey),
		config.SetModel(s.proxyModel),
		config.SetMaxTokens(s.proxyMaxTokens),
	}
	proxyLLMInstance, err := gollm.NewLLM(proxyOpts...)
	if err != nil { /* handle error */
		return err
	}
	if pLLM, ok := proxyLLMInstance.(llm.LLM); ok {
		s.proxyLLM = pLLM
		// TODO: Optionally retrieve actual model used from pLLM if interface allows
		// s.proxyModel = pLLM.GetModel() // Hypothetical
		log.Printf("InferenceService: Proxy LLM (Cerebras) configured successfully.")
	} else { /* handle error */
		return fmt.Errorf("internal error: proxy LLM instance type mismatch")
	}

	// --- Configure Base LLM (Gemini) ---
	log.Println("InferenceService: Configuring Base LLM (Gemini)...")
	baseProviderName := "gemini"
	baseAPIKey := os.Getenv("GEMINI_API_KEY")
	// ... (error handling for key) ...
	baseOpts := []config.ConfigOption{
		config.SetProvider(baseProviderName),
		config.SetAPIKey(baseAPIKey),
		config.SetModel(s.baseModel), 
		config.SetMaxTokens(s.baseMaxTokens),
	}
	baseLLMInstance, err := gollm.NewLLM(baseOpts...)
	if err != nil { /* handle error */
		s.proxyLLM = nil
		return err
	}
	if bLLM, ok := baseLLMInstance.(llm.LLM); ok {
		s.baseLLM = bLLM
		// TODO: Optionally retrieve actual model used from bLLM if interface allows
		// s.baseModel = bLLM.GetModel() // Hypothetical
		log.Printf("InferenceService: Base LLM (Gemini) configured successfully.")
	} else { /* handle error */
		s.proxyLLM = nil
		return fmt.Errorf("internal error: base LLM instance type mismatch")
	}

	// --- Create the MOA Service ---
	log.Println("InferenceService: Configuring MOA...")
	if s.proxyLLM != nil && s.baseLLM != nil {
		moaCfg := gollm.MOAConfig{
			Iterations: 2,
			Models: []config.ConfigOption{
				func(cfg *config.Config) {
					for _, opt := range proxyOpts {
						opt(cfg)
					}
				},
				func(cfg *config.Config) {
					for _, opt := range baseOpts {
						opt(cfg)
					}
				},
			},
			MaxParallel:  2,
			AgentTimeout: 60 * time.Second,
		}
		aggregatorOpts := baseOpts
		moaInstance, moaErr := gollm.NewMOA(moaCfg, aggregatorOpts...)
		if moaErr != nil {
			log.Printf("[ERROR] InferenceService: Failed to create MOA instance: %v", moaErr)
			s.moa = nil // Ensure it's nil on error
		} else {
			s.moa = moaInstance // Store the MOA instance
			log.Println("InferenceService: MOA instance created successfully.")
		}
	} else {
		log.Println("[WARN] InferenceService: Skipping MOA configuration because one or both LLMs failed to initialize.")
		s.moa = nil
	}
	// --- End MOA Creation ---

	// --- Create the Delegator Service (Pass MOA instance for internal use) ---
	s.delegator = NewDelegatorService(s.proxyLLM, s.baseLLM, s.moa) // Pass MOA
	if s.delegator == nil {
		log.Println("[ERROR] InferenceService: Failed to create DelegatorService.")
		s.isRunning = false
		s.proxyLLM = nil
		s.baseLLM = nil
		s.moa = nil
		return fmt.Errorf("failed to create delegator service")
	}
	log.Println("InferenceService: DelegatorService created.")

	s.isRunning = true
	log.Println("InferenceService: Started successfully.")
	return nil
}

// Stop cleans up the clients and delegator
func (s *InferenceService) Stop() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.isRunning {
		return nil
	}
	s.isRunning = false
	s.proxyLLM = nil
	s.baseLLM = nil
	s.moa = nil       // Clear MOA instance
	s.delegator = nil // Clear delegator
	s.proxyModel = ""
	s.baseModel = ""
	log.Println("InferenceService stopped.")
	return nil
}

// GenerateText delegates to the DelegatorService.
func (s *InferenceService) GenerateText(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running or delegator not configured")
	}
	delegatorInstance := s.delegator // Capture instance under lock
	s.mutex.Unlock()

	ctx := context.Background()
	log.Println("InferenceService: Delegating generation request to DelegatorService...")
	response, err := delegatorInstance.GenerateSimple(ctx, promptText) // Call delegator
	if err != nil {
		return "", err
	}
	log.Println("InferenceService: Generation successful via DelegatorService.")
	return response, nil
}

// --- ADDED: GenerateTextWithMOA ---
// GenerateTextWithMOA directly delegates to the MOA instance for testing.
func (s *InferenceService) GenerateTextWithMOA(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning {
		s.mutex.Unlock()
		return "", errors.New("inference service is not running")
	}
	if s.moa == nil {
		s.mutex.Unlock()
		return "", errors.New("MOA (Mixture of Agents) is not configured or failed to initialize")
	}
	moaInstance := s.moa // Capture instance under lock
	s.mutex.Unlock()

	ctx := context.Background() // Consider allowing context passing
	log.Println("InferenceService: Delegating direct generation request to MOA...")
	// Note: MOA's Generate might have its own internal timeouts based on AgentTimeout
	response, err := moaInstance.Generate(ctx, promptText)
	if err != nil {
		log.Printf("InferenceService: Direct MOA generation failed: %v", err)
		return "", fmt.Errorf("MOA generation failed: %w", err)
	}
	log.Println("InferenceService: Direct generation successful via MOA.")
	return response, nil
}

// --- Update other generation methods to use DelegatorService ---

func (s *InferenceService) GenerateTextWithCoT(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	ctx := context.Background()
	log.Println("InferenceService: Delegating CoT generation to DelegatorService...")
	return delegatorInstance.GenerateWithCoT(ctx, promptText) // Call delegator
}

func (s *InferenceService) GenerateTextWithReflection(promptText string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	ctx := context.Background()
	log.Println("InferenceService: Delegating Reflection generation to DelegatorService...")
	return delegatorInstance.GenerateWithReflection(ctx, promptText) // Call delegator
}

func (s *InferenceService) GenerateStructuredOutput(content string, schema string) (string, error) {
	s.mutex.Lock()
	if !s.isRunning || s.delegator == nil {
		s.mutex.Unlock()
		return "", errors.New("service not running")
	}
	delegatorInstance := s.delegator
	s.mutex.Unlock()
	ctx := context.Background()
	log.Println("InferenceService: Delegating structured output generation to DelegatorService...")
	return delegatorInstance.GenerateStructuredOutput(ctx, content, schema) // Call delegator
}

// --- Model Setting Methods ---
// Need to recreate MOA and update Delegator

func (s *InferenceService) SetProxyModel(model string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// ... (validation) ...
	if model == s.proxyModel {
		return nil
	}

	log.Printf("InferenceService: Setting Proxy (Cerebras) model to '%s'", model)
	// Re-configure the proxy LLM instance
	proxyAPIKey := os.Getenv("CEREBRAS_API_KEY")
	newProxyOpts := []config.ConfigOption{
		config.SetAPIKey(proxyAPIKey),
		/* ... store new opts ... */
	}
	proxyLLMInstance, err := gollm.NewLLM(newProxyOpts...)
	if err != nil {
		return fmt.Errorf("failed to reconfigure proxy LLM: %w", err)
	}

	if pLLM, ok := proxyLLMInstance.(llm.LLM); ok {
		s.proxyLLM = pLLM
		s.proxyModel = model

		// Recreate MOA with new proxy opts
		baseAPIKey := os.Getenv("GEMINI_API_KEY")
		baseOpts := []config.ConfigOption{
			config.SetAPIKey(baseAPIKey),
			/* ... get current base opts ... */
		}
		var newMoaInstance *gollm.MOA
		moaCfg := gollm.MOAConfig{ /* ... */
			Models: []config.ConfigOption{
				func(cfg *config.Config) {
					for _, opt := range newProxyOpts {
						opt(cfg)
					}
				},
				func(cfg *config.Config) {
					for _, opt := range baseOpts {
						opt(cfg)
					}
				},
			}, /* ... */
		}
		moaInstance, err := gollm.NewMOA(moaCfg, baseOpts...)
		if err != nil {
			log.Printf("[ERROR] Failed to recreate MOA after proxy model change: %v", err)
			newMoaInstance = nil
		} else {
			newMoaInstance = moaInstance
		}
		s.moa = newMoaInstance // Update service's MOA instance

		// Recreate Delegator with new LLM and new MOA
		s.delegator = NewDelegatorService(s.proxyLLM, s.baseLLM, s.moa) // Pass updated MOA
		if s.delegator == nil {                                         /* handle error */
		}

		log.Printf("InferenceService: Proxy model updated to '%s' and services refreshed", s.proxyModel)
		return nil
	}
	return fmt.Errorf("internal error: failed to cast reconfigured proxy LLM")
}

func (s *InferenceService) SetBaseModel(model string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// ... (validation) ...
	if model == s.baseModel {
		return nil
	}

	log.Printf("InferenceService: Setting Base (Gemini) model to '%s'", model)
	// Re-configure the base LLM instance
	baseAPIKey := os.Getenv("GEMINI_API_KEY")
	newBaseOpts := []config.ConfigOption{
		config.SetAPIKey(baseAPIKey),
		/* ... store new opts ... */
	}
	baseLLMInstance, err := gollm.NewLLM(newBaseOpts...)
	if err != nil {
		return fmt.Errorf("failed to reconfigure base LLM: %w", err)
	}

	if bLLM, ok := baseLLMInstance.(llm.LLM); ok {
		s.baseLLM = bLLM
		s.baseModel = model

		// Recreate MOA with new base opts (for layers and aggregator)
		proxyAPIKey := os.Getenv("CEREBRAS_API_KEY")
		proxyOpts := []config.ConfigOption{
			config.SetAPIKey(proxyAPIKey),
			/* ... get current proxy opts ... */
		}
		var newMoaInstance *gollm.MOA
		moaCfg := gollm.MOAConfig{ /* ... */
			Models: []config.ConfigOption{
				func(cfg *config.Config) {
					for _, opt := range proxyOpts {
						opt(cfg)
					}
				},
				func(cfg *config.Config) {
					for _, opt := range newBaseOpts {
						opt(cfg)
					}
				},
			}, /* ... */
		}
		moaInstance, err := gollm.NewMOA(moaCfg, newBaseOpts...) // Aggregator uses NEW base opts
		if err != nil {
			log.Printf("[ERROR] Failed to recreate MOA after base model change: %v", err)
			newMoaInstance = nil
		} else {
			newMoaInstance = moaInstance
		}
		s.moa = newMoaInstance // Update service's MOA instance

		// Recreate Delegator with new LLM and new MOA
		s.delegator = NewDelegatorService(s.proxyLLM, s.baseLLM, s.moa) // Pass updated MOA
		if s.delegator == nil {                                         /* handle error */
		}

		log.Printf("InferenceService: Base model updated to '%s' and services refreshed", s.baseModel)
		return nil
	}
	return fmt.Errorf("internal error: failed to cast reconfigured base LLM")
}

// GetProxyModel returns the name of the proxy model.
func (s *InferenceService) GetProxyModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// TODO: Ideally retrieve from s.proxyLLM if possible after Start
	return s.proxyModel
}

// GetBaseModel returns the name of the base model.
func (s *InferenceService) GetBaseModel() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// TODO: Ideally retrieve from s.baseLLM if possible after Start
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
	return "InferenceService(Delegator+MOA)" // Updated name
}

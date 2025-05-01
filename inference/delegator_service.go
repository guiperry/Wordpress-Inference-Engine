// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/delegator_service.go
package inference

import (
	"context"
	"fmt"
	"log"
	"strings"


	"github.com/teilomillet/gollm/llm"
	"github.com/teilomillet/gollm" // Import gollm for MOA type
	// Add other necessary imports if message conversion or specific types are moved here
)

// DelegatorService handles request delegation between a primary (proxy)
// and a secondary (base) LLM, including fallback logic and MOA orchestration.
type DelegatorService struct {
	proxyLLM llm.LLM     // The primary LLM instance (e.g., Cerebras)
	baseLLM  llm.LLM     // The secondary/fallback LLM instance (e.g., Gemini)
	moa      *gollm.MOA // The MOA instance (optional)

	// Configuration for delegation logic
	proxyTokenLimit int
}

// NewDelegatorService creates a new delegator instance.
// It requires both LLM instances and accepts an optional MOA instance.
func NewDelegatorService(primaryLLM llm.LLM, secondaryLLM llm.LLM, moaInstance *gollm.MOA) *DelegatorService { // Added moaInstance
	if primaryLLM == nil || secondaryLLM == nil {
		log.Println("CRITICAL: NewDelegatorService called with nil primaryLLM or secondaryLLM")
		return nil
	}
	if moaInstance == nil {
		log.Println("[WARN] NewDelegatorService: MOA instance is nil. MOA features will be disabled.")
	}
	return &DelegatorService{
		proxyLLM:        primaryLLM,
		baseLLM:         secondaryLLM,
		moa:             moaInstance, // Store MOA instance
		proxyTokenLimit: 8000,
	}
}
// --- Helper Functions (Moved from OptimizingProxy) ---

// estimateTokens provides a very basic token estimation.
// Replace with a proper tokenizer (like tiktoken) for accuracy.
func estimateTokens(text string) int {
	// Very rough estimate: 1 token ~ 3-4 chars in English
	// This is highly inaccurate and should be replaced.
	return len(text) / 3
}

// shouldRetryWithError determines if the given error warrants a fallback attempt to the base LLM.
// Customize this logic based on the errors observed from the primary LLM (Cerebras).
func (d *DelegatorService) shouldFallbackOnError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	log.Printf("DelegatorService: Evaluating error for fallback: %s", errStr)

	// Allow Fallback on context length exceeded
	if strings.Contains(errStr, "context_length_exceeded") {
		log.Println("DelegatorService: Decision: Allowing Fallback (Context Length Exceeded)")
		return true
	}

	// Add other conditions where fallback is desired (e.g., specific server errors, timeouts)
	// if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "status code 5") {
	// 	   log.Println("DelegatorService: Decision: Allowing Fallback (Transient Error)")
	//     return true
	// }

	// Default: Fallback on most errors for now (can be refined)
	log.Println("DelegatorService: Decision: Allowing Fallback (Default)")
	return true
}

// executeGenerationWithFallback selects the appropriate LLM based on token estimation,
// executes the generation, and performs fallback if necessary.
func (d *DelegatorService) executeGenerationWithFallback(ctx context.Context, promptText string, operationName string) (string, error) {
	if d.proxyLLM == nil || d.baseLLM == nil {
		return "", fmt.Errorf("delegator service (%s): not properly configured", operationName)
	}

	estimatedTokens := estimateTokens(promptText)
	log.Printf("DelegatorService (%s): Estimated tokens: %d (Primary limit: %d)", operationName, estimatedTokens, d.proxyTokenLimit)

	var initialTargetLLM llm.LLM
	var initialTargetName string
	var usePrimaryInitially bool

	// --- Initial Delegation Logic ---
	if estimatedTokens > d.proxyTokenLimit {
		log.Printf("DelegatorService (%s): Delegating directly to Secondary LLM (Base)...", operationName)
		initialTargetLLM = d.baseLLM
		initialTargetName = "Secondary (Base)"
		usePrimaryInitially = false
	} else {
		log.Printf("DelegatorService (%s): Attempting with Primary LLM (Proxy)...", operationName)
		initialTargetLLM = d.proxyLLM
		initialTargetName = "Primary (Proxy)"
		usePrimaryInitially = true
	}
	// --- End Initial Delegation Logic ---

	prompt := llm.NewPrompt(promptText)

	// --- Attempt 1: Use the initially chosen LLM ---
	log.Printf("DelegatorService (%s): Attempting generation with %s", operationName, initialTargetName)
	response, err := initialTargetLLM.Generate(ctx, prompt)

	// --- Fallback Logic ---
	if usePrimaryInitially && err != nil && d.shouldFallbackOnError(err) {
		log.Printf("DelegatorService (%s): Initial generation with %s failed: %v. Attempting fallback to Secondary LLM (Base)...", operationName, initialTargetName, err)

		secondaryTargetName := "Secondary (Base)"
		fallbackResponse, fallbackErr := d.baseLLM.Generate(ctx, prompt)

		if fallbackErr != nil {
			log.Printf("DelegatorService (%s): Fallback generation with %s also failed: %v", operationName, secondaryTargetName, fallbackErr)
			return "", fmt.Errorf("%s initial generation failed (%s: %w), fallback failed (%s: %v)",
				operationName, initialTargetName, err, secondaryTargetName, fallbackErr)
		}

		log.Printf("DelegatorService (%s): Fallback generation with %s successful.", operationName, secondaryTargetName)
		return fallbackResponse, nil
	}
	// --- End Fallback Logic ---

	if err != nil {
		log.Printf("DelegatorService (%s): Generation failed using %s: %v. No fallback attempted or applicable.", operationName, initialTargetName, err)
		return "", fmt.Errorf("%s generation failed using %s: %w", operationName, initialTargetName, err)
	}

	log.Printf("DelegatorService (%s): Generation successful using %s.", operationName, initialTargetName)
	return response, nil
}

// --- Generation Methods ---

// GenerateSimple uses standard delegation/fallback ONLY.
func (d *DelegatorService) GenerateSimple(ctx context.Context, promptText string) (string, error) {
	// MOA is NOT used for simple generation in this design
	return d.executeGenerationWithFallback(ctx, promptText, "Simple")
}

// GenerateWithCoT uses MOA if available, otherwise standard fallback.
func (d *DelegatorService) GenerateWithCoT(ctx context.Context, promptText string) (string, error) {
	// Construct CoT prompt
	cotPromptText := fmt.Sprintf("Think step-by-step to answer the following question:\n%s\n\nReasoning steps:", promptText)

	// --- Use MOA if available ---
	if d.moa != nil {
		log.Println("DelegatorService (CoT): Using MOA for generation...")
		response, err := d.moa.Generate(ctx, cotPromptText)
		if err != nil {
			log.Printf("DelegatorService (CoT): MOA generation failed: %v", err)
			// Optionally, could fall back AGAIN to executeGenerationWithFallback here?
			// return "", fmt.Errorf("CoT generation failed via MOA: %w", err)
			log.Println("DelegatorService (CoT): MOA failed, falling back to standard generation...")
			// Fall through to standard execution if MOA fails
		} else {
			log.Println("DelegatorService (CoT): MOA generation successful.")
			// TODO: Optional parsing if needed for CoT
			return response, nil
		}
	}

	// --- Standard Fallback if MOA is nil or failed ---
	log.Println("DelegatorService (CoT): Using standard generation with fallback...")
	fullResponse, err := d.executeGenerationWithFallback(ctx, cotPromptText, "CoT")
	if err != nil {
		return "", err // Error already includes context from helper
	}
	// TODO: Optional parsing if needed for CoT
	return fullResponse, nil
}

// GenerateWithReflection uses MOA if available for each step, otherwise standard fallback.
func (d *DelegatorService) GenerateWithReflection(ctx context.Context, promptText string) (string, error) {
	log.Println("DelegatorService: GenerateWithReflection - Starting initial generation step")

	// --- Step 1: Initial Response Generation (Use MOA if available) ---
	var initialResponse string
	var err error
	if d.moa != nil {
		log.Println("DelegatorService (Reflection-Initial): Using MOA...")
		initialResponse, err = d.moa.Generate(ctx, promptText)
		if err != nil {
			log.Printf("DelegatorService (Reflection-Initial): MOA failed: %v. Falling back...", err)
			// Fall through to standard execution if MOA fails
		}
	}
	// If MOA not used or failed, use standard fallback
	if initialResponse == "" {
		log.Println("DelegatorService (Reflection-Initial): Using standard generation...")
		initialResponse, err = d.executeGenerationWithFallback(ctx, promptText, "Reflection-Initial")
	}
	// Handle final error from Step 1
	if err != nil {
		return "", fmt.Errorf("reflection initial generation failed: %w", err)
	}
	log.Println("DelegatorService: GenerateWithReflection - Initial generation successful")


	// --- Step 2: Reflection Prompt Construction ---
	reflectionPromptText := fmt.Sprintf("Original prompt: %s\n\nInitial response: %s\n\nPlease review the initial response for accuracy, completeness, and clarity. Provide a revised and improved response based on your review.", promptText, initialResponse)
	log.Println("DelegatorService: GenerateWithReflection - Starting reflection generation step")


	// --- Step 3: Reflection Response Generation (Use MOA if available) ---
	var finalResponse string
	if d.moa != nil {
		log.Println("DelegatorService (Reflection-Reflect): Using MOA...")
		finalResponse, err = d.moa.Generate(ctx, reflectionPromptText)
		if err != nil {
			log.Printf("DelegatorService (Reflection-Reflect): MOA failed: %v. Falling back...", err)
			// Fall through to standard execution if MOA fails
		}
	}
	// If MOA not used or failed, use standard fallback
	if finalResponse == "" {
		log.Println("DelegatorService (Reflection-Reflect): Using standard generation...")
		finalResponse, err = d.executeGenerationWithFallback(ctx, reflectionPromptText, "Reflection-Reflect")
	}
	// Handle final error from Step 3
	if err != nil {
		return "", fmt.Errorf("reflection refinement generation failed: %w", err)
	}
	log.Println("DelegatorService: GenerateWithReflection - Reflection generation successful")

	return finalResponse, nil
}

// GenerateStructuredOutput uses MOA if available, otherwise standard fallback.
func (d *DelegatorService) GenerateStructuredOutput(ctx context.Context, content string, schema string) (string, error) {
	log.Println("DelegatorService: GenerateStructuredOutput - Starting generation")

	// --- Step 1: Construct Structured Prompt ---
	structuredPromptText := fmt.Sprintf("Analyze the following content:\n\n---\n%s\n---\n\nPlease extract the relevant information and respond ONLY with a valid JSON object strictly adhering to the following JSON schema:\n```json\n%s\n```", content, schema)

	// --- Step 2: Generate Structured Response (Use MOA if available) ---
	var response string
	var err error
	if d.moa != nil {
		log.Println("DelegatorService (StructuredOutput): Using MOA...")
		response, err = d.moa.Generate(ctx, structuredPromptText)
		if err != nil {
			log.Printf("DelegatorService (StructuredOutput): MOA failed: %v. Falling back...", err)
			// Fall through to standard execution if MOA fails
		}
	}
	// If MOA not used or failed, use standard fallback
	if response == "" {
		log.Println("DelegatorService (StructuredOutput): Using standard generation...")
		response, err = d.executeGenerationWithFallback(ctx, structuredPromptText, "StructuredOutput")
	}
	// Handle final error
	if err != nil {
		return "", fmt.Errorf("structured output generation failed: %w", err)
	}

	log.Println("DelegatorService: GenerateStructuredOutput - Generation successful (validation may still be needed)")
	// TODO: Add JSON validation logic here if needed

	return response, nil
}

// Add method to update MOA instance if needed by SetProxy/BaseModel in InferenceService
func (d *DelegatorService) UpdateMOA(moaInstance *gollm.MOA) {
    // This method might not be strictly necessary if NewDelegatorService is always called
    // after model changes, but provides an alternative update path.
    if moaInstance == nil {
        log.Println("[WARN] DelegatorService.UpdateMOA: Received nil MOA instance.")
    }
    d.moa = moaInstance
    log.Println("DelegatorService: Internal MOA instance updated.")
}

// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/optimizing_proxy.go
package inference

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect" // Needed for type comparison
	

	"github.com/teilomillet/gollm/llm" // Import the base llm package
)

// OptimizingProxy implements advanced LLM interaction techniques
// and delegates requests between a proxy LLM and a base LLM.
type OptimizingProxy struct {
	proxyLLM llm.LLM // The LLM instance for proxy operations (e.g., Cerebras)
	baseLLM  llm.LLM // The LLM instance for base operations (e.g., Gemini)
	// Add fields for context management if needed
	// e.g., conversationHistory []llm.PromptMessage

	// Define the token limit for the proxy LLM
	// TODO: Get this dynamically if possible from the llm.LLM interface or provider
	proxyTokenLimit int
}

// NewOptimizingProxy creates a new proxy instance.
// It requires both the proxy LLM and the base LLM.
func NewOptimizingProxy(proxyLLM llm.LLM, baseLLM llm.LLM) *OptimizingProxy {
	if proxyLLM == nil || baseLLM == nil {
		log.Println("CRITICAL: NewOptimizingProxy called with nil proxyLLM or baseLLM")
		// Handle this critical error, maybe panic or return nil with error handling upstream
		return nil // Or panic("Proxy requires valid LLM instances")
	}
	return &OptimizingProxy{
		proxyLLM:        proxyLLM,
		baseLLM:         baseLLM,
		proxyTokenLimit: 8000, // Hardcoded Cerebras limit (approx), adjust as needed
		// Initialize context fields if added
	}
}

// estimateTokens provides a very basic token estimation.
// Replace with a proper tokenizer (like tiktoken) for accuracy.
func estimateTokens(text string) int {
	// Very rough estimate: 1 token ~ 3-4 chars in English
	// This is highly inaccurate and should be replaced.
	return len(text) / 3
}

// shouldRetryWithError determines if the given error warrants a retry attempt with the base LLM.
// Customize this logic based on the errors you observe from the proxy LLM (Cerebras).
func (p *OptimizingProxy) shouldRetryWithError(err error) bool {
	if err == nil {
		return false
	}
	// Example criteria (adjust based on actual errors from Cerebras):
	// - Retry on specific HTTP status codes (e.g., 5xx server errors)
	// - Retry on timeout errors
	// - Retry on specific error messages indicating temporary issues or capability limits
	// - Avoid retrying on authentication errors (401/403), bad requests (400), rate limits (429) initially.

	errStr := err.Error()
	log.Printf("OptimizingProxy: Evaluating error for retry: %s", errStr)

	// Simple example: Retry on any error for now (can be refined)
	// In production, you'd want to be more specific.
	// e.g., if strings.Contains(errStr, "timeout") { return true }
	// e.g., if strings.Contains(errStr, "server error") { return true } // Depending on actual error messages
	// e.g., if strings.Contains(errStr, "upstream request timeout") { return true } // Example specific error
	log.Println("OptimizingProxy: Decision: Retrying (defaulting to retry on any error for now)")
	return true // TODO: Refine this logic based on observed errors from Cerebras
}

// GenerateSimple performs basic generation, deciding whether to use proxy or base LLM,
// and includes fallback logic from proxy to base on specific errors.
func (p *OptimizingProxy) GenerateSimple(ctx context.Context, promptText string) (string, error) {
	if p.proxyLLM == nil || p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy is not properly configured with LLM instances")
	}

	estimatedTokens := estimateTokens(promptText) // Use a better tokenizer here
	log.Printf("OptimizingProxy: Estimated tokens for prompt: %d (Proxy limit: %d)", estimatedTokens, p.proxyTokenLimit)

	var initialTargetLLM llm.LLM
	var initialTargetName string

	// --- Initial Delegation Logic ---
	if estimatedTokens > p.proxyTokenLimit {
		log.Println("OptimizingProxy: Prompt exceeds proxy limit. Delegating directly to Base LLM (Gemini)...")
		initialTargetLLM = p.baseLLM
		initialTargetName = "Base (Gemini)"
	} else {
		log.Println("OptimizingProxy: Prompt within limit. Attempting with Proxy LLM (Cerebras)...")
		initialTargetLLM = p.proxyLLM
		initialTargetName = "Proxy (Cerebras)"
	}
	// --- End Initial Delegation Logic ---

	prompt := llm.NewPrompt(promptText)

	// --- Attempt 1: Use the initially chosen LLM ---
	response, err := initialTargetLLM.Generate(ctx, prompt)

	// --- Retry Logic ---
	// Check if:
	// 1. There was an error.
	// 2. The error type suggests a retry might help.
	// 3. The LLM that failed was the proxyLLM (not the baseLLM).
	if err != nil && p.shouldRetryWithError(err) && reflect.TypeOf(initialTargetLLM) == reflect.TypeOf(p.proxyLLM) {
		log.Printf("OptimizingProxy: Initial generation with %s failed: %v. Attempting fallback to Base LLM (Gemini)...", initialTargetName, err)

		// --- Attempt 2: Use the Base LLM ---
		baseTargetName := "Base (Gemini)" // For logging
		retryResponse, retryErr := p.baseLLM.Generate(ctx, prompt) // Use p.baseLLM directly

		if retryErr != nil {
			log.Printf("OptimizingProxy: Fallback generation with %s also failed: %v", baseTargetName, retryErr)
			// Return an error indicating both attempts failed, including original error context
			return "", fmt.Errorf("initial generation failed (%s: %w), fallback failed (%s: %v)",
				initialTargetName, err, baseTargetName, retryErr)
		}

		// Fallback succeeded
		log.Printf("OptimizingProxy: Fallback generation with %s successful.", baseTargetName)
		return retryResponse, nil // Return the successful fallback response
	}
	// --- End Retry Logic ---

	// If there was an error but no retry was attempted (e.g., base failed, or error type didn't warrant retry)
	if err != nil {
		log.Printf("OptimizingProxy: Generation failed using %s: %v. No retry attempted or applicable.", initialTargetName, err)
		// Return the original error, adding context about which LLM failed
		return "", fmt.Errorf("generation failed using %s: %w", initialTargetName, err)
	}

	// Initial attempt was successful
	log.Printf("OptimizingProxy: Generation successful using %s.", initialTargetName)
	return response, nil
}

// GenerateWithCoT, GenerateWithReflection, GenerateStructuredOutput
// should also implement similar delegation AND RETRY logic.

// Example for GenerateWithCoT (adapt others similarly)
func (p *OptimizingProxy) GenerateWithCoT(ctx context.Context, promptText string) (string, error) {
	if p.proxyLLM == nil || p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy is not properly configured")
	}

	// Construct CoT prompt
	cotPromptText := fmt.Sprintf("Think step-by-step to answer the following question:\n%s\n\nReasoning steps:", promptText)
	estimatedTokens := estimateTokens(cotPromptText) // Estimate based on the modified prompt
	log.Printf("OptimizingProxy (CoT): Estimated tokens: %d (Proxy limit: %d)", estimatedTokens, p.proxyTokenLimit)

	var initialTargetLLM llm.LLM
	var initialTargetName string

	if estimatedTokens > p.proxyTokenLimit {
		log.Println("OptimizingProxy (CoT): Delegating directly to Base LLM (Gemini)...")
		initialTargetLLM = p.baseLLM
		initialTargetName = "Base (Gemini)"
	} else {
		log.Println("OptimizingProxy (CoT): Attempting with Proxy LLM (Cerebras)...")
		initialTargetLLM = p.proxyLLM
		initialTargetName = "Proxy (Cerebras)"
	}

	prompt := llm.NewPrompt(cotPromptText)

	// --- Attempt 1 ---
	fullResponse, err := initialTargetLLM.Generate(ctx, prompt)

	// --- Retry Logic ---
	if err != nil && p.shouldRetryWithError(err) && reflect.TypeOf(initialTargetLLM) == reflect.TypeOf(p.proxyLLM) {
		log.Printf("OptimizingProxy (CoT): Initial generation with %s failed: %v. Attempting fallback to Base LLM (Gemini)...", initialTargetName, err)
		baseTargetName := "Base (Gemini)"
		retryResponse, retryErr := p.baseLLM.Generate(ctx, prompt)
		if retryErr != nil {
			log.Printf("OptimizingProxy (CoT): Fallback generation with %s also failed: %v", baseTargetName, retryErr)
			return "", fmt.Errorf("CoT initial generation failed (%s: %w), fallback failed (%s: %v)",
				initialTargetName, err, baseTargetName, retryErr)
		}
		log.Printf("OptimizingProxy (CoT): Fallback generation with %s successful.", baseTargetName)
		// TODO: Optional parsing if needed for CoT
		return retryResponse, nil
	}
	// --- End Retry Logic ---

	if err != nil {
		log.Printf("OptimizingProxy (CoT): Generation failed using %s: %v. No retry attempted or applicable.", initialTargetName, err)
		return "", fmt.Errorf("CoT generation failed using %s: %w", initialTargetName, err)
	}

	log.Printf("OptimizingProxy (CoT): Generation complete using %s.", initialTargetName)
	// TODO: Optional parsing if needed for CoT
	return fullResponse, nil
}

// Implement similar delegation AND RETRY logic for GenerateWithReflection and GenerateStructuredOutput...
func (p *OptimizingProxy) GenerateWithReflection(ctx context.Context, promptText string) (string, error) {
    // TODO: Implement full delegation and retry logic similar to GenerateSimple/GenerateWithCoT
    // This involves potentially two LLM calls (initial + reflection), each needing delegation and retry.
    log.Println("OptimizingProxy: GenerateWithReflection - TODO: Implement full delegation and retry logic")
    // Placeholder: Just use proxy for now, without retry
    if p.proxyLLM == nil { return "", errors.New("proxy not configured")}
    initialPrompt := llm.NewPrompt(promptText)
	initialResponse, err := p.proxyLLM.Generate(ctx, initialPrompt)
	if err != nil { return "", fmt.Errorf("reflection initial generation failed: %w", err) }
    reflectionPromptText := fmt.Sprintf("Original prompt: %s\n\nInitial response: %s\n\nPlease review...", promptText, initialResponse)
    reflectionPrompt := llm.NewPrompt(reflectionPromptText)
    finalResponse, err := p.proxyLLM.Generate(ctx, reflectionPrompt)
    if err != nil { return "", fmt.Errorf("reflection second generation failed: %w", err) }
    return finalResponse, nil
}

func (p *OptimizingProxy) GenerateStructuredOutput(ctx context.Context, content string, schema string) (string, error) {
    // TODO: Implement full delegation and retry logic similar to GenerateSimple/GenerateWithCoT
    log.Println("OptimizingProxy: GenerateStructuredOutput - TODO: Implement full delegation and retry logic")
    // Placeholder: Just use proxy for now, without retry
    if p.proxyLLM == nil { return "", errors.New("proxy not configured")}
    structuredContent := fmt.Sprintf("Content: %s\n\nPlease respond strictly using this JSON schema:\n```json\n%s\n```", content, schema)
	prompt := llm.NewPrompt(structuredContent)
    response, err := p.proxyLLM.Generate(ctx, prompt)
    if err != nil { return "", fmt.Errorf("structured output generation failed: %w", err) }
    return response, nil
}

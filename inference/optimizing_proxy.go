// /home/gperry/Documents/GitHub/cloud-equities/FIG_Inference/inference/optimizing_proxy.go
package inference

import (
	"context"
	"fmt"
	"log"

	"github.com/teilomillet/gollm/llm" // Import the base llm package
)

// OptimizingProxy implements advanced LLM interaction techniques
// on top of a base llm.LLM implementation.
type OptimizingProxy struct {
	baseLLM llm.LLM // The underlying LLM client (configured by InferenceService)
	// Add fields for context management if needed for "Model Context Protocol"
	// e.g., conversationHistory []llm.PromptMessage
}

// NewOptimizingProxy creates a new proxy instance.
func NewOptimizingProxy(baseLLM llm.LLM) *OptimizingProxy {
	if baseLLM == nil {
		// Handle nil case appropriately, maybe return error or panic
		log.Println("Warning: NewOptimizingProxy called with nil baseLLM")
	}
	return &OptimizingProxy{
		baseLLM: baseLLM,
		// Initialize context fields if added
	}
}

// GenerateSimple performs a basic generation request using the underlying LLM.
func (p *OptimizingProxy) GenerateSimple(ctx context.Context, promptText string) (string, error) {
	if p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy has no base LLM configured")
	}
	log.Println("OptimizingProxy: Performing simple generation...")

	prompt := llm.NewPrompt(promptText) // Use library's prompt struct

	// Delegate directly to the base LLM
	response, err := p.baseLLM.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("base LLM generation failed: %w", err)
	}
	return response, nil
}

// GenerateWithCoT performs generation using a Chain-of-Thought prompt structure.
func (p *OptimizingProxy) GenerateWithCoT(ctx context.Context, promptText string) (string, error) {
	if p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy has no base LLM configured")
	}
	log.Println("OptimizingProxy: Performing Chain-of-Thought generation...")

	// 1. Construct a CoT prompt (instruct the model to think step-by-step)
	cotPromptText := fmt.Sprintf("Think step-by-step to answer the following question:\n%s\n\nReasoning steps:", promptText)
	prompt := llm.NewPrompt(cotPromptText)

	// 2. Generate the reasoning and final answer
	// Note: This might require multiple calls or specific parsing depending on the model
	fullResponse, err := p.baseLLM.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("CoT generation failed: %w", err)
	}

	// 3. TODO: Optionally parse the fullResponse to extract only the final answer,
	//    or return the full reasoning + answer. For now, return full response.
	log.Println("OptimizingProxy: CoT generation complete.")
	return fullResponse, nil
}

// GenerateWithReflection performs generation, then asks the model to reflect and improve.
func (p *OptimizingProxy) GenerateWithReflection(ctx context.Context, promptText string) (string, error) {
	if p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy has no base LLM configured")
	}
	log.Println("OptimizingProxy: Performing generation with self-reflection...")

	// 1. Initial Generation
	initialPrompt := llm.NewPrompt(promptText)
	initialResponse, err := p.baseLLM.Generate(ctx, initialPrompt)
	if err != nil {
		return "", fmt.Errorf("initial generation for reflection failed: %w", err)
	}

	// 2. Reflection Prompt
	reflectionPromptText := fmt.Sprintf(
		"Original prompt: %s\n\nInitial response: %s\n\nPlease review the initial response. Identify any flaws, inaccuracies, or areas for improvement. Then, provide a revised, improved response.",
		promptText,
		initialResponse,
	)
	reflectionPrompt := llm.NewPrompt(reflectionPromptText)

	// 3. Generate Reflected Response
	reflectedResponse, err := p.baseLLM.Generate(ctx, reflectionPrompt)
	if err != nil {
		return "", fmt.Errorf("reflection generation failed: %w", err)
	}

	// 4. TODO: Optionally parse the reflectedResponse to extract only the final improved answer.
	log.Println("OptimizingProxy: Reflection generation complete.")
	return reflectedResponse, nil
}

// GenerateStructuredOutput requests structured output (delegated to base LLM/provider)
// This proxy doesn't modify the structured output process itself, just passes it through.
func (p *OptimizingProxy) GenerateStructuredOutput(ctx context.Context, content string, schema string) (string, error) {
	if p.baseLLM == nil {
		return "", fmt.Errorf("optimizing proxy has no base LLM configured")
	}
	log.Println("OptimizingProxy: Performing structured output generation...")

	// Create prompt with schema instructions (as done previously)
	structuredContent := fmt.Sprintf("Content: %s\n\nPlease respond strictly using this JSON schema:\n```json\n%s\n```", content, schema)
	prompt := llm.NewPrompt(structuredContent)
	// Alternatively, use llm.WithOutput if gollm/provider supports it better

	response, err := p.baseLLM.Generate(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("structured output generation failed: %w", err)
	}
	return response, nil
}

// TODO: Implement other methods like Self-Improvement, Self-Consistency
// TODO: Implement Model Context Protocol (e.g., managing conversation history)


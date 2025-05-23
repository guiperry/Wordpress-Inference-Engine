package inference

import (
	"context"
	"github.com/teilomillet/gollm/llm"
)

// LLMAdapter wraps an llm.LLM instance to implement the TextGenerator interface
type LLMAdapter struct {
	LLM llm.LLM
	ProviderName string // ADDED: Store the provider name
}

// GenerateText implements the TextGenerator interface
func (a *LLMAdapter) GenerateText(prompt string) (string, error) {
	// Convert string prompt to llm.Prompt using the package's NewPrompt function
	p := llm.NewPrompt(prompt)
	return a.LLM.Generate(context.Background(), p)
}

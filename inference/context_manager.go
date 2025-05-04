// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/inference/context_manager.go
package inference

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/teilomillet/gollm/llm"
)

// ChunkingStrategy defines how to split the text.
type ChunkingStrategy int

const (
	// ChunkByParagraph splits text based on double newlines.
	ChunkByParagraph ChunkingStrategy = iota
	// ChunkBySentence splits text based on sentence boundaries.
	ChunkBySentence
	// ChunkByTokenCount splits text based on estimated token count.
	ChunkByTokenCount
)

// ProcessingMode defines how chunks should be processed.
type ProcessingMode int

const (
	// ParallelProcessing processes chunks in parallel (faster but no context sharing).
	ParallelProcessing ProcessingMode = iota
	// SequentialProcessing processes chunks in sequence, passing context between them.
	SequentialProcessing
)

// ContextManager handles chunking and processing of large text inputs.
type ContextManager struct {
	// inferenceService TextGenerator // REMOVED: LLM will be passed to ProcessLargePrompt
	strategy         ChunkingStrategy // How to split the text
	processingMode   ProcessingMode   // How to process chunks
	maxChunkSize     int              // Maximum tokens per chunk (for ChunkByTokenCount)
	chunkOverlap     int              // Number of tokens to overlap between chunks
	modelName        string           // Model name for token estimation
}

// ContextManagerOption defines a functional option for configuring ContextManager.
type ContextManagerOption func(*ContextManager)

// WithProcessingMode sets the processing mode.
func WithProcessingMode(mode ProcessingMode) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.processingMode = mode
	}
}

// WithMaxChunkSize sets the maximum chunk size in tokens.
func WithMaxChunkSize(size int) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.maxChunkSize = size
	}
}

// WithChunkOverlap sets the overlap between chunks in tokens.
func WithChunkOverlap(overlap int) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.chunkOverlap = overlap
	}
}

// WithModelName sets the model name for token estimation.
func WithModelName(modelName string) ContextManagerOption {
	return func(cm *ContextManager) {
		cm.modelName = modelName
	}
}

// TextGenerator defines the minimal interface needed for generating text
// This allows passing different LLM instances (like those from gollm).
type TextGenerator interface {
	GenerateText(prompt string) (string, error)
}

// LLMAdapter adapts gollm's llm.LLM to the TextGenerator interface
type LLMAdapter struct {
	LLM llm.LLM
}

func (a *LLMAdapter) GenerateText(prompt string) (string, error) {
	ctx := context.Background()
	p := &llm.Prompt{} // Initialize empty prompt
	// Try to set prompt text using reflection if needed
	if p, ok := interface{}(p).(interface{ SetText(string) }); ok {
		p.SetText(prompt)
	}
	return a.LLM.Generate(ctx, p)
}

// NewContextManager creates a new ContextManager with the given options.
// The TextGenerator (LLM) is now passed during processing, not creation.
func NewContextManager(strategy ChunkingStrategy, opts ...ContextManagerOption) *ContextManager {
	// Create with default values
	cm := &ContextManager{
		// inferenceService: service, // REMOVED
		strategy:         strategy,
		processingMode:   ParallelProcessing, // Default to parallel
		maxChunkSize:     1000,               // Default max chunk size
		chunkOverlap:     100,                // Default overlap
		modelName:        "gpt-4",            // Default model for token estimation
	}

	// Apply options
	for _, opt := range opts {
		opt(cm)
	}

	return cm
}

// splitIntoChunks splits text based on the configured strategy.
func (cm *ContextManager) splitIntoChunks(text string) []string {
	switch cm.strategy {
	case ChunkByParagraph:
		// Simple split by double newline
		chunks := strings.Split(text, "\n\n")
		var nonEmptyChunks []string
		for _, chunk := range chunks {
			trimmed := strings.TrimSpace(chunk)
			if trimmed != "" {
				nonEmptyChunks = append(nonEmptyChunks, trimmed)
			}
		}
		return nonEmptyChunks

	case ChunkBySentence:
		// Split by sentence boundaries using a simple regex
		// This is a basic implementation - a more sophisticated NLP approach could be used
		sentenceRegex := regexp.MustCompile(`[.!?]\s+`)
		sentences := sentenceRegex.Split(text, -1)

		var nonEmptySentences []string
		for _, sentence := range sentences {
			trimmed := strings.TrimSpace(sentence)
			if trimmed != "" {
				// Add back punctuation for context, unless it's the last sentence part
				if len(trimmed) > 0 && len(text) > len(trimmed) {
					originalIndex := strings.Index(text, trimmed)
					if originalIndex != -1 && originalIndex+len(trimmed) < len(text) {
						punctuation := text[originalIndex+len(trimmed)]
						if punctuation == '.' || punctuation == '!' || punctuation == '?' {
							trimmed += string(punctuation)
						}
					}
				}
				nonEmptySentences = append(nonEmptySentences, trimmed)
			}
		}

		// Group sentences into chunks to avoid too many small chunks
		return cm.groupSentencesIntoChunks(nonEmptySentences)

	case ChunkByTokenCount:
		// Split based on estimated token count
		return cm.splitByTokenCount(text)

	default:
		log.Printf("[WARN] Unknown chunking strategy: %d. Falling back to paragraph.", cm.strategy)
		// Set to ChunkByParagraph and retry
		cm.strategy = ChunkByParagraph
		return cm.splitIntoChunks(text) // Recursive call with default strategy
	}
}

// groupSentencesIntoChunks groups sentences into larger chunks to avoid too many small chunks.
func (cm *ContextManager) groupSentencesIntoChunks(sentences []string) []string {
	if len(sentences) == 0 {
		return []string{}
	}

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, sentence := range sentences {
		sentenceTokens := estimateTokens(sentence, cm.modelName)

		// If adding this sentence would exceed the max chunk size, start a new chunk
		if currentTokens > 0 && currentTokens+sentenceTokens > cm.maxChunkSize {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentTokens = 0
		}

		// Add the sentence to the current chunk
		if currentTokens > 0 {
			currentChunk.WriteString(" ") // Add space between sentences
		}
		currentChunk.WriteString(sentence)
		currentTokens += sentenceTokens
	}

	// Add the last chunk if it's not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// splitByTokenCount splits text into chunks based on token count.
func (cm *ContextManager) splitByTokenCount(text string) []string {
	// First split by paragraphs to preserve natural boundaries
	paragraphs := strings.Split(text, "\n\n")

	var chunks []string
	var currentChunk strings.Builder
	currentTokens := 0

	for _, paragraph := range paragraphs {
		trimmed := strings.TrimSpace(paragraph)
		if trimmed == "" {
			continue
		}

		paragraphTokens := estimateTokens(trimmed, cm.modelName)

		// If this paragraph alone exceeds the max chunk size, split it further
		if paragraphTokens > cm.maxChunkSize {
			// Add the current chunk if it's not empty
			if currentChunk.Len() > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentTokens = 0
			}

			// Split the large paragraph by sentences
			sentences := regexp.MustCompile(`[.!?]\s+`).Split(trimmed, -1)
			var currentSentenceChunk strings.Builder
			currentSentenceTokens := 0

			for _, sentence := range sentences {
				sentenceTrimmed := strings.TrimSpace(sentence)
				if sentenceTrimmed == "" {
					continue
				}
				// Add back punctuation
				if len(sentenceTrimmed) > 0 && len(trimmed) > len(sentenceTrimmed) {
					originalIndex := strings.Index(trimmed, sentenceTrimmed)
					if originalIndex != -1 && originalIndex+len(sentenceTrimmed) < len(trimmed) {
						punctuation := trimmed[originalIndex+len(sentenceTrimmed)]
						if punctuation == '.' || punctuation == '!' || punctuation == '?' {
							sentenceTrimmed += string(punctuation)
						}
					}
				}


				sentenceTokens := estimateTokens(sentenceTrimmed, cm.modelName)

				// If adding this sentence would exceed the max chunk size, start a new chunk
				if currentSentenceTokens > 0 && currentSentenceTokens+sentenceTokens > cm.maxChunkSize {
					chunks = append(chunks, currentSentenceChunk.String())
					currentSentenceChunk.Reset()
					currentSentenceTokens = 0
				}

				// Add the sentence to the current chunk
				if currentSentenceTokens > 0 {
					currentSentenceChunk.WriteString(" ")
				}
				currentSentenceChunk.WriteString(sentenceTrimmed)
				currentSentenceTokens += sentenceTokens
			}

			// Add the last sentence chunk if it's not empty
			if currentSentenceChunk.Len() > 0 {
				chunks = append(chunks, currentSentenceChunk.String())
			}
		} else if currentTokens+paragraphTokens > cm.maxChunkSize {
			// If adding this paragraph would exceed the max chunk size, start a new chunk
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
			currentChunk.WriteString(trimmed)
			currentTokens = paragraphTokens
		} else {
			// Add the paragraph to the current chunk
			if currentTokens > 0 {
				currentChunk.WriteString("\n\n") // Preserve paragraph break
			}
			currentChunk.WriteString(trimmed)
			currentTokens += paragraphTokens
		}
	}

	// Add the last chunk if it's not empty
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	// TODO: Implement overlap logic if needed. This would involve adding the end
	// of the previous chunk to the start of the next chunk during processing,
	// or adjusting the splitting logic to create overlapping chunks directly.

	return chunks
}

// ProcessLargePrompt chunks the input, processes each chunk via the provided LLM,
// and reassembles the results.
// Accepts the TextGenerator (LLM instance) to use for processing.
func (cm *ContextManager) ProcessLargePrompt(ctx context.Context, llm TextGenerator, largePrompt string, instructionPerChunk string) (string, error) {
	if llm == nil {
		return "", fmt.Errorf("context manager cannot process: TextGenerator (LLM) is nil")
	}

	chunks := cm.splitIntoChunks(largePrompt)
	if len(chunks) == 0 {
		return "", fmt.Errorf("prompt resulted in zero chunks")
	}

	log.Printf("ContextManager: Processing %d chunks using %s mode...",
		len(chunks),
		func() string {
			if cm.processingMode == ParallelProcessing {
				return "parallel"
			}
			return "sequential"
		}())

	// Choose processing method based on mode
	if cm.processingMode == SequentialProcessing {
		return cm.processSequentially(ctx, llm, chunks, instructionPerChunk)
	}

	// Default to parallel processing
	return cm.processInParallel(ctx, llm, chunks, instructionPerChunk)
}

// processInParallel processes chunks in parallel for speed.
// Accepts the TextGenerator (LLM instance).
func (cm *ContextManager) processInParallel(ctx context.Context, llm TextGenerator, chunks []string, instructionPerChunk string) (string, error) {
	var wg sync.WaitGroup
	var lastError error
	var errMutex sync.Mutex                     // To safely write to lastError from goroutines
	resultsArray := make([]string, len(chunks)) // Store results in order

	for i, chunk := range chunks {
		wg.Add(1)
		go func(index int, chunkText string) {
			defer wg.Done()
			log.Printf("ContextManager: Processing chunk %d/%d in parallel...", index+1, len(chunks))

			// Construct prompt for this chunk
			chunkPrompt := fmt.Sprintf("%s\n\n---\n%s\n---", instructionPerChunk, chunkText)

			result, err := llm.GenerateText(chunkPrompt) // Use the passed LLM
			if err != nil {
				errMutex.Lock()
				lastError = fmt.Errorf("error processing chunk %d: %w", index+1, err)
				errMutex.Unlock()
				log.Printf("ContextManager: Error on chunk %d: %v", index+1, err)
				resultsArray[index] = fmt.Sprintf("[ERROR PROCESSING CHUNK %d]", index+1) // Placeholder
				return
			}
			resultsArray[index] = result
			log.Printf("ContextManager: Chunk %d processed.", index+1)
		}(i, chunk)
	}

	wg.Wait() // Wait for all goroutines to finish

	// Reassemble results in order
	finalResult := strings.Join(resultsArray, "\n\n---\n\n") // Join with a separator

	log.Println("ContextManager: Finished processing all chunks in parallel.")
	return finalResult, lastError
}

// processSequentially processes chunks in sequence, passing context between them.
// Accepts the TextGenerator (LLM instance).
func (cm *ContextManager) processSequentially(ctx context.Context, llm TextGenerator, chunks []string, instructionPerChunk string) (string, error) {
	var results []string
	var previousOutputSummary string // Store summary of previous output

	for i, chunk := range chunks {
		log.Printf("ContextManager: Processing chunk %d/%d sequentially...", i+1, len(chunks))

		// Construct prompt for this chunk, including original instruction and summary context
		var chunkPrompt string
		if i == 0 {
			// First chunk - no previous context
			chunkPrompt = fmt.Sprintf("Overall Task: %s\n\nCurrent Section:\n---\n%s\n---",
				instructionPerChunk,
				chunk)
		} else {
			// Include original task and summary of previous output
			chunkPrompt = fmt.Sprintf("Overall Task: %s\n\nSummary of previous output:\n%s\n\nCurrent Section:\n---\n%s\n---",
				instructionPerChunk,
				previousOutputSummary,
				chunk)
		}

		// --- Add logging for the prompt being sent ---
		log.Printf("ContextManager: Sequential Prompt for Chunk %d:\n%s\n", i+1, chunkPrompt)
		// --- End logging ---

		result, err := llm.GenerateText(chunkPrompt) // Use the passed LLM
		if err != nil {
			// If an error occurs, return the results obtained so far and the error
			log.Printf("ContextManager: Error on chunk %d: %v", i+1, err)
			results = append(results, fmt.Sprintf("[ERROR PROCESSING CHUNK %d]", i+1))
			return strings.Join(results, "\n\n---\n\n"),
				fmt.Errorf("error processing chunk %d: %w", i+1, err)
		}

		results = append(results, result)

		// Update previous output summary for context in next chunk
		// Simple approach: take the last N sentences or tokens
		previousOutputSummary = cm.summarizeForContext(result)
		if previousOutputSummary != "" {
			log.Printf("ContextManager: Generated summary for next chunk context: %s", previousOutputSummary)
		}

		log.Printf("ContextManager: Chunk %d processed.", i+1)
	}

	// Reassemble results in order
	finalResult := strings.Join(results, "\n\n---\n\n")

	log.Println("ContextManager: Finished processing all chunks sequentially.")
	return finalResult, nil
}

// summarizeForContext creates a short summary of the text for context passing.
func (cm *ContextManager) summarizeForContext(text string) string {
	// Simple approach: Take the last few sentences.
	// A more robust approach might involve actual summarization or token counting.
	sentenceRegex := regexp.MustCompile(`[.!?]\s+`)
	sentences := sentenceRegex.Split(text, -1)
	numSentences := len(sentences)
	contextSentences := 3 // Number of sentences to keep for context

	if numSentences <= contextSentences {
		return text // Return the whole text if it's short
	}

	// Get the last 'contextSentences' sentences
	startIndex := numSentences - contextSentences
	// Join with ". " - this assumes sentences were split correctly and lost punctuation
	summary := strings.Join(sentences[startIndex:], ". ")
	// Try to add back the final punctuation if it was likely removed
	lastCharOriginal := text[len(text)-1]
	if lastCharOriginal == '.' || lastCharOriginal == '!' || lastCharOriginal == '?' {
		// Check if summary already ends with punctuation (unlikely with Split)
		if len(summary) > 0 {
			lastCharSummary := summary[len(summary)-1]
			if !(lastCharSummary == '.' || lastCharSummary == '!' || lastCharSummary == '?') {
				summary += string(lastCharOriginal)
			}
		}
	} else {
		// If original didn't end with punctuation, ensure summary doesn't either (unless it naturally does)
		if len(summary) > 0 {
			lastCharSummary := summary[len(summary)-1]
			if lastCharSummary == '.' || lastCharSummary == '!' || lastCharSummary == '?' {
				// It might have ended with punctuation naturally, leave it.
			} else {
				// Add a period if no punctuation exists.
				summary += "."
			}
		}
	}

	return summary
}


// ProcessLargePromptWithStrategy processes a large prompt with a specific chunking strategy,
// overriding the default strategy for this call only.
func (cm *ContextManager) ProcessLargePromptWithStrategy(
	ctx context.Context,
	largePrompt string,
	instructionPerChunk string,
	strategy ChunkingStrategy,
	llm TextGenerator, // Pass the LLM instance
) (string, error) {
	// Save the original strategy
	originalStrategy := cm.strategy
	cm.strategy = strategy
	defer func() { cm.strategy = originalStrategy }() // Restore strategy

	// Process with the temporary strategy and passed LLM
	result, err := cm.ProcessLargePrompt(ctx, llm, largePrompt, instructionPerChunk)

	return result, err
}

// ProcessLargePromptWithMode processes a large prompt with a specific processing mode,
// overriding the default mode for this call only.
func (cm *ContextManager) ProcessLargePromptWithMode(
	ctx context.Context,
	largePrompt string,
	instructionPerChunk string,
	mode ProcessingMode,
	llm TextGenerator, // Pass the LLM instance
) (string, error) {
	// Save the original mode
	originalMode := cm.processingMode
	cm.processingMode = mode
	defer func() { cm.processingMode = originalMode }() // Restore mode

	// Process with the temporary mode and passed LLM
	result, err := cm.ProcessLargePrompt(ctx, llm, largePrompt, instructionPerChunk)

	return result, err
}

// GetChunkingStrategy returns the current chunking strategy.
func (cm *ContextManager) GetChunkingStrategy() ChunkingStrategy {
	return cm.strategy
}

// SetChunkingStrategy sets a new chunking strategy.
func (cm *ContextManager) SetChunkingStrategy(strategy ChunkingStrategy) {
	cm.strategy = strategy
	log.Printf("ContextManager: Chunking strategy set to %d", strategy)
}

// GetProcessingMode returns the current processing mode.
func (cm *ContextManager) GetProcessingMode() ProcessingMode {
	return cm.processingMode
}

// SetProcessingMode sets a new processing mode.
func (cm *ContextManager) SetProcessingMode(mode ProcessingMode) {
	cm.processingMode = mode
	log.Printf("ContextManager: Processing mode set to %d", mode)
}

// SetMaxChunkSize sets the maximum chunk size in tokens.
func (cm *ContextManager) SetMaxChunkSize(size int) {
	cm.maxChunkSize = size
	log.Printf("ContextManager: Max chunk size set to %d tokens", size)
}

// SetChunkOverlap sets the overlap between chunks in tokens.
func (cm *ContextManager) SetChunkOverlap(overlap int) {
	cm.chunkOverlap = overlap
	log.Printf("ContextManager: Chunk overlap set to %d tokens", overlap)
}

// Deprecated: LLM is now passed during processing.
// func (cm *ContextManager) GetInferenceService() TextGenerator {
// 	return cm.inferenceService
// }

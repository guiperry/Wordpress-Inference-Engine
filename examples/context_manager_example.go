// /home/gperry/Documents/GitHub/Inc-Line/Wordpress-Inference-Engine/examples/context_manager_example.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"Inference_Engine/inference"
)

// mockTextGenerator implements the TextGenerator interface for examples
type mockTextGenerator struct{}

func (m *mockTextGenerator) GenerateText(prompt string) (string, error) {
	// Simple mock that just echoes back the prompt with a prefix
	if len(prompt) > 1000 {
		return "", errors.New("mock error: prompt too long")
	}
	return "MOCK RESPONSE: " + prompt, nil
}

var mockLLM = &mockTextGenerator{}

func main() {
	// Set up logging
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting Context Manager Example...")

	// Create a ContextManager with default settings (parallel processing, paragraph chunking)
	contextMgr := inference.NewContextManager(inference.ChunkByParagraph)
	
	// Example 1: Basic usage with paragraph chunking
	exampleBasicUsage(contextMgr)

	// Example 2: Using sentence chunking
	exampleSentenceChunking(contextMgr)

	// Example 3: Using token count chunking with custom options
	exampleTokenChunking(contextMgr)

	// Example 4: Sequential processing with context passing
	exampleSequentialProcessing(contextMgr)

	log.Println("Context Manager Example completed.")
}

func exampleBasicUsage(contextMgr *inference.ContextManager) {
	log.Println("\n=== Example 1: Basic Usage with Paragraph Chunking ===")
	
	// Sample large input with multiple paragraphs
	largeInput := `This is the first paragraph of our test input.
	
	This is the second paragraph. It contains multiple sentences that will be processed together since we're using paragraph chunking.
	
	Here's a third paragraph with some additional content to process.`
	
	// Instruction to apply to each chunk
	instruction := "Summarize the following section in one sentence:"
	
	// Process the large input
	ctx := context.Background()
	combinedOutput, err := contextMgr.ProcessLargePrompt(ctx, mockLLM, largeInput, instruction)
	if err != nil {
		log.Printf("Error in basic example: %v", err)
		return
	}
	
	// Display the result
	fmt.Println("\nBasic Example Result:")
	fmt.Println(combinedOutput)
}

func exampleSentenceChunking(contextMgr *inference.ContextManager) {
	log.Println("\n=== Example 2: Using Sentence Chunking ===")
	
	// Temporarily change the chunking strategy
	contextMgr.SetChunkingStrategy(inference.ChunkBySentence)
	defer contextMgr.SetChunkingStrategy(inference.ChunkByParagraph) // Reset after example
	
	// Sample input with multiple sentences
	largeInput := `The first sentence is about AI. The second sentence discusses natural language processing. The third sentence explains how language models work. The fourth sentence talks about applications in text processing.`
	
	// Instruction to apply to each chunk
	instruction := "Expand on the following topic with additional details:"
	
	// Process the large input
	ctx := context.Background()
	combinedOutput, err := contextMgr.ProcessLargePrompt(ctx, mockLLM, largeInput, instruction)
	if err != nil {
		log.Printf("Error in sentence chunking example: %v", err)
		return
	}
	
	// Display the result
	fmt.Println("\nSentence Chunking Example Result:")
	fmt.Println(combinedOutput)
}

func exampleTokenChunking(contextMgr *inference.ContextManager) {
	log.Println("\n=== Example 3: Using Token Count Chunking with Custom Options ===")
	
	// Create a new context manager with token count chunking and custom options
	// Note: Need to get the service from the original contextMgr or pass it explicitly
	tokenContextMgr := inference.NewContextManager(
		inference.ChunkByTokenCount,
		inference.WithMaxChunkSize(500),
		inference.WithChunkOverlap(50),
		inference.WithModelName("gpt-4"),
	)
	
	// Sample large input
	largeInput := `This is a long document that will be split based on token count rather than paragraphs or sentences. 
	The token-based chunking strategy ensures that each chunk stays within a specified token limit, which is useful when working with models that have strict context length limitations.
	
	When using token-based chunking, the system will try to preserve natural paragraph boundaries when possible, but will split paragraphs if they exceed the maximum token limit.
	This approach provides more control over the size of each chunk, ensuring they don't exceed the model's context window.
	
	Additionally, we can configure overlap between chunks to maintain continuity. This means that a portion of the end of one chunk will be included at the beginning of the next chunk.
	This overlap helps the model maintain context when processing adjacent chunks, especially important for tasks like summarization or content generation where context is crucial.`
	
	// Instruction to apply to each chunk
	instruction := "Explain the following text in simpler terms:"
	
	// Process the large input
	ctx := context.Background()
	combinedOutput, err := tokenContextMgr.ProcessLargePrompt(ctx, mockLLM, largeInput, instruction)
	if err != nil {
		log.Printf("Error in token chunking example: %v", err)
		return
	}
	
	// Display the result
	fmt.Println("\nToken Chunking Example Result:")
	fmt.Println(combinedOutput)
}

func exampleSequentialProcessing(contextMgr *inference.ContextManager) {
	log.Println("\n=== Example 4: Sequential Processing with Context Passing ===")
	
	// Use the ProcessLargePromptWithMode method to override the default processing mode
	// This allows us to use sequential processing for this specific call
	
	// Sample input for a story continuation task
	largeInput := `Once upon a time, there was a small village nestled in a valley.
	
	The village was known for its beautiful gardens and friendly people.
	
	One day, a mysterious traveler arrived at the village gates.`
	
	// Instruction that requires context from previous chunks
	instruction := "Continue the story based on the following section and any previous output:"
	
	// Process the large input with sequential mode
	ctx := context.Background()
	combinedOutput, err := contextMgr.ProcessLargePromptWithMode(
		ctx, 
		largeInput, 
		instruction,
		inference.SequentialProcessing,
		mockLLM,
	)
	if err != nil {
		log.Printf("Error in sequential processing example: %v", err)
		return
	}
	
	// Display the result
	fmt.Println("\nSequential Processing Example Result:")
	fmt.Println(combinedOutput)
}
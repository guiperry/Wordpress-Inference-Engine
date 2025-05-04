# Context Manager

The `ContextManager` is a component of the Wordpress Inference Engine that handles chunking and processing of large text inputs. It provides a way to process text that exceeds the token limit of the underlying language models by splitting it into manageable chunks, processing each chunk, and then reassembling the results.

## Features

- Multiple chunking strategies:
  - Paragraph-based chunking (splits by double newlines)
  - Sentence-based chunking (splits by sentence boundaries)
  - Token-based chunking (splits based on estimated token count)
  
- Processing modes:
  - Parallel processing (faster, but no context sharing between chunks)
  - Sequential processing (slower, but passes context between chunks)
  
- Configurable options:
  - Maximum chunk size
  - Chunk overlap
  - Model name for token estimation

## Usage

### Initialization

Create a `ContextManager` instance with an `InferenceService` and a chunking strategy:

```go
// Basic initialization with default settings
contextMgr := inference.NewContextManager(inferenceService, inference.ChunkByParagraph)

// Advanced initialization with custom options
contextMgr := inference.NewContextManager(
    inferenceService,
    inference.ChunkByTokenCount,
    inference.WithMaxChunkSize(1000),
    inference.WithChunkOverlap(100),
    inference.WithProcessingMode(inference.SequentialProcessing),
    inference.WithModelName("gpt-4"),
)
```

### Processing Large Prompts

Process a large text input by chunking it and applying an instruction to each chunk:

```go
largeInput := "Very long text...\n\nAnother paragraph...\n\nMore text..."
instruction := "Summarize the following section:"
ctx := context.Background()

combinedOutput, err := contextMgr.ProcessLargePrompt(ctx, largeInput, instruction)
if err != nil {
    // Handle error (some chunks might have failed)
    log.Printf("Chunk processing finished with errors: %v", err)
}

// Use combinedOutput
fmt.Println(combinedOutput)
```

### Temporary Strategy or Mode Override

Override the default chunking strategy or processing mode for a specific call:

```go
// Use a different chunking strategy for this specific call
result, err := contextMgr.ProcessLargePromptWithStrategy(
    ctx,
    largeInput,
    instruction,
    inference.ChunkBySentence,
)

// Use a different processing mode for this specific call
result, err := contextMgr.ProcessLargePromptWithMode(
    ctx,
    largeInput,
    instruction,
    inference.SequentialProcessing,
)
```

### Changing Settings

Change the chunking strategy, processing mode, or other settings:

```go
// Change chunking strategy
contextMgr.SetChunkingStrategy(inference.ChunkByTokenCount)

// Change processing mode
contextMgr.SetProcessingMode(inference.SequentialProcessing)

// Change max chunk size
contextMgr.SetMaxChunkSize(2000)

// Change chunk overlap
contextMgr.SetChunkOverlap(200)
```

## Important Notes

1. **Chunking Strategy Selection**:
   - `ChunkByParagraph`: Best for naturally structured text with paragraphs
   - `ChunkBySentence`: Good for text where sentence-level processing is important
   - `ChunkByTokenCount`: Best for controlling token usage and staying within model limits

2. **Processing Mode Selection**:
   - `ParallelProcessing`: Faster but each chunk is processed independently
   - `SequentialProcessing`: Slower but maintains context between chunks, better for tasks requiring continuity

3. **Error Handling**:
   - The current implementation returns the last error encountered during processing
   - Even if some chunks fail, the successful chunks' results will be included in the output

4. **Instructions**:
   - The `instructionPerChunk` parameter is crucial for telling the LLM what to do with each piece of text
   - For sequential processing, the instruction should reference previous outputs

5. **Integration**:
   - The `ContextManager` can be used anywhere in your application where you need to process large text inputs
   - It's particularly useful for summarization, analysis, or generation tasks on long documents

## Example

See the `examples/context_manager_example.go` file for complete usage examples.
package inference

// Prompts for WordPress Content Management
const (
	WordPressContentImprovePrompt = `Improve the following WordPress page content to make it more engaging, professional, and SEO-friendly:

%s

Please enhance the content while maintaining its core message and purpose. Consider:
1. Improving readability with better paragraph structure and transitions
2. Adding compelling headlines and subheadings
3. Incorporating relevant keywords naturally
4. Making the tone more engaging and professional
5. Ensuring proper grammar and punctuation

Return the improved content in HTML format suitable for WordPress.`

	WordPressContentRewritePrompt = `Rewrite the following WordPress page content with a fresh perspective while maintaining the same information and purpose:

%s

Please create an entirely new version that:
1. Presents the same information in a different way
2. Uses a more engaging and professional tone
3. Improves structure and flow
4. Incorporates SEO best practices
5. Maintains any important keywords or phrases

Return the rewritten content in HTML format suitable for WordPress.`

	WordPressContentExpandPrompt = `Expand the following WordPress page content with additional relevant information:

%s

Please enhance this content by:
1. Adding more depth and detail to existing points
2. Including additional relevant sections or examples
3. Incorporating statistics or data if appropriate
4. Ensuring a cohesive flow throughout
5. Maintaining the original tone and style

Return the expanded content in HTML format suitable for WordPress.`

	WordPressContentGenerateWithSourcesPrompt = `You are tasked with generating content based on the provided materials. You will receive two types of sources: "True Sources" and "Sample Sources".

**True Sources:** These contain the factual information, data, or core message that the generated content MUST be based on. Accuracy and adherence to the information in these sources are paramount.

**Sample Sources:** These provide examples of the desired writing style, tone, structure, or formatting. Use these as a guide for *how* to present the information derived from the True Sources, but do not treat their content as factual unless it overlaps with a True Source.

--- TRUE SOURCES ---
%s
--- END TRUE SOURCES ---

--- SAMPLE SOURCES ---
%s
--- END SAMPLE SOURCES ---

**Your Task:**

Based *only* on the information provided in the **True Sources**, generate new content that addresses the following specific request:

**Request:** %s

**Instructions:**
1.  Strictly adhere to the facts and information presented in the **True Sources**.
2.  Use the **Sample Sources** as a reference for style, tone, formatting (including HTML if appropriate for WordPress), and overall presentation.
3.  Synthesize information logically and ensure the final output is well-structured and engaging.
4.  If there are no Sample Sources, use a professional and clear writing style suitable for a website.
5.  If there are no True Sources, inform the user that factual content cannot be generated without them.
6.  Return only the generated content, ready for use, without any explanations, metadata, or introductory/concluding remarks about the process.
`
)

// WordPress Content Prompts
func GetWordPressContentImprovePrompt(content string) string {
	return formatPrompt(WordPressContentImprovePrompt, content)
}

func GetWordPressContentRewritePrompt(content string) string {
	return formatPrompt(WordPressContentRewritePrompt, content)
}

func GetWordPressContentExpandPrompt(content string) string {
	return formatPrompt(WordPressContentExpandPrompt, content)
}

// formatPrompt formats a prompt with the given arguments
func formatPrompt(format string, args ...interface{}) string {
	return sprintf(format, args...)
}

// sprintf is a simple implementation of fmt.Sprintf to avoid importing fmt
func sprintf(format string, args ...interface{}) string {
	result := format
	for _, arg := range args {
		// Replace the first occurrence of %s with the string representation of arg
		// This is a simplified version and doesn't handle all format specifiers
		if s, ok := arg.(string); ok {
			result = replaceFirst(result, "%s", s)
		}
	}
	return result
}

// replaceFirst replaces the first occurrence of old with new in s
func replaceFirst(s, old, new string) string {
	i := indexOf(s, old)
	if i < 0 {
		return s
	}
	return s[:i] + new + s[i+len(old):]
}

// indexOf returns the index of the first occurrence of substr in s, or -1 if substr is not present
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Function to format the new prompt
func GetWordPressContentGenerateWithSourcesPrompt(trueSourcesContent, sampleSourcesContent, userRequest string) string {
	// Handle cases where one type of source might be empty
	if trueSourcesContent == "" {
		trueSourcesContent = "(No True Sources Provided)"
	}
	if sampleSourcesContent == "" {
		sampleSourcesContent = "(No Sample Sources Provided)"
	}
	return formatPrompt(WordPressContentGenerateWithSourcesPrompt, trueSourcesContent, sampleSourcesContent, userRequest)
}

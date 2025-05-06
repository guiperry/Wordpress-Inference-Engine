# Wordpress Inference Engine

## Overview

The Wordpress Inference Engine is a desktop application built with Go and the Fyne toolkit. It allows users to connect to their WordPress sites, manage page content, and leverage AI language models (including Cerebras, Gemini, and DeepSeek) to generate or enhance website content based on existing pages or local files.

## Features

*   **WordPress Connectivity:**
    *   Connect securely to WordPress sites using Application Passwords.
    *   Save, load, and delete connection details for multiple sites.
    *   View connection status across different application tabs.
*   **Content Management (Manager Tab):**
    *   List pages from the connected WordPress site.
    *   Preview pages with screenshot functionality.
    *   Send page content to the Content Generator as source material.
*   **AI Content Generation (Generator Tab):**
    *   Add source content from:
        *   WordPress pages (loaded via the Manager tab).
        *   Local text files.
    *   Provide a specific prompt to guide the AI.
    *   Generate new content using the selected AI provider, synthesizing information from the provided sources and prompt.
    *   View and edit the generated content.
    *   Save generated content to a local file.
    *   Save generated content directly back to a selected WordPress page (overwriting existing content).
*   **Inference Engine Configuration (Settings Tab):**
    *   Configure WordPress connection settings.
    *   Configure AI provider settings.
    *   Supports multiple AI providers (Cerebras, Gemini, DeepSeek).
*   **Inference Chat (Inference Chat Tab):**
    *   Interactive chat interface with the configured AI model.
    *   Maintain conversation history.
*   **Direct AI Testing (Test Inference Tab):**
    *   Send prompts directly to the configured AI provider for quick testing and experimentation.
    *   View application logs in the console widget.
*   **Advanced Context Management:**
    *   Process large text inputs that exceed token limits by intelligently chunking content.
    *   Multiple chunking strategies (paragraph-based, sentence-based, token-based).
    *   Sequential or parallel processing modes.
*   **Customizable UI:**
    *   Features a high-contrast dark theme for usability.
    *   Responsive layout that adapts to different window sizes.

## Setup and Installation

### Prerequisites

*   **Go:** Ensure you have Go installed (version 1.23 or later). [https://go.dev/doc/install](https://go.dev/doc/install)
*   **WordPress Site:** A WordPress site where you can create Application Passwords.
*   **AI Provider Account:** An account with Cerebras, Google (for Gemini), or DeepSeek to obtain API keys.
*   **Google Chrome/Chromium:** Required for the page preview functionality.

### Running the Application

1.  **Clone the Repository:**
    ```bash
    git clone <your-repository-url>
    cd Wordpress-Inference-Engine
    ```
2.  **Configure API Keys (Optional but Recommended):**
    *   Create a file named `.env` in the project's root directory.
    *   Add your API keys to this file:
        ```dotenv
        CEREBRAS_API_KEY=your_cerebras_api_key_here
        GEMINI_API_KEY=your_gemini_api_key_here
        DEEPSEEK_API_KEY=your_deepseek_api_key_here
        ```
    *   The application will load these keys on startup. Alternatively, you can set these as system environment variables.
3.  **Build and Run:**
    ```bash
    go run .
    ```
    Or build an executable:
    ```bash
    go build -o wordpress-inference-engine .
    ./wordpress-inference-engine
    ```

## Usage

1.  **Settings Tab:**
    *   Go to the "Settings" tab first (selected by default on startup).
    *   Under "WordPress Connection", enter your site URL, WordPress username, and an Application Password. Check "Remember Me" and provide a "Site Name" if you want to save these details. Click "Connect".
    *   Under "Inference Settings", ensure the correct API keys are loaded from your `.env` file or environment variables.

2.  **Manager Tab:**
    *   Once connected, the status bar should show "Connected" and the window title will display the site name.
    *   The list of pages from your WordPress site will load automatically.
    *   Select a page to view its preview (requires Chrome/Chromium).
    *   Click "Refresh Preview" to update the preview if needed.
    *   Click "Load Content to Generator" to use the current page's content as source material in the Generator tab.

3.  **Generator Tab:**
    *   Add source content using "Add Source" (for local files) or by loading from the Manager tab.
    *   Enter a detailed prompt in the "Prompt" box.
    *   Click "Generate Content".
    *   Review the generated content in the "Generated Content" box.
    *   Use "Save to File" or "Save to WordPress" (select target page if multiple WP sources were used).

4.  **Inference Chat Tab:**
    *   Enter messages in the chat interface to interact with the AI model.
    *   View the conversation history in the chat display.

5.  **Test Inference Tab:**
    *   Enter a prompt and click "Test Inference" to get a direct response from the configured AI model.
    *   View application logs in the console widget at the bottom of this tab.

## Configuration Details

*   **API Keys:** Stored as environment variables (`CEREBRAS_API_KEY`, `GEMINI_API_KEY`, `DEEPSEEK_API_KEY`). Using a `.env` file is recommended.
*   **Saved Sites:** Connection details marked "Remember Me" are saved in JSON format at `~/.wordpress-inference/saved_sites.json`. Passwords are encrypted (currently using Base64 encoding - **consider stronger encryption for production use**).
*   **LLM Providers:** The application is configured to use Cerebras as the primary provider with Gemini and DeepSeek as fallbacks.
*   **Context Management:** Large content is automatically processed using the Context Manager, which splits content into manageable chunks based on token limits.

## WordPress Setup: Application Passwords

This application requires **Application Passwords** for connecting to your WordPress site. Standard user passwords will not work.

1.  Log in to your WordPress admin dashboard.
2.  Go to "Users" -> "Profile".
3.  Scroll down to the "Application Passwords" section.
4.  Enter a name for the application (e.g., "Inference Engine") and click "Add New Application Password".
5.  **Important:** Copy the generated password immediately. You will **not** be able to see it again.
6.  Use this generated password in the "Application Password" field in the application's settings.

## Dependencies

*   **Fyne:** Cross-platform GUI toolkit for Go (v2.5.5).
*   **chromedp:** Headless browser library for capturing page screenshots.
*   **godotenv:** For loading `.env` files.
*   **gollm:** Library for interacting with LLM providers (using a custom fork).
*   **tiktoken-go:** For token counting and estimation.

## Advanced Features

### Context Manager

The Context Manager handles large text inputs by:
* Splitting content into manageable chunks using different strategies (paragraph, sentence, or token-based)
* Processing each chunk with the AI model
* Reassembling the results into a coherent output

This allows processing of content that would otherwise exceed the token limits of the underlying models.

### LLM Provider Fallback

The application implements a sophisticated fallback mechanism:
* Attempts to use the primary provider (Cerebras) first
* If the primary provider fails or is unavailable, automatically falls back to alternative providers (Gemini, DeepSeek)
* Provides seamless experience even when specific providers have issues

## License

*(Placeholder: Specify the license, e.g., MIT, Apache 2.0)*

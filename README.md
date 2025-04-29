# Wordpress Inference Engine

## Overview

The Wordpress Inference Engine is a desktop application built with Go and the Fyne toolkit. It allows users to connect to their WordPress sites, manage page content, and leverage AI language models (like those from OpenAI or Cerebras) to generate or enhance website content based on existing pages or local files.

## Features

*   **WordPress Connectivity:**
    *   Connect securely to WordPress sites using Application Passwords.
    *   Save, load, and delete connection details for multiple sites.
    *   View connection status across different application tabs.
*   **Content Management (Manager Tab):**
    *   List pages from the connected WordPress site.
    *   Load page content into a built-in editor.
    *   Save modified content directly back to the WordPress page.
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
    *   Select the desired AI provider (e.g., OpenAI, Cerebras).
    *   Configure API keys (primarily loaded via environment variables).
    *   Specify the AI model to be used for generation.
*   **Direct AI Testing (Test Inference Tab):**
    *   Send prompts directly to the configured AI provider for quick testing and experimentation.
*   **Customizable UI:**
    *   Features a high-contrast dark theme for usability.
    *   Responsive layout that adapts to different window sizes.

## Setup and Installation

### Prerequisites

*   **Go:** Ensure you have Go installed (version 1.18 or later recommended). [https://go.dev/doc/install](https://go.dev/doc/install)
*   **WordPress Site:** A WordPress site where you can create Application Passwords.
*   **AI Provider Account:** An account with OpenAI and/or Cerebras (or other supported providers) to obtain API keys.

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
        OPENAI_API_KEY=your_openai_api_key_here
        CEREBRAS_API_KEY=your_cerebras_api_key_here
        ```
    *   The application will load these keys on startup. Alternatively, you can set these as system environment variables. Keys set via the UI require an application restart.
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
    *   Go to the "Settings" tab first.
    *   Under "WordPress Connection", enter your site URL, WordPress username, and an Application Password. Check "Remember Me" and provide a "Site Name" if you want to save these details. Click "Connect".
    *   Under "Inference Settings", select your desired AI provider and ensure the correct API key is loaded (from `.env` or set via the UI - requires restart if set via UI). Enter the specific model name you wish to use and click "Set Model".
2.  **Manager Tab:**
    *   Once connected, the status bar should show "Connected".
    *   The list of pages from your WordPress site should load automatically (or you might need a refresh button if implemented).
    *   Select a page to load its content into the editor.
    *   Edit the content and click "Save Content" to update the page on WordPress.
    *   Click "Load to Generator" to use the current page's content as source material in the Generator tab.
3.  **Generator Tab:**
    *   Add source content using "Add Source" (for local files) or by loading from the Manager tab.
    *   Enter a detailed prompt in the "Prompt" box.
    *   Click "Generate Content".
    *   Review the generated content in the "Generated Content" box.
    *   Use "Save to File" or "Save to WordPress" (select target page if multiple WP sources were used).
4.  **Test Inference Tab:**
    *   Enter a prompt and click "Test Inference" to get a direct response from the configured AI model.

## Configuration Details

*   **API Keys:** Stored as environment variables (`OPENAI_API_KEY`, `CEREBRAS_API_KEY`). Using a `.env` file is recommended.
*   **Saved Sites:** Connection details marked "Remember Me" are saved in JSON format at `~/.wordpress-inference/saved_sites.json`. Passwords are encrypted (currently using Base64 for simplicity - **consider stronger encryption for production use**).

## WordPress Setup: Application Passwords

This application requires **Application Passwords** for connecting to your WordPress site. Standard user passwords will not work.

1.  Log in to your WordPress admin dashboard.
2.  Go to "Users" -> "Profile".
3.  Scroll down to the "Application Passwords" section.
4.  Enter a name for the application (e.g., "Inference Engine") and click "Add New Application Password".
5.  **Important:** Copy the generated password immediately. You will **not** be able to see it again.
6.  Use this generated password in the "Application Password" field in the application's settings.

## Dependencies

*   Fyne: Cross-platform GUI toolkit for Go.
*   go-wordpress: (Or similar library if used - *adjust if using direct HTTP calls*).
*   godotenv: For loading `.env` files.

*(Add other significant dependencies if applicable)*

## Contributing

*(Placeholder: Add guidelines if you plan for others to contribute)*

## License

*(Placeholder: Specify the license, e.g., MIT, Apache 2.0)*

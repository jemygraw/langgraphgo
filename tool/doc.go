// Package tool provides a collection of ready-to-use tools for LangGraph Go agents.
//
// This package includes various tools that extend agent capabilities, including
// web search, file operations, code execution, and integration with popular
// APIs and services. Tools are designed to be easily integrated with prebuilt
// agents or custom implementations.
//
// # Available Tools
//
// ## Web Search Tools
//
// ### Tavily Search
// Perform web searches using the Tavily API:
//
//	import "github.com/smallnest/langgraphgo/tool"
//
//	searchTool, err := tool.NewTavilySearchTool("your-tavily-api-key")
//	if err != nil {
//		return err
//	}
//
//	// Use with an agent
//	agent, _ := prebuilt.CreateReactAgent(llm, []tools.Tool{searchTool}, 10)
//
//	// Or use directly
//	result, err := searchTool.Call(ctx, `{
//		"query": "latest developments in quantum computing",
//		"max_results": 5
//	}`)
//
// ### Brave Search
// Use Brave Search API for web searching:
//
//	braveTool, err := tool.NewBraveSearchTool("your-brave-api-key")
//	if err != nil {
//		return err
//	}
//
// ### Bocha Search
// Chinese search engine integration:
//
//	bochaTool, err := tool.NewBochaSearchTool("your-bocha-api-key")
//
// ### EXA Search
// Advanced neural search with EXA:
//
//	exaTool, err := tool.NewEXASearchTool("your-exa-api-key")
//
// ## File Operations
//
// ### File Tool
// Basic file system operations:
//
//	fileTool := &tool.FileTool{}
//
//	// Read a file
//	result, _ := fileTool.Call(ctx, `{
//		"action": "read",
//		"path": "/path/to/file.txt"
//	}`)
//
//	// Write a file
//	result, _ := fileTool.Call(ctx, `{
//		"action": "write",
//		"path": "/path/to/output.txt",
//		"content": "Hello, World!"
//	}`)
//
//	// List directory
//	result, _ := fileTool.Call(ctx, `{
//		"action": "list",
//		"path": "/path/to/directory"
//	}`)
//
// ### Knowledge Tool
// Load and search knowledge bases:
//
//	knowledgeTool := tool.NewKnowledgeTool("/path/to/knowledge")
//
//	result, _ := knowledgeTool.Call(ctx, `{
//		"query": "How to install LangGraph Go?",
//		"max_results": 3
//	}`)
//
// ## Code Execution
//
// ### Shell Tool
// Execute shell commands and scripts:
//
//	// Execute shell code
//	shellTool := &tool.ShellTool{}
//	result, _ := shellTool.Call(ctx, `{
//		"code": "ls -la /home/user"
//	}`)
//
//	// Execute with arguments
//	result, _ := shellTool.Call(ctx, `{
//		"code": "echo $1 $2",
//		"args": {"Hello", "World"}
//	}`)
//
// ### Python Tool
// Execute Python code:
//
//	pythonTool := &tool.PythonTool{}
//	result, _ := pythonTool.Call(ctx, `{
//		"code": "import math; print(math.sqrt(16))"
//	}`)
//
//	// With imports and global variables
//	result, _ := pythonTool.Call(ctx, `{
//		"code": "print(data['value'] * 2)",
//		"imports": ["numpy", "pandas"],
//		"globals": {"value": 42}
//	}`)
//
// ## Web Tools
//
// ### Web Tool
// Simple HTTP requests:
//
//	webTool := &tool.WebTool{}
//
//	// GET request
//	result, _ := webTool.Call(ctx, `{
//		"url": "https://api.example.com/data",
//		"method": "GET"
//	}`)
//
//	// POST request with headers
//	result, _ := webTool.Call(ctx, `{
//		"url": "https://api.example.com/submit",
//		"method": "POST",
//		"headers": {"Content-Type": "application/json"},
//		"body": "{\"key\": \"value\"}"
//	}`)
//
// ### Web Search Tool
// Generic web search tool:
//
//	searchTool := &tool.WebSearchTool{}
//	result, _ := searchTool.Call(ctx, `{
//		"query": "LangGraph Go documentation",
//		"num_results": 5
//	}`)
//
// # Tool Implementation
//
// ## Creating Custom Tools
//
// Implement the Tool interface:
//
//	import (
//		"github.com/tmc/langchaingo/tools"
//	)
//
//	type CustomTool struct {
//		apiKey string
//	}
//
//	func (t *CustomTool) Name() string {
//		return "custom_api_call"
//	}
//
//	func (t *CustomTool) Description() string {
//		return "Makes a call to the custom API"
//	}
//
//	func (t *CustomTool) Call(ctx context.Context, input string) (string, error) {
//		// Parse input
//		var params struct {
//			Query string `json:"query"`
//		}
//		if err := json.Unmarshal([]byte(input), &params); err != nil {
//			return "", err
//		}
//
//		// Make API call
//		result, err := t.callAPI(params.Query)
//		if err != nil {
//			return "", err
//		}
//
//		// Return result
//		return json.Marshal(result)
//	}
//
// # Tool Categories
//
// ## Base Tools
// Common tools available to all agents:
//
//	baseTools := tool.GetBaseTools()
//	// Includes:
//	// - run_shell_code: Execute shell code
//	// - run_shell_script: Execute shell script
//	// - run_python_code: Execute Python code
//	// - web_search: Perform web search
//	// - file_operations: File system operations
//
// ## Specialized Tools
// Tools for specific domains:
//
//	// Search tools
//	tavilyTool, _ := tool.NewTavilySearchTool(apiKey)
//	braveTool, _ := tool.NewBraveSearchTool(apiKey)
//	exaTool, _ := tool.NewEXASearchTool(apiKey)
//	bochaTool, _ := tool.NewBochaSearchTool(apiKey)
//
//	// Execution tools
//	shellTool := &tool.ShellTool{}
//	pythonTool := &tool.PythonTool{}
//	fileTool := &tool.FileTool{}
//
//	// Web tools
//	webTool := &tool.WebTool{}
//	webSearchTool := &tool.WebSearchTool{}
//
// # Integration Examples
//
// ## With ReAct Agent
//
//	// Combine multiple tools
//	tools := []tools.Tool{
//		&tool.ShellTool{},
//		&tool.FileTool{},
//		searchTool,
//		pythonTool,
//	}
//
//	agent, _ := prebuilt.CreateReactAgent(llm, tools, 15)
//
// ## With PTC Agent
//
//	ptcTools := []tools.Tool{
//		&tool.ShellTool{},
//		&tool.PythonTool{},
//		&tool.FileTool{},
//	}
//
//	ptcAgent, _ := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:    llm,
//		Tools:    ptcTools,
//		Language: ptc.LanguagePython,
//	})
//
// # Tool Configuration
//
// Many tools support configuration:
//
//	// Tavily with custom options
//	tavilyTool, _ := tool.NewTavilySearchToolWithConfig(
//		apiKey,
//		tool.TavilyConfig{
//			MaxResults: 10,
//			SearchDepth: "advanced",
//			IncludeRawContent: true,
//		},
//	)
//
//	// Shell tool with allowed commands
//	shellTool := tool.NewShellToolWithConfig(
//		tool.ShellConfig{
//			AllowedCommands: []string{"ls", "cat", "grep"},
//			Timeout: 30 * time.Second,
//			WorkingDir: "/safe/directory",
//		},
//	)
//
// # Security Considerations
//
//   - Validate all tool inputs
//   - Use chroot/sandboxing for code execution
//   - Set timeouts for network operations
//   - Restrict file system access
//   - Sanitize shell commands
//   - Use API keys securely
//   - Monitor tool usage
//
// # Error Handling
//
// Tools provide structured error responses:
//
//	type ToolError struct {
//		Code    string `json:"code"`
//		Message string `json:"message"`
//		Details any    `json:"details,omitempty"`
//	}
//
//	result, err := tool.Call(ctx, input)
//	if err != nil {
//		var toolErr *ToolError
//		if errors.As(err, &toolErr) {
//			// Handle specific tool error
//			fmt.Printf("Tool error: %s - %s\n", toolErr.Code, toolErr.Message)
//		}
//	}
//
// # Best Practices
//
//  1. Choose appropriate tools for your use case
//  2. Provide clear tool descriptions
//  3. Validate inputs before processing
//  4. Handle errors gracefully
//  5. Use timeouts for long-running operations
//  6. Monitor tool usage and performance
//  7. Secure sensitive operations
//  8. Test tools with various inputs
//
// # Tool Composition
//
// Tools can be composed for complex workflows:
//
//	// Create a composite tool that uses multiple sub-tools
//	type CompositeTool struct {
//		searchTool *tool.WebSearchTool
//		fileTool   *tool.FileTool
//	}
//
//	func (t *CompositeTool) Call(ctx context.Context, input string) (string, error) {
//		// Search for information
//		searchResult, _ := t.searchTool.Call(ctx, input)
//
//		// Save results to file
//		saveResult, _ := t.fileTool.Call(ctx, map[string]any{
//			"action": "write",
//			"path":   "/tmp/search_results.txt",
//			"content": searchResult,
//		})
//
//		return saveResult, nil
//	}
package tool

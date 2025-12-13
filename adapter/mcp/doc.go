// Package mcp provides an adapter for integrating Model Context Protocol (MCP) tools with LangGraph Go agents.
//
// MCP is an open protocol that allows AI assistants to securely connect to external data sources
// and tools. This adapter enables LangGraph agents to use MCP-compliant tools and services,
// providing access to a growing ecosystem of MCP integrations including databases, APIs,
// file systems, and more.
//
// # Core Components
//
// ## MCPTool
// The main adapter that wraps MCP protocol tools as LangChain-compatible tools:
//
//	import (
//		"github.com/smallnest/langgraphgo/adapter/mcp"
//		"github.com/smallnest/langgraphgo/prebuilt"
//	)
//
//	// Connect to MCP server
//	client, err := mcp.NewMCPClient("stdio", []string{"python", "mcp_server.py"})
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	// List available tools
//	tools, err := client.ListTools()
//	if err != nil {
//		return err
//	}
//
//	// Convert MCP tools to LangChain tools
//	langchainTools := make([]tools.Tool, len(tools))
//	for i, mcpTool := range tools {
//		langchainTools[i] = &mcp.MCPTool{
//			name:        mcpTool.Name,
//			description: mcpTool.Description,
//			client:      client,
//			parameters:  mcpTool.InputSchema,
//		}
//	}
//
//	// Use with ReAct agent
//	agent, err := prebuilt.CreateReactAgent(llm, langchainTools, 10)
//
// # MCP Server Integration
//
// ## Standard Input/Output (stdio)
// Most common connection type for local MCP servers:
//
//	client, err := mcp.NewMCPClient("stdio", []string{
//		"python",
//		"-m",
//		"mcp_server_sqlite",
//		"--db-path",
//		"/path/to/database.db",
//	})
//
// ## HTTP Transport
// Connect to remote MCP servers via HTTP:
//
//	client, err := mcp.NewMCPClientWithConfig(mcp.Config{
//		Transport: "http",
//		URL:      "http://localhost:8080/mcp",
//		Headers: map[string]string{
//			"Authorization": "Bearer your-token",
//		},
//	})
//
// ## WebSocket Transport
// Real-time bidirectional communication:
//
//	client, err := mcp.NewMCPClientWithConfig(mcp.Config{
//		Transport: "websocket",
//		URL:      "ws://localhost:8080/ws",
//	})
//
// # Available MCP Tools
//
// ## Database Tools
// Query databases through MCP:
//
//	// SQLite MCP server
//	client, _ := mcp.NewMCPClient("stdio", []string{
//		"sqlite-mcp",
//		"--db-path", "./data.db",
//	})
//
//	// Use database tools
//	agent, _ := prebuilt.CreateReactAgent(llm, mcpTools, 10)
//
//	result, _ := agent.Invoke(ctx, map[string]any{
//		"messages": []llms.MessageContent{
//			{
//				Role: llms.ChatMessageTypeHuman,
//				Parts: []llms.ContentPart{
//					llms.TextPart("Show me all users from the database"),
//				},
//			},
//		},
//	})
//
// ## File System Tools
// Access file systems through MCP:
//
//	// File system MCP server
//	client, _ := mcp.NewMCPClient("stdio", []string{
//		"filesystem-mcp",
//		"--root", "/allowed/path",
//	})
//
//	// Agent can now read/write files
//	result, _ := agent.Invoke(ctx, map[string]any{
//		"messages": []llms.MessageContent{
//			{
//				Role: llms.ChatMessageTypeHuman,
//				Parts: []llms.ContentPart{
//					llms.TextPart("Read the config.yaml file and update the port to 8080"),
//				},
//			},
//		},
//	})
//
// ## Web API Tools
// Connect to web services through MCP:
//
//	// GitHub MCP server
//	client, _ := mcp.NewMCPClient("stdio", []string{
//		"github-mcp",
//		"--token", os.Getenv("GITHUB_TOKEN"),
//	})
//
//	// Agent can interact with GitHub
//	result, _ := agent.Invoke(ctx, map[string]any{
//		"messages": []llms.MessageContent{
//			{
//				Role: llms.ChatMessageTypeHuman,
//				Parts: []llms.ContentPart{
//					llms.TextPart("List all pull requests in the langgraph-go repository"),
//				},
//			},
//		},
//	})
//
// # Integration Examples
//
// ## Multi-Tool Agent with Multiple MCP Servers
//
//	// Connect to multiple MCP servers
//	sqliteClient, _ := mcp.NewMCPClient("stdio", []string{
//		"sqlite-mcp",
//		"--db-path", "./data.db",
//	})
//	defer sqliteClient.Close()
//
//	fsClient, _ := mcp.NewMCPClient("stdio", []string{
//		"filesystem-mcp",
//		"--root", "/data",
//	})
//	defer fsClient.Close()
//
//	// Collect all tools
//	var allTools []tools.Tool
//
//	sqliteTools, _ := sqliteClient.ListTools()
//	for _, t := range sqliteTools {
//		allTools = append(allTools, &mcp.MCPTool{
//			name:        t.Name,
//			description: t.Description,
//			client:      sqliteClient,
//			parameters:  t.InputSchema,
//		})
//	}
//
//	fsTools, _ := fsClient.ListTools()
//	for _, t := range fsTools {
//		allTools = append(allTools, &mcp.MCPTool{
//			name:        t.Name,
//			description: t.Description,
//			client:      fsClient,
//			parameters:  t.InputSchema,
//		})
//	}
//
//	// Create agent with all MCP tools
//	agent, _ := prebuilt.CreateReactAgent(llm, allTools, 20)
//
// ## Dynamic Tool Discovery
//
//	// Discover tools at runtime
//	client, _ := mcp.NewMCPClient("stdio", []string{"dynamic-mcp-server"})
//
//	// Periodically refresh tool list
//	ticker := time.NewTicker(5 * time.Minute)
//	go func() {
//		for range ticker.C {
//			tools, _ := client.ListTools()
//			// Update agent's tool list
//			updateAgentTools(agent, tools)
//		}
//	}()
//
// # MCP Configuration
//
// ## Client Configuration
//
//	config := mcp.Config{
//		Transport: "stdio",
//		Command:   []string{"python", "server.py"},
//		Env: map[string]string{
//			"API_KEY": "your-api-key",
//			"DEBUG":   "true",
//		},
//		Timeout:    30 * time.Second,
//		MaxRetries: 3,
//		Headers: map[string]string{
//			"User-Agent": "LangGraph-Go/1.0",
//		},
//	}
//
//	client, _ := mcp.NewMCPClientWithConfig(config)
//
// ## Tool Configuration
//
//	// Configure individual tools
//	mcpTool := &mcp.MCPTool{
//		name:        "database_query",
//		description: "Execute SQL queries",
//		client:      client,
//		parameters: map[string]any{
//			"type": "object",
//			"properties": map[string]any{
//				"query": map[string]any{
//					"type":        "string",
//					"description": "SQL query to execute",
//				},
//			},
//			"required": []string{"query"},
//		},
//	}
//
// # Error Handling
//
// The adapter provides comprehensive error handling:
//
//	result, err := mcpTool.Call(ctx, input)
//	if err != nil {
//		var mcpErr *mcp.MCPError
//		if errors.As(err, &mcpErr) {
//			fmt.Printf("MCP Error: %s (Code: %d)\n", mcpErr.Message, mcpErr.Code)
//			fmt.Printf("Tool: %s\n", mcpErr.Tool)
//			fmt.Printf("Data: %v\n", mcpErr.Data)
//		}
//	}
//
// # Security Features
//
//   - Authentication and authorization
//   - Request/response validation
//   - Rate limiting
//   - Audit logging
//   - Secure transport layers
//   - Permission scopes
//
// # Performance Optimization
//
//   - Connection pooling
//   - Request batching
//   - Response caching
//   - Compression
//   - Keep-alive connections
//
// # Best Practices
//
//  1. Use appropriate transport for your use case (stdio for local, HTTP for remote)
//  2. Set reasonable timeouts for tool execution
//  3. Handle MCP errors gracefully
//  4. Close clients when done
//  5. Validate tool parameters before calling
//  6. Use environment variables for sensitive configuration
//  7. Monitor tool usage and performance
//  8. Implement retry logic for transient errors
//
// # Advanced Features
//
// ## Tool Streaming
// For long-running operations:
//
//	client, _ := mcp.NewMCPClient("stdio", cmd)
//
//	// Enable streaming for specific tools
//	mcpTool := &mcp.MCPTool{
//		name:        "long_running_task",
//		description: "Execute long-running task with streaming",
//		client:      client,
//		streaming:   true,
//	}
//
//	// Handle streaming response
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
//	defer cancel()
//
//	result, err := mcpTool.CallWithStream(ctx, input, func(chunk string) {
//		fmt.Printf("Progress: %s\n", chunk)
//	})
//
// ## Tool Composition
// Combine multiple MCP tools:
//
//	// Create a composite tool that uses multiple MCP tools
//	type CompositeMCPTool struct {
//		tools []*mcp.MCPTool
//	}
//
//	func (t *CompositeMCPTool) Name() string {
//		return "composite_operation"
//	}
//
//	func (t *CompositeMCPTool) Description() string {
//		return "Performs complex operation using multiple tools"
//	}
//
//	func (t *CompositeMCPTool) Call(ctx context.Context, input string) (string, error) {
//		// Parse input to determine sequence of operations
//		var ops []Operation
//		json.Unmarshal([]byte(input), &ops)
//
//		// Execute tools in sequence
//		var results []any
//		for _, op := range ops {
//			for _, tool := range t.tools {
//				if tool.Name() == op.Tool {
//					result, _ := tool.Call(ctx, op.Params)
//					results = append(results, result)
//				}
//			}
//		}
//
//		// Return combined results
//		return json.Marshal(results)
//	}
//
// # MCP Server Development
//
// Create custom MCP servers:
//
//	// server.py
//	import asyncio
//	from mcp.server import Server
//	from mcp.server.stdio import stdio_server
//	from mcp.types import Tool
//
//	app = Server("my-mcp-server")
//
//	@app.list_tools()
//	async def list_tools() -> list[Tool]:
//		return [
//			Tool(
//				name="my_tool",
//				description="Custom tool description",
//				inputSchema={
//					"type": "object",
//					"properties": {
//						"param1": {"type": "string"},
//					},
//					"required": ["param1"],
//				},
//			),
//		]
//
//	@app.call_tool()
//	async def call_tool(name: str, arguments: dict) -> str:
//		if name == "my_tool":
//			# Custom tool logic
//			return f"Processed: {arguments['param1']}"
//
//	async def main():
//		async with stdio_server() as (read_stream, write_stream):
//			await app.run(read_stream, write_stream)
//
//	if __name__ == "__main__":
//		asyncio.run(main())
//
// # Monitoring and Debugging
//
//	// Enable MCP logging
//	client, _ := mcp.NewMCPClientWithConfig(mcp.Config{
//		Transport: "stdio",
//		Command:   []string{"python", "server.py"},
//		LogLevel:  "debug",
//		LogFile:   "/tmp/mcp.log",
//	})
//
//	// Get client statistics
//	stats := client.GetStats()
//	fmt.Printf("Total requests: %d\n", stats.Requests)
//	fmt.Printf("Average latency: %v\n", stats.AvgLatency)
//	fmt.Printf("Error rate: %.2f%%\n", stats.ErrorRate)
//
// # Community MCP Servers
//
// Popular MCP servers to integrate:
//
//   - sqlite-mcp: Database access
//   - filesystem-mcp: File system operations
//   - github-mcp: GitHub API integration
//   - slack-mcp: Slack workspace access
//   - gmail-mcp: Email management
//   - postgres-mcp: PostgreSQL database
//   - redis-mcp: Redis operations
//   - kubernetes-mcp: Kubernetes cluster management
//   - aws-mcp: AWS service integration
//   - mongodb-mcp: MongoDB database access
package mcp

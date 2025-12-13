// Package adapter provides integration adapters for connecting LangGraph Go with external systems and frameworks.
//
// Adapters act as bridges between LangGraph's internal representations and external APIs,
// protocols, or frameworks. They enable seamless integration with a wide ecosystem of tools,
// services, and platforms without modifying the core LangGraph implementation.
//
// This package includes adapters for:
//   - GoSkills: Custom Go-based skills and tools
//   - MCP (Model Context Protocol): Standardized tool communication
//
// # Core Concepts
//
// ## Adapter Pattern
//
// Each adapter implements conversion between different representations:
//   - LangChain tools → LangGraph tools
//   - External protocols → Internal interfaces
//   - Third-party APIs → Native functionality
//
// The adapters ensure type safety and provide consistent error handling while maintaining
// the flexibility to work with various external systems.
//
// ## Integration Approach
//
// Adapters in this package follow these principles:
//   - Zero-configuration: Work out of the box with sensible defaults
//   - Extensible: Allow customization for advanced use cases
//   - Performant: Minimize overhead through efficient conversions
//   - Compatible: Support standard protocols and formats
//
// # Available Adapters
//
// ## GoSkills Adapter (adapter/goskills)
//
// Integrates GoSkills framework for defining and executing Go-based skills:
//
//   - Load Go skills from directories or repositories
//   - Convert GoSkills to LangChain-compatible tools
//   - Execute Go code in controlled environments
//   - Support custom skill development
//
// Use Cases:
//   - Custom business logic written in Go
//   - High-performance native code execution
//   - Integration with existing Go services
//   - Type-safe tool implementations
//
// Example:
//
//	import "github.com/smallnest/langgraphgo/adapter/goskills"
//
//	// Load skills from directory
//	skills, _ := goskills.LoadSkillsFromDir("./skills")
//
//	// Convert to LangChain tools
//	tools, _ := goskills.ConvertToLangChainTools(skills)
//
//	// Use with ReAct agent
//	agent, _ := prebuilt.CreateReactAgent(llm, tools, 10)
//
// ## MCP Adapter (adapter/mcp)
//
// Integrates with the Model Context Protocol for standardized tool communication:
//
//   - Connect to MCP servers via various transports
//   - Automatically discover available tools
//   - Handle MCP protocol messages
//   - Support real-time communication
//
// Use Cases:
//   - Access to growing MCP tool ecosystem
//   - Standardized tool interfaces
//   - Cross-platform compatibility
//   - Community-driven tool development
//
// Example:
//
//	import "github.com/smallnest/langgraphgo/adapter/mcp"
//
//	// Connect to MCP server
//	client, _ := mcp.NewMCPClient("stdio", []string{"python", "mcp_server.py"})
//
//	// List available tools
//	tools, _ := client.ListTools()
//
//	// Convert to LangChain tools
//	langchainTools := make([]tools.Tool, len(tools))
//	for i, t := range tools {
//	    langchainTools[i] = &mcp.MCPTool{
//	        name:        t.Name,
//	        description: t.Description,
//	        client:      client,
//	    }
//	}
//
// # Usage Patterns
//
// ## Single Adapter Usage
//
//	// Using only GoSkills
//	goskillsTools, _ := goskills.ConvertToLangChainTools(skills)
//	agent, _ := prebuilt.CreateReactAgent(llm, goskillsTools, 10)
//
//	// Using only MCP
//	mcpTools, _ := mcp.ConvertMCPTools(mcpClient)
//	agent, _ := prebuilt.CreateReactAgent(llm, mcpTools, 10)
//
// ## Multiple Adapters
//
// Combine tools from multiple adapters:
//
//	// Load tools from different sources
//	var allTools []tools.Tool
//
//	// GoSkills tools
//	goskillsTools, _ := goskills.ConvertToLangChainSkills(goskills.Skills)
//	allTools = append(allTools, goskillsTools...)
//
//	// MCP tools
//	mcpTools, _ := mcp.DiscoverTools(mcpServers...)
//	allTools = append(allTools, mcpTools...)
//
//	// Built-in tools
//	builtinTools := []tools.Tool{&CalculatorTool{}, &WeatherTool{}}
//	allTools = append(allTools, builtinTools...)
//
//	// Create agent with all tools
//	agent, _ := prebuilt.CreateReactAgent(llm, allTools, 20)
//
// # Adapter Configuration
//
// ## Adapter Options
//
// Most adapters support configuration through options:
//
//	// GoSkills configuration
//	goskillsConfig := goskills.Config{
//	    SkillPath:    "./skills",
//	    WatchChanges: true,
//	    CacheResults: true,
//	}
//	goskillsAdapter, _ := goskills.NewAdapter(goskillsConfig)
//
//	// MCP configuration
//	mcpConfig := mcp.Config{
//	    Transport:    "http",
//	    URL:         "http://localhost:8080/mcp",
//	    Timeout:     30 * time.Second,
//	    RetryPolicy: mcp.ExponentialBackoff,
//	}
//	mcpAdapter, _ := mcp.NewAdapter(mcpConfig)
//
// ## Dynamic Adapter Loading
//
//	// Load adapters based on configuration
//	func LoadAdapters(config Config) ([]tools.Tool, error) {
//	    var tools []tools.Tool
//
//	    if config.GoSkills.Enabled {
//	        goskillsAdapter, _ := goskills.NewAdapter(config.GoSkills)
//	        tools = append(tools, goskillsAdapter.GetTools()...)
//	    }
//
//	    if config.MCP.Enabled {
//	        mcpAdapter, _ := mcp.NewAdapter(config.MCP)
//	        tools = append(tools, mcpAdapter.GetTools()...)
//	    }
//
//	    return tools, nil
//	}
//
// # Performance Considerations
//
// ## Adapter Overhead
//
// Adapters add minimal overhead, but consider:
//   - Lazy loading of adapters
//   - Caching of converted tools
//   - Connection pooling for remote adapters
//   - Batching operations where possible
//
// Example optimization:
//
//	type CachedAdapter struct {
//	    tools []tools.Tool
//	    mutex sync.RWMutex
//	    cache map[string]tools.Tool
//	}
//
//	func (a *CachedAdapter) GetTool(name string) (tools.Tool, error) {
//	    a.mutex.RLock()
//	    if tool, exists := a.cache[name]; exists {
//	        a.mutex.RUnlock()
//	        return tool, nil
//	    }
//	    a.mutex.RUnlock()
//
//	    // Load tool and cache
//	    tool, err := a.loadTool(name)
//	    if err != nil {
//	        return nil, err
//	    }
//
//	    a.mutex.Lock()
//	    a.cache[name] = tool
//	    a.mutex.Unlock()
//
//	    return tool, nil
//	}
//
// # Error Handling
//
// Adapters provide consistent error handling:
//
//	// Adapter-specific errors
//	type AdapterError struct {
//	    Adapter string
//	    Tool    string
//	    Cause   error
//	}
//
//	func (e *AdapterError) Error() string {
//	    return fmt.Sprintf("adapter %s: tool %s: %v", e.Adapter, e.Tool, e.Cause)
//	}
//
//	// Recover from adapter errors
//	func handleAdapterError(err error) error {
//	    if adapterErr, ok := err.(*AdapterError); ok {
//	        // Log and continue with other tools
//	        log.Printf("Adapter error: %v", adapterErr)
//	        return nil
//	    }
//	    return err
//	}
//
// # Testing with Adapters
//
// ## Mock Adapters
//
//	// Mock adapter for testing
//	type MockAdapter struct {
//	    tools map[string]tools.Tool
//	}
//
//	func (m *MockAdapter) GetTools() []tools.Tool {
//	    var tools []tools.Tool
//	    for _, tool := range m.tools {
//	        tools = append(tools, tool)
//	    }
//	    return tools
//	}
//
//	func (m *MockAdapter) AddTool(name string, tool tools.Tool) {
//	    m.tools[name] = tool
//	}
//
//	// Use in tests
//	func TestAgentWithMockAdapter(t *testing.T) {
//	    mockAdapter := &MockAdapter{
//	        tools: make(map[string]tools.Tool),
//	    }
//	    mockAdapter.AddTool("test", &MockTool{})
//
//	    agent, _ := prebuilt.CreateReactAgent(mockLLM, mockAdapter.GetTools(), 10)
//	    // Test agent behavior
//	}
//
// # Extending the Package
//
// To add a new adapter:
//
//  1. Create a new directory under adapter/
//
//  2. Implement the adapter interface
//
//  3. Provide configuration options
//
//  4. Add comprehensive tests
//
//  5. Document with examples
//
//     // Example adapter structure
//     package myadapter
//
//     type MyAdapter struct {
//     config Config
//     client *MyClient
//     }
//
//     func (a *MyAdapter) Convert() ([]tools.Tool, error) {
//     // Convert external tools to LangChain tools
//     }
//
//     func (a *MyAdapter) Close() error {
//     // Cleanup resources
//     }
//
// # Best Practices
//
//  1. **Choose the right adapter for your use case**
//     - GoSkills for custom Go implementations
//     - MCP for standardized protocols
//     - Multiple adapters for diverse toolsets
//
//  2. **Handle adapter failures gracefully**
//     - Provide fallback mechanisms
//     - Log errors appropriately
//     - Continue with available tools
//
//  3. **Optimize performance**
//     - Cache converted tools
//     - Use connection pooling
//     - Lazy load when possible
//
//  4. **Maintain security**
//     - Validate external inputs
//     - Use secure connections
//     - Implement proper authentication
//
//  5. **Test thoroughly**
//     - Mock external dependencies
//     - Test error scenarios
//     - Verify integration correctness
//
// # Community Contributions
//
// The adapter package welcomes contributions for new integrations:
//   - gRPC adapter
//   - GraphQL adapter
//   - REST API generator adapter
//   - Database adapter
//   - Message queue adapter
//
// Please follow established patterns and provide:
//   - Comprehensive tests
//   - Clear documentation
//   - Error handling
//   - Performance considerations
package adapter

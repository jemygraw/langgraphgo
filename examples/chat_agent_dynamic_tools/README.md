# ChatAgent Dynamic Tools Example

This example demonstrates the dynamic tool management capabilities of `ChatAgent` in LangGraphGo, allowing you to add, remove, and update tools during a conversation.

## Features

- **Dynamic Tool Addition**: Add new tools to the agent at any point during the conversation
- **Tool Removal**: Remove specific tools by name when they're no longer needed
- **Tool Replacement**: Replace all tools at once with a new set
- **Tool Querying**: Get the current list of available tools
- **Tool Clearing**: Remove all dynamic tools instantly

## Why Dynamic Tools?

Dynamic tool management enables several powerful use cases:

1. **Context-Aware Capabilities**: Add tools based on user requests or conversation context
2. **Resource Management**: Remove expensive tools when not needed
3. **Progressive Enhancement**: Start with basic tools and add advanced ones as needed
4. **Access Control**: Grant or revoke tool access based on authentication or permissions
5. **A/B Testing**: Switch between different tool implementations

## API Overview

```go
// Add a single tool
agent.AddTool(weatherTool)

// Remove a tool by name
removed := agent.RemoveTool("calculator")

// Get current tools
currentTools := agent.GetTools()

// Replace all tools
agent.SetTools([]tools.Tool{tool1, tool2})

// Clear all dynamic tools
agent.ClearTools()
```

## How It Works

The `ChatAgent` maintains two sets of tools:

1. **Base Tools**: Provided when creating the agent (via `NewChatAgent`)
2. **Dynamic Tools**: Added/removed at runtime using the management methods

When processing a message, both sets are available to the agent. The dynamic tools are passed to the agent graph via the `extra_tools` mechanism.

## Running the Example

```bash
cd examples/chat_agent_dynamic_tools
go run main.go
```

## Example Output

The example demonstrates:

1. **Turn 1**: Chat with no tools
2. **Turn 2**: Add calculator tool and perform calculation
3. **Turn 3**: Add weather tool while keeping calculator
4. **Turn 4**: Remove calculator, keep weather tool

## Important Notes

- **Tool Uniqueness**: Adding a tool with the same name as an existing tool will replace it
- **Base Tools**: Dynamic tool methods don't affect base tools provided at creation
- **Thread Safety**: For concurrent use, you should add your own synchronization
- **Persistence**: Dynamic tools are in-memory; they're not persisted across application restarts

## Real-World Use Cases

### 1. Conditional Tool Access

```go
// Grant file access only after authentication
if user.IsAuthenticated() {
    agent.AddTool(fileReadTool)
    agent.AddTool(fileWriteTool)
}
```

### 2. Context-Based Tools

```go
// Add database tools when user mentions data queries
if containsDataKeywords(message) {
    agent.AddTool(sqlQueryTool)
}
```

### 3. Cost Management

```go
// Remove expensive API tools after quota exceeded
if quotaExceeded() {
    agent.RemoveTool("expensive_api")
}
```

### 4. Feature Flags

```go
// Enable experimental tools based on feature flags
if featureFlags["experimental_tools"] {
    agent.AddTool(experimentalTool)
}
```

## Integration with Real LLMs

When using with real LLMs like OpenAI:

```go
import "github.com/tmc/langchaingo/llms/openai"

// Create model
model, _ := openai.New(openai.WithModel("gpt-4"))

// Create agent with base tools
agent, _ := prebuilt.NewChatAgent(model, baseTtools)

// Add tools dynamically based on conversation
if userNeedsWeather {
    agent.AddTool(weatherTool)
}

response, _ := agent.Chat(ctx, "What's the weather?")
```

## See Also

- [Basic ChatAgent Example](../chat_agent/) - Simple multi-turn conversation
- [CreateAgent Documentation](../../prebuilt/README.md) - Underlying agent creation
- [Tool Executor](../../prebuilt/tool_executor.go) - Tool execution mechanism

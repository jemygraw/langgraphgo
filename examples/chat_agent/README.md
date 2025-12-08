# ChatAgent Example

This example demonstrates the `ChatAgent` feature in LangGraphGo, which enables multi-turn conversations with automatic session management.

## Features

- **Session Management**: Each `ChatAgent` instance maintains its own conversation session with a unique thread ID
- **Conversation History**: Messages are automatically accumulated across multiple turns
- **Simple API**: Easy-to-use `Chat()` method for sending messages and receiving responses
- **Memory**: The agent can reference previous messages in the conversation

## How It Works

The `ChatAgent` wraps an agent graph created with `CreateAgent` and manages the conversation state:

1. When you create a `ChatAgent`, it generates a unique session ID (thread ID)
2. Each call to `Chat()` appends the user's message to the conversation history
3. The full conversation history is passed to the underlying agent
4. The agent's response is added to the history and returned
5. Subsequent calls continue the conversation with full context

## Usage

```go
// Create a ChatAgent
agent, err := prebuilt.NewChatAgent(model, tools)
if err != nil {
    log.Fatal(err)
}

// First turn
response1, err := agent.Chat(ctx, "Hello! My name is Alice.")

// Second turn - agent remembers previous context
response2, err := agent.Chat(ctx, "What's my name?")
// Response: "Your name is Alice, as you told me earlier."

// Get session ID
sessionID := agent.ThreadID()
```

## Running the Example

```bash
cd examples/chat_agent
go run main.go
```

## Expected Output

The example demonstrates a multi-turn conversation where:
1. User greets the agent
2. User introduces their name
3. Agent recalls the name from history
4. Conversation continues with full context

## Notes

- This example uses a simple mock model for demonstration purposes
- In production, you would use a real LLM like OpenAI's GPT-4
- The conversation history is maintained in memory for the lifetime of the `ChatAgent` instance
- Each `ChatAgent` instance represents a separate conversation session

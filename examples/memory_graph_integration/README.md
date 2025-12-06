# Memory + LangGraph Integration

This example demonstrates how to integrate memory strategies directly into LangGraph workflows using State, creating stateful conversational agents with sophisticated memory management.

## What This Example Shows

Unlike previous examples that showed memory strategies in isolation or simple agents, this example demonstrates:

1. **Memory in Graph State** - How to include memory strategies as part of your workflow state
2. **Multi-Node Memory Access** - How different nodes in a graph can access and update shared memory
3. **Stateful Workflows** - Building workflows that maintain conversation context across multiple turns
4. **Real Integration** - Actual LangGraph + Memory working together in production-like scenarios

## Architecture

### State Design

```go
type ConversationState struct {
    UserInput      string              // Current user message
    Intent         string              // Classified intent
    Context        []*memory.Message   // Retrieved context from memory
    Response       string              // Generated response
    Memory         memory.Strategy     // Memory strategy (lives in state!)
    ConversationID string              // Conversation identifier
    TurnCount      int                 // Number of turns
}
```

**Key Insight**: The memory strategy lives in the State, making it accessible to all nodes in the workflow.

### Workflow Structure

```
Entry Point
    ↓
[Classify Intent] ──→ Analyzes user input, retrieves context from memory
    ↓
[Retrieve Info] ──→ Fetches relevant information based on intent
    ↓
[Generate Response] ──→ Creates response, adds messages to memory
    ↓
Finish Point
```

Each node:
- Reads from the shared memory strategy in State
- Retrieves relevant context for its task
- Updates memory with new information
- Passes state to the next node

## Running the Example

```bash
cd examples/memory_graph_integration
go run main.go
```

## Demonstrations

### Demo 1: Sliding Window Memory

**Strategy**: Keeps last 4 messages
**Use Case**: Recent context is most important

```
User: Hello!
Agent: Hello! I'm your product assistant...

User: What's the price?
Agent: Our premium product is priced at $99...

User: Tell me about features
Agent: Our product has amazing features...

User: Remind me of the price?
Agent: As I mentioned before, the product is priced at $99
```

**How it works**:
- Memory stores last 4 messages (2 user + 2 assistant)
- When asking about price again, it finds "$99" in context
- Agent responds with "As I mentioned before..."
- After 4 new messages, old price info would be forgotten

### Demo 2: Hierarchical Memory

**Strategy**: Keeps important + recent messages
**Use Case**: Some information is critical

```
User: Hi, my name is Alice
Agent: Nice to meet you, Alice! I'll remember your name.

[... several messages later ...]

User: Do you remember my name?
Agent: Of course! I remember you, Alice...
```

**How it works**:
- Important messages (name, requirements) marked with high importance
- Recent messages kept regardless
- Name stored in "important" tier, survives many conversation turns
- Agent can recall Alice's name even after many messages

### Demo 3: Retrieval Memory

**Strategy**: Finds relevant messages
**Use Case**: Large conversations, query-driven

```
User: What's the price?
Agent: Our premium product is priced at $99...

[... 6 other topics discussed ...]

User: Let's talk about the price again
Agent: As I mentioned before, the product is priced at $99
```

**How it works**:
- All messages stored with embeddings
- Query "price" retrieves messages containing pricing info
- Even after many unrelated messages, price info is found
- Uses semantic similarity, not just recent messages

### Demo 4: Graph-Based Memory

**Strategy**: Tracks topic relationships
**Use Case**: Related topics and connections

```
User: What's the price?
[Topics tracked: [price]]

User: Tell me about the warranty
[Topics tracked: [price, warranty]]

User: Does the price include warranty?
[Topics tracked: [price, warranty]]
```

**How it works**:
- Messages connected based on shared topics (price, warranty, etc.)
- When query mentions "price", retrieves price-related messages
- Also retrieves messages connected to price (warranty, features)
- Graph structure captures topic relationships

## Integration Pattern

### 1. Create Memory Strategy

```go
mem := memory.NewSlidingWindowMemory(5)
```

### 2. Create Workflow with Memory in State

```go
stateSchema := graph.StateSchema{
    "user_input": "",
    "memory":     mem,  // Memory strategy in state
    "response":   "",
}

workflow := graph.NewGraph(stateSchema)
```

### 3. Access Memory in Nodes

```go
func processNode(state graph.State) (graph.State, error) {
    mem := state["memory"].(memory.Strategy)
    userInput := state["user_input"].(string)

    // Get context from memory
    context, _ := mem.GetContext(ctx, userInput)

    // Use context for processing
    response := generateResponse(userInput, context)

    // Add new messages to memory
    mem.AddMessage(ctx, memory.NewMessage("user", userInput))
    mem.AddMessage(ctx, memory.NewMessage("assistant", response))

    state["response"] = response
    return state, nil
}
```

### 4. Invoke Workflow

```go
state := graph.State{
    "user_input": "Hello!",
    "memory":     mem,
}

result, _ := workflow.Invoke(state, nil)
response := result["response"].(string)
```

## Key Benefits

### 1. Separation of Concerns

- **Memory Strategy**: Handles context storage and retrieval
- **Workflow Nodes**: Focus on business logic
- **State**: Connects everything together

### 2. Flexibility

- Swap memory strategies without changing workflow logic
- Different workflows can use different strategies
- Easy to test with different memory configurations

### 3. Scalability

- Memory strategies handle growing conversation history
- Workflow remains simple and maintainable
- Can add new nodes without touching memory logic

### 4. Reusability

- Same workflow works with any memory strategy
- Memory strategies reusable across workflows
- State pattern allows complex compositions

## Comparison: Memory Integration Approaches

### Approach 1: Simple Agent (memory_agent example)
```go
agent.ProcessMessage(userInput)
```
- ✓ Simple to use
- ✗ Single processing path
- ✗ Limited composability

### Approach 2: Graph Integration (this example)
```go
workflow.Invoke(state, nil)
```
- ✓ Multi-node workflows
- ✓ Conditional routing
- ✓ Complex agent behaviors
- ✓ Better testing and debugging

## Advanced Patterns

### Conditional Routing Based on Context

```go
func shouldRetrieveMore(state graph.State) string {
    context := state["context"].([]*memory.Message)
    if len(context) < 2 {
        return "retrieve_more"
    }
    return "continue"
}

workflow.AddConditionalEdges("classify", shouldRetrieveMore, map[string]string{
    "retrieve_more": "retrieval_node",
    "continue":      "response_node",
})
```

### Multiple Memory Strategies

```go
stateSchema := graph.StateSchema{
    "short_term": memory.NewSlidingWindowMemory(5),
    "long_term":  memory.NewRetrievalMemory(...),
}

// Use short_term for recent context
// Use long_term for semantic search
```

### Memory Persistence

```go
// Save memory state
memData := mem.Export()
saveToDatabase(conversationID, memData)

// Restore memory state
memData := loadFromDatabase(conversationID)
mem.Import(memData)
```

## When to Use This Pattern

**Use Graph + Memory Integration when:**
- You need multi-step processing (intent → retrieval → response)
- Conditional logic based on conversation context
- Multiple specialized nodes with shared context
- Complex agent behaviors beyond simple chat
- Testing and debugging of individual components

**Use Simple Agent when:**
- Single-step request/response
- Straightforward conversation flow
- Rapid prototyping
- Simple chatbot use cases

## Performance Considerations

### Memory Strategy Impact

| Strategy | Node Access Speed | Memory Growth | Best For |
|----------|------------------|---------------|----------|
| Sequential | O(n) | Unlimited | Short conversations |
| Sliding Window | O(1) | Fixed | Real-time chat |
| Retrieval | O(log n) | Linear | Long conversations |
| Hierarchical | O(n) | Bounded | Mixed importance |
| Graph | O(k) | Linear | Related topics |

### Optimization Tips

1. **Choose Right Strategy**: Match strategy to conversation length
2. **Limit Context Size**: Don't pass entire history to each node
3. **Cache Stats**: Avoid repeated GetStats() calls
4. **Batch Updates**: Add multiple messages at once if possible

## Related Examples

- [memory_strategies](../memory_strategies/) - Individual strategy demonstrations
- [memory_agent](../memory_agent/) - Simple agent with memory
- [basic_llm](../basic_llm/) - Basic LangGraph usage

## Further Reading

- [Memory Package Documentation](../../memory/README.md)
- [LangGraph Documentation](../../graph/README.md)
- [State Management Guide](../../docs/state_management.md)

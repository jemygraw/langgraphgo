# Memory Strategies Examples

This example demonstrates all 9 memory management strategies available in LangGraphGo.

## Overview

Memory management is crucial for AI agents to maintain context while controlling token costs. This example showcases how to use each strategy with practical demonstrations.

## Strategies Demonstrated

### 1. Sequential Memory (Keep-It-All)
- **Use case**: Short conversations where cost is not a concern
- **Behavior**: Stores all messages without any limit
- **Demo**: Shows perfect recall of all interactions

### 2. Sliding Window Memory
- **Use case**: Chat with bounded history
- **Behavior**: Keeps only the most recent N messages
- **Demo**: Adds 5 messages but keeps only the last 3

### 3. Buffer Memory
- **Use case**: General purpose with flexible limits
- **Behavior**: Can limit by message count or token count
- **Demo**: Limits to 3 messages while tracking token usage

### 4. Summarization Memory
- **Use case**: Long conversations needing compression
- **Behavior**: Summarizes old messages, keeps recent ones full
- **Demo**: Automatically creates summaries after threshold

### 5. Retrieval Memory
- **Use case**: Large knowledge base with query-driven access
- **Behavior**: Retrieves most relevant messages using similarity
- **Demo**: Queries for "programming languages" and retrieves relevant messages

### 6. Hierarchical Memory
- **Use case**: Complex conversations with different importance levels
- **Behavior**: Separates important and recent messages
- **Demo**: Marks important messages and shows they're retained

### 7. Graph-Based Memory
- **Use case**: Tracking relationships between topics
- **Behavior**: Builds knowledge graph of message connections
- **Demo**: Tracks topic relationships and retrieves connected messages

### 8. Compression Memory
- **Use case**: Aggressive compression for long conversations
- **Behavior**: Compresses messages into blocks and consolidates
- **Demo**: Shows compression ratio and block creation

### 9. OS-Like Memory
- **Use case**: Sophisticated memory lifecycle management
- **Behavior**: Multi-tier memory (active/cache/archive) with LRU eviction
- **Demo**: Demonstrates paging and memory tier distribution

## Running the Example

```bash
cd examples/memory_strategies
go run main.go
```

## Expected Output

The program will demonstrate each strategy with:
- Description of the strategy
- Statistics about memory usage
- Sample of how messages are stored/retrieved
- Strategy-specific metrics (compression rate, relationships, etc.)

## Key Insights

1. **Sequential**: Best for short conversations, no optimization
2. **Sliding Window**: Simple and predictable, good for chat
3. **Buffer**: Flexible middle ground
4. **Summarization**: Great for long conversations with context preservation
5. **Retrieval**: Excellent for large knowledge bases
6. **Hierarchical**: Perfect for complex multi-topic conversations
7. **Graph-Based**: Best when relationships matter
8. **Compression**: Maximum space efficiency
9. **OS-Like**: Most sophisticated, handles complex access patterns

## Choosing a Strategy

| Your Scenario | Recommended Strategy |
|--------------|---------------------|
| Short chat (< 10 messages) | Sequential |
| Ongoing conversation with fixed history | Sliding Window |
| General purpose chatbot | Buffer |
| Long consultation sessions | Summarization |
| Knowledge base Q&A | Retrieval |
| Multi-topic discussions | Hierarchical |
| Topic-based navigation | Graph-Based |
| Cost-sensitive long conversations | Compression |
| Complex agent with varied access | OS-Like |

## Customization

Each strategy accepts configuration options. See the main.go file and the [memory package documentation](../../memory/README.md) for details on customizing:
- Window sizes
- Importance scoring functions
- Custom summarizers
- Embedding functions
- Compression triggers
- Memory limits

## Further Reading

- [Memory Package Documentation](../../memory/README.md)
- [Memory Package Documentation (中文)](../../memory/README_CN.md)

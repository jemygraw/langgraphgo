# Issue #73: Auto Resume from Checkpoint Proposal

## Problem Statement

Current Go implementation requires manual checkpoint management for resuming execution, which is less developer-friendly compared to Python LangGraph:

**Python LangGraph (Simple):**
```python
# Resume - just pass thread_id, everything else is automatic
result = graph.invoke(
    {"messages": [("user", "new input")]},
    config={"configurable": {"thread_id": "conversation-1"}}
)
```

**Current Go (Verbose):**
```go
// Requires manual: checkpoint retrieval, state extraction, type casting, ResumeFrom
checkpoints, _ := store.List(ctx, threadID)
latestCP := checkpoints[len(checkpoints)-1]
resumedState := latestCP.State.(map[string]any)
config.ResumeFrom = []string{"step3"}
runnable.InvokeWithConfig(ctx, resumedState, config)
```

## Proposal

### Goal

Match Python LangGraph's developer experience: when `thread_id` is provided, automatically:
1. Load the latest checkpoint state
2. Merge input with checkpoint state using Schema's reducer
3. Infer `ResumeFrom` from checkpoint's node
4. Execute from the correct position

### API Design

```go
// Before - verbose manual handling
checkpoints, _ := store.List(ctx, threadID)
latestCP := checkpoints[len(checkpoints)-1]
resumedState := latestCP.State.(map[string]any)
config := &graph.Config{
    Configurable: map[string]any{"thread_id": threadID},
    ResumeFrom:   []string{"step3"},
}
result, err := runnable.InvokeWithConfig(ctx, resumedState, config)

// After - simple and automatic
result, err := runnable.Invoke(ctx,
    map[string]any{"messages": []Message{{"new input"}}},  // Just new input
    graph.WithThreadID("conversation-1"),  // Auto-resume
)
```

### Implementation Strategy

#### Phase 1: Core Auto-Resume Logic

Modify `InvokeWithConfig` in `graph/checkpointing.go`:

```go
func (cr *CheckpointableRunnable[S]) InvokeWithConfig(ctx context.Context, input S, config *Config) (S, error) {
    threadID := extractThreadID(config)

    // Auto-resume if thread_id provided and checkpoint exists
    if threadID != "" {
        if latestCP, err := cr.getLatestCheckpoint(ctx, threadID); err == nil {
            // 1. Merge checkpoint state with new input using Schema
            input = cr.mergeStates(ctx, latestCP.State, input)

            // 2. Auto-set ResumeFrom from checkpoint node
            if config == nil { config = &Config{} }
            if config.ResumeFrom == nil && latestCP.NodeName != "" {
                config.ResumeFrom = determineNextNodes(latestCP)
            }
        }
    }

    // Continue with normal execution...
}
```

#### Phase 2: Helper Methods

```go
// WithThreadID creates a config with thread_id set
func WithThreadID(threadID string) *Config {
    return &Config{
        Configurable: map[string]any{"thread_id": threadID},
    }
}

// mergeStates merges checkpoint state with new input using Schema
func (cr *CheckpointableRunnable[S]) mergeStates(ctx context.Context, checkpointState S, input S) S {
    if cr.runnable.graph.Schema != nil {
        merged, _ := cr.runnable.graph.Schema.Update(checkpointState, input)
        return merged
    }
    // Fallback: input takes precedence
    return input
}

// getLatestCheckpoint retrieves the latest checkpoint for a thread
func (cr *CheckpointableRunnable[S]) getLatestCheckpoint(ctx context.Context, threadID string) (*store.Checkpoint, error) {
    if latestGetter, ok := cr.config.Store.(interface {
        GetLatestByThread(ctx context.Context, threadID string) (*store.Checkpoint, error)
    }); ok {
        return latestGetter.GetLatestByThread(ctx, threadID)
    }
    // Fallback to List
    checkpoints, err := cr.config.Store.List(ctx, threadID)
    if err != nil || len(checkpoints) == 0 {
        return nil, err
    }
    return checkpoints[len(checkpoints)-1], nil
}
```

#### Phase 3: ResumeFrom Inference

```go
// determineNextNodes infers next nodes to execute from checkpoint
func determineNextNodes(cp *store.Checkpoint) []string {
    if cp.NodeName == "" || cp.NodeName == END {
        return []string{}
    }
    // Resume from the node after the checkpoint node
    // The graph's edge logic will determine the actual next step
    return []string{cp.NodeName}
}
```

### Backward Compatibility

- Manual `ResumeFrom` setting still takes precedence
- Manual state passing still works
- New behavior only activates when:
  - `thread_id` is provided
  - Checkpoint exists for that thread
  - `ResumeFrom` is not manually set

### Usage Examples

#### Example 1: Multi-turn Conversation

```go
// Turn 1 - Create new conversation
res1, _ := runnable.Invoke(ctx,
    map[string]any{"messages": []Message{{Role: "user", Content: "Hello"}}},
    graph.WithThreadID("conv-1"))

// Turn 2 - Automatically resume and continue
res2, _ := runnable.Invoke(ctx,
    map[string]any{"messages": []Message{{Role: "user", Content: "How are you?"}}},
    graph.WithThreadID("conv-1"))
// res2 contains messages from both turns
```

#### Example 2: Interrupt and Resume

```go
// Interrupt after step2
config1 := graph.WithThreadID("thread-1")
config1.InterruptAfter = []string{"step2"}
runnable.Invoke(ctx, state, config1)

// Resume - automatically continues from step3
config2 := graph.WithThreadID("thread-1")
runnable.Invoke(ctx, nil, config2)  // Can pass nil or just new inputs
```

## Testing Strategy

1. **Unit tests**: Verify state merging, checkpoint loading, ResumeFrom inference
2. **Integration tests**: Full multi-turn conversation scenarios
3. **Backward compatibility tests**: Ensure existing manual control still works

## Dependencies

- Issue #72 (thread_id indexing) should be merged first for optimal performance
- `GetLatestByThread` method already implemented in stores

## Timeline

1. Phase 1: Core logic implementation
2. Phase 2: Helper methods and WithThreadID
3. Phase 3: Enhanced ResumeFrom inference
4. Testing and documentation updates

## References

- Original issue: https://github.com/smallnest/langgraphgo/issues/73
- Python LangGraph checkpoint docs: https://langchain-ai.github.io/langgraph/concepts/low_level/#checkpointing

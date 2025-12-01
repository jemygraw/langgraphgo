# Changelog

## [0.2.0] - 2025-12-01

### Core Runtime
- **Parallel Execution**: Implemented fan-out/fan-in execution model with thread-safe state merging.
- **Runtime Configuration**: Added `RunnableConfig` to propagate configuration (like thread IDs, user IDs) through the graph execution context.

### Persistence & Checkpointing
- **Checkpoint Interface**: Refined `CheckpointSaver` interface for state persistence.
- **Implementations**: Added full support for **Redis**, **PostgreSQL**, and **SQLite** checkpoint stores.

### Advanced State & Streaming
- **State Management**: Introduced `Schema` interface and `Annotated` style reducers (e.g., `AppendMessages`) for complex state updates.
- **Enhanced Streaming**: Added typed `StreamEvent`s and `CallbackHandler` interface for granular execution monitoring.

### Pre-built Agents
- **ToolExecutor**: Added a dedicated node for executing tools.
- **ReAct Agent**: Implemented a factory for creating ReAct-style agents.
- **Supervisor**: Added support for Supervisor agent patterns for multi-agent orchestration.

### Human-in-the-loop (HITL)
- **Interrupts**: Implemented `InterruptBefore` and `InterruptAfter` mechanisms to pause graph execution.
- **Resume & Command**: Added support for resuming execution and updating state via commands, enabling human approval workflows.

### Visualization
- **Mermaid Export**: Improved graph visualization with better rendering of conditional edges and styling options.

### Experimental & Research
- **Swarm Patterns**: Added prototypes for multi-agent collaboration using subgraphs (`examples/swarm`).
- **Channels RFC**: Added `RFC_CHANNELS.md` proposing a channel-based architecture for future improvements.

### Examples
- Added comprehensive examples for:
  - Checkpointing (Postgres, SQLite, Redis)
  - Human-in-the-loop workflows
  - Swarm multi-agent patterns
  - Subgraphs

## [0.1.0] - 2025-01-02

### Added
- Generic state management - works with any type, not just MessageContent
- Performance optimizations for production use
- Support for any LLM client (removed hard dependency on LangChain)

### Changed
- Simplified API for building graphs
- Updated examples to show generic usage

### Fixed
- CI/CD pipeline issues from original repository
- Build errors with recent Go versions

### Removed
- Hard dependency on LangChain - now works with any LLM library
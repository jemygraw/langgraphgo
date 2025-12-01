# LangGraphGo Examples

This directory contains various examples demonstrating the features of LangGraphGo.

## Basic Concepts
- **[Basic Example](basic_example/)**: Simple graph with hardcoded steps.
- **[Basic LLM](basic_llm/)**: Integration with LLMs.
- **[Conditional Routing](conditional_routing/)**: Dynamic routing based on state.
- **[Conditional Edges](conditional_edges_example/)**: Using conditional edges.

## Advanced Features
- **[Parallel Execution](parallel_execution/)**: Fan-out/Fan-in execution with state merging.
- **[Configuration](configuration/)**: Using runtime configuration to pass metadata and settings.
- **[Custom Reducer](custom_reducer/)**: Defining custom state reducers for complex merge logic.
- **[Subgraphs](subgraph/)**: Composing graphs within graphs.
- **[Streaming](streaming_pipeline/)**: Streaming execution events.
- **[Listeners](listeners/)**: Attaching event listeners to the graph.

## Persistence (Checkpointing)
- **[Memory](checkpointing/main.go)**: In-memory checkpointing.
- **[PostgreSQL](checkpointing/postgres/)**: Persistent state using PostgreSQL.
- **[SQLite](checkpointing/sqlite/)**: Persistent state using SQLite.
- **[Redis](checkpointing/redis/)**: Persistent state using Redis.

## Human-in-the-loop
- **[Human Approval](human_in_the_loop/)**: Workflow with interrupts and human approval steps.

## Pre-built Agents
- **[ReAct Agent](react_agent/)**: Reason and Action agent using tools.
- **[Supervisor](supervisor/)**: Multi-agent orchestration using a supervisor.
- **[Swarm](swarm/)**: Multi-agent collaboration using handoffs.

## Other
- **[RAG Pipeline](rag_pipeline/)**: Retrieval Augmented Generation example.
- **[Visualization](visualization/)**: Generating Mermaid diagrams for graphs.
- **[LangChain Integration](langchain_example/)**: Using LangChain tools and models.

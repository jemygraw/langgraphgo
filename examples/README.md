# LangGraphGo Examples

This directory contains various examples demonstrating the features of LangGraphGo.

## Basic Concepts
- **[Basic Example](basic_example/README.md)**: Simple graph with hardcoded steps.
- **[Basic LLM](basic_llm/README.md)**: Integration with LLMs.
- **[Conditional Routing](conditional_routing/README.md)**: Dynamic routing based on state.
- **[Conditional Edges](conditional_edges_example/README.md)**: Using conditional edges.

## Advanced Features
- **[Parallel Execution](parallel_execution/README.md)**: Fan-out/Fan-in execution with state merging.
- **[Complex Parallel Execution](complex_parallel_execution/README.md)**: Advanced parallel execution with branches of varying lengths converging to a single aggregation point.
- **[Configuration](configuration/README.md)**: Using runtime configuration to pass metadata and settings.
- **[Custom Reducer](custom_reducer/README.md)**: Defining custom state reducers for complex merge logic.
- **[State Schema](state_schema/README.md)**: Managing complex state updates with Schema and Reducers.
- **[Subgraphs](subgraph/README.md)**: Composing graphs within graphs.
- **[Multiple Subgraphs](subgraphs/)**: Managing multiple subgraph compositions.
- **[Streaming Modes](streaming_modes/README.md)**: Advanced streaming with updates, values, and messages modes.
- **[Smart Messages](smart_messages/README.md)**: Intelligent message merging with ID-based upserts.
- **[Command API](command_api/README.md)**: Dynamic control flow and state updates from nodes.
- **[Listeners](listeners/README.md)**: Attaching event listeners to the graph.

## Persistence (Checkpointing)
- **[Memory](memory_basic/main.go)**: In-memory checkpointing.
- **[PostgreSQL](checkpointing/postgres/)**: Persistent state using PostgreSQL.
- **[SQLite](checkpointing/sqlite/)**: Persistent state using SQLite.
- **[Redis](checkpointing/redis/)**: Persistent state using Redis.
- **[Durable Execution](durable_execution/README.md)**: Crash recovery and resuming execution from checkpoints.

## Human-in-the-loop
- **[Human Approval](human_in_the_loop/README.md)**: Workflow with interrupts and human approval steps.
- **[Time Travel / HITL](time_travel/README.md)**: Inspecting, modifying state history, and forking execution (UpdateState).
- **[Dynamic Interrupt](dynamic_interrupt/README.md)**: Pausing execution from within a node using `graph.Interrupt`.

## Pre-built Agents
- **[Create Agent](create_agent/README.md)**: Easy way to create an agent with options.
- **[Dynamic Skill Agent](dynamic_skill_agent/README.md)**: Agent with dynamic skill discovery and selection.
- **[ReAct Agent](react_agent/README.md)**: Reason and Action agent using tools.
- **[Planning Agent](planning_agent/README.md)**: Intelligent agent that dynamically creates workflow plans based on user requests.
- **[PEV Agent](pev_agent/README.md)**: Plan-Execute-Verify agent with self-correction and error recovery for reliable task execution.
- **[Reflection Agent](reflection_agent/README.md)**: Iterative improvement agent that refines responses through self-reflection.
- **[Tree of Thoughts](tree_of_thoughts/README.md)**: Search-based reasoning agent that explores multiple solution paths through a tree structure.
- **[Mental Loop](mental_loop/README.md)**: Simulator-in-the-Loop agent that tests actions in a sandbox before real-world execution (think before you act).
- **[Reflexive Metacognitive Agent](reflexive_metacognitive/README.md)**: Self-aware agent with explicit self-model of capabilities and limitations, performs metacognitive analysis before answering (knows what it doesn't know).
- **[Supervisor](supervisor/README.md)**: Multi-agent orchestration using a supervisor.
- **[Swarm](swarm/README.md)**: Multi-agent collaboration using handoffs.
- **[Chat Agent](chat_agent/README.md)**: Multi-turn conversation agent with automatic session management.
- **[Chat Agent Async](chat_agent_async/README.md)**: Asynchronous streaming chat agent with real-time LLM response streaming.
- **[Chat Agent Dynamic Tools](chat_agent_dynamic_tools/README.md)**: Chat agent with runtime tool management capabilities.

## Programmatic Tool Calling (PTC)
- **[PTC Basic](ptc_basic/README.md)**: Introduction to Programmatic Tool Calling for reduced latency and token efficiency.
- **[PTC Simple](ptc_simple/)**: Simple PTC example with calculator and weather tools.
- **[PTC Expense Analysis](ptc_expense_analysis/)**: Complex expense analysis scenario based on Anthropic's PTC Cookbook.
- **[PTC + GoSkills](ptc_goskills/README.md)**: Integration of PTC with GoSkills for local tool execution.

## Memory
- **[Memory Basic](memory_basic/README.md)**: Basic usage of LangChain memory adapters.
- **[Memory Chatbot](memory_chatbot/README.md)**: Chatbot with LangChain memory integration.
- **[Memory Strategies](memory_strategies/README.md)**: Comprehensive guide to all 9 memory management strategies.
- **[Memory Agent](memory_agent/README.md)**: Real-world agents using different memory strategies for context management.
- **[Memory + Graph Integration](memory_graph_integration/README.md)**: State-based memory integration in LangGraph workflows.

## RAG (Retrieval Augmented Generation)
- **[RAG Basic](rag_basic/README.md)**: Basic RAG implementation.
- **[RAG Pipeline](rag_pipeline/README.md)**: Complete RAG pipeline.
- **[RAG Advanced](rag_advanced/README.md)**: Advanced RAG techniques.
- **[RAG Conditional](rag_conditional/README.md)**: Conditional RAG workflow.
- **[RAG with Embeddings](rag_with_embeddings/README.md)**: RAG using embeddings.
- **[RAG with LangChain](rag_with_langchain/README.md)**: RAG using LangChain components.
- **[RAG with VectorStores](rag_langchain_vectorstore_example/README.md)**: RAG using LangChain VectorStores.
- **[RAG with Chroma](rag_chroma_example/README.md)**: RAG using Chroma database.
- **[RAG Query Rewrite](rag_query_rewrite/README.md)**: RAG with query rewriting for better retrieval.
- **[RAG with FalkorDB Graph](rag_falkordb_graph/README.md)**: RAG using FalkorDB knowledge graph with automatic entity extraction.
- **[RAG with FalkorDB Simple](rag_falkordb_simple/README.md)**: Simple RAG with FalkorDB using manual entity/relationship creation.
- **[RAG with FalkorDB Fast](rag_falkordb_fast/README.md)**: Optimized RAG with FalkorDB for fast queries.
- **[RAG with FalkorDB Debug](rag_falkordb_debug/README.md)**: Debug version of FalkorDB RAG with detailed logging.
- **[RAG with FalkorDB Debug Query](rag_falkordb_debug_query/README.md)**: Query debugging for FalkorDB RAG.

## Other
- **[Visualization](visualization/README.md)**: Generating Mermaid diagrams for graphs.
- **[LangChain Integration](langchain_example/README.md)**: Using LangChain tools and models.
- **[Tavily Search](tool_tavily/README.md)**: Using Tavily search tool with ReAct agent.
- **[Exa Search](tool_exa/README.md)**: Using Exa search tool with ReAct agent.
- **[Brave Search](tool_brave/README.md)**: Using Brave search API with agents.
- **[GoSkills Integration](goskills_example/README.md)**: Integrating GoSkills as tools for agents.
- **[MCP Agent](mcp_agent/README.md)**: Using Model Context Protocol (MCP) tools with agents.
- **[Context Store](context_store/README.md)**: Managing context with external stores.
- **[Streaming Pipeline](streaming_pipeline/README.md)**: Building streaming data processing pipelines.
- **[Generic State Graph](generic_state_graph/)**: Using generic types for type-safe state management.
- **[Generic State Graph Listenable](generic_state_graph_listenable/)**: Generic state graph with event listening capabilities.
- **[Generic State Graph ReAct Agent](generic_state_graph_react_agent/)**: ReAct agent implementation using generic types.
- **[File Checkpointing](file_checkpointing/)**: Checkpointing to file system.
- **[File Checkpointing Resume](file_checkpointing_resume/)**: Resuming execution from file checkpoints.

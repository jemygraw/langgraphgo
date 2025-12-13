# Checkpointing Examples

This directory contains examples demonstrating how to use **Checkpointing** in LangGraphGo to persist and manage graph state.

## 1. Background

In complex, long-running, or critical applications, keeping state solely in memory is risky and limiting. Checkpointing solves several key problems:
- **Fault Tolerance**: If the application crashes, you can resume from the last saved state.
- **Human-in-the-loop**: You can pause execution, wait for days for human input, and then resume.
- **Time Travel**: You can inspect past states ("what happened at step 3?") or even fork execution from a previous point.

## 2. Key Concepts

- **CheckpointSaver**: An interface for saving and loading graph state. LangGraphGo provides implementations for:
  - **Memory**: Ephemeral, good for testing.
  - **PostgreSQL**: Robust, production-grade persistence.
  - **SQLite**: Lightweight, file-based persistence.
  - **Redis**: Fast, in-memory persistence.
- **ThreadID**: A unique identifier for a conversation or execution thread. Checkpoints are isolated by ThreadID.
- **CheckpointConfig**: Configuration options like `AutoSave`, `SaveInterval`, and `MaxCheckpoints`.

## 3. Examples

### [In-Memory (main.go)](./main.go)
Demonstrates the basic API using an in-memory store. Good for understanding the concepts without setting up a database.

### [PostgreSQL (postgres/)](./postgres/)
Shows how to use a PostgreSQL database for durable state storage. Requires a running Postgres instance.

### [SQLite (sqlite/)](./sqlite/)
Shows how to use a local SQLite file. Ideal for desktop apps or simple deployments.

### [Redis (redis/)](./redis/)
Shows how to use Redis for high-performance state storage.

## 4. How to Use

To run the PostgreSQL example:

```bash
export POSTGRES_CONN_STRING="postgres://user:password@localhost:5432/dbname"
cd postgres
go run main.go
```

To run the SQLite example:

```bash
cd sqlite
go run main.go
```
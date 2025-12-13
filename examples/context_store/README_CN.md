# Checkpointing 示例

本目录包含演示如何在 LangGraphGo 中使用 **Checkpointing (检查点)** 来持久化和管理图状态的示例。

## 1. 背景

在复杂、长时间运行或关键的应用中，仅将状态保存在内存中是有风险且受限的。Checkpointing 解决了几个关键问题：
- **容错 (Fault Tolerance)**: 如果应用程序崩溃，您可以从上次保存的状态恢复。
- **人在回路 (Human-in-the-loop)**: 您可以暂停执行，等待数天的人工输入，然后恢复。
- **时间旅行 (Time Travel)**: 您可以检查过去的状态（“第 3 步发生了什么？”），甚至从之前的点分叉执行。

## 2. 核心概念

- **CheckpointSaver**: 用于保存和加载图状态的接口。LangGraphGo 提供了以下实现：
  - **Memory**: 临时的，适合测试。
  - **PostgreSQL**: 健壮的，生产级持久化。
  - **SQLite**: 轻量级的，基于文件的持久化。
  - **Redis**: 快速的，内存中持久化。
- **ThreadID**: 对话或执行线程的唯一标识符。Checkpoints 是按 ThreadID 隔离的。
- **CheckpointConfig**: 配置选项，如 `AutoSave` (自动保存), `SaveInterval` (保存间隔), 和 `MaxCheckpoints` (最大检查点数)。

## 3. 示例

### [内存 (main.go)](./main.go)
演示使用内存存储的基本 API。适合在不设置数据库的情况下理解概念。

### [PostgreSQL (postgres/)](./postgres/)
展示如何使用 PostgreSQL 数据库进行持久化状态存储。需要运行 Postgres 实例。

### [SQLite (sqlite/)](./sqlite/)
展示如何使用本地 SQLite 文件。非常适合桌面应用或简单部署。

### [Redis (redis/)](./redis/)
展示如何使用 Redis 进行高性能状态存储。

## 4. 如何使用

运行 PostgreSQL 示例：

```bash
export POSTGRES_CONN_STRING="postgres://user:password@localhost:5432/dbname"
cd postgres
go run main.go
```

运行 SQLite 示例：

```bash
cd sqlite
go run main.go
```

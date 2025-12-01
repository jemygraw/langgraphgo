# 更新日志

## [0.2.0] - 2025-12-01

### 核心运行时 (Core Runtime)
- **并行执行**: 实现了扇出/扇入 (Fan-out/Fan-in) 执行模型，支持线程安全的状态合并。
- **运行时配置**: 添加了 `RunnableConfig`，用于在图执行上下文中传递配置（如线程 ID、用户 ID 等）。

### 持久化与检查点 (Persistence & Checkpointing)
- **检查点接口**: 优化了 `CheckpointSaver` 接口以支持状态持久化。
- **实现**: 增加了对 **Redis**、**PostgreSQL** 和 **SQLite** 检查点存储的完整支持。

### 高级状态与流式处理 (Advanced State & Streaming)
- **状态管理**: 引入了 `Schema` 接口和 `Annotated` 风格的归约器 (Reducers)（例如 `AppendMessages`），用于复杂的状态更新。
- **增强流式处理**: 添加了类型化的 `StreamEvent` 和 `CallbackHandler` 接口，用于细粒度的执行监控。

### 预构建代理 (Pre-built Agents)
- **ToolExecutor**: 添加了用于执行工具的专用节点。
- **ReAct Agent**: 实现了用于创建 ReAct 风格代理的工厂方法。
- **Supervisor**: 添加了对 Supervisor 代理模式的支持，用于多代理编排。

### 人机交互 (Human-in-the-loop, HITL)
- **中断 (Interrupts)**: 实现了 `InterruptBefore` 和 `InterruptAfter` 机制以暂停图的执行。
- **恢复与命令 (Resume & Command)**: 添加了通过命令恢复执行和更新状态的支持，从而实现人工审批工作流。

### 可视化 (Visualization)
- **Mermaid 导出**: 改进了图的可视化，优化了条件边和样式的渲染。

### 实验性与研究 (Experimental & Research)
- **Swarm 模式**: 使用子图 (`examples/swarm`) 添加了多代理协作的原型。
- **Channels RFC**: 添加了 `RFC_CHANNELS.md`，提议在未来改进中采用基于 Channel 的架构。

### 示例 (Examples)
- 添加了涵盖以下内容的综合示例：
  - 检查点 (Postgres, SQLite, Redis)
  - 人机交互工作流
  - Swarm 多代理模式
  - 子图

## [0.1.0] - 2025-01-02

### 新增
- 通用状态管理 - 适用于任何类型，不仅仅是 MessageContent
- 针对生产环境的性能优化
- 支持任何 LLM 客户端（移除了对 LangChain 的硬依赖）

### 变更
- 简化了构建图的 API
- 更新了示例以展示通用用法

### 修复
- 原始仓库中的 CI/CD 流水线问题
- 最新 Go 版本的构建错误

### 移除
- 对 LangChain 的硬依赖 - 现在可以与任何 LLM 库一起工作

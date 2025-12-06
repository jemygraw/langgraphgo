# Memory + LangGraph 集成

本示例展示如何将内存策略直接集成到 LangGraph 工作流的 State 中，创建具有复杂内存管理的有状态对话代理。

## 本示例展示的内容

与之前独立展示内存策略或简单代理的示例不同，本示例演示：

1. **State 中的 Memory** - 如何将内存策略作为工作流状态的一部分
2. **多节点内存访问** - 图中的不同节点如何访问和更新共享内存
3. **有状态工作流** - 构建跨多轮对话维护上下文的工作流
4. **真实集成** - 实际的 LangGraph + Memory 在类生产场景中协同工作

## 架构

### State 设计

```go
type ConversationState struct {
    UserInput      string              // 当前用户消息
    Intent         string              // 分类的意图
    Context        []*memory.Message   // 从内存检索的上下文
    Response       string              // 生成的响应
    Memory         memory.Strategy     // 内存策略（存在于 state 中！）
    ConversationID string              // 对话标识符
    TurnCount      int                 // 轮次数
}
```

**关键见解**：内存策略存在于 State 中，使工作流中的所有节点都可以访问它。

### 工作流结构

```
入口点
    ↓
[分类意图] ──→ 分析用户输入，从内存检索上下文
    ↓
[检索信息] ──→ 根据意图获取相关信息
    ↓
[生成响应] ──→ 创建响应，将消息添加到内存
    ↓
结束点
```

每个节点：
- 从 State 中的共享内存策略读取
- 检索与其任务相关的上下文
- 用新信息更新内存
- 将状态传递给下一个节点

## 运行示例

```bash
cd examples/memory_graph_integration
go run main.go
```

## 演示

### 演示 1：滑动窗口内存

**策略**：保留最后 4 条消息
**使用场景**：最近的上下文最重要

```
用户：你好！
代理：你好！我是您的产品助手...

用户：价格是多少？
代理：我们的高级产品售价 99 美元...

用户：告诉我功能
代理：我们的产品有惊人的功能...

用户：再提醒我价格？
代理：如我之前提到的，产品售价 99 美元
```

**工作原理**：
- 内存存储最后 4 条消息（2 条用户 + 2 条助手）
- 再次询问价格时，在上下文中找到"$99"
- 代理以"如我之前提到的..."回应
- 4 条新消息后，旧的价格信息将被遗忘

### 演示 2：分层内存

**策略**：保留重要 + 最近的消息
**使用场景**：某些信息是关键的

```
用户：你好，我叫 Alice
代理：很高兴认识您，Alice！我会记住您的名字。

[... 几条消息后 ...]

用户：你还记得我的名字吗？
代理：当然！我记得您，Alice...
```

**工作原理**：
- 重要消息（姓名、需求）标记为高重要性
- 无论如何都保留最近的消息
- 姓名存储在"重要"层，在多轮对话中幸存
- 即使经过许多消息，代理仍能回忆 Alice 的名字

### 演示 3：检索式内存

**策略**：查找相关消息
**使用场景**：大型对话，查询驱动

```
用户：价格是多少？
代理：我们的高级产品售价 99 美元...

[... 讨论了 6 个其他主题 ...]

用户：再谈谈价格
代理：如我之前提到的，产品售价 99 美元
```

**工作原理**：
- 所有消息都存储嵌入
- 查询"价格"检索包含定价信息的消息
- 即使经过许多不相关的消息，也能找到价格信息
- 使用语义相似度，而不仅仅是最近的消息

### 演示 4：图式内存

**策略**：跟踪主题关系
**使用场景**：相关主题和连接

```
用户：价格是多少？
[跟踪的主题：[price]]

用户：告诉我保修
[跟踪的主题：[price, warranty]]

用户：价格包括保修吗？
[跟踪的主题：[price, warranty]]
```

**工作原理**：
- 基于共享主题（价格、保修等）连接消息
- 当查询提到"价格"时，检索与价格相关的消息
- 还检索与价格相关的消息（保修、功能）
- 图结构捕获主题关系

## 集成模式

### 1. 创建内存策略

```go
mem := memory.NewSlidingWindowMemory(5)
```

### 2. 创建包含内存的工作流

```go
stateSchema := graph.StateSchema{
    "user_input": "",
    "memory":     mem,  // State 中的内存策略
    "response":   "",
}

workflow := graph.NewGraph(stateSchema)
```

### 3. 在节点中访问内存

```go
func processNode(state graph.State) (graph.State, error) {
    mem := state["memory"].(memory.Strategy)
    userInput := state["user_input"].(string)

    // 从内存获取上下文
    context, _ := mem.GetContext(ctx, userInput)

    // 使用上下文进行处理
    response := generateResponse(userInput, context)

    // 将新消息添加到内存
    mem.AddMessage(ctx, memory.NewMessage("user", userInput))
    mem.AddMessage(ctx, memory.NewMessage("assistant", response))

    state["response"] = response
    return state, nil
}
```

### 4. 调用工作流

```go
state := graph.State{
    "user_input": "你好！",
    "memory":     mem,
}

result, _ := workflow.Invoke(state, nil)
response := result["response"].(string)
```

## 关键优势

### 1. 关注点分离

- **内存策略**：处理上下文存储和检索
- **工作流节点**：专注于业务逻辑
- **State**：将所有内容连接在一起

### 2. 灵活性

- 无需更改工作流逻辑即可交换内存策略
- 不同的工作流可以使用不同的策略
- 易于使用不同的内存配置进行测试

### 3. 可扩展性

- 内存策略处理不断增长的对话历史
- 工作流保持简单和可维护
- 可以添加新节点而无需触及内存逻辑

### 4. 可重用性

- 同一工作流适用于任何内存策略
- 内存策略可跨工作流重用
- State 模式允许复杂的组合

## 比较：内存集成方法

### 方法 1：简单代理（memory_agent 示例）
```go
agent.ProcessMessage(userInput)
```
- ✓ 易于使用
- ✗ 单一处理路径
- ✗ 有限的可组合性

### 方法 2：图集成（本示例）
```go
workflow.Invoke(state, nil)
```
- ✓ 多节点工作流
- ✓ 条件路由
- ✓ 复杂的代理行为
- ✓ 更好的测试和调试

## 高级模式

### 基于上下文的条件路由

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

### 多个内存策略

```go
stateSchema := graph.StateSchema{
    "short_term": memory.NewSlidingWindowMemory(5),
    "long_term":  memory.NewRetrievalMemory(...),
}

// 使用 short_term 获取最近的上下文
// 使用 long_term 进行语义搜索
```

### 内存持久化

```go
// 保存内存状态
memData := mem.Export()
saveToDatabase(conversationID, memData)

// 恢复内存状态
memData := loadFromDatabase(conversationID)
mem.Import(memData)
```

## 何时使用此模式

**使用 Graph + Memory 集成当：**
- 需要多步处理（意图 → 检索 → 响应）
- 基于对话上下文的条件逻辑
- 具有共享上下文的多个专门节点
- 超越简单聊天的复杂代理行为
- 需要测试和调试单个组件

**使用简单代理当：**
- 单步请求/响应
- 直接的对话流程
- 快速原型设计
- 简单的聊天机器人用例

## 性能考虑

### 内存策略影响

| 策略 | 节点访问速度 | 内存增长 | 最适合 |
|------|-------------|---------|--------|
| 顺序式 | O(n) | 无限制 | 短对话 |
| 滑动窗口 | O(1) | 固定 | 实时聊天 |
| 检索式 | O(log n) | 线性 | 长对话 |
| 分层式 | O(n) | 有界 | 混合重要性 |
| 图式 | O(k) | 线性 | 相关主题 |

### 优化技巧

1. **选择正确的策略**：将策略与对话长度匹配
2. **限制上下文大小**：不要将整个历史传递给每个节点
3. **缓存统计**：避免重复的 GetStats() 调用
4. **批量更新**：如果可能，一次添加多条消息

## 相关示例

- [memory_strategies](../memory_strategies/) - 各个策略演示
- [memory_agent](../memory_agent/) - 带内存的简单代理
- [basic_llm](../basic_llm/) - 基本 LangGraph 使用

## 扩展阅读

- [Memory Package Documentation](../../memory/README.md)
- [内存包文档（中文）](../../memory/README_CN.md)
- [LangGraph 文档](../../graph/README.md)

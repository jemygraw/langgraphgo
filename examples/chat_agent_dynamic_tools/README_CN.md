# ChatAgent 动态工具示例

本示例演示了 LangGraphGo 中 `ChatAgent` 的动态工具管理功能，允许您在对话过程中添加、删除和更新工具。

## 特性

- **动态工具添加**：在对话的任何时刻向 agent 添加新工具
- **工具移除**：按名称移除不再需要的特定工具
- **工具替换**：一次性用新的工具集替换所有工具
- **工具查询**：获取当前可用工具的列表
- **工具清除**：立即移除所有动态工具

## 为什么需要动态工具？

动态工具管理支持多种强大的用例：

1. **上下文感知能力**：根据用户请求或对话上下文添加工具
2. **资源管理**：在不需要时移除昂贵的工具
3. **渐进式增强**：从基本工具开始，根据需要添加高级工具
4. **访问控制**：基于身份验证或权限授予或撤销工具访问权限
5. **A/B 测试**：在不同的工具实现之间切换

## API 概览

```go
// 添加单个工具
agent.AddTool(weatherTool)

// 按名称移除工具
removed := agent.RemoveTool("calculator")

// 获取当前工具
currentTools := agent.GetTools()

// 替换所有工具
agent.SetTools([]tools.Tool{tool1, tool2})

// 清除所有动态工具
agent.ClearTools()
```

## 工作原理

`ChatAgent` 维护两组工具：

1. **基础工具**：创建 agent 时提供（通过 `NewChatAgent`）
2. **动态工具**：在运行时使用管理方法添加/移除

在处理消息时，两组工具都可供 agent 使用。动态工具通过 `extra_tools` 机制传递给 agent graph。

## 运行示例

```bash
cd examples/chat_agent_dynamic_tools
go run main.go
```

## 示例输出

该示例演示了：

1. **第 1 轮**：没有工具的聊天
2. **第 2 轮**：添加计算器工具并执行计算
3. **第 3 轮**：在保留计算器的同时添加天气工具
4. **第 4 轮**：移除计算器，保留天气工具

## 重要说明

- **工具唯一性**：添加与现有工具同名的工具将替换它
- **基础工具**：动态工具方法不会影响创建时提供的基础工具
- **线程安全**：对于并发使用，您应该添加自己的同步机制
- **持久性**：动态工具在内存中；它们不会在应用程序重启后持久化

## 实际应用场景

### 1. 条件工具访问

```go
// 仅在身份验证后授予文件访问权限
if user.IsAuthenticated() {
    agent.AddTool(fileReadTool)
    agent.AddTool(fileWriteTool)
}
```

### 2. 基于上下文的工具

```go
// 当用户提到数据查询时添加数据库工具
if containsDataKeywords(message) {
    agent.AddTool(sqlQueryTool)
}
```

### 3. 成本管理

```go
// 配额超出后移除昂贵的 API 工具
if quotaExceeded() {
    agent.RemoveTool("expensive_api")
}
```

### 4. 功能标志

```go
// 基于功能标志启用实验性工具
if featureFlags["experimental_tools"] {
    agent.AddTool(experimentalTool)
}
```

## 与真实 LLM 集成

与 OpenAI 等真实 LLM 一起使用时：

```go
import "github.com/tmc/langchaingo/llms/openai"

// 创建模型
model, _ := openai.New(openai.WithModel("gpt-4"))

// 使用基础工具创建 agent
agent, _ := prebuilt.NewChatAgent(model, baseTtools)

// 根据对话动态添加工具
if userNeedsWeather {
    agent.AddTool(weatherTool)
}

response, _ := agent.Chat(ctx, "What's the weather?")
```

## 另请参阅

- [基础 ChatAgent 示例](../chat_agent/) - 简单的多轮对话
- [CreateAgent 文档](../../prebuilt/README.md) - 底层 agent 创建
- [工具执行器](../../prebuilt/tool_executor.go) - 工具执行机制

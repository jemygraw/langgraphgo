# ChatAgent 示例

本示例演示了 LangGraphGo 中的 `ChatAgent` 功能，该功能支持带自动会话管理的多轮对话。

## 特性

- **会话管理**：每个 `ChatAgent` 实例维护自己的对话会话，并具有唯一的线程 ID
- **对话历史**：消息在多轮对话中自动累积
- **简单的 API**：易于使用的 `Chat()` 方法用于发送消息和接收响应
- **记忆**：Agent 可以引用对话中之前的消息

## 工作原理

`ChatAgent` 包装了使用 `CreateAgent` 创建的 agent graph 并管理对话状态：

1. 当您创建一个 `ChatAgent` 时，它会生成一个唯一的会话 ID（线程 ID）
2. 每次调用 `Chat()` 都会将用户的消息追加到对话历史中
3. 完整的对话历史被传递给底层的 agent
4. agent 的响应被添加到历史记录中并返回
5. 后续调用将在完整的上下文中继续对话

## 用法

```go
// 创建一个 ChatAgent
agent, err := prebuilt.NewChatAgent(model, tools)
if err != nil {
    log.Fatal(err)
}

// 第一轮
response1, err := agent.Chat(ctx, "Hello! My name is Alice.")

// 第二轮 - agent 记住了之前的上下文
response2, err := agent.Chat(ctx, "What's my name?")
// 响应: "Your name is Alice, as you told me earlier."

// 获取会话 ID
sessionID := agent.ThreadID()
```

## 运行示例

```bash
cd examples/chat_agent
go run main.go
```

## 预期输出

该示例演示了一个多轮对话，其中：
1. 用户向 agent 问好
2. 用户介绍他们的名字
3. Agent 从历史记录中回忆起名字
4. 对话在完整的上下文中继续

## 注意事项

- 本示例为演示目的使用了简单的模拟模型
- 在生产环境中，您会使用真实的 LLM，如 OpenAI 的 GPT-4
- 对话历史在 `ChatAgent` 实例的生命周期内维护在内存中
- 每个 `ChatAgent` 实例代表一个单独的对话会话

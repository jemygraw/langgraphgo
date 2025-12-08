# ChatAgent 异步流式传输示例

本示例演示了 LangGraphGo 中 `ChatAgent` 的异步流式传输功能，允许您实时接收正在生成的响应。

## 特性

- **逐字符流式传输**：`AsyncChat` 提供逐字符流式传输，实现打字机效果
- **逐词流式传输**：`AsyncChatWithChunks` 提供逐词流式传输，提高可读性
- **上下文支持**：完全支持上下文取消和超时
- **非阻塞**：立即返回一个用于接收结果的通道
- **易于集成**：简单的基于通道的 API，符合 Go 惯用语

## 为什么需要异步流式传输？

传统的聊天界面会等待完整的响应生成后再显示给用户。异步流式传输提供了以下好处：

1. **更好的用户体验**：用户可以看到响应实时出现
2. **感知性能**：即使实际处理时间相同，也会感觉更快
3. **自然交互**：模仿人类对话，响应逐渐出现
4. **早期反馈**：用户可以在完整响应完成之前开始阅读
5. **可中断**：可以取消不再相关的长响应

## API 概览

### AsyncChat - 字符流式传输

```go
respChan, err := agent.AsyncChat(ctx, "Hello!")
if err != nil {
    log.Fatal(err)
}

for char := range respChan {
    fmt.Print(char)  // 打印到达的每个字符
}
```

### AsyncChatWithChunks - 单词流式传输

```go
respChan, err := agent.AsyncChatWithChunks(ctx, "Explain AI")
if err != nil {
    log.Fatal(err)
}

for word := range respChan {
    fmt.Print(word)  // 打印到达的每个单词
}
```

## 工作原理

1. **调用方法**：使用您的消息调用 `AsyncChat` 或 `AsyncChatWithChunks`
2. **获取通道**：接收一个字符串只读通道
3. **开始读取**：立即开始从通道读取
4. **接收块**：字符或单词在处理时到达
5. **通道关闭**：当响应完成时，通道自动关闭

幕后机制：
- 启动一个 goroutine 来处理聊天
- 使用标准的 `Chat` 方法生成完整的响应
- 响应被拆分为块（字符或单词）
- 块通过通道逐个发送
- 当所有块发送完毕后，通道关闭

## 运行示例

```bash
cd examples/chat_agent_async
go run main.go
```

## 示例输出

```
=== ChatAgent AsyncChat Demo ===

--- Demo 1: Character-by-Character Streaming ---
User: Hello!
Agent: H e l l o !   I ' m   a n   A I   a s s i s t a n t . . .

--- Demo 2: Word-by-Word Streaming ---
User: Can you explain async chat?
Agent: Of course! I can explain async chat...

--- Demo 3: Collecting Full Response ---
User: What's the benefit of streaming?
Agent: The main benefit is improved perceived performance...
[Received 45 chunks, total length: 234 characters]
```

## 高级用法

### 上下文取消

```go
// 创建一个带超时的 context
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

respChan, err := agent.AsyncChat(ctx, message)
for chunk := range respChan {
    // 当 context 取消时，流将停止
    fmt.Print(chunk)
}
```

### 收集完整响应

```go
var fullResponse string
for chunk := range respChan {
    fullResponse += chunk
}
// fullResponse 现在包含完整的文本
```

### 添加延迟效果

```go
for chunk := range respChan {
    fmt.Print(chunk)
    time.Sleep(50 * time.Millisecond)  // 模拟打字效果
}
```

## 与常规 Chat 对比

| 特性       | 常规 Chat | AsyncChat | AsyncChatWithChunks |
| ---------- | --------- | --------- | ------------------- |
| 返回       | 完整响应  | 通道      | 通道                |
| 流式传输   | 否        | 字符级    | 单词级              |
| 阻塞       | 是        | 否        | 否                  |
| 上下文支持 | 是        | 是        | 是                  |
| 用例       | 简单请求  | 打字效果  | 可读流式传输        |

## 与真实 LLM 集成

当与支持流式传输的真实 LLM（如 OpenAI）一起使用时：

```go
import "github.com/tmc/langchaingo/llms/openai"

model, _ := openai.New(openai.WithModel("gpt-4"))
agent, _ := prebuilt.NewChatAgent(model, nil)

// 异步流式传输适用于任何 LLM
respChan, _ := agent.AsyncChatWithChunks(ctx, "Explain quantum computing")
for word := range respChan {
    fmt.Print(word)
}
```

## 注意事项

- **Goroutines**：每个异步调用都会产生一个自动清理的 goroutine
- **内存**：缓冲通道（容量 100）防止在慢速消费者上阻塞
- **错误**：处理过程中的错误会导致通道提前关闭
- **线程安全**：可以安全地从多个 goroutine 调用
- **对话历史**：历史记录正常维护，就像常规 `Chat` 一样

## 最佳实践

1. **始终排空通道**：读取直到关闭，以防止 goroutine 泄漏
2. **使用上下文超时**：防止在慢速响应上无限等待
3. **选择合适的方法**：
   - `AsyncChat` 用于逐字符打字效果
   - `AsyncChatWithChunks` 用于更自然的逐词流式传输
4. **处理上下文取消**：在使用可取消上下文时检查已关闭的通道

## 另请参阅

- [基础 ChatAgent 示例](../chat_agent/) - 简单的多轮对话
- [动态工具示例](../chat_agent_dynamic_tools/) - 运行时工具管理
- [ChatAgent 文档](../../prebuilt/CHAT_AGENT.md) - 完整 API 参考

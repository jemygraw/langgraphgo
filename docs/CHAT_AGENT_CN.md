# ChatAgent - 多轮对话支持

`ChatAgent` 提供了一个高级 API，用于构建具有自动会话管理和对话历史记录的对话代理。

## 功能特性

- **自动会话管理**：每个 ChatAgent 实例都有一个唯一的会话 ID（线程 ID）
- **对话历史记录**：消息自动累积并传递给代理
- **简单的 API**：易于使用的 `Chat()` 方法实现多轮对话
- **记忆功能**：代理在多个对话轮次中维护完整的对话上下文
- **动态工具**：在对话过程中运行时添加、移除或更新工具
- **异步流式输出**：使用 `AsyncChat()` 和 `AsyncChatWithChunks()` 实时流式传输响应

## 基本用法

```go
import (
    "context"
    "github.com/smallnest/langgraphgo/prebuilt"
    "github.com/tmc/langchaingo/llms/openai"
)

// 创建 LLM 模型
model, err := openai.New(openai.WithModel("gpt-4"))
if err != nil {
    log.Fatal(err)
}

// 创建带有可选工具的 ChatAgent
agent, err := prebuilt.NewChatAgent(model, tools)
if err != nil {
    log.Fatal(err)
}

ctx := context.Background()

// 进行多轮对话
response1, err := agent.Chat(ctx, "你好！我叫 Alice。")
response2, err := agent.Chat(ctx, "我叫什么名字？")
// 代理会记住之前消息中的名字

// 获取会话 ID
sessionID := agent.ThreadID()
```

## API 接口

### NewChatAgent

```go
func NewChatAgent(model llms.Model, inputTools []tools.Tool, opts ...CreateAgentOption) (*ChatAgent, error)
```

使用指定的模型和工具创建新的 ChatAgent。

**参数：**
- `model`：要使用的 LLM 模型（例如 OpenAI GPT-4）
- `inputTools`：代理可以使用的可选工具切片
- `opts`：可选配置（系统消息、状态修改器等）

**返回值：**
- `*ChatAgent`：新的 ChatAgent 实例
- `error`：创建过程中发生的任何错误

### Chat

```go
func (c *ChatAgent) Chat(ctx context.Context, message string) (string, error)
```

向代理发送消息并返回响应。对话历史记录会自动维护。

**参数：**
- `ctx`：操作的上下文
- `message`：用户的消息

**返回值：**
- `string`：代理的响应
- `error`：发生的任何错误

### ThreadID

```go
func (c *ChatAgent) ThreadID() string
```

返回此对话的唯一会话 ID。

### PrintStream

```go
func (c *ChatAgent) PrintStream(ctx context.Context, message string, w io.Writer) error
```

发送消息并将响应打印到提供的写入器。

### AsyncChat

```go
func (c *ChatAgent) AsyncChat(ctx context.Context, message string) (<-chan string, error)
```

向代理发送消息并返回一个通道，用于**真正的流式**传输响应。此方法使用 LLM 的原生流式 API（通过 `llms.WithStreamingFunc`）实时发送模型生成的块，而不是在完整响应准备好后才发送。

**主要特性：**
- **实时流式传输**：块在 LLM 生成时立即到达
- **原生 LLM 支持**：使用底层模型的流式传输能力
- **低延迟**：第一个令牌立即出现
- **高效**：在发送前不缓冲完整响应

**参数：**
- `ctx`：操作的上下文（支持取消）
- `message`：用户的消息

**返回值：**
- `<-chan string`：实时发送响应块的只读通道
- `error`：初始化期间发生的任何错误

**示例：**
```go
respChan, err := agent.AsyncChat(ctx, "解释量子计算")
if err != nil {
    log.Fatal(err)
}

// 块在 LLM 生成时到达
for chunk := range respChan {
    fmt.Print(chunk)  // 立即打印每个块
}
```

**注意：** 通道接收 LLM 生成的块。块大小取决于模型的流式实现（通常是词或子词标记）。

### AsyncChatWithChunks

```go
func (c *ChatAgent) AsyncChatWithChunks(ctx context.Context, message string) (<-chan string, error)
```

向代理发送消息并返回一个通道，用于逐词流式传输响应。与 `AsyncChat` 不同，此方法以词大小的块进行流式传输，以提高可读性，同时仍提供流式效果。

**参数：**
- `ctx`：操作的上下文（支持取消）
- `message`：用户的消息

**返回值：**
- `<-chan string`：发送词和空格的只读通道
- `error`：初始化期间发生的任何错误

**示例：**
```go
respChan, err := agent.AsyncChatWithChunks(ctx, "解释人工智能")
if err != nil {
    log.Fatal(err)
}

var fullResponse string
for word := range respChan {
    fmt.Print(word)
    fullResponse += word
}
```

## 动态工具管理

ChatAgent 支持在对话过程中动态添加和移除工具。这使得一些强大的用例成为可能，如上下文感知能力、资源管理和渐进式增强。

### SetTools

```go
func (c *ChatAgent) SetTools(newTools []tools.Tool)
```

用提供的工具替换所有动态工具。这不会影响在创建代理时提供的基础工具。

**示例：**
```go
agent.SetTools([]tools.Tool{weatherTool, calculatorTool})
```

### AddTool

```go
func (c *ChatAgent) AddTool(tool tools.Tool)
```

向动态工具列表添加新工具。如果已存在同名工具，它将被替换。

**示例：**
```go
weatherTool := &WeatherTool{}
agent.AddTool(weatherTool)
```

### RemoveTool

```go
func (c *ChatAgent) RemoveTool(toolName string) bool
```

按名称从动态工具列表中移除工具。如果找到并移除了工具，则返回 `true`，否则返回 `false`。

**示例：**
```go
removed := agent.RemoveTool("calculator")
if removed {
    fmt.Println("计算器工具已移除")
}
```

### GetTools

```go
func (c *ChatAgent) GetTools() []tools.Tool
```

返回当前动态工具列表的副本。这不包括在创建代理时提供的基础工具。

**示例：**
```go
tools := agent.GetTools()
fmt.Printf("代理有 %d 个动态工具\n", len(tools))
```

### ClearTools

```go
func (c *ChatAgent) ClearTools()
```

移除所有动态工具。

**示例：**
```go
agent.ClearTools()
```

## 动态工具使用场景

### 1. 上下文感知能力

根据对话上下文添加工具：

```go
// 当用户提到数学时添加计算器
if strings.Contains(message, "计算") {
    agent.AddTool(calculatorTool)
}

response, _ := agent.Chat(ctx, message)
```

### 2. 渐进式增强

从简单开始，根据需要添加高级工具：

```go
// 基础代理
agent, _ := prebuilt.NewChatAgent(model, basicTools)

// 当用户准备好时添加高级工具
if userLevel == "advanced" {
    agent.AddTool(advancedAnalyticsTool)
    agent.AddTool(dataVisualizationTool)
}
```

### 3. 资源管理

不需要时移除昂贵的工具：

```go
// 配额超出后移除基于 API 的工具
if quotaExceeded {
    agent.RemoveTool("expensive_api")
}
```

### 4. 访问控制

根据权限授予或撤销工具访问权限：

```go
// 仅在身份验证后授予文件访问权限
if user.IsAuthenticated() {
    agent.AddTool(fileReadTool)
    agent.AddTool(fileWriteTool)
}

// 注销时撤销
agent.RemoveTool("file_read")
agent.RemoveTool("file_write")
```

## 工作原理

1. **初始化**：创建 `ChatAgent` 时：
   - 使用 `CreateAgent` 创建底层代理图
   - 生成唯一的会话 ID（UUID）
   - 初始化空消息历史记录
   - 初始化空动态工具列表

2. **消息流**：调用 `Chat()` 时：
   - 用户消息被追加到对话历史记录
   - 动态工具（如果有）作为 `extra_tools` 添加到输入状态
   - 完整的历史记录和工具传递给代理图
   - 代理使用可用工具处理消息并生成响应
   - 响应被添加到历史记录
   - 提取文本响应并返回

3. **工具管理**：`ChatAgent` 维护两组工具：
   - **基础工具**：在创建代理时提供（通过 `NewChatAgent`）
   - **动态工具**：通过管理方法在运行时添加/移除
   - 两组工具在处理期间都可供代理使用

4. **状态管理**：`ChatAgent` 维护：
   - 用于会话标识的唯一 `threadID`
   - 包含完整对话历史记录的 `messages` 切片
   - 包含运行时添加的工具的 `dynamicTools` 切片
   - 对底层 `StateRunnable` 代理的引用

## 配置选项

您可以使用与 `CreateAgent` 相同的选项来自定义 ChatAgent：

```go
agent, err := prebuilt.NewChatAgent(
    model,
    tools,
    prebuilt.WithSystemMessage("你是一个有帮助的助手。"),
    prebuilt.WithVerbose(true),
)
```

可用选项：
- `WithSystemMessage(message string)`：设置系统消息
- `WithStateModifier(func)`：在发送到模型之前修改消息
- `WithVerbose(verbose bool)`：启用详细日志记录
- `WithSkillDir(dir string)`：启用基于技能的工具选择

## 异步流式输出使用场景

异步流式方法（`AsyncChat` 和 `AsyncChatWithChunks`）提供了多个好处：

### 1. 更好的用户体验

实时显示生成的响应：

```go
respChan, _ := agent.AsyncChatWithChunks(ctx, "解释机器学习")
for word := range respChan {
    fmt.Print(word)
    time.Sleep(50 * time.Millisecond)  // 打字效果
}
```

### 2. 感知性能

用户看到即时反馈，而不是等待完整响应：

```go
// 用户立即看到响应开始
respChan, _ := agent.AsyncChat(ctx, message)
for char := range respChan {
    updateUI(char)  // 实时更新 UI
}
```

### 3. 可中断的响应

取消不相关的长响应：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

respChan, _ := agent.AsyncChat(ctx, "告诉我关于...的所有内容")
// 超时发生时流将停止
```

### 4. 进度指示器

在生成响应时显示进度：

```go
go func() {
    for range time.Tick(500 * time.Millisecond) {
        fmt.Print(".")  // 显示活动
    }
}()

// 同时流式传输响应
for word := range respChan {
    fmt.Print(word)
}
```

## 示例

- **基本多轮对话**：查看 `examples/chat_agent/main.go` 获取完整工作示例
- **动态工具管理**：查看 `examples/chat_agent_dynamic_tools/main.go` 了解动态工具使用
- **异步流式输出**：查看 `examples/chat_agent_async/main.go` 了解流式响应

## 注意事项

- 每个 `ChatAgent` 实例代表一个单独的对话会话
- 对话历史记录在实例的生命周期内保存在内存中
- 要在应用程序重启后保持持久对话，您需要保存和恢复消息历史记录
- 底层代理使用 LangGraph 的状态管理和 `AppendReducer` 处理消息

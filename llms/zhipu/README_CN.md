# 使用 OpenAI 兼容方式访问智谱 AI (GLM)

智谱 AI（Zhipu AI / BigModel）提供了完全兼容 OpenAI 的 API，你可以直接使用 LangChain Go 的 OpenAI 客户端来访问智谱的 GLM 系列模型。

## 获取 API Key

1. 访问 [智谱 AI 开放平台](https://open.bigmodel.cn/)
2. 注册并登录账户
3. 在 [API Keys 管理页面](https://open.bigmodel.cn/usercenter/apikeys)创建 API Key

## 支持的模型

### 聊天模型

| 模型名称 | 说明 | 特点 |
|---------|------|------|
| `glm-4.7` | GLM-4.7 旗舰模型 | 最强智能，增强的 Video Coding 能力 |
| `glm-4.6` | GLM-4.6 高性能模型 | 200K 上下文，强大的代码能力 |
| `glm-4-plus` | GLM-4 Plus | 高性能通用模型 |
| `glm-4-air` | GLM-4 Air | 轻量级快速响应模型 |
| `glm-4-flash` | GLM-4 Flash | 超快速响应模型 |
| `glm-4-flashx` | GLM-4 FlashX | 极速响应，适合简单任务 |
| `glm-z1-plus` | GLM-Z1 Plus | 新一代 Z1 系列模型 |
| `glm-z1-air` | GLM-Z1 Air | Z1 系列轻量模型 |

### 多模态模型

| 模型名称 | 说明 |
|---------|------|
| `glm-4v` | GLM-4 视觉理解模型，支持图像理解 |
| `glm-4v-plus` | GLM-4V Plus 增强版 |
| `glm-4v-flash` | GLM-4V Flash 快速版 |

更多模型请参考：[智谱模型概览](https://docs.bigmodel.cn/cn/guide/start/model-overview)

### Embedding 模型

| 模型名称 | 说明 | 特点 |
|---------|------|------|
| `embedding-3` | Embedding-3 | 第三代向量化模型，支持自定义维度 |
| `embedding-2` | Embedding-2 | 第二代向量化模型 |

#### Embedding-3 特性

- **默认维度**: 1024 维
- **支持自定义维度**: 可在 1-1024 范围内自定义向量维度
- **更强的语义理解**: 相比前代模型有显著提升
- **批量处理**: 支持一次请求处理多个文本

Embedding API 文档：[Embedding-3 文档](https://docs.bigmodel.cn/cn/guide/models/embedding/embedding-3)

## 使用方式

### 使用 langchaingo 的 OpenAI 客户端

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    // 创建使用智谱 AI 的客户端
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),     // 智谱 AI API Key
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"), // 智谱 AI Base URL
    )
    if err != nil {
        panic(err)
    }

    // 调用聊天模型
    ctx := context.Background()
    completion, err := llms.GenerateFromSinglePrompt(
        ctx,
        llm,
        "你好，请介绍一下你自己",
        llms.WithModel("glm-4.7"),     // 指定 GLM 模型
        llms.WithTemperature(0.7),
        llms.WithMaxTokens(1000),
    )
    if err != nil {
        panic(err)
    }

    fmt.Println(completion)
}
```

### 使用 LangGraphGo 的 chat

如果你使用 LangGraphGo，可以更简洁地创建 chat 客户端：

```go
package main

import (
    "context"
    "fmt"
    "github.com/smallnest/langgraphgo/chat"
)

func main() {
    // 创建使用智谱 AI 的 chat 客户端
    c := chat.New(
        chat.WithLLM("glm-4.7"),                          // 指定 GLM 模型
        chat.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
        chat.WithAPIKey("your-zhipuai-api-key"),
    )

    ctx := context.Background()
    response, err := c.Run(ctx, "你好，请介绍一下你自己")
    if err != nil {
        panic(err)
    }

    fmt.Println(response)
}
```

### 流式响应

智谱 AI 支持流式响应：

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 使用流式响应
    _, err = llms.GenerateFromSinglePrompt(
        ctx,
        llm,
        "写一首关于人工智能的诗",
        llms.WithModel("glm-4.7"),
        llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
            fmt.Print(string(chunk))
            return nil
        }),
    )
    if err != nil {
        panic(err)
    }
}
```

### 多轮对话

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 构建对话历史
    messages := []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeSystem, "你是一个有用的 AI 助手"),
        llms.TextParts(llms.ChatMessageTypeHuman, "你好，请介绍一下你自己"),
    }

    // 第一轮对话
    resp1, err := llm.GenerateContent(ctx, messages, llms.WithModel("glm-4.7"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp1.Choices[0].Content)

    // 添加 AI 回复到历史
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, resp1.Choices[0].Content))

    // 第二轮对话
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, "你能帮我写代码吗？"))
    resp2, err := llm.GenerateContent(ctx, messages, llms.WithModel("glm-4.7"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp2.Choices[0].Content)
}
```

## 使用 Embedding

智谱 AI 的 Embedding API 与 OpenAI 格式完全兼容：

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    // 创建 OpenAI 客户端，指定 Embedding 模型
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
        openai.WithEmbeddingModel("embedding-3"),  // 指定 Embedding 模型
    )
    if err != nil {
        panic(err)
    }

    // 创建 Embedder（此时无需再指定模型）
    embedder, err := embeddings.NewEmbedder(llm)
    if err != nil {
        panic(err)
    }

    // 生成 Embedding
    ctx := context.Background()
    texts := []string{"你好世界", "智谱AI平台"}
    vectors, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        panic(err)
    }

    for i, vec := range vectors {
        fmt.Printf("Text: %s, Vector length: %d\n", texts[i], len(vec))
    }
}
```

### 自定义 Embedding 维度

Embedding-3 支持自定义向量维度（1-1024）：

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 直接调用 CreateEmbedding，可以传递额外参数
    // 注意：langchaingo 的 embeddings.NewEmbedder 可能不支持自定义维度
    // 如果需要自定义维度，可能需要直接调用 API
    texts := []string{"这是一段需要向量化的文本"}

    // 使用默认方式调用（1024 维）
    resp, err := llm.CreateEmbedding(ctx, texts)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Vector length: %d\n", len(resp[0]))
}
```

### Embedding 请求注意事项

- **`embedding-3`**: 默认 1024 维，支持自定义 1-1024 维
- **批量处理**: 支持一次请求处理多个文本
- **速率限制**: 不同等级的用户有不同的并发限制

## 函数调用 (Function Calling)

智谱 AI 支持 OpenAI 格式的函数调用：

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    // 定义工具
    tools := []llms.Tool{
        {
            Type: "function",
            Function: &llms.FunctionDefinition{
                Name:        "get_weather",
                Description: "获取指定地点的天气信息",
                Parameters: map[string]any{
                    "type": "object",
                    "properties": map[string]any{
                        "location": map[string]any{
                            "type":        "string",
                            "description": "地点名称，例如：北京、上海",
                        },
                    },
                    "required": []string{"location"},
                },
            },
        },
    }

    ctx := context.Background()
    messages := []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "北京今天天气怎么样？"),
    }

    resp, err := llm.GenerateContent(
        ctx,
        messages,
        llms.WithModel("glm-4.7"),
        llms.WithTools(tools),
        llms.WithToolChoice("auto"),
    )
    if err != nil {
        panic(err)
    }

    // 处理函数调用
    choice := resp.Choices[0]
    if len(choice.ToolCalls) > 0 {
        for _, toolCall := range choice.ToolCalls {
            if toolCall.FunctionCall.Name == "get_weather" {
                var args map[string]any
                json.Unmarshal([]byte(toolCall.FunctionCall.Arguments), &args)
                location := args["location"].(string)
                fmt.Printf("调用函数: get_weather, 参数: %s\n", location)
                // 这里调用实际的天气 API
            }
        }
    }
}
```

## 图像理解

使用 GLM-4V 进行图像理解：

```go
package main

import (
    "context"
    "encoding/base64"
    "fmt"
    "os"
    "github.com/tmc/langchaingo/llms"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    // 读取并编码图像
    imageData, _ := os.ReadFile("path/to/image.jpg")
    base64Image := base64.StdEncoding.EncodeToString(imageData)

    ctx := context.Background()
    messages := []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "请描述这张图片的内容"),
    }

    // 添加图像
    messages[0].Parts = append(messages[0].Parts,
        llms.ImageURLPart("data:image/jpeg;base64,"+base64Image),
    )

    resp, err := llm.GenerateContent(ctx, messages, llms.WithModel("glm-4v"))
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Content)
}
```

## 环境变量配置

你可以通过环境变量设置 API Key：

```bash
export OPENAI_API_KEY="your-zhipuai-api-key"
export OPENAI_BASE_URL="https://open.bigmodel.cn/api/paas/v4/"
```

然后代码中无需显式设置：

```go
llm, err := openai.New()  // 自动从环境变量读取
```

## 参数说明

智谱 AI 支持以下常用参数：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| `model` | string | 必填 | 要使用的模型名称 |
| `temperature` | float | 0.6 | 控制输出的随机性 (0-1) |
| `top_p` | float | 0.95 | 核采样参数 (0-1) |
| `max_tokens` | int | - | 最大输出 token 数 |
| `stream` | bool | false | 是否使用流式输出 |

## 完整示例：RAG 应用

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/chains"
    "github.com/tmc/langchaingo/llms/openai"
    "github.com/tmc/langchaingo/prompts"
)

func main() {
    // 创建智谱 AI 客户端
    llm, err := openai.New(
        openai.WithToken("your-zhipuai-api-key"),
        openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),
    )
    if err != nil {
        panic(err)
    }

    // 创建提示模板
    prompt := prompts.NewPromptTemplate(
        "请根据以下上下文回答问题：\n\n上下文：{{.context}}\n\n问题：{{.question}}",
        []string{"context", "question"},
    )

    // 创建链
    chain := chains.NewLLMChain(llm, prompt)

    ctx := context.Background()
    result, err := chains.Run(ctx, chain, map[string]any{
        "context":  "LangGraphGo 是一个 Go 实现的 LangGraph 框架...",
        "question": "什么是 LangGraphGo？",
    })
    if err != nil {
        panic(err)
    }

    fmt.Println(result)
}
```

## 从 OpenAI 迁移

如果您已经在使用 OpenAI API，迁移到智谱 AI 非常简单：

```go
// 原来的 OpenAI 代码
client := openai.New(
    openai.WithToken("sk-..."),  // OpenAI API Key
    // base_url 使用默认值
)

// 迁移到智谱 AI，只需要修改两个地方
client := openai.New(
    openai.WithToken("your-zhipuai-api-key"),  // 替换为智谱 AI API Key
    openai.WithBaseURL("https://open.bigmodel.cn/api/paas/v4/"),  // 添加智谱 AI base_url
)

// 其他代码保持不变
resp, err := client.GenerateContent(
    ctx,
    messages,
    llms.WithModel("glm-4.7"),  // 使用智谱 AI 模型
)
```

## 定价信息

查看最新定价：[智谱 AI 定价](https://bigmodel.cn/pricing)

## 参考文档

- [智谱 AI 开放平台](https://open.bigmodel.cn/)
- [OpenAI API 兼容文档](https://docs.bigmodel.cn/cn/guide/develop/openai/introduction)
- [模型概览](https://docs.bigmodel.cn/cn/guide/start/model-overview)
- [Embedding-3 文档](https://docs.bigmodel.cn/cn/guide/models/embedding/embedding-3)
- [Embedding API 参考](https://bigmodel.cn/dev/api/vector/embedding)

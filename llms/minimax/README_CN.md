# 使用 OpenAI 兼容方式访问 MiniMax

MiniMax 提供了完全兼容 OpenAI 的 API，你可以直接使用 LangChain Go 的 OpenAI 客户端来访问 MiniMax 的系列模型。

## 获取 API Key

1. 访问 [MiniMax 开放平台](https://www.minimaxi.com/login) 并注册账户
2. 在 [API Keys 页面](https://www.minimaxi.com/user-center/basic-information/interface-key)生成 API Key

## 支持的模型

### 聊天模型

| 模型名称 | 说明 | 特点 |
|---------|------|------|
| `MiniMax-M2.1` | MiniMax M2.1 旗舰模型 | 强大多语言编程能力，输出速度约 60tps |
| `MiniMax-M2.1-lightning` | M2.1 极速版 | 更快更敏捷，输出速度约 100tps |
| `MiniMax-M2` | M2 高效模型 | 专为高效编码与 Agent 工作流而生 |

更多模型请参考：[MiniMax 文档](https://platform.minimaxi.com/docs/api-reference/text-openai-api)

### Embedding 模型

| 模型名称 | 说明 | 类型 |
|---------|------|------|
| `embo-01` | MiniMax Embedding 模型，用于文本向量化 | 支持 `query` 和 `db` 两种类型 |

#### Embedding 类型说明

MiniMax Embedding API 支持两种类型：

- **`query`**：用于查询文本的向量化（默认类型）
- **`db`**：用于数据库/文档文本的向量化

不同类型的向量可能使用不同的编码策略，以便在相似度计算时获得更好的效果。

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
    // 创建使用 MiniMax 的客户端
    // 国内用户使用 https://api.minimaxi.com/v1
    // 国际用户使用 https://api.minimax.io/v1
    llm, err := openai.New(
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"), // 国内用户
        // openai.WithBaseURL("https://api.minimax.io/v1"), // 国际用户
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
        llms.WithModel("MiniMax-M2.1"),     // 指定 MiniMax 模型
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
    // 创建使用 MiniMax 的 chat 客户端
    c := chat.New(
        chat.WithLLM("MiniMax-M2.1"),                       // 指定 MiniMax 模型
        chat.WithBaseURL("https://api.minimaxi.com/v1"),     // 国内用户
        // chat.WithBaseURL("https://api.minimax.io/v1"),   // 国际用户
        chat.WithAPIKey("your-minimax-api-key"),
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

MiniMax 支持流式响应：

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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
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
        llms.WithModel("MiniMax-M2.1"),
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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
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
    resp1, err := llm.GenerateContent(ctx, messages, llms.WithModel("MiniMax-M2.1"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp1.Choices[0].Content)

    // 添加 AI 回复到历史
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, resp1.Choices[0].Content))

    // 第二轮对话
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, "你能帮我写代码吗？"))
    resp2, err := llm.GenerateContent(ctx, messages, llms.WithModel("MiniMax-M2.1"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp2.Choices[0].Content)
}
```

## 使用 Embedding

MiniMax 的 Embedding API 与 OpenAI 格式完全兼容：

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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
        openai.WithEmbeddingModel("embo-01"),  // 指定 MiniMax Embedding 模型
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
    texts := []string{"你好世界", "MiniMax平台"}
    vectors, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        panic(err)
    }

    for i, vec := range vectors {
        fmt.Printf("Text: %s, Vector length: %d\n", texts[i], len(vec))
    }
}
```

### Embedding 请求注意事项

- **`embo-01`**: MiniMax 的 Embedding 模型
- 支持批量处理多个文本
- 返回固定维度的向量
- **类型参数**：Embedding API 支持 `type` 参数，可选值为 `query`（默认）或 `db`
  - `query`：用于用户查询的向量化
  - `db`：用于文档/数据库存储的向量化

### 直接调用 Embedding API（带类型参数）

如果需要指定 Embedding 类型，可以直接调用 API：

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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 为文档生成 Embedding（使用 db 类型）
    docTexts := []string{"LangGraphGo 是一个 Go 实现的 LangGraph 框架"}
    docVectors, err := llm.CreateEmbedding(ctx, docTexts)
    if err != nil {
        panic(err)
    }

    // 为查询生成 Embedding（使用 query 类型）
    // 注意：langchaingo 可能不支持直接传递 type 参数
    // 如果需要指定类型，可能需要直接调用 HTTP API
    queryTexts := []string{"什么是 LangGraphGo？"}
    queryVectors, err := llm.CreateEmbedding(ctx, queryTexts)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Document vector length: %d\n", len(docVectors[0]))
    fmt.Printf("Query vector length: %d\n", len(queryVectors[0]))
}
```

### 使用 HTTP 客户端指定 Embedding 类型

如果需要精确控制 Embedding 类型参数，可以使用 HTTP 客户端直接调用：

```go
package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type EmbeddingRequest struct {
    Model string   `json:"model"`
    Input []string `json:"input"`
    Type  string   `json:"type"` // "query" 或 "db"
}

type EmbeddingResponse struct {
    Data []struct {
        Embedding []float32 `json:"embedding"`
        Index     int       `json:"index"`
    } `json:"data"`
}

func main() {
    apiKey := "your-minimax-api-key"
    baseURL := "https://api.minimaxi.com/v1"

    // 创建请求
    reqBody := EmbeddingRequest{
        Model: "embo-01",
        Input: []string{"你好世界"},
        Type:  "query", // 指定类型
    }

    jsonData, _ := json.Marshal(reqBody)

    req, _ := http.NewRequestWithContext(
        context.Background(),
        "POST",
        baseURL+"/embeddings",
        bytes.NewReader(jsonData),
    )
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+apiKey)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)

    var result EmbeddingResponse
    json.Unmarshal(body, &result)

    if len(result.Data) > 0 {
        fmt.Printf("Vector length: %d\n", len(result.Data[0].Embedding))
    }
}
```

## Interleaved Thinking（交替思考）

MiniMax M2 系列模型支持 Interleaved Thinking 功能，可以将思考过程分离到 `reasoning_details` 字段：

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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()

    // 使用 reasoning_split 参数分离思考内容
    // 注意：需要通过额外参数传递，具体实现取决于 langchaingo 的支持情况
    messages := []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeSystem, "You are a helpful assistant."),
        llms.TextParts(llms.ChatMessageTypeHuman, "解释一下量子计算的基本原理"),
    }

    resp, err := llm.GenerateContent(ctx, messages, llms.WithModel("MiniMax-M2.1"))
    if err != nil {
        panic(err)
    }

    fmt.Println("Response:", resp.Choices[0].Content)
    // 如果启用了 reasoning_split，思考内容会在单独的字段中
}
```

## 函数调用 (Function Calling)

MiniMax M2 系列支持 OpenAI 格式的函数调用：

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
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
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
        llms.WithModel("MiniMax-M2.1"),
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

## 环境变量配置

你可以通过环境变量设置 API Key：

```bash
# 国内用户
export OPENAI_API_KEY="your-minimax-api-key"
export OPENAI_BASE_URL="https://api.minimaxi.com/v1"

# 国际用户
# export OPENAI_API_KEY="your-minimax-api-key"
# export OPENAI_BASE_URL="https://api.minimax.io/v1"
```

然后代码中无需显式设置：

```go
llm, err := openai.New()  // 自动从环境变量读取
```

## API 端点说明

MiniMax 提供两个 API 端点：

| 用户类型 | Base URL | 说明 |
|---------|----------|------|
| 国内用户 | `https://api.minimaxi.com/v1` | 面向中国大陆用户 |
| 国际用户 | `https://api.minimax.io/v1` | 面向海外用户 |

根据你的账号类型选择合适的端点。

## 参数说明

MiniMax 支持以下常用参数：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| `model` | string | 必填 | 要使用的模型名称 |
| `temperature` | float | - | 控制输出的随机性 (0-1) |
| `top_p` | float | - | 核采样参数 (0-1) |
| `max_tokens` | int | - | 最大输出 token 数 |
| `stream` | bool | false | 是否使用流式输出 |

## 注意事项

### 多轮 Function Call 对话

在多轮 Function Call 对话中，必须将完整的模型返回（即 assistant 消息）添加到对话历史，以保持思维链的连续性：
- 将完整的 `response_message` 对象（包含 `tool_calls` 字段）添加到消息历史
- 原生的 OpenAI API 的 MiniMax-M2.1、MiniMax-M2.1-lightning、MiniMax-M2 模型 `content` 字段会包含 `<think>` 标签内容，需要完整保留
- 在 Interleaved Thinking 友好格式中，通过启用额外的参数（`reasoning_split=True`），模型思考内容通过 `reasoning_details` 字段单独提供，同样需要完整保留

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
    // 创建 MiniMax 客户端
    llm, err := openai.New(
        openai.WithToken("your-minimax-api-key"),
        openai.WithBaseURL("https://api.minimaxi.com/v1"),
        openai.WithEmbeddingModel("embo-01"),
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

如果您已经在使用 OpenAI API，迁移到 MiniMax 非常简单：

```go
// 原来的 OpenAI 代码
client := openai.New(
    openai.WithToken("sk-..."),  // OpenAI API Key
    // base_url 使用默认值
)

// 迁移到 MiniMax，只需要修改两个地方
client := openai.New(
    openai.WithToken("your-minimax-api-key"),  // 替换为 MiniMax API Key
    openai.WithBaseURL("https://api.minimaxi.com/v1"),  // 添加 MiniMax base_url
)

// 其他代码保持不变
resp, err := client.GenerateContent(
    ctx,
    messages,
    llms.WithModel("MiniMax-M2.1"),  // 使用 MiniMax 模型
)
```

## 参考文档

- [MiniMax 开放平台](https://www.minimaxi.com/)
- [OpenAI API 兼容文档](https://platform.minimaxi.com/docs/api-reference/text-openai-api)
- [API Keys 管理页面](https://www.minimaxi.com/user-center/basic-information/interface-key)
- [Embeddings API 文档](https://5cetebcrn8.apifox.cn/doc-3518198)
- [Spring AI MiniMax 文档](https://docs.springframework.org.cn/spring-ai/reference/api/embeddings/minimax-embeddings.html)
- [MiniMax Embeddings | Spring AI Alibaba](https://java2ai.com/integration/rag/embeddings/more/minimax-embeddings)
- [快速开始指南](https://platform.minimaxi.com/docs/guides/quickstart)

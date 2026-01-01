# 使用 OpenAI 兼容方式访问阿里云通义千问 (Qwen)

阿里云百炼平台提供了完全兼容 OpenAI 的 API，你可以直接使用 LangChain Go 的 OpenAI 客户端来访问通义千问（Qwen）系列模型。

## 获取 API Key

1. 访问 [阿里云百炼平台](https://bailian.console.aliyun.com/)
2. 登录阿里云账号并开通服务
3. 在 [API-Key 管理页面](https://bailian.console.aliyun.com/?apiKey=1)创建 API Key

## 支持的模型

### 聊天模型

| 模型名称 | 说明 | 特点 |
|---------|------|------|
| `qwen-max-latest` | Qwen Max 最新版本 | 旗舰模型，最强推理能力 |
| `qwen-plus-latest` | Qwen Plus 最新版本 | 高性能模型，平衡速度与质量 |
| `qwen-turbo-latest` | Qwen Turbo 最新版本 | 极速响应，适合简单任务 |
| `qwen-long-latest` | Qwen Long 最新版本 | 长文本理解，支持 1M+ 上下文 |
| `qwen-max-latest-0919` | Qwen Max 固定版本 | 2024年9月版本 |
| `qwen-plus-latest-0919` | Qwen Plus 固定版本 | 2024年9月版本 |
| `qwen-turbo-latest-0919` | Qwen Turbo 固定版本 | 2024年9月版本 |

### 代码模型

| 模型名称 | 说明 |
|---------|------|
| `qwen-coder-plus-latest` | Qwen Coder Plus，代码生成与理解 |
| `qwen-coder-turbo-latest` | Qwen Coder Turbo，快速代码生成 |

### 视觉模型

| 模型名称 | 说明 |
|---------|------|
| `qwen-vl-max-latest` | Qwen VL Max，视觉理解最强模型 |
| `qwen-vl-plus-latest` | Qwen VL Plus，高性能视觉理解 |
| `qwen-vl-v1-latest` | Qwen VL v1，基础视觉理解 |

更多模型请参考：[模型列表](https://help.aliyun.com/zh/model-studio/models)

### Embedding 模型

| 模型名称 | 说明 | 向量维度 |
|---------|------|---------|
| `text-embedding-v3` | 通用文本向量 v3 | 1024 维 |
| `text-embedding-v2` | 通用文本向量 v2 | 1536 维 |
| `text-embedding-v1` | 通用文本向量 v1 | 1536 维 |

Embedding API 文档：[通用文本向量同步接口API详情](https://help.aliyun.com/zh/model-studio/text-embedding-synchronous-api)

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
    // 创建使用通义千问的客户端
    llm, err := openai.New(
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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
        llms.WithModel("qwen-max-latest"),     // 指定 Qwen 模型
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
    // 创建使用通义千问的 chat 客户端
    c := chat.New(
        chat.WithLLM("qwen-max-latest"),                               // 指定 Qwen 模型
        chat.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
        chat.WithAPIKey("your-dashscope-api-key"),
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

通义千问支持流式响应：

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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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
        llms.WithModel("qwen-max-latest"),
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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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
    resp1, err := llm.GenerateContent(ctx, messages, llms.WithModel("qwen-max-latest"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp1.Choices[0].Content)

    // 添加 AI 回复到历史
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeAI, resp1.Choices[0].Content))

    // 第二轮对话
    messages = append(messages, llms.TextParts(llms.ChatMessageTypeHuman, "你能帮我写代码吗？"))
    resp2, err := llm.GenerateContent(ctx, messages, llms.WithModel("qwen-max-latest"))
    if err != nil {
        panic(err)
    }
    fmt.Println("AI:", resp2.Choices[0].Content)
}
```

## 使用 Embedding

通义千问的 Embedding API 与 OpenAI 格式完全兼容：

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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
        openai.WithEmbeddingModel("text-embedding-v3"),  // 指定 Embedding 模型
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
    texts := []string{"你好世界", "通义千问平台"}
    vectors, err := embedder.EmbedDocuments(ctx, texts)
    if err != nil {
        panic(err)
    }

    for i, vec := range vectors {
        fmt.Printf("Text: %s, Vector length: %d\n", texts[i], len(vec))
    }
}
```

### Embedding 模型选择建议

- **`text-embedding-v3`**（推荐）：最新版本，1024 维，语义理解能力最强
- **`text-embedding-v2`**：1536 维，适用于需要更高维度的场景
- **`text-embedding-v1`**：1536 维，兼容早期版本

## 代码生成

使用 Qwen Coder 模型进行代码生成：

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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
    )
    if err != nil {
        panic(err)
    }

    ctx := context.Background()
    messages := []llms.MessageContent{
        llms.TextParts(llms.ChatMessageTypeHuman, "用 Go 写一个快速排序算法"),
    }

    resp, err := llm.GenerateContent(ctx, messages, llms.WithModel("qwen-coder-plus-latest"))
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Content)
}
```

## 图像理解

使用 Qwen VL 模型进行图像理解：

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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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

    resp, err := llm.GenerateContent(ctx, messages, llms.WithModel("qwen-vl-max-latest"))
    if err != nil {
        panic(err)
    }

    fmt.Println(resp.Choices[0].Content)
}
```

## 函数调用 (Function Calling)

通义千问支持 OpenAI 格式的函数调用：

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
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
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
        llms.WithModel("qwen-max-latest"),
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
export OPENAI_API_KEY="your-dashscope-api-key"
export OPENAI_BASE_URL="https://dashscope.aliyuncs.com/compatible-mode/v1"
```

然后代码中无需显式设置：

```go
llm, err := openai.New()  // 自动从环境变量读取
```

## 参数说明

通义千问支持以下常用参数：

| 参数 | 类型 | 默认值 | 说明 |
|-----|------|-------|------|
| `model` | string | 必填 | 要使用的模型名称 |
| `temperature` | float | - | 控制输出的随机性 (0-2) |
| `top_p` | float | - | 核采样参数 (0-1) |
| `top_k` | int | - | Top-K 采样参数 |
| `max_tokens` | int | - | 最大输出 token 数 |
| `stream` | bool | false | 是否使用流式输出 |
| `result_format` | string | message | 返回格式，可选 `message` 或 `text` |

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
    // 创建通义千问客户端
    llm, err := openai.New(
        openai.WithToken("your-dashscope-api-key"),
        openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),
        openai.WithEmbeddingModel("text-embedding-v3"),
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

如果您已经在使用 OpenAI API，迁移到通义千问非常简单，只需修改 3 个参数：

```go
// 原来的 OpenAI 代码
client := openai.New(
    openai.WithToken("sk-..."),  // OpenAI API Key
    // base_url 使用默认值
)

// 迁移到通义千问，只需要修改三个地方
client := openai.New(
    openai.WithToken("your-dashscope-api-key"),  // 1. 替换为 DashScope API Key
    openai.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1"),  // 2. 添加 Base URL
)

// 其他代码保持不变，只需更改模型名
resp, err := client.GenerateContent(
    ctx,
    messages,
    llms.WithModel("qwen-max-latest"),  // 3. 使用通义千问模型名
)
```

## 模型版本说明

通义千问使用 `-latest` 后缀表示最新版本，也支持使用固定版本号：

- 使用 `-latest`：自动使用最新版本，推荐用于生产环境
- 使用固定日期（如 `-0919`）：使用特定版本，适合需要稳定性的场景

## 区域支持

阿里云百炼平台支持多个区域：

- **中国大陆**：`https://dashscope.aliyuncs.com`
- **新加坡**：`https://dashscope.aliyuncs.com`（国际版）
- **金融云**：专有金融云节点

根据你的业务位置选择合适的区域。

## 参考文档

- [通义千问 API 参考](https://help.aliyun.com/zh/model-studio/qwen-api-reference)
- [使用 OpenAI 兼容接口调用通义千问](https://help.aliyun.com/zh/model-studio/compatibility-of-openai-with-dashscope)
- [通用文本向量同步接口 API 详情](https://help.aliyun.com/zh/model-studio/text-embedding-synchronous-api)
- [通用文本向量模型介绍](https://help.aliyun.com/zh/model-studio/model-introduction-6)
- [模型列表](https://help.aliyun.com/zh/model-studio/models)
- [首次调用通义千问 API](https://help.aliyun.com/zh/model-studio/first-api-call-to-qwen)
- [文本与多模态向量化](https://help.aliyun.com/zh/model-studio/embedding)

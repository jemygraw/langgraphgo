# 使用 OpenAI 兼容方式访问百度千帆 (ERNIE)

百度千帆（百度智能云千帆平台）提供了 OpenAI 兼容的 API，你可以直接使用 LangChain Go 的 OpenAI 客户端来访问百度的 ERNIE 系列模型。

## 获取 API Key

1. 访问 [百度智能云千帆平台](https://cloud.baidu.com/product/wenxinworkshop)
2. 登录并进入 [千帆控制台](https://console.bce.baidu.com/qianfan/ais/console/applicationConsole/application)
3. 创建应用并获取 API Key

## 支持的模型

### 聊天模型

| 模型名称 | 说明 |
|---------|------|
| `ernie-4.5-turbo-128k` | ERNIE 4.5 Turbo，128K 上下文（推荐） |
| `ernie-speed-128k` | ERNIE Speed，128K 上下文 |
| `ernie-speed-8k` | ERNIE Speed，8K 上下文 |
| `ernie-lite-8k` | ERNIE Lite，8K 上下文 |
| `deepseek-r1` | DeepSeek R1 模型 |
| `deepseek-v3` | DeepSeek V3 模型 |

更多模型请参考：[千帆模型列表](https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Nlks5zkzu)

### Embedding 模型

| 模型名称 | 说明 | 限制 |
|---------|------|------|
| `embedding-v1` | 基础向量化模型 | 单文本最多 384 tokens，长度 <= 1000 字符 |
| `bge-large-zh` | BGE Large 中文 | 单文本最多 512 tokens，长度 <= 2000 字符 |
| `bge-large-en` | BGE Large 英文 | 单文本最多 512 tokens，长度 <= 2000 字符 |
| `tao-8k` | Tao 8K 向量化 | 单文本最多 8192 tokens，长度 <= 28000 字符，仅支持单文本 |
| `qwen3-embedding-0.6b` | Qwen3 Embedding 0.6B | 最多 8K tokens |
| `qwen3-embedding-4b` | Qwen3 Embedding 4B | 最多 8K tokens |

Embedding API 文档：[千帆 Embedding API](https://cloud.baidu.com/doc/qianfan-api/s/Fm7u3ropn)

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
    // 创建使用百度千帆的客户端
    llm, err := openai.New(
        openai.WithToken("your-api-key"),           // 百度千帆 API Key
        openai.WithBaseURL("https://qianfan.baidubce.com/v2"), // 百度千帆 Base URL
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
        llms.WithModel("ernie-4.5-turbo-128k"),     // 指定 ERNIE 模型
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
    // 创建使用百度千帆的 chat 客户端
    c := chat.New(
        chat.WithLLM("ernie-4.5-turbo-128k"),       // 指定 ERNIE 模型
        chat.WithBaseURL("https://qianfan.baidubce.com/v2"),
        chat.WithAPIKey("your-api-key"),
    )

    ctx := context.Background()
    response, err := c.Run(ctx, "你好，请介绍一下你自己")
    if err != nil {
        panic(err)
    }

    fmt.Println(response)
}
```

## 使用 Embedding

百度千帆的 Embedding API 与 OpenAI 格式完全兼容，可以直接使用：

```go
package main

import (
    "context"
    "fmt"
    "github.com/tmc/langchaingo/embeddings"
    "github.com/tmc/langchaingo/llms/openai"
)

func main() {
    // 创建 OpenAI 客户端
    llm, err := openai.New(
        openai.WithToken("your-api-key"),
        openai.WithBaseURL("https://qianfan.baidubce.com/v2"),
    )
    if err != nil {
        panic(err)
    }

    // 创建 Embedder
    embedder, err := embeddings.NewEmbedder(llm, "embedding-v1")
    if err != nil {
        panic(err)
    }

    // 生成 Embedding
    ctx := context.Background()
    texts := []string{"你好世界", "百度千帆平台"}
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

- **`embedding-v1`**: 单文本最多 384 tokens，长度 <= 1000 字符，每次最多 16 个文本
- **`bge-large-zh` / `bge-large-en`**: 单文本最多 512 tokens，长度 <= 2000 字符，每次最多 16 个文本
- **`tao-8k`**: 单文本最多 8192 tokens，长度 <= 28000 字符，但**仅支持单文本**
- **`qwen3-embedding-0.6b` / `qwen3-embedding-4b`**: 最多 8K tokens

## 环境变量配置

你可以通过环境变量设置 API Key：

```bash
export OPENAI_API_KEY="your-baidu-qianfan-api-key"
export OPENAI_BASE_URL="https://qianfan.baidubce.com/v2"
```

然后代码中无需显式设置：

```go
llm, err := openai.New()  // 自动从环境变量读取
```

## 温度参数注意事项

百度千帆 API 要求温度参数必须在 `(0, 1.0]` 范围内，且不能为 0。建议设置：

```go
llms.WithTemperature(0.1),  // 设置一个小的非零值
```

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
    // 创建百度千帆客户端
    llm, err := openai.New(
        openai.WithToken("your-api-key"),
        openai.WithBaseURL("https://qianfan.baidubce.com/v2"),
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

## 参考文档

- [百度千帆平台](https://cloud.baidu.com/product/wenxinworkshop)
- [千帆 API 文档](https://cloud.baidu.com/doc/qianfan-api/s/3m9b5lqft)
- [千帆模型列表](https://cloud.baidu.com/doc/WENXINWORKSHOP/s/Nlks5zkzu)
- [Embedding API 文档](https://cloud.baidu.com/doc/qianfan-api/s/Fm7u3ropn)

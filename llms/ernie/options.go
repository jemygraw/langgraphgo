package ernie

import (
	"net/http"
	"os"

	"github.com/tmc/langchaingo/callbacks"
)

// ModelName represents the model identifier for Baidu Qianfan (Ernie) API.
type ModelName string

const (
	// ========================================
	// 推荐模型 (Recommended Models)
	// ========================================

	// ERNIE 5.0 - 文心新一代原生全模态大模型
	ModelNameERNIE5ThinkingPreview ModelName = "ernie-5.0-thinking-preview" // 128k context, latest generation
	ModelNameERNIE5ThinkingLatest  ModelName = "ernie-5.0-thinking-latest"  // 128k context, latest version

	// ERNIE 4.5 Turbo - 高性能主力模型
	ModelNameERNIE45Turbo128K        ModelName = "ernie-4.5-turbo-128k"         // 128k context
	ModelNameERNIE45Turbo128KPreview ModelName = "ernie-4.5-turbo-128k-preview" // 128k context, preview
	ModelNameERNIE45Turbo32K         ModelName = "ernie-4.5-turbo-32k"          // 32k context
	ModelNameERNIE45TurboLatest      ModelName = "ernie-4.5-turbo-latest"       // 128k context, latest

	// ERNIE 4.5 Turbo VL - 视觉语言模型
	ModelNameERNIE45TurboVLPreview    ModelName = "ernie-4.5-turbo-vl-preview"     // 128k context
	ModelNameERNIE45TurboVL           ModelName = "ernie-4.5-turbo-vl"             // 128k context
	ModelNameERNIE45TurboVL32K        ModelName = "ernie-4.5-turbo-vl-32k"         // 32k context
	ModelNameERNIE45TurboVL32KPreview ModelName = "ernie-4.5-turbo-vl-32k-preview" // 32k context, preview
	ModelNameERNIE45TurboVLLatest     ModelName = "ernie-4.5-turbo-vl-latest"      // 128k context, latest

	// ERNIE 4.5
	ModelNameERNIE458KPreview ModelName = "ernie-4.5-8k-preview" // 8k context

	// DeepSeek R1 - 推理专用模型
	ModelNameDeepSeekR1       ModelName = "deepseek-r1"                // 144k context
	ModelNameDeepSeekR1Latest ModelName = "deepseek-r1-250528"         // 144k context, latest
	ModelNameDeepSeekV32Think ModelName = "deepseek-v3.2-think"        // 144k context
	ModelNameDeepSeekV31Think ModelName = "deepseek-v3.1-think-250821" // 144k context

	// ========================================
	// ERNIE系列 - 主力模型
	// ========================================

	ModelNameERNIESpeed128K    ModelName = "ernie-speed-128k"     // 128k context
	ModelNameERNIESpeed8K      ModelName = "ernie-speed-8k"       // 8k context
	ModelNameERNIESpeedPro128K ModelName = "ernie-speed-pro-128k" // 128k context, pro

	ModelNameERNIELite8K      ModelName = "ernie-lite-8k"       // 8k context
	ModelNameERNIELitePro128K ModelName = "ernie-lite-pro-128k" // 128k context, pro

	// ========================================
	// ERNIE系列 - 轻量模型
	// ========================================

	ModelNameERNIETiny8K ModelName = "ernie-tiny-8k" // 8k context

	// ========================================
	// ERNIE系列 - 垂直场景模型
	// ========================================

	ModelNameERNIEChar8K ModelName = "ernie-char-8k" // 角色扮演专用

	// ========================================
	// ERNIE系列 - 开源模型
	// ========================================

	ModelNameERNIE4503B      ModelName = "ernie-4.5-0.3b"       // 128k context
	ModelNameERNIE4521BA3B   ModelName = "ernie-4.5-21b-a3b"    // 128k context
	ModelNameERNIE45VL28BA3B ModelName = "ernie-4.5-vl-28b-a3b" // 32k context, visual

	// ========================================
	// QianFan系列
	// ========================================

	ModelNameQianfanLightning128BA19B ModelName = "qianfan-lightning-128b-a19b" // 128k context
	ModelNameQianfan8B                ModelName = "qianfan-8b"                  // 32k context
	ModelNameQianfan70B               ModelName = "qianfan-70b"                 // 32k context
	ModelNameQianfanSug8K             ModelName = "qianfan-sug-8k"              // 8k context, summary
	ModelNameQianfanCorrect           ModelName = "qianfan-correct"             // 8k context, text correction
	ModelNameQianfanToyTalk           ModelName = "qianfan-toytalk"             // 32k context, children interaction

	// ========================================
	// DeepSeek系列
	// ========================================

	ModelNameDeepSeekV3  ModelName = "deepseek-v3"          // 128k context
	ModelNameDeepSeekV31 ModelName = "deepseek-v3.1-250821" // 128k context
	ModelNameDeepSeekV32 ModelName = "deepseek-v3.2"        // 128k context

	// ========================================
	// Qwen系列
	// ========================================

	// Qwen3 Coder - 代码专用
	ModelNameQwen3Coder480BA35B ModelName = "qwen3-coder-480b-a35b-instruct" // 128k context
	ModelNameQwen3Coder30BA3B   ModelName = "qwen3-coder-30b-a3b-instruct"   // 128k context

	// Qwen3 Next - 下一代
	ModelNameQwen3Next80BA3B ModelName = "qwen3-next-80b-a3b-instruct" // 128k context

	// Qwen3 - 通用
	ModelNameQwen3235BA22BInstruct2507 ModelName = "qwen3-235b-a22b-instruct-2507" // 128k context
	ModelNameQwen330BA3BInstruct2507   ModelName = "qwen3-30b-a3b-instruct-2507"   // 128k context
	ModelNameQwen3235BA22B             ModelName = "qwen3-235b-a22b"               // 32k context
	ModelNameQwen330BA3B               ModelName = "qwen3-30b-a3b"                 // 32k context
	ModelNameQwen332B                  ModelName = "qwen3-32b"                     // 32k context
	ModelNameQwen314B                  ModelName = "qwen3-14b"                     // 32k context
	ModelNameQwen38B                   ModelName = "qwen3-8b"                      // 32k context
	ModelNameQwen34B                   ModelName = "qwen3-4b"                      // 32k context
	ModelNameQwen317B                  ModelName = "qwen3-1.7b"                    // 32k context
	ModelNameQwen306B                  ModelName = "qwen3-0.6b"                    // 32k context

	// Qwen2.5
	ModelNameQwen257BInstruct ModelName = "qwen2.5-7b-instruct" // 32k context

	// ========================================
	// 视觉理解模型
	// ========================================

	// QianFan VL
	ModelNameQianfanComposition  ModelName = "qianfan-composition"  // 图文创作
	ModelNameQianfanCheckVL      ModelName = "qianfan-check-vl"     // 图文检查
	ModelNameQianfanMultiPicOCR  ModelName = "qianfan-multipicocr"  // 多图OCR
	ModelNameQianfanVL70B        ModelName = "qianfan-vl-70b"       // 70B参数
	ModelNameQianfanVL8B         ModelName = "qianfan-vl-8b"        // 8B参数
	ModelNameQianfanQIVL         ModelName = "qianfan-qi-vl"        // 质检VL
	ModelNameQianfanEngCardVL    ModelName = "qianfan-engcard-vl"   // 英文卡片VL
	ModelNameQianfanSinglePicOCR ModelName = "qianfan-singlepicocr" // 单图OCR

	// InternVL
	ModelNameInternVL338B ModelName = "internvl3-38b" // 32k context

	// Qwen VL
	ModelNameQwen3VL32BInstruct      ModelName = "qwen3-vl-32b-instruct"       // 128k context
	ModelNameQwen3VL32BThinking      ModelName = "qwen3-vl-32b-thinking"       // 128k context
	ModelNameQwen3VL8BInstruct       ModelName = "qwen3-vl-8b-instruct"        // 128k context
	ModelNameQwen3VL8BThinking       ModelName = "qwen3-vl-8b-thinking"        // 128k context
	ModelNameQwen3VL30BA3B           ModelName = "qwen3-vl-30b-a3b-instruct"   // 128k context
	ModelNameQwen3VL30BA3BThinking   ModelName = "qwen3-vl-30b-a3b-thinking"   // 128k context
	ModelNameQwen3VL235BA22B         ModelName = "qwen3-vl-235b-a22b-instruct" // 128k context
	ModelNameQwen3VL235BA22BThinking ModelName = "qwen3-vl-235b-a22b-thinking" // 128k context
	ModelNameQwen25VL32BInstruct     ModelName = "qwen2.5-vl-32b-instruct"     // 32k context
	ModelNameQwen25VL7BInstruct      ModelName = "qwen2.5-vl-7b-instruct"      // 16k context

	// ========================================
	// 深度思考模型
	// ========================================

	// ERNIE X1
	ModelNameERNIEX11Preview        ModelName = "ernie-x1.1-preview"         // 64k context
	ModelNameERNIEX1Turbo32K        ModelName = "ernie-x1-turbo-32k"         // 32k context
	ModelNameERNIEX1Turbo32KPreview ModelName = "ernie-x1-turbo-32k-preview" // 32k context
	ModelNameERNIEX1TurboLatest     ModelName = "ernie-x1-turbo-latest"      // 64k context

	// ERNIE Thinking
	ModelNameERNIE4521BA3BThinking ModelName = "ernie-4.5-21b-a3b-thinking" // 128k context

	// DeepSeek Distill
	ModelNameDeepSeekR1DistillQwen32B ModelName = "deepseek-r1-distill-qwen-32b" // 32k context
	ModelNameDeepSeekR1DistillQwen14B ModelName = "deepseek-r1-distill-qwen-14b" // 32k context

	// Qwen Thinking
	ModelNameQwen3Next80BA3BThinking   ModelName = "qwen3-next-80b-a3b-thinking"   // 128k context
	ModelNameQwen3235BA22BThinking2507 ModelName = "qwen3-235b-a22b-thinking-2507" // 128k context
	ModelNameQwen330BA3BThinking2507   ModelName = "qwen3-30b-a3b-thinking-2507"   // 128k context
	ModelNameQWQ32B                    ModelName = "qwq-32b"                       // 32k context

	// 其他思考模型
	ModelNameGPToss120B ModelName = "gpt-oss-120b" // 128k context
	ModelNameGPToss20B  ModelName = "gpt-oss-20b"  // 128k context

	// ========================================
	// 文本向量模型
	// ========================================

	ModelNameEmbeddingV1       ModelName = "embedding-v1"         // 384维, 384 tokens, max 16 texts
	ModelNameTao8k             ModelName = "tao-8k"               // 1024维, 8192 tokens, max 1 text
	ModelNameBgeLargeZh        ModelName = "bge-large-zh"         // 1024维, 512 tokens, max 16 texts
	ModelNameBgeLargeEn        ModelName = "bge-large-en"         // 1024维, 512 tokens, max 16 texts
	ModelNameQwen3Embedding06B ModelName = "qwen3-embedding-0.6b" // 1024维, 8192 tokens
	ModelNameQwen3Embedding4B  ModelName = "qwen3-embedding-4b"   // 2560维, 8192 tokens
	ModelNameQwen3Embedding8B  ModelName = "qwen3-embedding-8b"   // 4096维, 8192 tokens

	// ========================================
	// 多模态向量模型
	// ========================================

	ModelNameGmeQwen2VL2B ModelName = "gme-qwen2-vl-2b-instruct" // 1536维, 支持文本+图片

	// ========================================
	// 重排序模型
	// ========================================

	ModelNameBCERerankerBase  ModelName = "bce-reranker-base"   // 基础重排序
	ModelNameQwen3Reranker06B ModelName = "qwen3-reranker-0.6b" // 0.6B重排序
	ModelNameQwen3Reranker4B  ModelName = "qwen3-reranker-4b"   // 4B重排序
	ModelNameQwen3Reranker8B  ModelName = "qwen3-reranker-8b"   // 8B重排序

	// ========================================
	// OCR模型
	// ========================================

	ModelNamePaddleOCRVL09B ModelName = "paddleocr-vl-0.9b" // PaddleOCR
	ModelNameDeepSeekOCR    ModelName = "deepseek-ocr"      // DeepSeek OCR

	// ========================================
	// 图像生成模型
	// ========================================

	ModelNameMuseSteamerAirImage ModelName = "musesteamer-air-image" // 百度蒸汽机Air图像生成
	ModelNameFLUX1Schnell        ModelName = "flux.1-schnell"        // FLUX快速图像生成
	ModelNameQwenImage           ModelName = "qwen-image"            // 通义图像生成
	ModelNameQwenImageEdit       ModelName = "qwen-image-edit"       // 通义图像编辑

	// ========================================
	// 兼容旧版本的别名 (Legacy aliases for backward compatibility)
	// ========================================

	// Deprecated: Use ModelNameERNIESpeed8K
	ModelNameERNIEBot ModelName = "ernie-speed-8k"
	// Deprecated: Use ModelNameERNIESpeed8K
	ModelNameERNIEBotTurbo ModelName = "ernie-speed-8k"
	// Deprecated: Use ModelNameERNIESpeed8K
	ModelNameERNIEBotPro ModelName = "ernie-speed-pro-128k"
	// Deprecated: Use ModelNameERNIE45Turbo128K
	ModelNameERNIEBot4 ModelName = "ernie-4.5-turbo-128k"
	// Deprecated: Use ModelNameERNIESpeed128K
	ModelNameERNIESpeed ModelName = "ernie-speed-128k"
	// Deprecated: Use ModelNameERNIESpeed8K
	ModelNameERNIESpeed128k ModelName = "ernie-speed-8k"
	// Deprecated: Use ModelNameERNIETiny8K
	ModelNameERNIETiny ModelName = "ernie-tiny-8k"
	// Deprecated: Use ModelNameERNIELite8K
	ModelNameERNIELite ModelName = "ernie-lite-8k"
	// Deprecated: Use ModelNameERNIELitePro128K
	ModelNameERNIELite8k ModelName = "ernie-lite-pro-128k"
	// Deprecated: Use ModelNameQwen330BA3B
	ModelNameERNIE3_5 ModelName = "qwen3-30b-a3b"
)

type options struct {
	apiKey           string
	modelName        ModelName
	httpClient       *http.Client
	callbacksHandler callbacks.Handler
	baseURL          string
}

// Option is a function that configures an LLM.
type Option func(*options)

// WithAPIKey sets the API key for the LLM.
func WithAPIKey(apiKey string) Option {
	return func(opts *options) {
		opts.apiKey = apiKey
	}
}

// WithModel sets the model name for the LLM.
func WithModel(model ModelName) Option {
	return func(opts *options) {
		opts.modelName = model
	}
}

// WithHTTPClient sets the HTTP client for the LLM.
func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) {
		opts.httpClient = client
	}
}

// WithCallbacks sets the callbacks handler for the LLM.
func WithCallbacks(handler callbacks.Handler) Option {
	return func(opts *options) {
		opts.callbacksHandler = handler
	}
}

// WithBaseURL sets the base URL for the LLM API.
// Default is "https://qianfan.baidubce.com".
func WithBaseURL(baseURL string) Option {
	return func(opts *options) {
		opts.baseURL = baseURL
	}
}

// getEnvOrDefault retrieves an environment variable or returns the default value.
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

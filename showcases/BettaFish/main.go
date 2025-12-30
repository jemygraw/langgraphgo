package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/forum_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/insight_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/media_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/query_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/report_engine"
	"github.com/smallnest/langgraphgo/showcases/BettaFish/schema"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <查询>")
		return
	}

	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("错误: 未设置 OPENAI_API_KEY 环境变量。")
	}
	if os.Getenv("TAVILY_API_KEY") == "" {
		log.Fatal("错误: 未设置 TAVILY_API_KEY 环境变量。")
	}
	// Optional: Check for API Base if using alternative providers (e.g., DeepSeek)
	if os.Getenv("OPENAI_API_BASE") != "" {
		fmt.Printf("使用自定义 API Base: %s\n", os.Getenv("OPENAI_API_BASE"))
	}
	if os.Getenv("OPENAI_MODEL") != "" {
		fmt.Printf("使用自定义模型: %s\n", os.Getenv("OPENAI_MODEL"))
	}

	query := os.Args[1]

	// Initialize state as map[string]any
	initialState := map[string]any{
		"query":      query,
		"paragraphs": make([]*schema.Paragraph, 0),
	}

	// Create graph using the custom state type
	workflow := graph.NewStateGraph[map[string]any]()

	// Wrap node functions to match typed signature
	wrapNode := func(fn func(ctx context.Context, state any) (any, error)) func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return func(ctx context.Context, s map[string]any) (map[string]any, error) {
			result, err := fn(ctx, s)
			if err != nil {
				return nil, err
			}
			if resultMap, ok := result.(map[string]any); ok {
				return resultMap, nil
			}
			return s, nil
		}
	}

	// Add nodes
	workflow.AddNode("query_engine", "Query analysis engine", wrapNode(query_engine.QueryEngineNode))
	workflow.AddNode("media_engine", "Media search engine", wrapNode(media_engine.MediaEngineNode))
	workflow.AddNode("insight_engine", "Insight generation engine", wrapNode(insight_engine.InsightEngineNode))
	workflow.AddNode("forum_engine", "Forum search engine", wrapNode(forum_engine.ForumEngineNode))
	workflow.AddNode("report_engine", "Report generation engine", wrapNode(report_engine.ReportEngineNode))

	// Add edges (Sequential for now)
	workflow.SetEntryPoint("query_engine")
	workflow.AddEdge("query_engine", "media_engine")
	workflow.AddEdge("media_engine", "insight_engine")
	workflow.AddEdge("insight_engine", "forum_engine")
	workflow.AddEdge("forum_engine", "report_engine")
	workflow.AddEdge("report_engine", graph.END)

	// Compile graph
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("编译图失败: %v", err)
	}

	// Run graph
	ctx := context.Background()
	finalState, err := app.Invoke(ctx, initialState)
	if err != nil {
		log.Fatalf("运行图失败: %v", err)
	}

	// Print result
	paragraphs, _ := finalState["paragraphs"].([]*schema.Paragraph)
	fmt.Println("\n=== 执行完成 ===")
	fmt.Printf("报告已生成，包含 %d 个段落。\n", len(paragraphs))
}

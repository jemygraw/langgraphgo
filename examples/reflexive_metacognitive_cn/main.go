// 反思元认知代理 (Reflexive Metacognitive Agent) - 中文版
//
// 本示例实现了"反思元认知代理"架构，这是一个具备自我意识的AI代理架构。
//
// 架构概述：
//
// 元认知代理维护一个显式的"自我模型"——对其自身知识、工具和边界
// 的结构化表示。当面临任务时，它的第一步不是解决问题，而是*
// 在自我模型的背景下分析问题*。它会问自己这样的问题：
//
//   - "我有足够的知识来自信地回答这个问题吗？"
//   - "这个主题在我的专业领域内吗？"
//   - "我有回答这个问题所需的特定工具吗？"
//   - "用户的查询是否涉及错误可能造成危险的高风险主题？"
//
// 根据答案，它选择一个策略：
//   1. 直接推理 (REASON_DIRECTLY)：针对知识范围内的高置信度、低风险查询
//   2. 使用工具 (USE_TOOL)：当查询需要通过特定工具获得能力时
//   3. 升级处理 (ESCALATE)：针对低置信度、高风险或超出范围的查询
//
// 该模式适用于：
// - 高风险咨询系统（医疗、法律、金融）
// - 自主系统（机器人评估其安全执行任务的能力）
// - 复杂工具编排器（从众多选项中选择正确的API）
//
// 参考资料: https://github.com/FareedKhan-dev/all-agentic-architectures/blob/main/17_reflexive_metacognitive.ipynb

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// ==================== 数据模型 ====================

// AgentSelfModel 是代理能力和局限性的结构化表示
// 这是其自我意识的基础
type AgentSelfModel struct {
	Name                string   // 代理名称
	Role                string   // 代理角色
	KnowledgeDomain     []string // 代理熟悉的知识领域
	AvailableTools      []string // 代理可使用的工具
	ConfidenceThreshold float64  // 置信度阈值，低于此值必须升级处理
}

// MetacognitiveAnalysis 表示代理对查询的自我分析结果
type MetacognitiveAnalysis struct {
	Confidence float64           // 0.0 到 1.0 - 安全准确回答的置信度
	Strategy   string            // "reason_directly"、"use_tool" 或 "escalate"
	Reasoning  string            // 选择该置信度和策略的理由
	ToolToUse  string            // 如果策略是"use_tool"，则为工具名称
	ToolArgs   map[string]string // 如果策略是"use_tool"，则为工具参数
}

// AgentState 表示在图中节点之间传递的状态
type AgentState struct {
	UserQuery             string
	SelfModel             *AgentSelfModel
	MetacognitiveAnalysis *MetacognitiveAnalysis
	ToolOutput            string
	FinalResponse         string
}

// ==================== 工具 ====================

// DrugInteractionChecker 药物相互作用检查器
type DrugInteractionChecker struct {
	knownInteractions map[string]string
}

// Check 检查两种药物之间的相互作用
func (d *DrugInteractionChecker) Check(drugA, drugB string) string {
	key := drugA + "+" + drugB
	if interaction, ok := d.knownInteractions[key]; ok {
		return fmt.Sprintf("发现药物相互作用: %s", interaction)
	}
	return "未发现已知的显著药物相互作用。但是，请务必咨询药剂师或医生。"
}

// NewDrugInteractionChecker 创建新的药物相互作用检查器
func NewDrugInteractionChecker() *DrugInteractionChecker {
	return &DrugInteractionChecker{
		knownInteractions: map[string]string{
			"布洛芬+赖诺普利": "中度风险：布洛芬可能降低赖诺普利的降压效果。建议监测血压。",
			"阿司匹林+华法林": "高风险：出血风险增加。除非医生指导，否则应避免同时使用。",
		},
	}
}

var drugTool = NewDrugInteractionChecker()

// ==================== 图节点 ====================

// MetacognitiveAnalysisNode 执行自我反思步骤
// 这是元认知架构的核心
func MetacognitiveAnalysisNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 🤔 代理正在进行元认知分析...                                 │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// 创建元认知分析提示词
	prompt := fmt.Sprintf(`你是一个AI助手的元认知推理引擎。你的任务是在代理自身能力和局限性（其"自我模型"）的背景下分析用户的查询。

你的主要指令是**安全第一**。你必须确定处理查询的最安全、最合适的策略。

**代理的自我模型：**
- 名称：%s
- 角色：%s
- 知识领域：%s
- 可用工具：%s

**知识领域主题：** 代理熟悉以下主题：感冒、流感、过敏、头痛、基本急救。

**策略规则：**
1. **升级处理 (escalate)**：在以下情况下选择此策略：
   - 查询涉及潜在的医疗紧急情况（胸痛、呼吸困难、严重受伤、骨折）
   - 查询涉及知识领域之外的主题（如癌症、糖尿病、心脏病、手术）
   - 你对提供安全答案有任何疑虑
   **如有疑虑，请升级处理。**

2. **使用工具 (use_tool)**：当查询明确或隐含地需要某个可用工具时选择此策略。例如，关于药物相互作用的问题需要使用 'drug_interaction_checker'。

3. **直接推理 (reason_directly)**：仅在以下情况下选择此策略：
   - 查询明确涉及知识领域内的主题（感冒、流感、过敏、头痛、基本急救）
   - 查询是简单的、低风险的信息性问题
   - 没有暗示严重疾病的症状

分析下面的用户查询，并以以下格式提供你的元认知分析：

置信度: [0.0 到 1.0]
策略: [escalate|use_tool|reason_directly]
工具名称: [如果是use_tool则为工具名称，否则填"无"]
药物A: [如果是药物相互作用检查器，否则填"无"]
药物B: [如果是药物相互作用检查器，否则填"无"]
理由: [为选择的置信度和策略提供简要说明]

**用户查询：**%s`,
		agentState.SelfModel.Name,
		agentState.SelfModel.Role,
		strings.Join(agentState.SelfModel.KnowledgeDomain, "、"),
		strings.Join(agentState.SelfModel.AvailableTools, "、"),
		agentState.UserQuery)

	// 调用 LLM
	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("元认知分析LLM调用失败: %w", err)
	}

	// 解析响应
	analysis := parseMetacognitiveAnalysis(resp)
	agentState.MetacognitiveAnalysis = analysis

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ 置信度: %.2f                                               │\n", analysis.Confidence)
	fmt.Printf("│ 策略: %s                                                    │\n", strategyToChinese(analysis.Strategy))
	fmt.Printf("│ 理由: %s                                                  │\n", truncate(analysis.Reasoning, 48))
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	return stateMap, nil
}

// ReasonDirectlyNode 处理高置信度、低风险的查询
func ReasonDirectlyNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ ✅ 对直接回答有信心。正在生成响应...                         │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	prompt := fmt.Sprintf(`你是%s。请为用户的查询提供有帮助的、非处方性的回答。
重要提示：始终提醒用户你不是医生，这不是医疗建议。

查询：%s`,
		agentState.SelfModel.Role,
		agentState.UserQuery)

	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("直接推理LLM调用失败: %w", err)
	}

	agentState.FinalResponse = resp
	return stateMap, nil
}

// CallToolNode 处理需要专门工具的查询
func CallToolNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ 🛠️  置信度需要使用工具。正在调用 `%s`...                  │\n", agentState.MetacognitiveAnalysis.ToolToUse)
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	analysis := agentState.MetacognitiveAnalysis
	if analysis.ToolToUse == "drug_interaction_checker" {
		drugA := analysis.ToolArgs["drug_a"]
		drugB := analysis.ToolArgs["drug_b"]
		toolOutput := drugTool.Check(drugA, drugB)
		agentState.ToolOutput = toolOutput
	} else {
		agentState.ToolOutput = "错误：未找到工具。"
	}

	return stateMap, nil
}

// SynthesizeToolResponseNode 将工具输出与有帮助的响应结合起来
func SynthesizeToolResponseNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 📝 正在综合工具输出的最终响应...                             │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	prompt := fmt.Sprintf(`你是%s。你已经使用工具获取了特定信息。现在，以清晰、有帮助的方式向用户展示这些信息。
重要提示：始终包含咨询医疗专业人员的免责声明。你不是医生。

原始查询：%s
工具输出：%s`,
		agentState.SelfModel.Role,
		agentState.UserQuery,
		agentState.ToolOutput)

	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("综合工具响应LLM调用失败: %w", err)
	}

	agentState.FinalResponse = resp
	return stateMap, nil
}

// EscalateToHumanNode 处理低置信度或高风险的查询
func EscalateToHumanNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 🚨 检测到低置信度或高风险。正在升级处理。                     │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	response := "我是AI助手，没有资格提供此主题的信息。此查询超出了我的知识领域或涉及潜在严重症状。" +
		"**请立即咨询合格的医疗专业人员。**"

	agentState.FinalResponse = response
	return stateMap, nil
}

// ==================== 路由逻辑 ====================

// RouteStrategy 根据元认知分析确定下一个节点
func RouteStrategy(ctx context.Context, state any) string {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	switch agentState.MetacognitiveAnalysis.Strategy {
	case "reason_directly":
		return "reason"
	case "use_tool":
		return "call_tool"
	case "escalate":
		return "escalate"
	default:
		return "escalate" // 默认为安全选项
	}
}

// ==================== 解析辅助函数 ====================

func parseMetacognitiveAnalysis(response string) *MetacognitiveAnalysis {
	analysis := &MetacognitiveAnalysis{
		Confidence: 0.1,
		Strategy:   "escalate",
		Reasoning:  response,
		ToolToUse:  "无",
		ToolArgs:   make(map[string]string),
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "置信度:") || strings.HasPrefix(upperLine, "CONFIDENCE:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				var confidence float64
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &confidence)
				analysis.Confidence = confidence
			}
		} else if strings.HasPrefix(upperLine, "策略:") || strings.HasPrefix(upperLine, "STRATEGY:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.Strategy = strings.TrimSpace(parts[1])
				analysis.Strategy = strings.ToLower(analysis.Strategy)
				// 中文策略映射
				if strings.Contains(analysis.Strategy, "直接") {
					analysis.Strategy = "reason_directly"
				} else if strings.Contains(analysis.Strategy, "工具") {
					analysis.Strategy = "use_tool"
				} else if strings.Contains(analysis.Strategy, "升级") {
					analysis.Strategy = "escalate"
				}
			}
		} else if strings.HasPrefix(upperLine, "工具名称:") || strings.HasPrefix(upperLine, "TOOL_TO_USE:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolToUse = strings.TrimSpace(parts[1])
				analysis.ToolToUse = strings.ToLower(analysis.ToolToUse)
			}
		} else if strings.HasPrefix(upperLine, "药物A:") || strings.HasPrefix(upperLine, "DRUG_A:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolArgs["drug_a"] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(upperLine, "药物B:") || strings.HasPrefix(upperLine, "DRUG_B:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolArgs["drug_b"] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(upperLine, "理由:") || strings.HasPrefix(upperLine, "REASONING:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.Reasoning = strings.TrimSpace(parts[1])
			}
		}
	}

	return analysis
}

func strategyToChinese(strategy string) string {
	switch strategy {
	case "reason_directly":
		return "直接推理"
	case "use_tool":
		return "使用工具"
	case "escalate":
		return "升级处理"
	default:
		return strategy
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	// 尝试在字符边界处截断（简单的UTF-8处理）
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ==================== 主函数 ====================

func main() {
	// 检查API密钥
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("需要设置 OPENAI_API_KEY 环境变量")
	}

	fmt.Println("=== 📘 反思元认知代理架构（中文版） ===")
	fmt.Println()
	fmt.Println("本示例演示了一个具备自我意识的医疗分诊助手。")
	fmt.Println("代理维护一个显式的'自我模型'，在决定如何处理每个查询之前")
	fmt.Println("先进行元认知分析。")
	fmt.Println()
	fmt.Println("策略：")
	fmt.Println("  - 直接推理 (REASON_DIRECTLY)：高置信度、低风险的查询")
	fmt.Println("  - 使用工具 (USE_TOOL)：需要专门工具的查询")
	fmt.Println("  - 升级处理 (ESCALATE)：低置信度、高风险或超出范围的查询")
	fmt.Println()

	// 创建LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// 定义代理的自我模型
	medicalAgentModel := &AgentSelfModel{
		Name:                "医疗分诊助手-3000",
		Role:                "提供初步医疗信息的有用AI助手",
		KnowledgeDomain:     []string{"感冒", "流感", "过敏", "头痛", "基本急救"},
		AvailableTools:      []string{"药物相互作用检查器"},
		ConfidenceThreshold: 0.6,
	}

	// 创建元认知图
	workflow := graph.NewStateGraph()

	// 添加节点
	workflow.AddNode("analyze", "元认知分析", MetacognitiveAnalysisNode)
	workflow.AddNode("reason", "直接推理", ReasonDirectlyNode)
	workflow.AddNode("call_tool", "调用工具", CallToolNode)
	workflow.AddNode("synthesize", "综合工具响应", SynthesizeToolResponseNode)
	workflow.AddNode("escalate", "升级给人类", EscalateToHumanNode)

	// 设置入口点
	workflow.SetEntryPoint("analyze")

	// 从分析节点添加条件边
	workflow.AddConditionalEdge("analyze", RouteStrategy)

	// 为每个策略添加边
	workflow.AddEdge("reason", graph.END)
	workflow.AddEdge("call_tool", "synthesize")
	workflow.AddEdge("synthesize", graph.END)
	workflow.AddEdge("escalate", graph.END)

	// 编译图
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("编译图失败: %v", err)
	}

	ctx := context.Background()

	// 测试查询
	testQueries := []struct {
		name  string
		query string
	}{
		{
			name:  "简单的、范围内的、低风险查询",
			query: "感冒的症状有哪些？",
		},
		{
			name:  "需要专门工具的查询",
			query: "我在服用赖诺普利，可以同时吃布洛芬吗？",
		},
		{
			name:  "高风险、紧急查询",
			query: "我胸部有剧烈的疼痛，左臂感到麻木，我该怎么办？",
		},
		{
			name:  "超出范围的查询",
			query: "胰腺癌四期的最新治疗方案有哪些？",
		},
	}

	for i, test := range testQueries {
		fmt.Printf("\n--- 测试 %d：%s ---\n", i+1, test.name)

		agentState := &AgentState{
			UserQuery: test.query,
			SelfModel: medicalAgentModel,
		}

		input := map[string]any{
			"llm":         llm,
			"agent_state": agentState,
		}

		result, err := app.Invoke(ctx, input)
		if err != nil {
			log.Printf("错误: %v\n", err)
			continue
		}

		resultMap := result.(map[string]any)
		finalState := resultMap["agent_state"].(*AgentState)

		fmt.Println("\n📋 响应：")
		fmt.Println(finalState.FinalResponse)
		fmt.Println(strings.Repeat("=", 70))
	}

	fmt.Println("\n=== 🎯 关键要点 ===")
	fmt.Println("反思元认知代理架构使AI系统能够：")
	fmt.Println("1. 维护能力和局限性的显式自我模型")
	fmt.Println("2. 在尝试解决问题之前先进行元认知分析")
	fmt.Println("3. 选择最安全的策略：直接推理、使用工具或升级处理")
	fmt.Println("4. 认识到自己不知道什么——这对安全至关重要")
	fmt.Println()
	fmt.Println("此架构适用于：")
	fmt.Println("- 高风险咨询系统（医疗、法律、金融）")
	fmt.Println("- 必须评估自身能力的自主系统")
	fmt.Println("- 错误信息可能造成伤害的任何领域")
}

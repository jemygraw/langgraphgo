// Reflexive Metacognitive Agent
//
// This example implements the "Reflexive Metacognitive Agent" architecture
// from the Agentic Architectures series by Fareed Khan.
//
// Architecture Overview:
//
// A metacognitive agent maintains an explicit "self-model" — a structured
// representation of its own knowledge, tools, and boundaries. When faced with
// a task, its first step is not to solve the problem, but to *analyze the
// problem in the context of its self-model*. It asks internal questions like:
//
//   - "Do I have sufficient knowledge to answer this confidently?"
//   - "Is this topic within my designated area of expertise?"
//   - "Do I have a specific tool that is required to answer this safely?"
//   - "Is the user's query about a high-stakes topic where an error would be dangerous?"
//
// Based on the answers, it chooses a strategy:
//   1. REASON_DIRECTLY: For high-confidence, low-risk queries within its knowledge
//   2. USE_TOOL: When the query requires a specific capability via a tool
//   3. ESCALATE: For low-confidence, high-risk, or out-of-scope queries
//
// This pattern is essential for:
// - High-Stakes Advisory Systems (healthcare, law, finance)
// - Autonomous Systems (robots assessing their ability to perform tasks safely)
// - Complex Tool Orchestrators (choosing the right API from many options)
//
// Reference: https://github.com/FareedKhan-dev/all-agentic-architectures/blob/main/17_reflexive_metacognitive.ipynb

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

// ==================== Data Models ====================

// AgentSelfModel is a structured representation of the agent's capabilities
// and limitations — the foundation of its self-awareness.
type AgentSelfModel struct {
	Name                string
	Role                string
	KnowledgeDomain     []string // Topics the agent is knowledgeable about
	AvailableTools      []string // Tools the agent can use
	ConfidenceThreshold float64  // Confidence below which the agent must escalate
}

// MetacognitiveAnalysis represents the agent's self-analysis of a query
type MetacognitiveAnalysis struct {
	Confidence float64           // 0.0 to 1.0 - confidence in ability to answer safely
	Strategy   string            // "reason_directly", "use_tool", or "escalate"
	Reasoning  string            // Justification for the chosen confidence and strategy
	ToolToUse  string            // If strategy is "use_tool", the name of the tool
	ToolArgs   map[string]string // If strategy is "use_tool", the arguments
}

// AgentState represents the state passed between nodes in the graph
type AgentState struct {
	UserQuery             string
	SelfModel             *AgentSelfModel
	MetacognitiveAnalysis *MetacognitiveAnalysis
	ToolOutput            string
	FinalResponse         string
}

// ==================== Tools ====================

// DrugInteractionChecker is a mock tool to check for drug interactions
type DrugInteractionChecker struct {
	knownInteractions map[string]string
}

// Check checks for interactions between two drugs
func (d *DrugInteractionChecker) Check(drugA, drugB string) string {
	key := drugA + "+" + drugB
	if interaction, ok := d.knownInteractions[key]; ok {
		return fmt.Sprintf("Interaction Found: %s", interaction)
	}
	return "No known significant interactions found. However, always consult a pharmacist or doctor."
}

// NewDrugInteractionChecker creates a new drug interaction checker
func NewDrugInteractionChecker() *DrugInteractionChecker {
	return &DrugInteractionChecker{
		knownInteractions: map[string]string{
			"ibuprofen+lisinopril": "Moderate risk: Ibuprofen may reduce the blood pressure-lowering effects of lisinopril. Monitor blood pressure.",
			"aspirin+warfarin":     "High risk: Increased risk of bleeding. This combination should be avoided unless directed by a doctor.",
		},
	}
}

var drugTool = NewDrugInteractionChecker()

// ==================== Graph Nodes ====================

// MetacognitiveAnalysisNode performs the self-reflection step
// This is the core of the metacognitive architecture
func MetacognitiveAnalysisNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("\n┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 🤔 Agent is performing metacognitive analysis...            │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	// Create prompt for metacognitive analysis
	prompt := fmt.Sprintf(`You are a metacognitive reasoning engine for an AI assistant. Your task is to analyze a user's query in the context of the agent's own capabilities and limitations (its 'self-model').

Your primary directive is **SAFETY**. You must determine the safest and most appropriate strategy for handling the query.

**Agent's Self-Model:**
- Name: %s
- Role: %s
- Knowledge Domain: %s
- Available Tools: %s

**Knowledge Domain Topics:** The agent is knowledgeable about: common_cold, influenza, allergies, headaches, basic_first_aid.

**Strategy Rules:**
1. **escalate**: Choose this strategy if:
   - The query involves a potential medical emergency (chest pain, difficulty breathing, severe injury, broken bones)
   - The query is about topics NOT in the knowledge domain (e.g., cancer, diabetes, heart disease, surgery)
   - You have ANY doubt about providing a safe answer
   **WHEN IN DOUBT, ESCALATE.**

2. **use_tool**: Choose this strategy if the query explicitly or implicitly requires one of the available tools. For example, a question about drug interactions requires the 'drug_interaction_checker'.

3. **reason_directly**: Choose this strategy ONLY if:
   - The query is clearly about topics in the knowledge domain (cold, flu, allergies, headaches, basic first aid)
   - The query is a simple, low-risk informational question
   - There are no symptoms suggesting a serious condition

Analyze the user query below and provide your metacognitive analysis in the following format:

CONFIDENCE: [0.0 to 1.0]
STRATEGY: [escalate|use_tool|reason_directly]
TOOL_TO_USE: [if use_tool, the tool name; otherwise "none"]
DRUG_A: [if drug_interaction_checker, or "none"]
DRUG_B: [if drug_interaction_checker, or "none"]
REASONING: [brief justification for the chosen confidence and strategy]

**User Query:** %s`,
		agentState.SelfModel.Name,
		agentState.SelfModel.Role,
		strings.Join(agentState.SelfModel.KnowledgeDomain, ", "),
		strings.Join(agentState.SelfModel.AvailableTools, ", "),
		agentState.UserQuery)

	// Call LLM
	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("metacognitive analysis LLM call failed: %w", err)
	}

	// Parse the response
	analysis := parseMetacognitiveAnalysis(resp)
	agentState.MetacognitiveAnalysis = analysis

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ Confidence: %.2f                                            │\n", analysis.Confidence)
	fmt.Printf("│ Strategy: %s                                                │\n", analysis.Strategy)
	fmt.Printf("│ Reasoning: %s                                              │\n", truncate(analysis.Reasoning, 50))
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	return stateMap, nil
}

// ReasonDirectlyNode handles high-confidence, low-risk queries
func ReasonDirectlyNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ ✅ Confident in direct answer. Generating response...       │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	prompt := fmt.Sprintf(`You are %s. Provide a helpful, non-prescriptive answer to the user's query.
IMPORTANT: Always remind the user that you are not a doctor and this is not medical advice.

Query: %s`,
		agentState.SelfModel.Role,
		agentState.UserQuery)

	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("reason directly LLM call failed: %w", err)
	}

	agentState.FinalResponse = resp
	return stateMap, nil
}

// CallToolNode handles queries that require specialized tools
func CallToolNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Printf("│ 🛠️  Confidence requires tool use. Calling `%s`...        │\n", agentState.MetacognitiveAnalysis.ToolToUse)
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	analysis := agentState.MetacognitiveAnalysis
	if analysis.ToolToUse == "drug_interaction_checker" {
		drugA := analysis.ToolArgs["drug_a"]
		drugB := analysis.ToolArgs["drug_b"]
		toolOutput := drugTool.Check(drugA, drugB)
		agentState.ToolOutput = toolOutput
	} else {
		agentState.ToolOutput = "Error: Tool not found."
	}

	return stateMap, nil
}

// SynthesizeToolResponseNode combines tool output with a helpful response
func SynthesizeToolResponseNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 📝 Synthesizing final response from tool output...          │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	prompt := fmt.Sprintf(`You are %s. You have used a tool to get specific information. Now, present this information to the user in a clear and helpful way.
IMPORTANT: ALWAYS include a disclaimer to consult a healthcare professional. You are not a doctor.

Original Query: %s
Tool Output: %s`,
		agentState.SelfModel.Role,
		agentState.UserQuery,
		agentState.ToolOutput)

	llm := stateMap["llm"].(llms.Model)
	resp, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return nil, fmt.Errorf("synthesize tool response LLM call failed: %w", err)
	}

	agentState.FinalResponse = resp
	return stateMap, nil
}

// EscalateToHumanNode handles low-confidence or high-risk queries
func EscalateToHumanNode(ctx context.Context, state any) (any, error) {
	stateMap := state.(map[string]any)
	agentState := stateMap["agent_state"].(*AgentState)

	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│ 🚨 Low confidence or high risk detected. Escalating.       │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")

	response := "I am an AI assistant and not qualified to provide information on this topic. " +
		"This query is outside my knowledge domain or involves potentially serious symptoms. " +
		"**Please consult a qualified medical professional immediately.**"

	agentState.FinalResponse = response
	return stateMap, nil
}

// ==================== Routing Logic ====================

// RouteStrategy determines the next node based on the metacognitive analysis
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
		return "escalate" // Default to safe option
	}
}

// ==================== Parsing Helpers ====================

func parseMetacognitiveAnalysis(response string) *MetacognitiveAnalysis {
	analysis := &MetacognitiveAnalysis{
		Confidence: 0.1,
		Strategy:   "escalate",
		Reasoning:  response,
		ToolToUse:  "none",
		ToolArgs:   make(map[string]string),
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		upperLine := strings.ToUpper(line)

		if strings.HasPrefix(upperLine, "CONFIDENCE:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				var confidence float64
				fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &confidence)
				analysis.Confidence = confidence
			}
		} else if strings.HasPrefix(upperLine, "STRATEGY:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.Strategy = strings.TrimSpace(parts[1])
				analysis.Strategy = strings.ToLower(analysis.Strategy)
			}
		} else if strings.HasPrefix(upperLine, "TOOL_TO_USE:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolToUse = strings.TrimSpace(parts[1])
				analysis.ToolToUse = strings.ToLower(analysis.ToolToUse)
			}
		} else if strings.HasPrefix(upperLine, "DRUG_A:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolArgs["drug_a"] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(upperLine, "DRUG_B:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.ToolArgs["drug_b"] = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(upperLine, "REASONING:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) > 1 {
				analysis.Reasoning = strings.TrimSpace(parts[1])
			}
		}
	}

	return analysis
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ==================== Main Function ====================

func main() {
	// Check for API key
	if os.Getenv("OPENAI_API_KEY") == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	fmt.Println("=== 📘 Reflexive Metacognitive Agent Architecture ===")
	fmt.Println()
	fmt.Println("This demo implements a medical triage assistant with self-awareness.")
	fmt.Println("The agent maintains an explicit 'self-model' and performs metacognitive")
	fmt.Println("analysis before deciding how to handle each query.")
	fmt.Println()
	fmt.Println("Strategies:")
	fmt.Println("  - REASON_DIRECTLY: High-confidence, low-risk queries")
	fmt.Println("  - USE_TOOL: Queries requiring specialized tools")
	fmt.Println("  - ESCALATE: Low-confidence, high-risk, or out-of-scope queries")
	fmt.Println()

	// Create LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatal(err)
	}

	// Define the agent's self-model
	medicalAgentModel := &AgentSelfModel{
		Name:                "TriageBot-3000",
		Role:                "A helpful AI assistant for providing preliminary medical information",
		KnowledgeDomain:     []string{"common_cold", "influenza", "allergies", "headaches", "basic_first_aid"},
		AvailableTools:      []string{"drug_interaction_checker"},
		ConfidenceThreshold: 0.6,
	}

	// Create the metacognitive graph
	workflow := graph.NewStateGraph()

	// Add nodes
	workflow.AddNode("analyze", "Metacognitive analysis", MetacognitiveAnalysisNode)
	workflow.AddNode("reason", "Reason directly", ReasonDirectlyNode)
	workflow.AddNode("call_tool", "Call tool", CallToolNode)
	workflow.AddNode("synthesize", "Synthesize tool response", SynthesizeToolResponseNode)
	workflow.AddNode("escalate", "Escalate to human", EscalateToHumanNode)

	// Set entry point
	workflow.SetEntryPoint("analyze")

	// Add conditional edges from analyze node
	workflow.AddConditionalEdge("analyze", RouteStrategy)

	// Add edges for each strategy
	workflow.AddEdge("reason", graph.END)
	workflow.AddEdge("call_tool", "synthesize")
	workflow.AddEdge("synthesize", graph.END)
	workflow.AddEdge("escalate", graph.END)

	// Compile the graph
	app, err := workflow.Compile()
	if err != nil {
		log.Fatalf("Failed to compile graph: %v", err)
	}

	ctx := context.Background()

	// Test queries
	testQueries := []struct {
		name  string
		query string
	}{
		{
			name:  "Simple, In-Scope, Low-Risk Query",
			query: "What are the symptoms of a common cold?",
		},
		{
			name:  "Specific Query Requiring a Tool",
			query: "Is it safe to take Ibuprofen if I am also taking Lisinopril?",
		},
		{
			name:  "High-Stakes, Emergency Query",
			query: "I have a crushing pain in my chest and my left arm feels numb, what should I do?",
		},
		{
			name:  "Out-of-Scope Query",
			query: "What are the latest treatment options for stage 4 pancreatic cancer?",
		},
	}

	for i, test := range testQueries {
		fmt.Printf("\n--- Test %d: %s ---\n", i+1, test.name)

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
			log.Printf("Error: %v\n", err)
			continue
		}

		resultMap := result.(map[string]any)
		finalState := resultMap["agent_state"].(*AgentState)

		fmt.Println("\n📋 Response:")
		fmt.Println(finalState.FinalResponse)
		fmt.Println(strings.Repeat("=", 70))
	}

	fmt.Println("\n=== 🎯 Key Takeaways ===")
	fmt.Println("The Reflexive Metacognitive Agent architecture enables AI systems to:")
	fmt.Println("1. Maintain an explicit self-model of capabilities and limitations")
	fmt.Println("2. Perform metacognitive analysis BEFORE attempting to solve problems")
	fmt.Println("3. Choose the safest strategy: reason directly, use tools, or escalate")
	fmt.Println("4. Recognize when they don't know something — critical for safety")
	fmt.Println()
	fmt.Println("This architecture is essential for:")
	fmt.Println("- High-stakes advisory systems (healthcare, law, finance)")
	fmt.Println("- Autonomous systems that must assess their own capabilities")
	fmt.Println("- Any domain where incorrect information could cause harm")
}

package prebuilt

import (
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// PlanningAgentState represents the state for a planning agent.
// The planning agent first generates a workflow plan using LLM,
// then executes according to the generated plan.
type PlanningAgentState struct {
	// Messages contains the conversation history
	Messages []llms.MessageContent

	// WorkflowPlan contains the parsed workflow plan from LLM
	WorkflowPlan *WorkflowPlan
}

// ReflectionAgentState represents the state for a reflection agent.
// The reflection agent iteratively improves its response through
// self-reflection and revision.
type ReflectionAgentState struct {
	// Messages contains the conversation history
	Messages []llms.MessageContent

	// Iteration counts the current iteration number
	Iteration int

	// Reflection contains the agent's self-reflection on its draft
	Reflection string

	// Draft contains the current draft response being refined
	Draft string
}

// PEVAgentState represents the state for a Plan-Execute-Verify agent.
// This agent follows a three-step process: plan, execute, and verify.
type PEVAgentState struct {
	// Messages contains the conversation history
	Messages []llms.MessageContent

	// Plan contains the list of steps to execute
	Plan []string

	// CurrentStep is the index of the current step being executed
	CurrentStep int

	// LastToolResult contains the result of the last tool execution
	LastToolResult string

	// IntermediateSteps contains results from intermediate steps
	IntermediateSteps []string

	// Retries counts the number of retries attempted
	Retries int

	// VerificationResult contains the verification status
	VerificationResult string

	// FinalAnswer contains the final answer after verification
	FinalAnswer string
}

// TreeOfThoughtsState represents the state for a tree-of-thoughts agent.
// This agent explores multiple reasoning paths in parallel to find
// the best solution.
type TreeOfThoughtsState struct {
	// ActivePaths contains all active reasoning paths being explored
	ActivePaths map[string]*SearchPath

	// Solution contains the best solution found so far
	Solution string

	// VisitedStates tracks which states have been visited to avoid cycles
	VisitedStates map[string]bool

	// Iteration counts the current iteration number
	Iteration int
}

// ChatAgentState represents the state for a chat agent.
// This is a conversational agent that maintains message history.
type ChatAgentState struct {
	// Messages contains the conversation history
	Messages []llms.MessageContent

	// SystemPrompt is the system prompt for the chat agent
	SystemPrompt string

	// ExtraTools contains additional tools available to the agent
	ExtraTools []tools.Tool
}

// AgentState represents the general agent state.
// This is the default state type for generic agents.
type AgentState struct {
	// Messages contains the conversation history
	Messages []llms.MessageContent

	// ExtraTools contains additional tools available to the agent
	ExtraTools []tools.Tool
}

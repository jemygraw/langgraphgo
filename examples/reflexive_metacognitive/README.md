# Reflexive Metacognitive Agent

This example demonstrates the **Reflexive Metacognitive Agent** architecture, which endows an AI agent with self-awareness by maintaining an explicit model of its own capabilities, knowledge boundaries, and confidence levels.

## Architecture Overview

A metacognitive agent maintains an explicit **"self-model"** — a structured representation of its own knowledge, tools, and boundaries. When faced with a task, its first step is not to solve the problem, but to *analyze the problem in the context of its self-model*.

### Metacognitive Questions

The agent asks itself internal questions like:
- Do I have sufficient knowledge to answer this confidently?
- Is this topic within my designated area of expertise?
- Do I have a specific tool that is required to answer this safely?
- Is the user's query about a high-stakes topic where an error would be dangerous?

### Strategy Selection

Based on the answers, it chooses one of three strategies:

1. **REASON_DIRECTLY**: For high-confidence, low-risk queries within its knowledge base
2. **USE_TOOL**: When the query requires a specific capability via a specialized tool
3. **ESCALATE**: For low-confidence, high-risk, or out-of-scope queries

## Use Cases

This architecture is essential for:

- **High-Stakes Advisory Systems**: Healthcare, law, finance — where agents must be able to say "I don't know"
- **Autonomous Systems**: Robots that must assess their own ability to perform physical tasks safely
- **Complex Tool Orchestrators**: Agents that must choose the right API from many options

## Strengths & Weaknesses

### Strengths
- **Enhanced Safety**: The agent is explicitly designed to avoid confident assertions in areas where it's not an expert
- **Improved Decision Making**: Forces a deliberate choice of strategy instead of naive, direct attempts

### Weaknesses
- **Complexity of Self-Model**: Defining and maintaining an accurate self-model can be complex
- **Metacognitive Overhead**: The initial analysis step adds latency and computational cost

## How to Run

```bash
cd examples/reflexive_metacognitive
go run .
```

Make sure you have set the `OPENAI_API_KEY` environment variable.

## Example Output

The demo runs four test cases:

1. **Simple Query** (cold symptoms) → Routes to `reason_directly`
2. **Drug Interaction Query** → Routes to `use_tool` with drug_interaction_checker
3. **Emergency Query** (chest pain) → Routes to `escalate` to human professional
4. **Out-of-Scope Query** (cancer treatment) → Routes to `escalate`

```
=== Test 1: Simple, In-Scope, Low-Risk Query ===
┌─────────────────────────────────────────────────────────────┐
│ 🤔 Agent is performing metacognitive analysis...            │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│ Confidence: 1.00                                            │
│ Strategy: reason_directly                                   │
│ Reasoning: The query is about symptoms of a common cold... │
└─────────────────────────────────────────────────────────────┘
```

## Architecture Diagram

```
                    User Query
                         │
                         ▼
              ┌─────────────────────┐
              │ Metacognitive       │
              │ Analysis Node       │
              │ (Self-Reflection)   │
              └─────────────────────┘
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
   ┌───────────┐  ┌───────────┐  ┌───────────┐
   │  Reason   │  │   Use     │  │ Escalate  │
   │ Directly  │  │   Tool    │  │  to Human │
   └───────────┘  └─────┬─────┘  └───────────┘
                        │
                        ▼
                 ┌─────────────┐
                 │  Synthesize │
                 │  Response   │
                 └─────────────┘
```

## Key Components

### AgentSelfModel
Defines what the agent knows and can do:
- **Knowledge Domain**: common_cold, influenza, allergies, headaches, basic_first_aid
- **Available Tools**: drug_interaction_checker
- **Confidence Threshold**: 0.6 (below this, must escalate)

### Metacognitive Analysis Node
The core of the architecture. Uses LLM to analyze:
- Query complexity and risk level
- Relevance to knowledge domain
- Tool requirements
- Confidence in providing a safe answer

### Strategy Routing
Conditional edge that directs flow based on the chosen strategy.

## Reference

Based on the Agentic Architectures series by Fareed Khan:
https://github.com/FareedKhan-dev/all-agentic-architectures/blob/main/17_reflexive_metacognitive.ipynb

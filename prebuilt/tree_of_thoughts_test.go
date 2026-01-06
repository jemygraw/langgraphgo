package prebuilt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock implementations for testing

type MockThoughtState struct {
	hash    string
	isValid bool
	isGoal  bool
	desc    string
}

func (m *MockThoughtState) IsValid() bool          { return m.isValid }
func (m *MockThoughtState) IsGoal() bool           { return m.isGoal }
func (m *MockThoughtState) GetDescription() string { return m.desc }
func (m *MockThoughtState) Hash() string           { return m.hash }

type MockThoughtGenerator struct {
	generateFunc func(ctx context.Context, current ThoughtState) ([]ThoughtState, error)
}

func (m *MockThoughtGenerator) Generate(ctx context.Context, current ThoughtState) ([]ThoughtState, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, current)
	}
	return []ThoughtState{
		&MockThoughtState{hash: "state1", isValid: true, isGoal: false, desc: "Thought 1"},
		&MockThoughtState{hash: "state2", isValid: true, isGoal: false, desc: "Thought 2"},
	}, nil
}

type MockThoughtEvaluator struct {
	evaluateFunc func(ctx context.Context, state ThoughtState, pathLength int) (float64, error)
}

func (m *MockThoughtEvaluator) Evaluate(ctx context.Context, state ThoughtState, pathLength int) (float64, error) {
	if m.evaluateFunc != nil {
		return m.evaluateFunc(ctx, state, pathLength)
	}
	return 0.5, nil
}

type MockFailingGenerator struct{}

func (m *MockFailingGenerator) Generate(ctx context.Context, current ThoughtState) ([]ThoughtState, error) {
	return nil, nil // Return empty slice to test edge cases
}

type MockInvalidStateGenerator struct{}

func (m *MockInvalidStateGenerator) Generate(ctx context.Context, current ThoughtState) ([]ThoughtState, error) {
	return []ThoughtState{
		&MockThoughtState{hash: "invalid", isValid: false, isGoal: false, desc: "Invalid"},
	}, nil
}

func TestCreateTreeOfThoughtsAgentMap(t *testing.T) {
	t.Run("Create agent with valid config", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
			MaxDepth:     3,
			MaxPaths:     2,
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create agent with goal as initial state", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "goal", isValid: true, isGoal: true, desc: "Goal"},
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create agent with default MaxDepth and MaxPaths", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
			// MaxDepth and MaxPaths are 0, should use defaults
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create agent with missing generator", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "generator")
	})

	t.Run("Create agent with missing evaluator", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "evaluator")
	})

	t.Run("Create agent with missing initial state", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator: &MockThoughtGenerator{},
			Evaluator: &MockThoughtEvaluator{},
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.Error(t, err)
		assert.Nil(t, agent)
		assert.Contains(t, err.Error(), "initial state")
	})

	t.Run("Create agent with verbose option", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
			Verbose:      true,
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})
}

func TestTreeOfThoughtsAgentMap_Execution(t *testing.T) {
	t.Run("Execute agent - finds goal immediately", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator: &MockThoughtGenerator{},
			Evaluator: &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "goal", isValid: true, isGoal: true,
				desc: "Goal reached"},
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)

		ctx := context.Background()
		result, err := agent.Invoke(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Execute agent - expands and evaluates", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Start"},
			MaxDepth:     2,
			MaxPaths:     3,
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)

		ctx := context.Background()
		result, err := agent.Invoke(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Execute agent with empty generator response", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockFailingGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Start"},
			MaxDepth:     2,
			MaxPaths:     3,
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)

		ctx := context.Background()
		result, err := agent.Invoke(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Execute agent with invalid states", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockInvalidStateGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Start"},
			MaxDepth:     2,
			MaxPaths:     3,
		}
		agent, err := CreateTreeOfThoughtsAgentMap(config)
		assert.NoError(t, err)

		ctx := context.Background()
		result, err := agent.Invoke(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestCreateTreeOfThoughtsAgent_Generic(t *testing.T) {
	type TOTState struct {
		ActivePaths map[string]*SearchPath `json:"active_paths"`
		Solution    string                 `json:"solution"`
		Visited     map[string]bool        `json:"visited"`
		Iteration   int                    `json:"iteration"`
	}

	t.Run("Create generic agent with valid config", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
			MaxDepth:     3,
			MaxPaths:     2,
		}

		agent, err := CreateTreeOfThoughtsAgent[TOTState](
			config,
			func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
			func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
			func(s TOTState) string { return s.Solution },
			func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
			func(s TOTState) map[string]bool { return s.Visited },
			func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
			func(s TOTState) int { return s.Iteration },
			func(s TOTState, i int) TOTState { s.Iteration = i; return s },
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create generic agent with goal state", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "goal", isValid: true, isGoal: true, desc: "Goal"},
		}

		agent, err := CreateTreeOfThoughtsAgent[TOTState](
			config,
			func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
			func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
			func(s TOTState) string { return s.Solution },
			func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
			func(s TOTState) map[string]bool { return s.Visited },
			func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
			func(s TOTState) int { return s.Iteration },
			func(s TOTState, i int) TOTState { s.Iteration = i; return s },
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create generic agent with default values", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
		}

		agent, err := CreateTreeOfThoughtsAgent[TOTState](
			config,
			func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
			func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
			func(s TOTState) string { return s.Solution },
			func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
			func(s TOTState) map[string]bool { return s.Visited },
			func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
			func(s TOTState) int { return s.Iteration },
			func(s TOTState, i int) TOTState { s.Iteration = i; return s },
		)
		assert.NoError(t, err)
		assert.NotNil(t, agent)
	})

	t.Run("Create generic agent with missing config", func(t *testing.T) {
		tests := []struct {
			name   string
			config TreeOfThoughtsConfig
			errMsg string
		}{
			{
				name: "missing generator",
				config: TreeOfThoughtsConfig{
					Evaluator:    &MockThoughtEvaluator{},
					InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
				},
				errMsg: "generator",
			},
			{
				name: "missing evaluator",
				config: TreeOfThoughtsConfig{
					Generator:    &MockThoughtGenerator{},
					InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
				},
				errMsg: "evaluator",
			},
			{
				name: "missing initial state",
				config: TreeOfThoughtsConfig{
					Generator: &MockThoughtGenerator{},
					Evaluator: &MockThoughtEvaluator{},
				},
				errMsg: "initial state",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				agent, err := CreateTreeOfThoughtsAgent[TOTState](
					tt.config,
					func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
					func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
					func(s TOTState) string { return s.Solution },
					func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
					func(s TOTState) map[string]bool { return s.Visited },
					func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
					func(s TOTState) int { return s.Iteration },
					func(s TOTState, i int) TOTState { s.Iteration = i; return s },
				)
				assert.Error(t, err)
				assert.Nil(t, agent)
				assert.Contains(t, err.Error(), tt.errMsg)
			})
		}
	})
}

func TestTreeOfThoughtsAgent_Generic_Execution(t *testing.T) {
	type TOTState struct {
		ActivePaths map[string]*SearchPath `json:"active_paths"`
		Solution    string                 `json:"solution"`
		Visited     map[string]bool        `json:"visited"`
		Iteration   int                    `json:"iteration"`
	}

	t.Run("Execute generic agent", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Start"},
			MaxDepth:     2,
			MaxPaths:     3,
		}

		agent, err := CreateTreeOfThoughtsAgent[TOTState](
			config,
			func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
			func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
			func(s TOTState) string { return s.Solution },
			func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
			func(s TOTState) map[string]bool { return s.Visited },
			func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
			func(s TOTState) int { return s.Iteration },
			func(s TOTState, i int) TOTState { s.Iteration = i; return s },
		)
		assert.NoError(t, err)

		ctx := context.Background()
		state := TOTState{}
		result, err := agent.Invoke(ctx, state)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("Execute generic agent with goal reached", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			InitialState: &MockThoughtState{hash: "goal", isValid: true, isGoal: true, desc: "Goal"},
		}

		agent, err := CreateTreeOfThoughtsAgent[TOTState](
			config,
			func(s TOTState) map[string]*SearchPath { return s.ActivePaths },
			func(s TOTState, p map[string]*SearchPath) TOTState { s.ActivePaths = p; return s },
			func(s TOTState) string { return s.Solution },
			func(s TOTState, sol string) TOTState { s.Solution = sol; return s },
			func(s TOTState) map[string]bool { return s.Visited },
			func(s TOTState, v map[string]bool) TOTState { s.Visited = v; return s },
			func(s TOTState) int { return s.Iteration },
			func(s TOTState, i int) TOTState { s.Iteration = i; return s },
		)
		assert.NoError(t, err)

		ctx := context.Background()
		state := TOTState{}
		result, err := agent.Invoke(ctx, state)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Solution)
	})
}

func TestSearchPath(t *testing.T) {
	t.Run("Create search path", func(t *testing.T) {
		states := []ThoughtState{
			&MockThoughtState{hash: "s1", isValid: true, isGoal: false, desc: "State 1"},
			&MockThoughtState{hash: "s2", isValid: true, isGoal: false, desc: "State 2"},
		}
		path := SearchPath{States: states, Score: 0.8}
		assert.NotNil(t, path.States)
		assert.Len(t, path.States, 2)
		assert.Equal(t, 0.8, path.Score)
	})

	t.Run("Search path with empty states", func(t *testing.T) {
		path := SearchPath{States: []ThoughtState{}, Score: 0}
		assert.NotNil(t, path.States)
		assert.Empty(t, path.States)
	})
}

func TestThoughtGenerator(t *testing.T) {
	t.Run("Custom generator with function", func(t *testing.T) {
		generator := &MockThoughtGenerator{
			generateFunc: func(ctx context.Context, current ThoughtState) ([]ThoughtState, error) {
				return []ThoughtState{
					&MockThoughtState{hash: "custom1", isValid: true, isGoal: false, desc: "Custom 1"},
				}, nil
			},
		}

		ctx := context.Background()
		current := &MockThoughtState{hash: "current", isValid: true, isGoal: false, desc: "Current"}
		states, err := generator.Generate(ctx, current)
		assert.NoError(t, err)
		assert.Len(t, states, 1)
		assert.Equal(t, "custom1", states[0].Hash())
	})
}

func TestThoughtEvaluator(t *testing.T) {
	t.Run("Custom evaluator with function", func(t *testing.T) {
		evaluator := &MockThoughtEvaluator{
			evaluateFunc: func(ctx context.Context, state ThoughtState, pathLength int) (float64, error) {
				return 0.95, nil
			},
		}

		ctx := context.Background()
		state := &MockThoughtState{hash: "test", isValid: true, isGoal: false, desc: "Test"}
		score, err := evaluator.Evaluate(ctx, state, 3)
		assert.NoError(t, err)
		assert.Equal(t, 0.95, score)
	})

	t.Run("Default evaluator", func(t *testing.T) {
		evaluator := &MockThoughtEvaluator{}

		ctx := context.Background()
		state := &MockThoughtState{hash: "test", isValid: true, isGoal: false, desc: "Test"}
		score, err := evaluator.Evaluate(ctx, state, 1)
		assert.NoError(t, err)
		assert.Equal(t, 0.5, score)
	})
}

func TestThoughtState(t *testing.T) {
	t.Run("MockThoughtState methods", func(t *testing.T) {
		state := &MockThoughtState{
			hash:    "test-hash",
			isValid: true,
			isGoal:  false,
			desc:    "Test description",
		}

		assert.True(t, state.IsValid())
		assert.False(t, state.IsGoal())
		assert.Equal(t, "Test description", state.GetDescription())
		assert.Equal(t, "test-hash", state.Hash())
	})

	t.Run("Invalid state", func(t *testing.T) {
		state := &MockThoughtState{
			hash:    "invalid",
			isValid: false,
			isGoal:  false,
			desc:    "Invalid state",
		}

		assert.False(t, state.IsValid())
		assert.False(t, state.IsGoal())
	})

	t.Run("Goal state", func(t *testing.T) {
		state := &MockThoughtState{
			hash:    "goal",
			isValid: true,
			isGoal:  true,
			desc:    "Goal reached",
		}

		assert.True(t, state.IsValid())
		assert.True(t, state.IsGoal())
	})
}

func TestTreeOfThoughtsConfig(t *testing.T) {
	t.Run("Config with all fields", func(t *testing.T) {
		config := TreeOfThoughtsConfig{
			Generator:    &MockThoughtGenerator{},
			Evaluator:    &MockThoughtEvaluator{},
			MaxDepth:     5,
			MaxPaths:     10,
			Verbose:      true,
			InitialState: &MockThoughtState{hash: "init", isValid: true, isGoal: false, desc: "Initial"},
		}

		assert.NotNil(t, config.Generator)
		assert.NotNil(t, config.Evaluator)
		assert.Equal(t, 5, config.MaxDepth)
		assert.Equal(t, 10, config.MaxPaths)
		assert.True(t, config.Verbose)
		assert.NotNil(t, config.InitialState)
	})
}

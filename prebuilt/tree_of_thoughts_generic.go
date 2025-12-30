package prebuilt

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/log"
)

// CreateTreeOfThoughtsAgentTyped creates a Tree of Thoughts search agent with full type safety.
//
// Tree of Thoughts (ToT) is a search-based reasoning framework where problem-solving
// is modeled as a search through a tree. At each step, multiple candidate "thoughts"
// are generated, evaluated for feasibility, and the most promising branches are expanded
// while unpromising ones are pruned.
//
// The ToT pattern involves:
// 1. Decomposition: Break down the problem into steps
// 2. Thought Generation: Generate multiple possible next steps (branches)
// 3. State Evaluation: Evaluate each thought for validity and promise
// 4. Pruning & Expansion: Remove bad branches, expand good ones
// 5. Solution: Continue until a goal state is reached
func CreateTreeOfThoughtsAgentTyped(config TreeOfThoughtsConfig) (*graph.StateRunnable[TreeOfThoughtsState], error) {
	if config.Generator == nil {
		return nil, fmt.Errorf("generator is required")
	}

	if config.Evaluator == nil {
		return nil, fmt.Errorf("evaluator is required")
	}

	if config.InitialState == nil {
		return nil, fmt.Errorf("initial state is required")
	}

	if config.MaxDepth == 0 {
		config.MaxDepth = 10 // Default max depth
	}

	if config.MaxPaths == 0 {
		config.MaxPaths = 5 // Default max active paths
	}

	// Create the workflow with generic state type
	workflow := graph.NewStateGraph[TreeOfThoughtsState]()

	// Define state schema for merging
	schema := graph.NewStructSchema(
		TreeOfThoughtsState{},
		func(current, new TreeOfThoughtsState) (TreeOfThoughtsState, error) {
			// Merge active paths
			if new.ActivePaths != nil {
				current.ActivePaths = new.ActivePaths
			}
			// Overwrite other fields
			if new.Solution != "" {
				current.Solution = new.Solution
			}
			if new.VisitedStates != nil {
				current.VisitedStates = new.VisitedStates
			}
			current.Iteration = new.Iteration
			return current, nil
		},
	)
	workflow.SetSchema(schema)

	// Add initialize node
	workflow.AddNode("initialize", "Initialize search with starting state", func(ctx context.Context, state TreeOfThoughtsState) (TreeOfThoughtsState, error) {
		return initializeNodeTyped(ctx, state, config)
	})

	// Add expand node
	workflow.AddNode("expand", "Expand active paths by generating new thoughts", func(ctx context.Context, state TreeOfThoughtsState) (TreeOfThoughtsState, error) {
		return expandNodeTyped(ctx, state, config)
	})

	// Add evaluate node
	workflow.AddNode("evaluate", "Evaluate and prune paths", func(ctx context.Context, state TreeOfThoughtsState) (TreeOfThoughtsState, error) {
		return evaluateNodeTyped(ctx, state, config)
	})

	// Set entry point
	workflow.SetEntryPoint("initialize")

	// Add edges
	workflow.AddEdge("initialize", "expand")
	workflow.AddConditionalEdge("expand", func(ctx context.Context, state TreeOfThoughtsState) string {
		return routeAfterExpandTyped(state, config)
	})
	workflow.AddConditionalEdge("evaluate", func(ctx context.Context, state TreeOfThoughtsState) string {
		return routeAfterEvaluateTyped(state, config)
	})

	return workflow.Compile()
}

// initializeNodeTyped sets up the initial search state (typed version)
func initializeNodeTyped(ctx context.Context, state TreeOfThoughtsState, config TreeOfThoughtsConfig) (TreeOfThoughtsState, error) {
	if config.Verbose {
		log.Info("initializing Tree of Thoughts search")
		log.Info("initial state: %s\n", config.InitialState.GetDescription())
	}

	// Create initial path
	initialPath := SearchPath{
		States: []ThoughtState{config.InitialState},
		Score:  0,
	}

	activePaths := make(map[string]*SearchPath)
	activePaths["initial"] = &initialPath

	visited := make(map[string]bool)
	visited[config.InitialState.Hash()] = true

	return TreeOfThoughtsState{
		ActivePaths:   activePaths,
		Solution:      "",
		VisitedStates: visited,
		Iteration:     0,
	}, nil
}

// expandNodeTyped generates new thoughts from active paths (typed version)
func expandNodeTyped(ctx context.Context, state TreeOfThoughtsState, config TreeOfThoughtsConfig) (TreeOfThoughtsState, error) {
	if len(state.ActivePaths) == 0 {
		return state, fmt.Errorf("no active paths to expand")
	}

	if config.Verbose {
		log.Info("iteration %d: expanding %d active paths", state.Iteration+1, len(state.ActivePaths))
	}

	newPaths := make(map[string]*SearchPath)
	visitedStates := state.VisitedStates
	if visitedStates == nil {
		visitedStates = make(map[string]bool)
	}

	// Expand each active path
	for pathID, path := range state.ActivePaths {
		if path == nil || len(path.States) == 0 {
			continue
		}

		currentState := path.States[len(path.States)-1]

		// Check if already at goal
		if currentState.IsGoal() {
			if config.Verbose {
				log.Info("path %s reached goal!", pathID)
			}
			// Build solution string from path
			solution := buildSolutionString(path)
			return TreeOfThoughtsState{
				ActivePaths:   state.ActivePaths,
				Solution:      solution,
				VisitedStates: state.VisitedStates,
				Iteration:     state.Iteration,
			}, nil
		}

		// Check max depth
		if len(path.States) >= config.MaxDepth {
			if config.Verbose {
				log.Warn("path %s reached max depth, skipping", pathID)
			}
			continue
		}

		// Generate next states
		nextStates, err := config.Generator.Generate(ctx, currentState)
		if err != nil {
			if config.Verbose {
				log.Warn("error generating next states for path %s: %v", pathID, err)
			}
			continue
		}

		if config.Verbose {
			log.Info("  path %s: generated %d candidate states", pathID, len(nextStates))
		}

		// Create new paths for each valid next state
		for i, nextState := range nextStates {
			// Skip if invalid
			if !nextState.IsValid() {
				continue
			}

			// Skip if already visited (cycle detection)
			hash := nextState.Hash()
			if visitedStates[hash] {
				continue
			}

			// Create new path
			newPathStates := make([]ThoughtState, len(path.States))
			copy(newPathStates, path.States)
			newPathStates = append(newPathStates, nextState)

			newPath := &SearchPath{
				States: newPathStates,
				Score:  0, // Will be evaluated in next step
			}

			newPathID := fmt.Sprintf("%s-%d", pathID, i)
			newPaths[newPathID] = newPath
			visitedStates[hash] = true
		}
	}

	if config.Verbose {
		log.Info("  Total new paths generated: %d\n", len(newPaths))
	}

	return TreeOfThoughtsState{
		ActivePaths:   newPaths,
		Solution:      state.Solution,
		VisitedStates: visitedStates,
		Iteration:     state.Iteration + 1,
	}, nil
}

// evaluateNodeTyped evaluates and prunes paths (typed version)
func evaluateNodeTyped(ctx context.Context, state TreeOfThoughtsState, config TreeOfThoughtsConfig) (TreeOfThoughtsState, error) {
	if len(state.ActivePaths) == 0 {
		return TreeOfThoughtsState{
			ActivePaths:   make(map[string]*SearchPath),
			Solution:      state.Solution,
			VisitedStates: state.VisitedStates,
			Iteration:     state.Iteration,
		}, nil
	}

	if config.Verbose {
		log.Info("evaluating %d paths", len(state.ActivePaths))
	}

	// Evaluate each path and convert to slice for sorting
	pathSlice := make([]*SearchPath, 0, len(state.ActivePaths))
	for _, path := range state.ActivePaths {
		if path == nil || len(path.States) == 0 {
			continue
		}

		lastState := path.States[len(path.States)-1]
		score, err := config.Evaluator.Evaluate(ctx, lastState, len(path.States))
		if err != nil {
			if config.Verbose {
				log.Warn("error evaluating path: %v", err)
			}
			score = -1
		}
		path.Score = score
		pathSlice = append(pathSlice, path)
	}

	// Prune paths with negative scores
	var prunedPaths []*SearchPath
	for _, path := range pathSlice {
		if path.Score >= 0 {
			prunedPaths = append(prunedPaths, path)
		}
	}

	if config.Verbose {
		log.Info("  pruned %d paths with negative scores", len(pathSlice)-len(prunedPaths))
	}

	// Keep only top MaxPaths paths
	if len(prunedPaths) > config.MaxPaths {
		// Sort by score (descending) using simple bubble sort
		for i := 0; i < len(prunedPaths)-1; i++ {
			for j := i + 1; j < len(prunedPaths); j++ {
				if prunedPaths[j].Score > prunedPaths[i].Score {
					prunedPaths[i], prunedPaths[j] = prunedPaths[j], prunedPaths[i]
				}
			}
		}
		prunedPaths = prunedPaths[:config.MaxPaths]

		if config.Verbose {
			log.Info("  kept top %d paths", config.MaxPaths)
		}
	}

	// Convert back to map
	resultPaths := make(map[string]*SearchPath)
	for i, path := range prunedPaths {
		resultPaths[fmt.Sprintf("path-%d", i)] = path
	}

	if config.Verbose {
		log.Info("  active paths remaining: %d\n", len(resultPaths))
	}

	return TreeOfThoughtsState{
		ActivePaths:   resultPaths,
		Solution:      state.Solution,
		VisitedStates: state.VisitedStates,
		Iteration:     state.Iteration,
	}, nil
}

// Routing functions (typed versions)

func routeAfterExpandTyped(state TreeOfThoughtsState, config TreeOfThoughtsConfig) string {
	// Check if solution found
	if state.Solution != "" {
		if config.Verbose {
			log.Info("solution found!")
		}
		return graph.END
	}

	// Check if any active paths remain
	if len(state.ActivePaths) == 0 {
		if config.Verbose {
			log.Error("no more paths to explore")
		}
		return graph.END
	}

	// Check iteration limit
	if state.Iteration >= config.MaxDepth {
		if config.Verbose {
			log.Warn("reached max iterations (%d)", config.MaxDepth)
		}
		return graph.END
	}

	// Continue to evaluation
	return "evaluate"
}

func routeAfterEvaluateTyped(state TreeOfThoughtsState, config TreeOfThoughtsConfig) string {
	// Check if any active paths remain
	if len(state.ActivePaths) == 0 {
		if config.Verbose {
			log.Error("no paths remaining after pruning")
		}
		return graph.END
	}

	// Continue expanding
	return "expand"
}

// Helper function to build solution string from path
func buildSolutionString(path *SearchPath) string {
	if path == nil || len(path.States) == 0 {
		return ""
	}

	var result string
	result += fmt.Sprintf("=== Solution Found ===\n")
	result += fmt.Sprintf("Path length: %d steps\n\n", len(path.States)-1)

	for i, s := range path.States {
		if i == 0 {
			result += fmt.Sprintf("Start: %s\n", s.GetDescription())
		} else {
			result += fmt.Sprintf("Step %d: %s\n", i, s.GetDescription())
		}
	}

	result += fmt.Sprintf("======================")
	return result
}

// Helper function to print solution
func PrintSolutionTyped(solution string) {
	if solution == "" {
		log.Info("no solution found")
		return
	}

	log.Info(solution)
}

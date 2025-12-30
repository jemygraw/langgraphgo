package graph

import (
	"context"

	g "github.com/smallnest/langgraphgo/graph"
)

// Helper functions to convert between map[string]any and *State
func mapToState(m map[string]any) *State {
	s := &State{}
	if v, ok := m["username"].(string); ok {
		s.Username = v
	}
	if v, ok := m["user_id"].(string); ok {
		s.UserID = v
	}
	if v, ok := m["tavily_search_data"].(string); ok {
		s.TavilySearchData = v
	}
	if v, ok := m["social_data"].([]Result); ok {
		s.SocialData = v
	}
	if v, ok := m["profile_data"].(string); ok {
		s.ProfileData = v
	}
	if v, ok := m["profile_text"].(string); ok {
		s.ProfileText = v
	}
	if v, ok := m["log_chan"].(chan string); ok {
		s.LogChan = v
	}
	return s
}

func stateToMap(s *State) map[string]any {
	return map[string]any{
		"username":           s.Username,
		"user_id":            s.UserID,
		"tavily_search_data": s.TavilySearchData,
		"social_data":        s.SocialData,
		"profile_data":       s.ProfileData,
		"profile_text":       s.ProfileText,
		"log_chan":           s.LogChan,
	}
}

func NewGraph() (*g.StateRunnable[map[string]any], error) {
	workflow := g.NewStateGraph[map[string]any]()

	// Wrap node functions to convert between map[string]any and *State
	workflow.AddNode("account", "提取用户ID", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		s := mapToState(state)
		result, err := AccountNode(ctx, s)
		if err != nil {
			return nil, err
		}
		if resultState, ok := result.(*State); ok {
			return stateToMap(resultState), nil
		}
		return state, nil
	})
	workflow.AddNode("search", "搜索社交资料", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		s := mapToState(state)
		result, err := SearchNode(ctx, s)
		if err != nil {
			return nil, err
		}
		if resultState, ok := result.(*State); ok {
			return stateToMap(resultState), nil
		}
		return state, nil
	})
	workflow.AddNode("profile", "生成用户画像", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		s := mapToState(state)
		result, err := ProfileNode(ctx, s)
		if err != nil {
			return nil, err
		}
		if resultState, ok := result.(*State); ok {
			return stateToMap(resultState), nil
		}
		return state, nil
	})

	workflow.SetEntryPoint("account")
	workflow.AddEdge("account", "search")
	workflow.AddEdge("search", "profile")
	workflow.AddEdge("profile", g.END)

	return workflow.Compile()
}

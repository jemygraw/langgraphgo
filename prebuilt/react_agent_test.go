package prebuilt

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/tools"
)

// MockLLM implements llms.Model for testing
type MockLLM struct {
	responses []llms.ContentResponse
	callCount int
}

func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.callCount >= len(m.responses) {
		return &llms.ContentResponse{
			Choices: []*llms.ContentChoice{
				{Content: "No more responses"},
			},
		}, nil
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return &resp, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return "", nil
}

// WeatherTool implements tools.Tool for testing
type WeatherTool struct {
	currentTemp int
}

func NewWeatherTool(temp int) *WeatherTool {
	return &WeatherTool{currentTemp: temp}
}

func (t *WeatherTool) Name() string {
	return "get_weather"
}

func (t *WeatherTool) Description() string {
	return "Get the current weather information for a given location"
}

func (t *WeatherTool) Call(ctx context.Context, input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("location is required")
	}

	// Simulate weather data based on location
	weatherData := map[string]string{
		"beijing":   fmt.Sprintf("北京天气: %d°C, 晴天", t.currentTemp),
		"shanghai":  fmt.Sprintf("上海天气: %d°C, 多云", t.currentTemp-2),
		"guangzhou": fmt.Sprintf("广州天气: %d°C, 阴天", t.currentTemp+5),
		"shenzhen":  fmt.Sprintf("深圳天气: %d°C, 小雨", t.currentTemp+3),
		"hangzhou":  fmt.Sprintf("杭州天气: %d°C, 晴天", t.currentTemp-1),
		"chengdu":   fmt.Sprintf("成都天气: %d°C, 雾霾", t.currentTemp-3),
		"wuhan":     fmt.Sprintf("武汉天气: %d°C, 晴天", t.currentTemp),
		"xian":      fmt.Sprintf("西安天气: %d°C, 多云", t.currentTemp-4),
		"new york":  fmt.Sprintf("New York weather: %d°F, sunny", t.currentTemp*2+32),
		"london":    fmt.Sprintf("London weather: %d°C, cloudy", t.currentTemp-10),
		"tokyo":     fmt.Sprintf("Tokyo weather: %d°C, rainy", t.currentTemp-5),
		"paris":     fmt.Sprintf("Paris weather: %d°C, partly cloudy", t.currentTemp-8),
	}

	if weather, ok := weatherData[input]; ok {
		return weather, nil
	}

	return fmt.Sprintf("%s: %d°C, data not available", input, t.currentTemp), nil
}

func TestReactAgentWithWeatherTool(t *testing.T) {
	// Setup Weather Tool
	weatherTool := NewWeatherTool(25)

	// Setup Mock LLM for weather query
	// 1. First call: calls get_weather tool
	// 2. Second call: provides weather summary
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "weather-call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "beijing"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "根据查询结果，北京当前天气为25°C，晴天。",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{weatherTool}, 5)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "北京现在的天气怎么样？"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	// Expected messages:
	// 0: Human query about Beijing weather
	// 1: AI with tool call
	// 2: Tool response with weather data
	// 3: AI with final weather summary
	assert.Equal(t, 4, len(messages))
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)
	assert.Equal(t, llms.ChatMessageTypeTool, messages[2].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[3].Role)

	// Verify Tool Response
	toolMsg := messages[2]
	assert.Equal(t, 1, len(toolMsg.Parts))
	toolResp, ok := toolMsg.Parts[0].(llms.ToolCallResponse)
	assert.True(t, ok)
	assert.Equal(t, "weather-call-1", toolResp.ToolCallID)
	assert.Equal(t, "get_weather", toolResp.Name)
	assert.Contains(t, toolResp.Content, "北京天气: 25°C, 晴天")

	// Verify Final Answer
	finalMsg := messages[3]
	assert.Equal(t, 1, len(finalMsg.Parts))
	textPart, ok := finalMsg.Parts[0].(llms.TextContent)
	assert.True(t, ok)
	assert.Contains(t, textPart.Text, "25°C")
	assert.Contains(t, textPart.Text, "晴天")
}

func TestReactAgentWithMultipleWeatherQueries(t *testing.T) {
	// Setup Weather Tool
	weatherTool := NewWeatherTool(20)

	// Setup Mock LLM for multiple weather queries
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			// First query: Shanghai weather
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "weather-call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "shanghai"}`,
								},
							},
						},
					},
				},
			},
			// Second query: Shenzhen weather
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "weather-call-2",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "shenzhen"}`,
								},
							},
						},
					},
				},
			},
			// Final summary
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "上海18°C多云，深圳23°C小雨。深圳比上海暖和5度，但有小雨。",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{weatherTool}, 10)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "请查询上海和深圳的天气，并比较一下"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	// Expected 6 messages:
	// 0: Human query
	// 1: AI with Shanghai weather tool call
	// 2: Tool response for Shanghai
	// 3: AI with Shenzhen weather tool call
	// 4: Tool response for Shenzhen
	// 5: AI with final comparison
	assert.Equal(t, 6, len(messages))

	// Verify Shanghai tool response
	shanghaiToolMsg := messages[2]
	toolResp1 := shanghaiToolMsg.Parts[0].(llms.ToolCallResponse)
	assert.Equal(t, "weather-call-1", toolResp1.ToolCallID)
	assert.Contains(t, toolResp1.Content, "上海天气: 18°C, 多云")

	// Verify Shenzhen tool response
	shenzhenToolMsg := messages[4]
	toolResp2 := shenzhenToolMsg.Parts[0].(llms.ToolCallResponse)
	assert.Equal(t, "weather-call-2", toolResp2.ToolCallID)
	assert.Contains(t, toolResp2.Content, "深圳天气: 23°C, 小雨")

	// Verify final comparison
	finalMsg := messages[5]
	textPart := finalMsg.Parts[0].(llms.TextContent)
	assert.Contains(t, textPart.Text, "上海")
	assert.Contains(t, textPart.Text, "深圳")
}

func TestReactAgentWithInternationalWeather(t *testing.T) {
	// Setup Weather Tool
	weatherTool := NewWeatherTool(15)

	// Setup Mock LLM for international weather query
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "weather-call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "london"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "weather-call-2",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "new york"}`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "London: 5°C cloudy, New York: 62°F sunny. New York is warmer than London.",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{weatherTool}, 10)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "What's the weather in London and New York?"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	assert.Equal(t, 6, len(messages))

	// Verify London response in Celsius
	londonToolMsg := messages[2]
	londonResp := londonToolMsg.Parts[0].(llms.ToolCallResponse)
	assert.Contains(t, londonResp.Content, "London weather: 5°C")

	// Verify New York response in Fahrenheit
	nyToolMsg := messages[4]
	nyResp := nyToolMsg.Parts[0].(llms.ToolCallResponse)
	assert.Contains(t, nyResp.Content, "New York weather: 62°F")
}

func TestReactAgentMaxIterations(t *testing.T) {
	// Setup Weather Tool
	weatherTool := NewWeatherTool(10)

	// Setup Mock LLM that provides a final answer after tool calls
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			// Tool call
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `{"input": "beijing"}`,
								},
							},
						},
					},
				},
			},
			// Final answer
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "北京当前天气为10°C，晴天。",
					},
				},
			},
		},
	}

	// Create Agent with max iterations of 5 (should not be reached)
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{weatherTool}, 5)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "北京天气怎么样？"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	// Should have exactly 4 messages: Human -> AI(toolcall) -> Tool -> AI(answer)
	assert.Equal(t, 4, len(messages))
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)
	assert.Equal(t, llms.ChatMessageTypeTool, messages[2].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[3].Role)

	// Verify tool call was made
	toolCallMsg := messages[1]
	assert.Equal(t, 1, len(toolCallMsg.Parts))
	toolCall := toolCallMsg.Parts[0].(llms.ToolCall)
	assert.Equal(t, "get_weather", toolCall.FunctionCall.Name)

	// Verify tool response
	toolResponseMsg := messages[2]
	toolResponse := toolResponseMsg.Parts[0].(llms.ToolCallResponse)
	assert.Contains(t, toolResponse.Content, "北京天气: 10°C")

	// Verify final answer
	finalMsg := messages[3]
	finalText := finalMsg.Parts[0].(llms.TextContent)
	assert.Contains(t, finalText.Text, "10°C")
}

func TestReactAgentDirectAnswer(t *testing.T) {
	// Setup Weather Tool (won't be used)
	weatherTool := NewWeatherTool(25)

	// Setup Mock LLM that answers directly without tools
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "我无法实时查询天气信息，建议您查看天气应用或网站获取最新信息。",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{weatherTool}, 5)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "你能告诉我天气吗？"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	// Expected 2 messages:
	// 0: Human query
	// 1: AI direct answer (no tool calls)
	assert.Equal(t, 2, len(messages))
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)

	// Verify no tool calls were made
	aiMsg := messages[1]
	for _, part := range aiMsg.Parts {
		_, isToolCall := part.(llms.ToolCall)
		assert.False(t, isToolCall, "Expected no tool calls in direct answer")
	}
}

func TestReactAgentErrorHandling(t *testing.T) {
	// Weather Tool that returns an error
	errorWeatherTool := &WeatherTool{currentTemp: 25}

	// Setup Mock LLM with invalid JSON arguments
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "error-call",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "get_weather",
									Arguments: `invalid-json`,
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "天气查询出现问题，请重试。",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{errorWeatherTool}, 5)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "查询天气"),
		},
	}

	// Run Agent - should handle the error gracefully
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	// Should have 4 messages even with error
	assert.Equal(t, 4, len(messages))
	assert.Equal(t, llms.ChatMessageTypeHuman, messages[0].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[1].Role)
	assert.Equal(t, llms.ChatMessageTypeTool, messages[2].Role)
	assert.Equal(t, llms.ChatMessageTypeAI, messages[3].Role)
}

func TestCreateReactAgent(t *testing.T) {
	// Test with original MockTool for backward compatibility
	mockTool := &MockTool{name: "legacy-test-tool"}

	// Setup Mock LLM
	mockLLM := &MockLLM{
		responses: []llms.ContentResponse{
			{
				Choices: []*llms.ContentChoice{
					{
						ToolCalls: []llms.ToolCall{
							{
								ID:   "legacy-call-1",
								Type: "function",
								FunctionCall: &llms.FunctionCall{
									Name:      "legacy-test-tool",
									Arguments: "test-input",
								},
							},
						},
					},
				},
			},
			{
				Choices: []*llms.ContentChoice{
					{
						Content: "Legacy test completed",
					},
				},
			},
		},
	}

	// Create Agent
	agent, err := CreateReactAgent(mockLLM, []tools.Tool{mockTool}, 3)
	assert.NoError(t, err)

	// Initial State
	initialState := map[string]any{
		"messages": []llms.MessageContent{
			llms.TextParts(llms.ChatMessageTypeHuman, "Run legacy test"),
		},
	}

	// Run Agent
	res, err := agent.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	// Verify Result
	messages := res["messages"].([]llms.MessageContent)

	assert.Equal(t, 4, len(messages))
	assert.Equal(t, "Legacy test completed", messages[3].Parts[0].(llms.TextContent).Text)
}

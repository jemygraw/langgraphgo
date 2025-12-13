// Package goskills provides an adapter for integrating GoSkills with LangGraph Go agents.
//
// GoSkills is a framework for defining and executing skills in Go. This adapter allows
// GoSkills-defined skills to be used as tools within LangGraph agents, enabling agents
// to execute Go code, shell commands, and custom operations safely.
//
// # Core Components
//
// ## SkillTool
// The main adapter that wraps GoSkills operations as LangChain-compatible tools:
//
//	import (
//		"github.com/smallnest/langgraphgo/adapter/goskills"
//		"github.com/smallnest/langgraphgo/prebuilt"
//	)
//
//	// Load skills from a directory
//	skills, err := goskills.LoadSkillsFromDir("/path/to/skills")
//	if err != nil {
//		return err
//	}
//
//	// Convert skills to LangChain tools
//	tools, err := goskills.ConvertToLangChainTools(skills)
//	if err != nil {
//		return err
//	}
//
//	// Use with ReAct agent
//	agent, err := prebuilt.CreateReactAgent(llm, tools, 10)
//
// # Available Skills
//
// The adapter provides built-in skills for common operations:
//
// ## Shell Code Execution
// Execute shell code with arguments:
//
//	tool := &goskills.SkillTool{
//		name: "run_shell_code",
//	}
//
//	result, err := tool.Call(ctx, `{
//		"code": "echo $1 $2",
//		"args": {"Hello": "World"}
//	}`)
//
// ## Shell Script Execution
// Execute existing shell scripts:
//
//	tool := &goskills.SkillTool{
//		name: "run_shell_script",
//	}
//
//	result, err := tool.Call(ctx, `{
//		"scriptPath": "/path/to/script.sh",
//		"args": ["arg1", "arg2"]
//	}`)
//
// ## Python Code Execution
// Execute Python code with imports:
//
//	tool := &goskills.SkillTool{
//		name: "run_python_code",
//	}
//
//	result, err := tool.Call(ctx, `{
//		"code": "import math; print(math.sqrt(16))",
//		"imports": ["math", "numpy"],
//		"globals": {"value": 42}
//	}`)
//
// ## Python Script Execution
// Execute Python scripts:
//
//	tool := &goskills.SkillTool{
//		name: "run_python_script",
//	}
//
//	result, err := tool.Call(ctx, `{
//		"scriptPath": "/path/to/script.py",
//		"args": ["--input", "data.txt"]
//	}`)
//
// ## Web Search
// Perform web searches:
//
//	tool := &goskills.SkillTool{
//		name: "web_search",
//	}
//
//	result, err := tool.Call(ctx, `{
//		"query": "latest AI developments",
//		"num_results": 5
//	}`)
//
// ## File Operations
// Read and write files:
//
//	tool := &goskills.SkillTool{
//		name: "file_operations",
//	}
//
//	// Read file
//	result, err := tool.Call(ctx, `{
//		"action": "read",
//		"path": "/path/to/file.txt"
//	}`)
//
//	// Write file
//	result, err := tool.Call(ctx, `{
//		"action": "write",
//		"path": "/path/to/output.txt",
//		"content": "Hello, World!"
//	}`)
//
// # Custom Skills
//
// Define custom Go skills for specific tasks:
//
//	// custom_skill.go
//	package main
//
//	import (
//		"fmt"
//		"github.com/smallnest/goskills/skill"
//	)
//
//	type MySkill struct{}
//
//	func (s *MySkill) Execute(ctx skill.Context) (any, error) {
//		// Extract parameters
//		input := ctx.Params["input"].(string)
//
//		// Custom logic
//		result := fmt.Sprintf("Processed: %s", strings.ToUpper(input))
//
//		return result, nil
//	}
//
//	func NewMySkill() *MySkill {
//		return &MySkill{}
//	}
//
// Register the skill:
//
//	skills := []goskills.Skill{
//		goskills.NewSkill("my_custom_skill", "Custom processing skill", NewMySkill),
//	}
//
// # Integration Examples
//
// ## With ReAct Agent
//
//	// Load skills
//	skills, _ := goskills.LoadSkillsFromDir("./skills")
//
//	// Convert to tools
//	langchainTools, _ := goskills.ConvertToLangChainTools(skills)
//
//	// Create agent
//	agent, _ := prebuilt.CreateReactAgent(llm, langchainTools, 15)
//
//	// Execute
//	result, _ := agent.Invoke(ctx, map[string]any{
//		"messages": []llms.MessageContent{
//			{
//				Role: llms.ChatMessageTypeHuman,
//				Parts: []llms.ContentPart{
//					llms.TextPart("Analyze the data in data.csv and create a plot"),
//				},
//			},
//		},
//	})
//
// ## With PTC Agent
//
//	ptcTools, _ := goskills.ConvertToLangChainTools(skills)
//
//	ptcAgent, _ := ptc.CreatePTCAgent(ptc.PTCAgentConfig{
//		Model:    llm,
//		Tools:    ptcTools,
//		Language: ptc.LanguagePython,
//	})
//
// # Skill Configuration
//
// Skills can be configured with parameters:
//
//	type SkillConfig struct {
//		Name        string            `json:"name"`
//		Description string            `json:"description"`
//		Parameters  map[string]any    `json:"parameters"`
//		Timeout     time.Duration     `json:"timeout"`
//		Retry       int               `json:"retry"`
//		Env         map[string]string `json:"env"`
//	}
//
//	// Create configured skill
//	skill := goskills.NewSkillWithConfig(SkillConfig{
//		Name:        "data_processor",
//		Description: "Process large datasets",
//		Parameters: map[string]any{
//			"batch_size": 1000,
//			"format":    "json",
//		},
//		Timeout: 30 * time.Second,
//		Env: map[string]string{
//			"DATA_PATH": "/data",
//		},
//	})
//
// # Error Handling
//
// The adapter provides structured error handling:
//
//	type SkillError struct {
//		Code      string `json:"code"`
//		Message   string `json:"message"`
//		Skill     string `json:"skill"`
//		Timestamp string `json:"timestamp"`
//	}
//
//	result, err := tool.Call(ctx, input)
//	if err != nil {
//		var skillErr *SkillError
//		if errors.As(err, &skillErr) {
//			fmt.Printf("Skill %s failed: %s\n", skillErr.Skill, skillErr.Message)
//		}
//	}
//
// # Security Features
//
//   - Sandboxed execution environments
//   - Resource limits (CPU, memory, time)
//   - Input validation and sanitization
//   - Restricted file system access
//   - Network access controls
//   - Audit logging
//
// # Performance Optimization
//
//   - Skill caching for reuse
//   - Parallel execution support
//   - Connection pooling for external services
//   - Result streaming for large outputs
//   - Memory management for long-running operations
//
// # Best Practices
//
//  1. Organize skills by functionality
//  2. Provide clear descriptions and examples
//  3. Implement proper error handling
//  4. Use timeouts for long operations
//  5. Validate all inputs
//  6. Log skill executions for debugging
//  7. Test skills with various inputs
//  8. Document skill parameters and return values
//
// # Advanced Features
//
// ## Skill Composition
// Combine multiple skills for complex operations:
//
//	type CompositeSkill struct {
//		skills []goskills.Skill
//	}
//
//	func (s *CompositeSkill) Execute(ctx skill.Context) (any, error) {
//		// Execute skills in sequence
//		for _, sk := range s.skills {
//			result, err := sk.Execute(ctx)
//			if err != nil {
//				return nil, err
//			}
//			ctx.Params["previous_result"] = result
//		}
//		return ctx.Params["previous_result"], nil
//	}
//
// ## Dynamic Skill Loading
// Load skills from multiple sources:
//
//	// From directory
//	dirSkills, _ := goskills.LoadSkillsFromDir("./skills")
//
//	// From remote repository
//	remoteSkills, _ := goskills.LoadSkillsFromRepo("github.com/user/skills")
//
//	// From configuration
//	configSkills, _ := goskills.LoadSkillsFromConfig("./skills.yaml")
//
//	// Combine all skills
//	allSkills := append(dirSkills, remoteSkills...)
//	allSkills = append(allSkills, configSkills...)
//
// # Monitoring and Debugging
//
// Skills include built-in monitoring:
//
//	// Enable metrics collection
//	goskills.EnableMetrics()
//
//	// Get skill statistics
//	stats := goskills.GetSkillStats()
//	fmt.Printf("Total executions: %d\n", stats.Total)
//	fmt.Printf("Success rate: %.2f%%\n", stats.SuccessRate)
//
//	// Trace skill execution
//	trace := goskills.TraceSkill("my_skill")
//	defer trace.Finish()
//
// # Integration with External Services
//
// Skills can integrate with external APIs:
//
//	type APISkill struct {
//		client *http.Client
//		apiKey string
//		baseURL string
//	}
//
//	func (s *APISkill) Execute(ctx skill.Context) (any, error) {
//		// Make API call
//		req, _ := http.NewRequest(
//			"GET",
//			s.baseURL + "/endpoint",
//			nil,
//		)
//		req.Header.Set("Authorization", "Bearer "+s.apiKey)
//
//		resp, err := s.client.Do(req)
//		if err != nil {
//			return nil, err
//		}
//		defer resp.Body.Close()
//
//		// Process response
//		var result map[string]any
//		json.NewDecoder(resp.Body).Decode(&result)
//
//		return result, nil
//	}
package goskills

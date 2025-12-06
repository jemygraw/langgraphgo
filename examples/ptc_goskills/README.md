# PTC + GoSkills Integration Example

This example demonstrates how to use **goskills tools** with **PTC (Programmatic Tool Calling)**.

## Architecture

```
┌─────────────┐
│     LLM     │  Generates Python/Go code
└──────┬──────┘
       │
       v
┌─────────────────┐
│  PTC Executor   │  Executes generated code
└──────┬──────────┘
       │
       v
┌─────────────────┐
│  Tool Server    │  HTTP server (internal)
└──────┬──────────┘
       │
       v
┌─────────────────┐
│ goskills Adapter│  Converts to tools.Tool
└──────┬──────────┘
       │
       v
┌─────────────────┐
│ goskills Tools  │  Local tool execution
└──────┬──────────┘
       │
       v
┌─────────────────┐
│  exec.Command   │  Shell/Python/etc.
└─────────────────┘
```

## How It Works

### 1. **goskills Provides Local Tools**

goskills implements various local tools:
- **Shell execution**: `run_shell_code`, `run_shell_script`
- **Python execution**: `run_python_code`, `run_python_script`
- **File operations**: `read_file`, `write_file`
- **Web search**: `duckduckgo_search`, `wikipedia_search`, `tavily_search`
- **Web fetching**: `web_fetch`

Each tool uses Go's `exec.Command` to run commands locally:
```go
// From goskills/tool/shell_tool.go
cmd := exec.Command("bash", scriptPath, args...)
output, err := cmd.CombinedOutput()
```

### 2. **goskills Adapter Wraps Tools**

The adapter (`adapter/goskills/goskills.go`) implements the `tools.Tool` interface:

```go
func (t *SkillTool) Call(ctx context.Context, input string) (string, error) {
    switch t.name {
    case "run_shell_code":
        shellTool := tool.ShellTool{}
        return shellTool.Run(params.Args, params.Code)

    case "run_python_code":
        pythonTool := tool.PythonTool{}
        return pythonTool.Run(params.Args, params.Code)

    // ... other tools
    }
}
```

### 3. **PTC Executes LLM-Generated Code**

When the LLM generates code like:

```python
# Write a file
result = write_file({
    "filePath": "test.txt",
    "content": "Hello from PTC!"
})

# Read it back
content = read_file({"filePath": "test.txt"})
print(content)
```

PTC will:
1. Execute this Python code
2. Call `write_file()` and `read_file()` functions
3. These functions make HTTP requests to the tool server
4. The tool server calls the goskills adapter
5. The adapter calls the actual goskills tool
6. The tool executes locally using `exec.Command`

## Key Benefits

### ✅ **Real Local Execution**
- Not placeholders or mocks
- Actual shell/Python/file operations
- Direct system access (sandboxed by PTC)

### ✅ **Flexible Tool System**
- Use any `tools.Tool` implementation
- Mix goskills with custom tools
- Easy to extend

### ✅ **Safety & Isolation**
- PTC provides execution sandboxing
- Tool server adds security boundary
- Timeout and error handling built-in

### ✅ **LLM-Friendly**
- LLM generates code that calls tools
- Reduces API round-trips (up to 10x faster)
- Natural programming interface

## Usage

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="your-api-key"

# Run the example
go run main.go
```

## Example Output

```
Loaded 10 tools from goskills:
  - run_shell_code: Executes a shell code snippet
  - run_python_code: Executes a Python code snippet
  - read_file: Reads content from a file
  - write_file: Writes content to a file
  - duckduckgo_search: Searches the web using DuckDuckGo
  ...

Query: Create a file named 'test.txt' with content 'Hello from PTC + goskills!' and then read it back.

Processing...

=== Execution Results ===

[Message 1 - human]
Create a file named 'test.txt' with content 'Hello from PTC + goskills!' and then read it back.

[Message 2 - ai]
```python
# Write the file
write_result = write_file({
    "filePath": "test.txt",
    "content": "Hello from PTC + goskills!"
})

# Read it back
content = read_file({"filePath": "test.txt"})
print(f"File content: {content}")
```

[Message 3 - human]
[Code Execution Result]
Successfully wrote to file: test.txt
File content: Hello from PTC + goskills!

[Message 4 - ai]
I've successfully created the file 'test.txt' and verified its content. The file contains: "Hello from PTC + goskills!"
```

## Comparison: ModeDirect vs ModeServer

Both modes use goskills tools, but differ in server exposure:

| Feature | ModeDirect | ModeServer |
|---------|-----------|------------|
| Tool Server | Internal (hidden) | Exposed to user code |
| Startup | Automatic | Automatic |
| Use Case | **Recommended** | Advanced scenarios |
| Performance | Excellent | Excellent |

**Recommendation**: Use `ModeDirect` for most cases - it's simpler and the server is managed automatically.

## Available GoSkills Tools

After loading with `goskills.SkillsToTools()`:

1. **`run_shell_code`** - Execute shell commands
2. **`run_shell_script`** - Run shell script files
3. **`run_python_code`** - Execute Python code
4. **`run_python_script`** - Run Python script files
5. **`read_file`** - Read file contents
6. **`write_file`** - Write to files
7. **`duckduckgo_search`** - Web search via DuckDuckGo
8. **`wikipedia_search`** - Search Wikipedia
9. **`tavily_search`** - Search using Tavily API
10. **`web_fetch`** - Fetch web page content

## See Also

- [PTC README](../../ptc/README.md) - PTC documentation
- [goskills](https://github.com/smallnest/goskills) - goskills repository
- [LangGraphGo Examples](../) - More examples

# 时间旅行 (Time Travel) 示例

本示例演示 LangGraphGo 使用 `GetState` 和 `UpdateState` 的 **时间旅行** 能力。

## 1. 背景

**时间旅行** 允许您在任何时间点检查和修改图执行的状态。这对于以下场景非常有用：
- **调试**: 检查特定步骤的状态。
- **修正**: 修复不正确的状态（例如，更正用户输入或工具输出）并恢复执行。
- **分支**: 创建替代执行路径（“如果...会怎样”场景）。

## 2. 核心概念

- **GetState**: 获取线程的当前状态快照，包括值和配置。
- **UpdateState**: 修改线程的状态，创建一个新的检查点。这实际上“分叉”了历史记录。

## 3. 工作原理

1.  **初始运行**: 我们运行一个简单的图，它会增加计数器。
2.  **获取状态**: 我们使用 `GetState` 检查最终状态。
3.  **更新状态**: 我们使用 `UpdateState` 手动将计数器设置为新值（例如 10），模拟修正或分支。
4.  **恢复**: 然后我们可以从这个新状态恢复执行（此处通过检查恢复的状态来演示）。

## 4. 代码亮点

### 获取状态
```go
snapshot, err := app.GetState(ctx, nil)
fmt.Printf("Current State: %v\n", snapshot.Values)
```

### 更新状态
```go
newValues := map[string]interface{}{"count": 10}
// 更新状态，就好像我们在 "step_1" 一样
newConfig, err := app.UpdateState(ctx, nil, newValues, "step_1")
```

## 5. 运行示例

```bash
go run main.go
```

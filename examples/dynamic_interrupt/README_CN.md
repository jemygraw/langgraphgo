# 动态中断 (Dynamic Interrupt) 示例

本示例演示 LangGraphGo 中的 **动态中断 (Dynamic Interrupt)** 功能，类似于 Python LangGraph 的 `interrupt` 函数。

## 1. 背景

动态中断允许节点在运行时暂停执行并等待外部输入（例如来自人类的输入）。与静态的 `InterruptBefore` 配置不同，这允许在节点逻辑内部做出中断决定。

## 2. 核心概念

- **`graph.Interrupt(ctx, value)`**: 一个可以在节点内部调用的函数。
  - 如果是第一次调用，它会停止执行并返回包含 `value` 的 `GraphInterrupt` 错误。
  - 如果在恢复期间调用（提供了 `ResumeValue`），它会返回 `ResumeValue` 并继续执行。
- **`Config.ResumeValue`**: 恢复执行时提供的值，该值将由 `graph.Interrupt` 返回。

## 3. 工作原理

1.  **初始运行**: 图运行直到遇到 `graph.Interrupt("What is your name?")`。
2.  **中断**: `Invoke` 方法返回一个 `GraphInterrupt` 错误。我们捕获这个错误并查看查询内容。
3.  **用户输入**: 我们模拟获取用户输入（"Alice"）。
4.  **恢复**: 我们再次调用 `Invoke`，但这次在配置中传递 `ResumeValue: "Alice"`。
5.  **重放**: 节点再次运行。`graph.Interrupt` 看到恢复值并立即返回 "Alice"。节点继续执行直至完成。

## 4. 代码亮点

### 节点内部
```go
name, err := graph.Interrupt(ctx, "What is your name?")
if err != nil {
    return nil, err // 传播中断
}
// 使用输入
return fmt.Sprintf("Hello, %v!", name), nil
```

### 处理中断
```go
_, err := runnable.Invoke(ctx, nil)
if interrupt, ok := err.(*graph.GraphInterrupt); ok {
    // 获取用户输入...
    config := &graph.Config{ResumeValue: userInput}
    // 恢复
    runnable.InvokeWithConfig(ctx, nil, config)
}
```

## 5. 运行示例

```bash
go run main.go
```

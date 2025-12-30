package graph

import (
	"context"
	"errors"
	"fmt"
)

// END is a special constant used to represent the end node in the graph.
const END = "END"

var (
	// ErrEntryPointNotSet is returned when the entry point of the graph is not set.
	ErrEntryPointNotSet = errors.New("entry point not set")

	// ErrNodeNotFound is returned when a node is not found in the graph.
	ErrNodeNotFound = errors.New("node not found")

	// ErrNoOutgoingEdge is returned when no outgoing edge is found for a node.
	ErrNoOutgoingEdge = errors.New("no outgoing edge found for node")
)

// GraphInterrupt is returned when execution is interrupted by configuration or dynamic interrupt
type GraphInterrupt struct {
	// Node that caused the interruption
	Node string
	// State at the time of interruption
	State any
	// NextNodes that would have been executed if not interrupted
	NextNodes []string
	// InterruptValue is the value provided by the dynamic interrupt (if any)
	InterruptValue any
}

func (e *GraphInterrupt) Error() string {
	if e.InterruptValue != nil {
		return fmt.Sprintf("graph interrupted at node %s with value: %v", e.Node, e.InterruptValue)
	}
	return fmt.Sprintf("graph interrupted at node %s", e.Node)
}

// Interrupt pauses execution and waits for input.
// If resuming, it returns the value provided in the resume command.
func Interrupt(ctx context.Context, value any) (any, error) {
	if resumeVal := GetResumeValue(ctx); resumeVal != nil {
		return resumeVal, nil
	}
	return nil, &NodeInterrupt{Value: value}
}

// Node represents a node in the graph.
type Node struct {
	// Name is the unique identifier for the node.
	Name string

	// Description describes the functionality of the node.
	Description string

	// Function is the function associated with the node.
	// It takes a context and any state as input and returns the updated state and an error.
	Function func(ctx context.Context, state any) (any, error)
}

// Edge represents an edge in the graph.
type Edge struct {
	// From is the name of the node from which the edge originates.
	From string

	// To is the name of the node to which the edge points.
	To string
}

// StateMerger merges multiple state updates into a single state.
type StateMerger func(ctx context.Context, currentState any, newStates []any) (any, error)

// RetryPolicy defines how to handle node failures
type RetryPolicy struct {
	MaxRetries      int
	BackoffStrategy BackoffStrategy
	RetryableErrors []string
}

// BackoffStrategy defines different backoff strategies
type BackoffStrategy int

const (
	FixedBackoff BackoffStrategy = iota
	ExponentialBackoff
	LinearBackoff
)

// Runnable is an alias for StateRunnable[map[string]any] for convenience.
type Runnable = StateRunnable[map[string]any]

// StateGraphMap is an alias for StateGraph[map[string]any] for convenience.
// Use NewStateGraph[map[string]any]() or NewStateGraph[S]() for other types.
type StateGraphMap = StateGraph[map[string]any]

// ListenableStateGraphMap is an alias for ListenableStateGraphTyped[map[string]any].
type ListenableStateGraphMap = ListenableStateGraphTyped[map[string]any]

// ListenableRunnableMap is an alias for ListenableRunnableTyped[map[string]any].
type ListenableRunnableMap = ListenableRunnableTyped[map[string]any]

// StateGraphUntyped is a wrapper around StateGraph[map[string]any] with convenience methods for untyped functions.
// Deprecated: Use StateGraph[S] or StateGraphMap directly
type StateGraphUntyped struct {
	*StateGraphMap
}

// NewStateGraphUntyped creates a new StateGraph with map[string]any state type
// Deprecated: Use NewStateGraph[map[string]any]() or NewStateGraph[S]() instead
func NewStateGraphUntyped() *StateGraphUntyped {
	return &StateGraphUntyped{
		StateGraphMap: NewStateGraph[map[string]any](),
	}
}

// ListenableStateGraphUntyped is a wrapper around ListenableStateGraphTyped[map[string]any] with convenience methods.
// Deprecated: Use ListenableStateGraphTyped[S] or ListenableStateGraphMap directly
type ListenableStateGraphUntyped struct {
	*ListenableStateGraphMap
}

// NewListenableStateGraphUntyped creates a new listenable state graph
// Deprecated: Use NewListenableStateGraphTyped[map[string]any]() instead
func NewListenableStateGraphUntyped() *ListenableStateGraphUntyped {
	return &ListenableStateGraphUntyped{
		ListenableStateGraphMap: NewListenableStateGraphTyped[map[string]any](),
	}
}

// StateRunnableUntyped is an alias for Runnable
// Deprecated: Use StateRunnable[S] or Runnable directly
type StateRunnableUntyped = Runnable

// ListenableRunnable is an alias for ListenableRunnableMap
// Deprecated: Use ListenableRunnableTyped[S] or ListenableRunnableMap directly
type ListenableRunnable = ListenableRunnableMap

// MapSchemaAdapter adapts a MapSchema to StateSchemaTyped[map[string]any]
type MapSchemaAdapter struct {
	Schema *MapSchema
}

// Init returns the initial state
func (a *MapSchemaAdapter) Init() map[string]any {
	result := a.Schema.Init()
	if result == nil {
		return make(map[string]any)
	}
	if m, ok := result.(map[string]any); ok {
		return m
	}
	return make(map[string]any)
}

// Update merges the new state into the current state
func (a *MapSchemaAdapter) Update(current, new map[string]any) (map[string]any, error) {
	result, err := a.Schema.Update(current, new)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return make(map[string]any), nil
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return current, nil
}

// NewMessageGraph creates a new instance of StateGraph[map[string]any] with a default schema
// that handles "messages" using the AddMessages reducer.
// This is the recommended constructor for chat-based agents that use
// map[string]any as state with a "messages" key.
//
// Deprecated: Use NewStateGraph[MessageState]() for type-safe state management.
func NewMessageGraph() *StateGraph[map[string]any] {
	g := NewStateGraph[map[string]any]()

	// Initialize default schema for message handling
	schema := NewMapSchema()
	schema.RegisterReducer("messages", AddMessages)

	// Wrap in adapter to match StateSchemaTyped[map[string]any]
	adapter := &MapSchemaAdapter{Schema: schema}
	g.SetSchema(adapter)

	return g
}

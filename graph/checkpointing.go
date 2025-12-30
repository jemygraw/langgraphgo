package graph

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"
	"github.com/smallnest/langgraphgo/store"
	"github.com/smallnest/langgraphgo/store/file"
	"github.com/smallnest/langgraphgo/store/memory"
)

// Checkpoint represents a saved state at a specific point in execution
type Checkpoint = store.Checkpoint

// CheckpointStore defines the interface for checkpoint persistence
type CheckpointStore = store.CheckpointStore

// NewMemoryCheckpointStore creates a new in-memory checkpoint store
func NewMemoryCheckpointStore() CheckpointStore {
	return memory.NewMemoryCheckpointStore()
}

// NewFileCheckpointStore creates a new file-based checkpoint store
func NewFileCheckpointStore(path string) (CheckpointStore, error) {
	return file.NewFileCheckpointStore(path)
}

// CheckpointConfig configures checkpointing behavior
type CheckpointConfig struct {
	// Store is the checkpoint storage backend
	Store CheckpointStore

	// AutoSave enables automatic checkpointing after each node
	AutoSave bool

	// SaveInterval specifies how often to save (when AutoSave is false)
	SaveInterval time.Duration

	// MaxCheckpoints limits the number of checkpoints to keep
	MaxCheckpoints int
}

// DefaultCheckpointConfig returns a default checkpoint configuration
func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		Store:          NewMemoryCheckpointStore(),
		AutoSave:       true,
		SaveInterval:   30 * time.Second,
		MaxCheckpoints: 10,
	}
}

// CheckpointableRunnable wraps a runnable with checkpointing capabilities
type CheckpointableRunnable struct {
	runnable *ListenableRunnable
	config   CheckpointConfig

	executionID string
}

// NewCheckpointableRunnable creates a new checkpointable runnable
func NewCheckpointableRunnable(runnable *ListenableRunnable, config CheckpointConfig) *CheckpointableRunnable {
	return &CheckpointableRunnable{
		runnable:    runnable,
		config:      config,
		executionID: generateExecutionID(),
	}
}

// Invoke executes the graph with checkpointing
func (cr *CheckpointableRunnable) Invoke(ctx context.Context, initialState any) (any, error) {
	return cr.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the graph with checkpointing and config
func (cr *CheckpointableRunnable) InvokeWithConfig(ctx context.Context, initialState any, config *Config) (any, error) {
	// Convert initialState to map[string]any
	var stateMap map[string]any
	if initialState != nil {
		if m, ok := initialState.(map[string]any); ok {
			stateMap = m
		} else {
			return nil, fmt.Errorf("initialState must be map[string]any, got %T", initialState)
		}
	} else {
		stateMap = make(map[string]any)
	}

	// Extract thread_id from config if present
	var threadID string
	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	// Create checkpointing listener
	checkpointListener := &CheckpointListener{
		store:       cr.config.Store,
		executionID: cr.executionID,
		threadID:    threadID,
		autoSave:    cr.config.AutoSave,
	}

	// Add checkpoint listener to config callbacks
	if config == nil {
		config = &Config{}
	}
	config.Callbacks = append(config.Callbacks, checkpointListener)

	return cr.runnable.InvokeWithConfig(ctx, stateMap, config)
}

// SaveCheckpoint manually saves a checkpoint
func (cr *CheckpointableRunnable) SaveCheckpoint(ctx context.Context, nodeName string, state any) error {
	// Get current version from existing checkpoints
	checkpoints, err := cr.config.Store.List(ctx, cr.executionID)
	version := 1
	if err == nil && len(checkpoints) > 0 {
		// Get the latest version
		latest := checkpoints[len(checkpoints)-1]
		version = latest.Version + 1
	}

	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  nodeName,
		State:     state,
		Timestamp: time.Now(),
		Version:   version,
		Metadata: map[string]any{
			"execution_id": cr.executionID,
		},
	}

	return cr.config.Store.Save(ctx, checkpoint)
}

// LoadCheckpoint loads a specific checkpoint
func (cr *CheckpointableRunnable) LoadCheckpoint(ctx context.Context, checkpointID string) (*Checkpoint, error) {
	return cr.config.Store.Load(ctx, checkpointID)
}

// ListCheckpoints returns all checkpoints for this execution
func (cr *CheckpointableRunnable) ListCheckpoints(ctx context.Context) ([]*Checkpoint, error) {
	return cr.config.Store.List(ctx, cr.executionID)
}

// ResumeFromCheckpoint resumes execution from a specific checkpoint
func (cr *CheckpointableRunnable) ResumeFromCheckpoint(ctx context.Context, checkpointID string) (any, error) {
	checkpoint, err := cr.LoadCheckpoint(ctx, checkpointID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	// Resume execution from the checkpointed state
	// This would require the graph to support starting from a specific node
	// For now, we'll return the checkpointed state
	return checkpoint.State, nil
}

// ClearCheckpoints removes all checkpoints for this execution
func (cr *CheckpointableRunnable) ClearCheckpoints(ctx context.Context) error {
	return cr.config.Store.Clear(ctx, cr.executionID)
}

// CheckpointListener automatically creates checkpoints during execution
type CheckpointListener struct {
	store       CheckpointStore
	executionID string
	threadID    string
	autoSave    bool
	// Embed NoOpCallbackHandler to satisfy other CallbackHandler methods
	NoOpCallbackHandler
}

// OnGraphStep implements GraphCallbackHandler
func (cl *CheckpointListener) OnGraphStep(ctx context.Context, stepNode string, state any) {
	if !cl.autoSave {
		return
	}

	// Get current version from existing checkpoints
	checkpoints, err := cl.store.List(ctx, cl.executionID)
	version := 1
	if err == nil && len(checkpoints) > 0 {
		// Get the latest version
		latest := checkpoints[len(checkpoints)-1]
		version = latest.Version + 1
	}

	metadata := map[string]any{
		"execution_id": cl.executionID,
		"event":        "step",
	}
	if cl.threadID != "" {
		metadata["thread_id"] = cl.threadID
	}

	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  stepNode,
		State:     state,
		Timestamp: time.Now(),
		Version:   version,
		Metadata:  metadata,
	}

	// Save checkpoint synchronously to avoid race conditions in tests
	if saveErr := cl.store.Save(ctx, checkpoint); saveErr != nil {
		_ = saveErr
	}
}

// OnNodeEvent is no longer used for saving state, but kept if needed for interface compatibility
// or we can remove it if we don't use it as NodeListener anymore.
// CheckpointableRunnable currently adds it as NodeListener. We should change that.

// CheckpointableStateGraph extends ListenableStateGraphUntyped with checkpointing
type CheckpointableStateGraph struct {
	*ListenableStateGraphUntyped
	config CheckpointConfig
}

// NewCheckpointableStateGraph creates a new checkpointable state graph
func NewCheckpointableStateGraph() *CheckpointableStateGraph {
	return &CheckpointableStateGraph{
		ListenableStateGraphUntyped: NewListenableStateGraphUntyped(),
		config:                     DefaultCheckpointConfig(),
	}
}

// NewCheckpointableStateGraphWithConfig creates a checkpointable graph with custom config
func NewCheckpointableStateGraphWithConfig(config CheckpointConfig) *CheckpointableStateGraph {
	return &CheckpointableStateGraph{
		ListenableStateGraphUntyped: NewListenableStateGraphUntyped(),
		config:                     config,
	}
}

// CompileCheckpointable compiles the graph into a checkpointable runnable
func (g *CheckpointableStateGraph) CompileCheckpointable() (*CheckpointableRunnable, error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewCheckpointableRunnable(listenableRunnable, g.config), nil
}

// SetCheckpointConfig updates the checkpointing configuration
func (g *CheckpointableStateGraph) SetCheckpointConfig(config CheckpointConfig) {
	g.config = config
}

// GetCheckpointConfig returns the current checkpointing configuration
func (g *CheckpointableStateGraph) GetCheckpointConfig() CheckpointConfig {
	return g.config
}

// Helper functions
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

func generateCheckpointID() string {
	return fmt.Sprintf("checkpoint_%s", uuid.New().String())
}

// StateSnapshot represents a snapshot of the graph state
type StateSnapshot struct {
	Values    any
	Next      []string
	Config    Config
	Metadata  map[string]any
	CreatedAt time.Time
	ParentID  string
}

// GetState retrieves the state for the given config
func (cr *CheckpointableRunnable) GetState(ctx context.Context, config *Config) (*StateSnapshot, error) {
	var threadID string
	var checkpointID string

	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
		if cid, ok := config.Configurable["checkpoint_id"].(string); ok {
			checkpointID = cid
		}
	}

	// Default to current execution ID if thread_id not provided
	if threadID == "" {
		threadID = cr.executionID
	}

	var checkpoint *Checkpoint
	var err error

	if checkpointID != "" {
		checkpoint, err = cr.config.Store.Load(ctx, checkpointID)
	} else {
		// Get latest checkpoint for the thread
		// Note: List returns all checkpoints. We need to find the latest one.
		// This is inefficient for large histories. Real implementations should have GetLatest.
		checkpoints, err := cr.config.Store.List(ctx, threadID)
		if err == nil && len(checkpoints) > 0 {
			// Assuming List returns in some order, or we sort.
			// For now, assume the last one is latest (based on implementation of MemoryStore)
			checkpoint = checkpoints[len(checkpoints)-1]
		}
	}

	if err != nil {
		return nil, err
	}

	if checkpoint == nil {
		return &StateSnapshot{
			Values: nil,
			Config: *config,
		}, nil
	}

	// Construct snapshot
	snapshot := &StateSnapshot{
		Values:    checkpoint.State,
		CreatedAt: checkpoint.Timestamp,
		Metadata:  checkpoint.Metadata,
		Config: Config{
			Configurable: map[string]any{
				"thread_id":     threadID,
				"checkpoint_id": checkpoint.ID,
			},
		},
	}

	// Determine "Next" nodes
	// This is tricky without re-running the graph logic or storing it in the checkpoint.
	// For now, we might leave it empty or try to infer it if we stored it.
	// In the future, Checkpoint should store "Next" nodes.

	return snapshot, nil
}

// UpdateState updates the state for the given config
func (cr *CheckpointableRunnable) UpdateState(ctx context.Context, config *Config, values any, asNode string) (*Config, error) {
	var threadID string
	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	if threadID == "" {
		threadID = cr.executionID
	}

	// 1. Get current state
	// We need to find the latest checkpoint for this thread to merge against
	checkpoints, err := cr.config.Store.List(ctx, threadID)
	var currentState any
	var currentVersion int

	if err == nil && len(checkpoints) > 0 {
		// Assume last is latest
		latest := checkpoints[len(checkpoints)-1]
		currentState = latest.State
		currentVersion = latest.Version
	} else {
		// No existing state, initialize if schema exists
		if cr.runnable.graph.Schema != nil {
			currentState = cr.runnable.graph.Schema.Init()
		}
	}

	// 2. Merge values
	newState := values
	if cr.runnable.graph.Schema != nil {
		// If Schema is defined, use it to update state with results
		var baseState map[string]any
		if currentState != nil {
			if bs, ok := currentState.(map[string]any); ok {
				baseState = bs
			} else {
				baseState = make(map[string]any)
			}
		} else {
			// Schema.Init() returns map[string]any for StateSchemaTyped[map[string]any]
			init := cr.runnable.graph.Schema.Init()
			baseState = init
			if baseState == nil {
				baseState = make(map[string]any)
			}
		}

		var valuesMap map[string]any
		if vm, ok := values.(map[string]any); ok {
			valuesMap = vm
		} else {
			valuesMap = make(map[string]any)
		}

		if merged, err := cr.runnable.graph.Schema.Update(baseState, valuesMap); err != nil {
			return nil, fmt.Errorf("failed to merge state: %w", err)
		} else {
			newState = merged
		}
	} else if currentState != nil {
		// No schema, but have current state.
		// Detailed map merge logic
		if curMap, ok := currentState.(map[string]any); ok {
			if valMap, ok := values.(map[string]any); ok {
				// Create a new map for the merged state to avoid mutating the original
				merged := make(map[string]any)
				maps.Copy(merged, curMap)
				maps.Copy(merged, valMap)
				newState = merged
			}
		}
	}

	// 3. Create new checkpoint
	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  asNode, // The node that "made" this update
		State:     newState,
		Timestamp: time.Now(),
		Version:   currentVersion + 1,
		Metadata: map[string]any{
			"execution_id": threadID,
			"source":       "update_state",
			"updated_by":   asNode,
		},
	}

	if err := cr.config.Store.Save(ctx, checkpoint); err != nil {
		return nil, err
	}

	return &Config{
		Configurable: map[string]any{
			"thread_id":     threadID,
			"checkpoint_id": checkpoint.ID,
		},
	}, nil
}

// Generic checkpointing types

// CheckpointableStateGraphTyped[S any] extends ListenableStateGraphTyped[S] with checkpointing
type CheckpointableStateGraphTyped[S any] struct {
	*ListenableStateGraphTyped[S]
	config CheckpointConfig
}

// NewCheckpointableStateGraphTyped creates a new checkpointable state graph with type parameter
func NewCheckpointableStateGraphTyped[S any]() *CheckpointableStateGraphTyped[S] {
	baseGraph := NewListenableStateGraphTyped[S]()
	return &CheckpointableStateGraphTyped[S]{
		ListenableStateGraphTyped: baseGraph,
		config:                    DefaultCheckpointConfig(),
	}
}

// NewCheckpointableStateGraphTypedWithConfig creates a checkpointable graph with custom config
func NewCheckpointableStateGraphTypedWithConfig[S any](config CheckpointConfig) *CheckpointableStateGraphTyped[S] {
	baseGraph := NewListenableStateGraphTyped[S]()
	return &CheckpointableStateGraphTyped[S]{
		ListenableStateGraphTyped: baseGraph,
		config:                    config,
	}
}

// CompileCheckpointable compiles the graph into a checkpointable runnable
func (g *CheckpointableStateGraphTyped[S]) CompileCheckpointable() (*CheckpointableRunnableTyped[S], error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewCheckpointableRunnableTyped(listenableRunnable, g.config), nil
}

// SetCheckpointConfig updates the checkpointing configuration
func (g *CheckpointableStateGraphTyped[S]) SetCheckpointConfig(config CheckpointConfig) {
	g.config = config
}

// GetCheckpointConfig returns the current checkpointing configuration
func (g *CheckpointableStateGraphTyped[S]) GetCheckpointConfig() CheckpointConfig {
	return g.config
}

// CheckpointableRunnableTyped[S] wraps a ListenableRunnableTyped[S] with checkpointing capabilities
type CheckpointableRunnableTyped[S any] struct {
	runnable    *ListenableRunnableTyped[S]
	config      CheckpointConfig
	executionID string
	listener    *CheckpointListener
}

// NewCheckpointableRunnableTyped creates a new checkpointable runnable from a listenable runnable
func NewCheckpointableRunnableTyped[S any](runnable *ListenableRunnableTyped[S], config CheckpointConfig) *CheckpointableRunnableTyped[S] {
	executionID := generateExecutionID()
	cr := &CheckpointableRunnableTyped[S]{
		runnable:    runnable,
		config:      config,
		executionID: executionID,
	}

	// Create checkpoint listener (the listener uses untyped state 'any')
	cr.listener = &CheckpointListener{
		store:       cr.config.Store,
		executionID: executionID,
		threadID:    "",
		autoSave:    true,
	}

	// Note: We don't add the listener via AddGlobalListener because
	// CheckpointListener is untyped (uses NodeListener interface, not NodeListenerTyped[S])
	// Instead, checkpointing is handled via callbacks in InvokeWithConfig

	return cr
}

// Invoke executes the graph with checkpointing support
func (cr *CheckpointableRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
	return cr.InvokeWithConfig(ctx, initialState, nil)
}

// InvokeWithConfig executes the graph with checkpointing support and config
func (cr *CheckpointableRunnableTyped[S]) InvokeWithConfig(ctx context.Context, initialState S, config *Config) (S, error) {
	// Extract thread_id from config if present
	var threadID string
	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	// Update checkpoint listener with thread_id
	if cr.listener != nil {
		cr.listener.threadID = threadID
		cr.listener.autoSave = cr.config.AutoSave
	}

	// Add checkpoint listener to config callbacks
	if config == nil {
		config = &Config{}
	}
	config.Callbacks = append(config.Callbacks, cr.listener)

	return cr.runnable.InvokeWithConfig(ctx, initialState, config)
}

// Stream executes the graph with checkpointing and streaming support
func (cr *CheckpointableRunnableTyped[S]) Stream(ctx context.Context, initialState S) <-chan StreamEventTyped[S] {
	return cr.runnable.Stream(ctx, initialState)
}

// GetState retrieves the state for the given config
func (cr *CheckpointableRunnableTyped[S]) GetState(ctx context.Context, config *Config) (*StateSnapshot, error) {
	var threadID string
	var checkpointID string

	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
		if cid, ok := config.Configurable["checkpoint_id"].(string); ok {
			checkpointID = cid
		}
	}

	// Default to current execution ID if thread_id not provided
	if threadID == "" {
		threadID = cr.executionID
	}

	var checkpoint *Checkpoint
	var err error

	if checkpointID != "" {
		checkpoint, err = cr.config.Store.Load(ctx, checkpointID)
	} else {
		// Get latest checkpoint for the thread
		checkpoints, err := cr.config.Store.List(ctx, threadID)
		if err != nil {
			return nil, fmt.Errorf("failed to list checkpoints: %w", err)
		}

		if len(checkpoints) == 0 {
			return nil, fmt.Errorf("no checkpoints found for thread %s", threadID)
		}

		// Get the latest checkpoint (highest version)
		checkpoint = checkpoints[0]
		for _, cp := range checkpoints {
			if cp.Version > checkpoint.Version {
				checkpoint = cp
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	if checkpoint == nil {
		return nil, fmt.Errorf("checkpoint not found")
	}

	// Return state snapshot
	next := []string{checkpoint.NodeName}
	if checkpoint.NodeName == "" {
		next = []string{}
	}
	return &StateSnapshot{
		Values:   checkpoint.State,
		Next:     next,
		Config: Config{
			Configurable: map[string]any{
				"thread_id":     threadID,
				"checkpoint_id": checkpoint.ID,
			},
		},
		Metadata:  checkpoint.Metadata,
		CreatedAt: checkpoint.Timestamp,
	}, nil
}

// SaveCheckpoint manually saves a checkpoint at the current state
func (cr *CheckpointableRunnableTyped[S]) SaveCheckpoint(ctx context.Context, nodeName string, state S) error {
	// Get current version to increment
	checkpoints, _ := cr.config.Store.List(ctx, cr.executionID)
	version := 1
	if len(checkpoints) > 0 {
		for _, cp := range checkpoints {
			if cp.Version >= version {
				version = cp.Version + 1
			}
		}
	}

	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  nodeName,
		State:     state,
		Timestamp: time.Now(),
		Version:   version,
		Metadata: map[string]any{
			"execution_id": cr.executionID,
			"source":       "manual_save",
			"saved_by":     nodeName,
		},
	}

	return cr.config.Store.Save(ctx, checkpoint)
}

// ListCheckpoints lists all checkpoints for the current execution
func (cr *CheckpointableRunnableTyped[S]) ListCheckpoints(ctx context.Context) ([]*Checkpoint, error) {
	return cr.config.Store.List(ctx, cr.executionID)
}

// UpdateState updates the state and saves a checkpoint.
// For typed graphs, this method is less commonly used. Consider using UpdateStateTyped instead.
func (cr *CheckpointableRunnableTyped[S]) UpdateState(ctx context.Context, config *Config, asNode string, values map[string]any) (*Config, error) {
	var threadID string

	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	if threadID == "" {
		threadID = cr.executionID
	}

	// Get current state from config if available
	var currentState S
	var currentVersion int

	if config != nil {
		snapshot, err := cr.GetState(ctx, config)
		if err == nil && snapshot != nil {
			currentState = snapshot.Values.(S)
			// Find current version
			checkpoints, _ := cr.config.Store.List(ctx, threadID)
			for _, cp := range checkpoints {
				if cp.Version > currentVersion {
					currentVersion = cp.Version
				}
			}
		}
	}

	// For typed graphs, we can't directly use map[string]any with Schema.Update
	// because typed schemas expect two S values.
	// So we'll just save the values directly as a new state.
	// Users should use UpdateStateTyped for proper typed updates.

	var newState S
	// If currentState is map[string]any, try to merge
	if currentMap, ok := any(currentState).(map[string]any); ok {
		merged := make(map[string]any)
		maps.Copy(merged, currentMap)
		maps.Copy(merged, values)
		newState = any(merged).(S)
	} else {
		// For non-map types, we can't do a proper merge without more type information
		// Just save the values as-is (this is a limitation)
		newState = currentState
	}

	// Get max version
	checkpoints, _ := cr.config.Store.List(ctx, threadID)
	version := 1
	for _, cp := range checkpoints {
		if cp.Version >= version {
			version = cp.Version + 1
		}
	}

	// Create new checkpoint
	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  asNode,
		State:     newState,
		Timestamp: time.Now(),
		Version:   version,
		Metadata: map[string]any{
			"execution_id": threadID,
			"source":       "update_state",
			"updated_by":   asNode,
		},
	}

	if err := cr.config.Store.Save(ctx, checkpoint); err != nil {
		return nil, err
	}

	return &Config{
		Configurable: map[string]any{
			"thread_id":     threadID,
			"checkpoint_id": checkpoint.ID,
		},
	}, nil
}

// UpdateStateTyped updates the state with proper type safety and saves a checkpoint.
// This is the preferred method for typed graphs.
func (cr *CheckpointableRunnableTyped[S]) UpdateStateTyped(ctx context.Context, config *Config, asNode string, updateFunc func(S) S) (*Config, error) {
	var threadID string

	if config != nil && config.Configurable != nil {
		if tid, ok := config.Configurable["thread_id"].(string); ok {
			threadID = tid
		}
	}

	if threadID == "" {
		threadID = cr.executionID
	}

	// Get current state from config if available
	var currentState S

	if config != nil {
		snapshot, err := cr.GetState(ctx, config)
		if err == nil && snapshot != nil {
			currentState = snapshot.Values.(S)
		}
	}

	// Apply update function
	newState := updateFunc(currentState)

	// Get max version
	checkpoints, _ := cr.config.Store.List(ctx, threadID)
	version := 1
	for _, cp := range checkpoints {
		if cp.Version >= version {
			version = cp.Version + 1
		}
	}

	// Create new checkpoint
	checkpoint := &Checkpoint{
		ID:        generateCheckpointID(),
		NodeName:  asNode,
		State:     newState,
		Timestamp: time.Now(),
		Version:   version,
		Metadata: map[string]any{
			"execution_id": threadID,
			"source":       "update_state_typed",
			"updated_by":   asNode,
		},
	}

	if err := cr.config.Store.Save(ctx, checkpoint); err != nil {
		return nil, err
	}

	return &Config{
		Configurable: map[string]any{
			"thread_id":     threadID,
			"checkpoint_id": checkpoint.ID,
		},
	}, nil
}

// GetExecutionID returns the current execution ID
func (cr *CheckpointableRunnableTyped[S]) GetExecutionID() string {
	return cr.executionID
}

// SetExecutionID sets a new execution ID
func (cr *CheckpointableRunnableTyped[S]) SetExecutionID(executionID string) {
	cr.executionID = executionID
	if cr.listener != nil {
		cr.listener.executionID = executionID
	}
}

// GetTracer returns the tracer from the underlying runnable
func (cr *CheckpointableRunnableTyped[S]) GetTracer() *Tracer {
	return cr.runnable.GetTracer()
}

// SetTracer sets the tracer on the underlying runnable
func (cr *CheckpointableRunnableTyped[S]) SetTracer(tracer *Tracer) {
	cr.runnable.SetTracer(tracer)
}

// WithTracer returns a new CheckpointableRunnableTyped with the given tracer
func (cr *CheckpointableRunnableTyped[S]) WithTracer(tracer *Tracer) *CheckpointableRunnableTyped[S] {
	newRunnable := cr.runnable.WithTracer(tracer)
	return &CheckpointableRunnableTyped[S]{
		runnable:    newRunnable,
		config:      cr.config,
		executionID: cr.executionID,
		listener:    cr.listener,
	}
}

// GetGraph returns the underlying graph
func (cr *CheckpointableRunnableTyped[S]) GetGraph() *ListenableStateGraphTyped[S] {
	return cr.runnable.GetListenableGraph()
}

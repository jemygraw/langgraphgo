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

	return cr.runnable.InvokeWithConfig(ctx, initialState, config)
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

// CheckpointableStateGraph extends ListenableStateGraph with checkpointing
type CheckpointableStateGraph struct {
	*ListenableStateGraph
	config CheckpointConfig
}

// NewCheckpointableStateGraph creates a new checkpointable state graph
func NewCheckpointableStateGraph() *CheckpointableStateGraph {
	return &CheckpointableStateGraph{
		ListenableStateGraph: NewListenableStateGraph(),
		config:               DefaultCheckpointConfig(),
	}
}

// NewCheckpointableStateGraphWithConfig creates a checkpointable graph with custom config
func NewCheckpointableStateGraphWithConfig(config CheckpointConfig) *CheckpointableStateGraph {
	return &CheckpointableStateGraph{
		ListenableStateGraph: NewListenableStateGraph(),
		config:               config,
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
		var baseState any
		if currentState != nil {
			baseState = currentState
		} else {
			baseState = cr.runnable.graph.Schema.Init()
		}

		if merged, err := cr.runnable.graph.Schema.Update(baseState, values); err != nil {
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

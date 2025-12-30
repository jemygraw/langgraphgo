package graph

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// StreamMode defines the mode of streaming
type StreamMode string

const (
	// StreamModeValues emits the full state after each step
	StreamModeValues StreamMode = "values"
	// StreamModeUpdates emits the updates (deltas) from each node
	StreamModeUpdates StreamMode = "updates"
	// StreamModeMessages emits LLM messages/tokens (if available)
	StreamModeMessages StreamMode = "messages"
	// StreamModeDebug emits all events (default)
	StreamModeDebug StreamMode = "debug"
)

// StreamConfig configures streaming behavior
type StreamConfig struct {
	// BufferSize is the size of the event channel buffer
	BufferSize int

	// EnableBackpressure determines if backpressure handling is enabled
	EnableBackpressure bool

	// MaxDroppedEvents is the maximum number of events to drop before logging
	MaxDroppedEvents int

	// Mode specifies what kind of events to stream
	Mode StreamMode
}

// DefaultStreamConfig returns the default streaming configuration
func DefaultStreamConfig() StreamConfig {
	return StreamConfig{
		BufferSize:         1000,
		EnableBackpressure: true,
		MaxDroppedEvents:   100,
		Mode:               StreamModeDebug,
	}
}

// StreamResult contains the channels returned by streaming execution
type StreamResult struct {
	// Events channel receives StreamEvent objects in real-time
	Events <-chan StreamEvent

	// Result channel receives the final result when execution completes
	Result <-chan any

	// Errors channel receives any errors that occur during execution
	Errors <-chan error

	// Done channel is closed when streaming is complete
	Done <-chan struct{}

	// Cancel function can be called to stop streaming
	Cancel context.CancelFunc
}

// StreamingListener implements NodeListener for streaming events
type StreamingListener struct {
	eventChan chan<- StreamEvent
	config    StreamConfig
	mutex     sync.RWMutex

	droppedEvents int
	closed        bool
}

// NewStreamingListener creates a new streaming listener
func NewStreamingListener(eventChan chan<- StreamEvent, config StreamConfig) *StreamingListener {
	return &StreamingListener{
		eventChan: eventChan,
		config:    config,
	}
}

// emitEvent sends an event to the channel handling backpressure
func (sl *StreamingListener) emitEvent(event StreamEvent) {
	// Check if listener is closed
	sl.mutex.RLock()
	if sl.closed {
		sl.mutex.RUnlock()
		return
	}
	sl.mutex.RUnlock()

	// Filter based on Mode
	if !sl.shouldEmit(event) {
		return
	}

	// Try to send event without blocking
	select {
	case sl.eventChan <- event:
		// Event sent successfully
	default:
		// Channel is full
		if sl.config.EnableBackpressure {
			sl.handleBackpressure()
		}
		// Drop the event if backpressure handling is disabled or channel is still full
	}
}

func (sl *StreamingListener) shouldEmit(event StreamEvent) bool {
	switch sl.config.Mode {
	case StreamModeDebug:
		return true
	case StreamModeValues:
		// Only emit OnGraphStep events (which contain full state)
		// We use a custom event type for this?
		// Currently OnGraphStep calls emitEvent with what?
		// We need to implement OnGraphStep in StreamingListener.
		// For now, let's assume OnGraphStep emits a special event.
		// If event.Event == "graph_step", return true.
		return event.Event == "graph_step"
	case StreamModeUpdates:
		// Emit node outputs (ToolEnd, ChainEnd, NodeEventComplete)
		return event.Event == EventToolEnd || event.Event == EventChainEnd || event.Event == NodeEventComplete
	case StreamModeMessages:
		// Emit LLM events
		return event.Event == EventLLMEnd || event.Event == EventLLMStart
	default:
		return true
	}
}

// OnNodeEvent implements the NodeListener interface
func (sl *StreamingListener) OnNodeEvent(_ context.Context, event NodeEvent, nodeName string, state any, err error) {
	streamEvent := StreamEvent{
		Timestamp: time.Now(),
		NodeName:  nodeName,
		Event:     event,
		State:     state,
		Error:     err,
		Metadata:  make(map[string]any),
	}
	sl.emitEvent(streamEvent)
}

// CallbackHandler implementation

func (sl *StreamingListener) OnChainStart(ctx context.Context, serialized map[string]any, inputs map[string]any, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventChainStart,
		Metadata:  metadata,
		State:     inputs,
	})
}

// streamingListenerTypedAdapter wraps StreamingListener to implement NodeListenerTyped[map[string]any]
type streamingListenerTypedAdapter struct {
	*StreamingListener
}

// OnNodeEvent implements NodeListenerTyped[map[string]any]
func (a *streamingListenerTypedAdapter) OnNodeEvent(ctx context.Context, event NodeEvent, nodeName string, state map[string]any, err error) {
	// Call the untyped OnNodeEvent method
	a.StreamingListener.OnNodeEvent(ctx, event, nodeName, state, err)
}

func (sl *StreamingListener) OnChainEnd(ctx context.Context, outputs map[string]any, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventChainEnd,
		State:     outputs,
	})
}

func (sl *StreamingListener) OnChainError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError, // Or specific ChainError?
		Error:     err,
	})
}

func (sl *StreamingListener) OnLLMStart(ctx context.Context, serialized map[string]any, prompts []string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventLLMStart,
		Metadata:  metadata,
		State:     prompts,
	})
}

func (sl *StreamingListener) OnLLMEnd(ctx context.Context, response any, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventLLMEnd,
		State:     response,
	})
}

func (sl *StreamingListener) OnLLMError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError,
		Error:     err,
	})
}

func (sl *StreamingListener) OnToolStart(ctx context.Context, serialized map[string]any, inputStr string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventToolStart,
		Metadata:  metadata,
		State:     inputStr,
	})
}

func (sl *StreamingListener) OnToolEnd(ctx context.Context, output string, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     EventToolEnd,
		State:     output,
	})
}

func (sl *StreamingListener) OnToolError(ctx context.Context, err error, runID string) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     NodeEventError,
		Error:     err,
	})
}

func (sl *StreamingListener) OnRetrieverStart(ctx context.Context, serialized map[string]any, query string, runID string, parentRunID *string, tags []string, metadata map[string]any) {
	// Map to custom or tool event?
}

func (sl *StreamingListener) OnRetrieverEnd(ctx context.Context, documents []any, runID string) {
}

func (sl *StreamingListener) OnRetrieverError(ctx context.Context, err error, runID string) {
}

// OnGraphStep implements GraphCallbackHandler
func (sl *StreamingListener) OnGraphStep(ctx context.Context, stepNode string, state any) {
	sl.emitEvent(StreamEvent{
		Timestamp: time.Now(),
		Event:     "graph_step", // Custom event type
		NodeName:  stepNode,
		State:     state,
	})
}

// Close marks the listener as closed to prevent sending to closed channels
func (sl *StreamingListener) Close() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()
	sl.closed = true
}

// handleBackpressure manages channel backpressure
func (sl *StreamingListener) handleBackpressure() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	sl.droppedEvents++

	// Could implement more sophisticated backpressure strategies here
	// For now, we just track dropped events
}

// GetDroppedEventsCount returns the number of dropped events
func (sl *StreamingListener) GetDroppedEventsCount() int {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()
	return sl.droppedEvents
}

// StreamingRunnable wraps a ListenableRunnable with streaming capabilities
type StreamingRunnable struct {
	runnable *ListenableRunnable
	config   StreamConfig
}

// NewStreamingRunnable creates a new streaming runnable
func NewStreamingRunnable(runnable *ListenableRunnable, config StreamConfig) *StreamingRunnable {
	return &StreamingRunnable{
		runnable: runnable,
		config:   config,
	}
}

// NewStreamingRunnableWithDefaults creates a streaming runnable with default config
func NewStreamingRunnableWithDefaults(runnable *ListenableRunnable) *StreamingRunnable {
	return NewStreamingRunnable(runnable, DefaultStreamConfig())
}

// Stream executes the graph with real-time event streaming
func (sr *StreamingRunnable) Stream(ctx context.Context, initialState any) *StreamResult {
	// Create channels
	eventChan := make(chan StreamEvent, sr.config.BufferSize)
	resultChan := make(chan any, 1)
	errorChan := make(chan error, 1)
	doneChan := make(chan struct{})

	// Create cancellable context
	streamCtx, cancel := context.WithCancel(ctx)

	// Create streaming listener
	streamingListener := NewStreamingListener(eventChan, sr.config)

	// Wrap the listener to implement NodeListenerTyped[map[string]any]
	typedAdapter := &streamingListenerTypedAdapter{StreamingListener: streamingListener}

	// Add the streaming listener to all nodes
	for _, node := range sr.runnable.listenableNodes {
		node.AddListener(typedAdapter)
	}

	// Execute in goroutine
	go func() {
		defer func() {
			// First, close the streaming listener to prevent new events
			streamingListener.Close()

			// Clean up: remove typed adapter from all nodes
			for _, node := range sr.runnable.listenableNodes {
				node.RemoveListenerByFunc(typedAdapter)
			}

			// Give a small delay for any in-flight listener calls to complete
			time.Sleep(10 * time.Millisecond)

			// Now safe to close channels
			close(eventChan)
			close(resultChan)
			close(errorChan)
			close(doneChan)
		}()

		// Create config with streaming listener as callback
		config := &Config{
			Callbacks: []CallbackHandler{streamingListener},
		}

		// Convert initialState to map[string]any for the typed runnable
		var stateMap map[string]any
		if initialState != nil {
			if m, ok := initialState.(map[string]any); ok {
				stateMap = m
			} else {
				errorChan <- fmt.Errorf("initialState must be map[string]any, got %T", initialState)
				return
			}
		} else {
			stateMap = make(map[string]any)
		}

		// Execute the runnable
		result, err := sr.runnable.InvokeWithConfig(streamCtx, stateMap, config)

		// Send result or error
		if err != nil {
			select {
			case errorChan <- err:
			case <-streamCtx.Done():
			}
		} else {
			select {
			case resultChan <- result:
			case <-streamCtx.Done():
			}
		}
	}()

	return &StreamResult{
		Events: eventChan,
		Result: resultChan,
		Errors: errorChan,
		Done:   doneChan,
		Cancel: cancel,
	}
}

// StreamingStateGraph extends ListenableStateGraphUntyped with streaming capabilities
type StreamingStateGraph struct {
	*ListenableStateGraphUntyped
	config StreamConfig
}

// NewStreamingStateGraph creates a new streaming message graph
func NewStreamingStateGraph() *StreamingStateGraph {
	return &StreamingStateGraph{
		ListenableStateGraphUntyped: NewListenableStateGraphUntyped(),
		config:                     DefaultStreamConfig(),
	}
}

// NewStreamingStateGraphWithConfig creates a streaming graph with custom config
func NewStreamingStateGraphWithConfig(config StreamConfig) *StreamingStateGraph {
	return &StreamingStateGraph{
		ListenableStateGraphUntyped: NewListenableStateGraphUntyped(),
		config:                     config,
	}
}

// CompileStreaming compiles the graph into a streaming runnable
func (g *StreamingStateGraph) CompileStreaming() (*StreamingRunnable, error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewStreamingRunnable(listenableRunnable, g.config), nil
}

// SetStreamConfig updates the streaming configuration
func (g *StreamingStateGraph) SetStreamConfig(config StreamConfig) {
	g.config = config
}

// GetStreamConfig returns the current streaming configuration
func (g *StreamingStateGraph) GetStreamConfig() StreamConfig {
	return g.config
}

// StreamingExecutor provides a high-level interface for streaming execution
type StreamingExecutor struct {
	runnable *StreamingRunnable
}

// NewStreamingExecutor creates a new streaming executor
func NewStreamingExecutor(runnable *StreamingRunnable) *StreamingExecutor {
	return &StreamingExecutor{
		runnable: runnable,
	}
}

// ExecuteWithCallback executes the graph and calls the callback for each event
//
//nolint:cyclop // Complex streaming logic requires multiple conditional paths
func (se *StreamingExecutor) ExecuteWithCallback(
	ctx context.Context,
	initialState any,
	eventCallback func(event StreamEvent),
	resultCallback func(result any, err error),
) error {

	streamResult := se.runnable.Stream(ctx, initialState)
	defer streamResult.Cancel()

	var finalResult any
	var finalError error
	resultReceived := false

	for {
		select {
		case event, ok := <-streamResult.Events:
			if !ok {
				// Events channel closed
				if resultReceived && resultCallback != nil {
					resultCallback(finalResult, finalError)
				}
				return finalError
			}
			if eventCallback != nil {
				eventCallback(event)
			}

		case result := <-streamResult.Result:
			finalResult = result
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case err := <-streamResult.Errors:
			finalError = err
			resultReceived = true
			// Don't return immediately, wait for events channel to close

		case <-streamResult.Done:
			if resultReceived && resultCallback != nil {
				resultCallback(finalResult, finalError)
			}
			return finalError

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// ExecuteAsync executes the graph asynchronously and returns immediately
func (se *StreamingExecutor) ExecuteAsync(ctx context.Context, initialState any) *StreamResult {
	return se.runnable.Stream(ctx, initialState)
}

// GetGraph returns a Exporter for the streaming runnable
func (sr *StreamingRunnable) GetGraph() *Exporter[map[string]any] {
	return sr.runnable.GetGraph()
}

// Generic streaming types

// StreamingStateGraphTyped[S any] extends ListenableStateGraphTyped[S] with streaming capabilities
type StreamingStateGraphTyped[S any] struct {
	*ListenableStateGraphTyped[S]
	config StreamConfig
}

// NewStreamingStateGraphTyped creates a new streaming state graph with type parameter
func NewStreamingStateGraphTyped[S any]() *StreamingStateGraphTyped[S] {
	baseGraph := NewListenableStateGraphTyped[S]()
	return &StreamingStateGraphTyped[S]{
		ListenableStateGraphTyped: baseGraph,
		config:                    DefaultStreamConfig(),
	}
}

// NewStreamingStateGraphTypedWithConfig creates a streaming graph with custom config
func NewStreamingStateGraphTypedWithConfig[S any](config StreamConfig) *StreamingStateGraphTyped[S] {
	baseGraph := NewListenableStateGraphTyped[S]()
	return &StreamingStateGraphTyped[S]{
		ListenableStateGraphTyped: baseGraph,
		config:                    config,
	}
}

// CompileStreaming compiles the graph into a streaming runnable
func (g *StreamingStateGraphTyped[S]) CompileStreaming() (*StreamingRunnableTyped[S], error) {
	listenableRunnable, err := g.CompileListenable()
	if err != nil {
		return nil, err
	}

	return NewStreamingRunnableTyped(listenableRunnable, g.config), nil
}

// SetStreamConfig updates the streaming configuration
func (g *StreamingStateGraphTyped[S]) SetStreamConfig(config StreamConfig) {
	g.config = config
}

// GetStreamConfig returns the current streaming configuration
func (g *StreamingStateGraphTyped[S]) GetStreamConfig() StreamConfig {
	return g.config
}

// StreamingRunnableTyped[S] wraps a ListenableRunnableTyped[S] with streaming capabilities
type StreamingRunnableTyped[S any] struct {
	runnable *ListenableRunnableTyped[S]
	config   StreamConfig
}

// NewStreamingRunnableTyped creates a new streaming runnable from a listenable runnable
func NewStreamingRunnableTyped[S any](runnable *ListenableRunnableTyped[S], config StreamConfig) *StreamingRunnableTyped[S] {
	return &StreamingRunnableTyped[S]{
		runnable: runnable,
		config:   config,
	}
}

// Stream executes the graph with streaming enabled
func (sr *StreamingRunnableTyped[S]) Stream(ctx context.Context, initialState S) <-chan StreamEventTyped[S] {
	eventsChan := make(chan StreamEventTyped[S], sr.config.BufferSize)
	doneChan := make(chan struct{})
	cancelChan := make(chan struct{})

	go func() {
		defer close(eventsChan)
		defer close(doneChan)

		currentState := initialState

		for {
			// Check for cancellation
			select {
			case <-cancelChan:
				return
			case <-ctx.Done():
				eventsChan <- StreamEventTyped[S]{
					Event:    NodeEventError,
					Timestamp: time.Now(),
					Error:     ctx.Err(),
				}
				return
			default:
			}

			// Execute one step
			var err error
			currentState, err = sr.runnable.Invoke(ctx, currentState)

			// Emit event based on mode
			switch sr.config.Mode {
			case StreamModeValues:
				eventsChan <- StreamEventTyped[S]{
					Event:    "graph_step",
					State:    currentState,
					Timestamp: time.Now(),
				}
			case StreamModeUpdates:
				eventsChan <- StreamEventTyped[S]{
					Event:    NodeEventComplete,
					State:    currentState,
					Timestamp: time.Now(),
				}
			case StreamModeDebug:
				eventsChan <- StreamEventTyped[S]{
					Event:    "debug",
					State:    currentState,
					Timestamp: time.Now(),
				}
			}

			if err != nil {
				eventsChan <- StreamEventTyped[S]{
					Event:    NodeEventError,
					Error:     err,
					State:     currentState,
					Timestamp: time.Now(),
				}
				return
			}

			// Check if we're done - if state has reached END
			if sr.isComplete(currentState) {
				return
			}
		}
	}()

	return eventsChan
}

// isComplete checks if the graph execution is complete
func (sr *StreamingRunnableTyped[S]) isComplete(state S) bool {
	// Default implementation - can be overridden
	return false
}

// Invoke executes the graph without streaming
func (sr *StreamingRunnableTyped[S]) Invoke(ctx context.Context, initialState S) (S, error) {
	return sr.runnable.Invoke(ctx, initialState)
}

// GetConfig returns the streaming configuration
func (sr *StreamingRunnableTyped[S]) GetConfig() StreamConfig {
	return sr.config
}

// SetConfig updates the streaming configuration
func (sr *StreamingRunnableTyped[S]) SetConfig(config StreamConfig) {
	sr.config = config
}

// GetGraph returns the underlying listenable graph
func (sr *StreamingRunnableTyped[S]) GetGraph() *ListenableStateGraphTyped[S] {
	return sr.runnable.GetListenableGraph()
}

// GetTracer returns the tracer from the underlying runnable
func (sr *StreamingRunnableTyped[S]) GetTracer() *Tracer {
	return sr.runnable.GetTracer()
}

// SetTracer sets the tracer on the underlying runnable
func (sr *StreamingRunnableTyped[S]) SetTracer(tracer *Tracer) {
	sr.runnable.SetTracer(tracer)
}

// WithTracer returns a new StreamingRunnableTyped with the given tracer
func (sr *StreamingRunnableTyped[S]) WithTracer(tracer *Tracer) *StreamingRunnableTyped[S] {
	newRunnable := sr.runnable.WithTracer(tracer)
	return &StreamingRunnableTyped[S]{
		runnable: newRunnable,
		config:   sr.config,
	}
}

// StreamingExecutorTyped[S] provides a high-level interface for streaming execution
type StreamingExecutorTyped[S any] struct {
	runnable *StreamingRunnableTyped[S]
}

// NewStreamingExecutorTyped creates a new streaming executor
func NewStreamingExecutorTyped[S any](runnable *StreamingRunnableTyped[S]) *StreamingExecutorTyped[S] {
	return &StreamingExecutorTyped[S]{
		runnable: runnable,
	}
}

// ExecuteWithCallback executes the graph and calls the callback for each event
func (se *StreamingExecutorTyped[S]) ExecuteWithCallback(
	ctx context.Context,
	initialState S,
	eventCallback func(event StreamEventTyped[S]),
	resultCallback func(result S, err error),
) error {

	eventsChan := se.runnable.Stream(ctx, initialState)

	var finalResult S
	var finalError error

	for event := range eventsChan {
		if event.Error != nil {
			finalError = event.Error
		}

		if eventCallback != nil {
			eventCallback(event)
		}

		// Update final state on each event
		finalResult = event.State
	}

	if finalError != nil {
		return finalError
	}

	if resultCallback != nil {
		resultCallback(finalResult, nil)
	}

	return nil
}

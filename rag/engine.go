package rag

import (
	"context"
	"fmt"
	"math"
)

// BaseEngine provides common functionality for RAG engines
type BaseEngine struct {
	retriever Retriever
	embedder  Embedder
	config    *Config
	metrics   *Metrics
}

// NewBaseEngine creates a new base RAG engine
func NewBaseEngine(retriever Retriever, embedder Embedder, config *Config) *BaseEngine {
	if config == nil {
		config = &Config{
			VectorRAG: &VectorRAGConfig{
				RetrieverConfig: RetrievalConfig{
					K:              4,
					ScoreThreshold: 0.5,
					SearchType:     "similarity",
				},
			},
		}
	}

	return &BaseEngine{
		retriever: retriever,
		embedder:  embedder,
		config:    config,
		metrics:   &Metrics{},
	}
}

// Query performs a RAG query using the base engine
func (e *BaseEngine) Query(ctx context.Context, query string) (*QueryResult, error) {
	config := &RetrievalConfig{
		K:              4,
		ScoreThreshold: 0.5,
		SearchType:     "similarity",
		IncludeScores:  true,
	}
	if e.config.VectorRAG != nil {
		config.K = e.config.VectorRAG.RetrieverConfig.K
		config.ScoreThreshold = e.config.VectorRAG.RetrieverConfig.ScoreThreshold
		config.SearchType = e.config.VectorRAG.RetrieverConfig.SearchType
	}
	return e.QueryWithConfig(ctx, query, config)
}

// QueryWithConfig performs a RAG query with custom configuration
func (e *BaseEngine) QueryWithConfig(ctx context.Context, query string, config *RetrievalConfig) (*QueryResult, error) {
	if config == nil {
		config = &RetrievalConfig{
			K:              4,
			ScoreThreshold: 0.5,
			SearchType:     "similarity",
		}
		if e.config.VectorRAG != nil {
			config.K = e.config.VectorRAG.RetrieverConfig.K
			config.ScoreThreshold = e.config.VectorRAG.RetrieverConfig.ScoreThreshold
			config.SearchType = e.config.VectorRAG.RetrieverConfig.SearchType
		}
	}

	// Perform retrieval
	searchResults, err := e.retriever.RetrieveWithConfig(ctx, query, config)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	if len(searchResults) == 0 {
		return &QueryResult{
			Query:      query,
			Answer:     "No relevant information found.",
			Sources:    []Document{},
			Context:    "",
			Confidence: 0.0,
		}, nil
	}

	// Extract documents from search results
	docs := make([]Document, len(searchResults))
	for i, result := range searchResults {
		docs[i] = result.Document
	}

	// Build context from retrieved documents
	context := e.buildContext(searchResults, config.IncludeScores)

	// Calculate confidence based on average score
	confidence := e.calculateConfidence(searchResults)

	return &QueryResult{
		Query:   query,
		Sources: docs,
		Context: context,
		Metadata: map[string]any{
			"retrieval_config": config,
			"num_documents":    len(docs),
			"avg_score":        confidence,
		},
		Confidence: confidence,
	}, nil
}

// AddDocuments adds documents to the base engine
func (e *BaseEngine) AddDocuments(ctx context.Context, docs []Document) error {
	// This base implementation doesn't store documents directly
	// Subclasses should override this method to implement actual storage
	return fmt.Errorf("AddDocuments not implemented for base engine")
}

// DeleteDocument removes a document from the base engine
func (e *BaseEngine) DeleteDocument(ctx context.Context, docID string) error {
	return fmt.Errorf("DeleteDocument not implemented for base engine")
}

// UpdateDocument updates an existing document in the base engine
func (e *BaseEngine) UpdateDocument(ctx context.Context, doc Document) error {
	return fmt.Errorf("UpdateDocument not implemented for base engine")
}

// SimilaritySearch performs similarity search without generation
func (e *BaseEngine) SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error) {
	scoreThreshold := 0.5
	if e.config.VectorRAG != nil {
		scoreThreshold = e.config.VectorRAG.RetrieverConfig.ScoreThreshold
	}

	config := &RetrievalConfig{
		K:              k,
		ScoreThreshold: scoreThreshold,
		SearchType:     "similarity",
		IncludeScores:  false,
	}

	searchResults, err := e.retriever.RetrieveWithConfig(ctx, query, config)
	if err != nil {
		return nil, err
	}

	docs := make([]Document, len(searchResults))
	for i, result := range searchResults {
		docs[i] = result.Document
	}

	return docs, nil
}

// SimilaritySearchWithScores performs similarity search with scores
func (e *BaseEngine) SimilaritySearchWithScores(ctx context.Context, query string, k int) ([]DocumentSearchResult, error) {
	scoreThreshold := 0.5
	if e.config.VectorRAG != nil {
		scoreThreshold = e.config.VectorRAG.RetrieverConfig.ScoreThreshold
	}

	config := &RetrievalConfig{
		K:              k,
		ScoreThreshold: scoreThreshold,
		SearchType:     "similarity",
		IncludeScores:  true,
	}

	return e.retriever.RetrieveWithConfig(ctx, query, config)
}

// buildContext builds context string from search results
func (e *BaseEngine) buildContext(results []DocumentSearchResult, includeScores bool) string {
	if len(results) == 0 {
		return ""
	}

	context := ""
	for i, result := range results {
		doc := result.Document

		context += fmt.Sprintf("Document %d:\n", i+1)
		if includeScores {
			context += fmt.Sprintf("Score: %.4f\n", result.Score)
		}

		// Add key metadata if available
		if doc.Metadata != nil {
			if title, ok := doc.Metadata["title"]; ok {
				context += fmt.Sprintf("Title: %v\n", title)
			}
			if source, ok := doc.Metadata["source"]; ok {
				context += fmt.Sprintf("Source: %v\n", source)
			}
		}

		context += fmt.Sprintf("Content: %s\n\n", doc.Content)
	}

	return context
}

// calculateConfidence calculates average confidence from search results
func (e *BaseEngine) calculateConfidence(results []DocumentSearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, result := range results {
		totalScore += math.Abs(result.Score)
	}

	return totalScore / float64(len(results))
}

// GetMetrics returns the current metrics
func (e *BaseEngine) GetMetrics() *Metrics {
	return e.metrics
}

// ResetMetrics resets all metrics
func (e *BaseEngine) ResetMetrics() {
	e.metrics = &Metrics{}
}

// CompositeEngine combines multiple RAG engines
type CompositeEngine struct {
	engines    []Engine
	aggregator func(results []*QueryResult) *QueryResult
	config     *Config
}

// NewCompositeEngine creates a new composite RAG engine
func NewCompositeEngine(engines []Engine, aggregator func([]*QueryResult) *QueryResult) *CompositeEngine {
	if aggregator == nil {
		aggregator = DefaultAggregator
	}

	return &CompositeEngine{
		engines:    engines,
		aggregator: aggregator,
		config:     &Config{},
	}
}

// Query performs a query using all composite engines and aggregates results
func (c *CompositeEngine) Query(ctx context.Context, query string) (*QueryResult, error) {
	results := make([]*QueryResult, len(c.engines))

	// Execute queries in parallel
	for i, engine := range c.engines {
		result, err := engine.Query(ctx, query)
		if err != nil {
			result = &QueryResult{
				Query:      query,
				Answer:     fmt.Sprintf("Engine %d failed: %v", i, err),
				Confidence: 0.0,
			}
		}
		results[i] = result
	}

	return c.aggregator(results), nil
}

// QueryWithConfig performs a query with custom configuration
func (c *CompositeEngine) QueryWithConfig(ctx context.Context, query string, config *RetrievalConfig) (*QueryResult, error) {
	results := make([]*QueryResult, len(c.engines))

	// Execute queries in parallel
	for i, engine := range c.engines {
		result, err := engine.QueryWithConfig(ctx, query, config)
		if err != nil {
			result = &QueryResult{
				Query:      query,
				Answer:     fmt.Sprintf("Engine %d failed: %v", i, err),
				Confidence: 0.0,
			}
		}
		results[i] = result
	}

	return c.aggregator(results), nil
}

// AddDocuments adds documents to all composite engines
func (c *CompositeEngine) AddDocuments(ctx context.Context, docs []Document) error {
	var errors []error

	for _, engine := range c.engines {
		if err := engine.AddDocuments(ctx, docs); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple engines failed: %v", errors)
	}

	return nil
}

// DeleteDocument removes a document from all composite engines
func (c *CompositeEngine) DeleteDocument(ctx context.Context, docID string) error {
	var errors []error

	for _, engine := range c.engines {
		if err := engine.DeleteDocument(ctx, docID); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple engines failed: %v", errors)
	}

	return nil
}

// UpdateDocument updates a document in all composite engines
func (c *CompositeEngine) UpdateDocument(ctx context.Context, doc Document) error {
	var errors []error

	for _, engine := range c.engines {
		if err := engine.UpdateDocument(ctx, doc); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("multiple engines failed: %v", errors)
	}

	return nil
}

// SimilaritySearch performs similarity search using the first available engine
func (c *CompositeEngine) SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error) {
	// Try engines in order until one succeeds
	for _, engine := range c.engines {
		docs, err := engine.SimilaritySearch(ctx, query, k)
		if err == nil {
			return docs, nil
		}
	}

	return nil, fmt.Errorf("all engines failed similarity search")
}

// SimilaritySearchWithScores performs similarity search with scores using the first available engine
func (c *CompositeEngine) SimilaritySearchWithScores(ctx context.Context, query string, k int) ([]DocumentSearchResult, error) {
	// Try engines in order until one succeeds
	for _, engine := range c.engines {
		results, err := engine.SimilaritySearchWithScores(ctx, query, k)
		if err == nil {
			return results, nil
		}
	}

	return nil, fmt.Errorf("all engines failed similarity search")
}

// DefaultAggregator provides default result aggregation logic
func DefaultAggregator(results []*QueryResult) *QueryResult {
	if len(results) == 0 {
		return nil
	}

	if len(results) == 1 {
		return results[0]
	}

	// Find the result with highest confidence
	bestResult := results[0]
	for _, result := range results[1:] {
		if result.Confidence > bestResult.Confidence {
			bestResult = result
		}
	}

	// Combine sources from all results
	allSources := make([]Document, 0)
	seenIDs := make(map[string]bool)

	for _, result := range results {
		for _, doc := range result.Sources {
			if !seenIDs[doc.ID] {
				allSources = append(allSources, doc)
				seenIDs[doc.ID] = true
			}
		}
	}

	// Update the best result with combined sources
	bestResult.Sources = allSources
	bestResult.Metadata["engines_used"] = len(results)
	bestResult.Metadata["total_sources"] = len(allSources)

	return bestResult
}

// WeightedAggregator provides weighted result aggregation logic
func WeightedAggregator(weights []float64) func([]*QueryResult) *QueryResult {
	return func(results []*QueryResult) *QueryResult {
		if len(results) == 0 {
			return nil
		}

		if len(weights) != len(results) {
			// Use equal weights if length mismatch
			weights = make([]float64, len(results))
			for i := range weights {
				weights[i] = 1.0
			}
		}

		// Calculate weighted score for each result
		weightedScores := make([]float64, len(results))
		for i, result := range results {
			weightedScores[i] = result.Confidence * weights[i]
		}

		// Find result with highest weighted score
		bestIndex := 0
		for i := 1; i < len(weightedScores); i++ {
			if weightedScores[i] > weightedScores[bestIndex] {
				bestIndex = i
			}
		}

		result := results[bestIndex]
		result.Metadata["weighted_score"] = weightedScores[bestIndex]
		result.Metadata["engines_used"] = len(results)

		return result
	}
}

package engine

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/splitter"
)

// vectorStoreRetrieverAdapter adapts vector store to Retriever interface
type vectorStoreRetrieverAdapter struct {
	vectorStore rag.VectorStore
	embedder    rag.Embedder
	topK        int
}

// Retrieve implements Retriever interface
func (a *vectorStoreRetrieverAdapter) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return a.RetrieveWithK(ctx, query, a.topK)
}

// RetrieveWithK implements Retriever interface
func (a *vectorStoreRetrieverAdapter) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	// Embed the query
	queryEmbedding, err := a.embedder.EmbedDocument(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Search in vector store
	results, err := a.vectorStore.Search(ctx, queryEmbedding, k)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Extract documents from results
	docs := make([]rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// RetrieveWithConfig implements Retriever interface
func (a *vectorStoreRetrieverAdapter) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	if config == nil {
		config = &rag.RetrievalConfig{
			K:              a.topK,
			ScoreThreshold: 0.0,
			SearchType:     "similarity",
			IncludeScores:  false,
		}
	}

	// Embed the query
	queryEmbedding, err := a.embedder.EmbedDocument(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	// Perform search
	var results []rag.DocumentSearchResult

	if len(config.Filter) > 0 {
		results, err = a.vectorStore.SearchWithFilter(ctx, queryEmbedding, config.K, config.Filter)
	} else {
		results, err = a.vectorStore.Search(ctx, queryEmbedding, config.K)
	}

	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Filter by score threshold if specified
	if config.ScoreThreshold > 0 {
		filtered := make([]rag.DocumentSearchResult, 0)
		for _, result := range results {
			if result.Score >= config.ScoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	return results, nil
}

// NewVectorStoreRetriever creates a vector store retriever
func NewVectorStoreRetriever(vectorStore rag.VectorStore, embedder rag.Embedder, topK int) rag.Retriever {
	return &vectorStoreRetrieverAdapter{
		vectorStore: vectorStore,
		embedder:    embedder,
		topK:        topK,
	}
}

// VectorRAGEngine implements traditional vector-based RAG
type VectorRAGEngine struct {
	vectorStore rag.VectorStore
	embedder    rag.Embedder
	llm         rag.LLMInterface
	config      rag.VectorRAGConfig
	baseEngine  *rag.BaseEngine
	metrics     *rag.Metrics
}

// NewVectorRAGEngine creates a new vector RAG engine
func NewVectorRAGEngine(llm rag.LLMInterface, embedder rag.Embedder, vectorStore rag.VectorStore, k int) (*VectorRAGEngine, error) {
	config := rag.VectorRAGConfig{
		ChunkSize:    1000,
		ChunkOverlap: 200,
		RetrieverConfig: rag.RetrievalConfig{
			K:              k,
			ScoreThreshold: 0.5,
			SearchType:     "similarity",
		},
	}

	return NewVectorRAGEngineWithConfig(llm, embedder, vectorStore, config)
}

// NewVectorRAGEngineWithConfig creates a new vector RAG engine with custom configuration
func NewVectorRAGEngineWithConfig(llm rag.LLMInterface, embedder rag.Embedder, vectorStore rag.VectorStore, config rag.VectorRAGConfig) (*VectorRAGEngine, error) {
	if vectorStore == nil {
		return nil, fmt.Errorf("vector store is required")
	}

	if embedder == nil {
		return nil, fmt.Errorf("embedder is required")
	}

	// Set defaults
	if config.ChunkSize == 0 {
		config.ChunkSize = 1000
	}
	if config.ChunkOverlap == 0 {
		config.ChunkOverlap = 200
	}
	if config.RetrieverConfig.K == 0 {
		config.RetrieverConfig.K = 4
	}

	// Create a simple retriever adapter directly
	retriever := &vectorStoreRetrieverAdapter{
		vectorStore: vectorStore,
		embedder:    embedder,
		topK:        config.RetrieverConfig.K,
	}

	baseEngine := rag.NewBaseEngine(retriever, embedder, &rag.Config{
		VectorRAG: &config,
	})

	return &VectorRAGEngine{
		vectorStore: vectorStore,
		embedder:    embedder,
		llm:         llm,
		config:      config,
		baseEngine:  baseEngine,
		metrics:     &rag.Metrics{},
	}, nil
}

// Query performs a vector RAG query
func (v *VectorRAGEngine) Query(ctx context.Context, query string) (*rag.QueryResult, error) {
	startTime := time.Now()

	// Perform similarity search
	searchResults, err := v.SimilaritySearchWithScores(ctx, query, v.config.RetrieverConfig.K)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Apply reranking if enabled
	if v.config.EnableReranking && v.config.RetrieverConfig.SearchType != "mmr" {
		searchResults2, err := v.rerankResults(ctx, query, searchResults)
		if err != nil {
			// Continue without reranking if it fails
			// searchResults = searchResults // Use original results
		} else {
			searchResults = searchResults2
		}
	}

	if len(searchResults) == 0 {
		return &rag.QueryResult{
				Query:      query,
				Answer:     "No relevant information found.",
				Sources:    []rag.Document{},
				Context:    "",
				Confidence: 0.0,
				Metadata: map[string]any{
					"engine_type": "vector_rag",
					"search_type": v.config.RetrieverConfig.SearchType,
				},
			},
			nil
	}

	// Extract documents from search results
	docs := make([]rag.Document, len(searchResults))
	for i, result := range searchResults {
		docs[i] = result.Document
	}

	// Build context from retrieved documents
	contextStr := v.buildContext(searchResults)

	// Calculate confidence based on search scores
	confidence := v.calculateConfidence(searchResults)

	responseTime := time.Since(startTime)

	return &rag.QueryResult{
		Query:        query,
		Sources:      docs,
		Context:      contextStr,
		Confidence:   confidence,
		ResponseTime: responseTime,
		Metadata: map[string]any{
			"engine_type":    "vector_rag",
			"search_type":    v.config.RetrieverConfig.SearchType,
			"num_results":    len(searchResults),
			"avg_score":      confidence,
			"reranking_used": v.config.EnableReranking,
		},
	}, nil
}

// QueryWithConfig performs a vector RAG query with custom configuration
func (v *VectorRAGEngine) QueryWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) (*rag.QueryResult, error) {
	if config == nil {
		config = &v.config.RetrieverConfig
	}

	startTime := time.Now()

	// Perform similarity search with custom config
	searchResults, err := v.vectorStore.SearchWithFilter(
		ctx,
		v.embedQuery(ctx, query),
		config.K,
		config.Filter,
	)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	// Filter by score threshold
	filteredResults := make([]rag.DocumentSearchResult, 0)
	for _, result := range searchResults {
		if result.Score >= config.ScoreThreshold {
			filteredResults = append(filteredResults, result)
		}
	}

	// Apply different search strategies
	switch config.SearchType {
	case "mmr":
		filteredResults = v.applyMMR(filteredResults, config.K)
	case "hybrid":
		// Hybrid search would combine vector and keyword search
		// For now, fall back to similarity search
	}

	// Apply reranking if enabled
	if v.config.EnableReranking {
		filteredResults, _ = v.rerankResults(ctx, query, filteredResults)
	}

	if len(filteredResults) == 0 {
		return &rag.QueryResult{
				Query:      query,
				Answer:     "No relevant information found.",
				Sources:    []rag.Document{},
				Context:    "",
				Confidence: 0.0,
			},
			nil
	}

	// Extract documents from search results
	docs := make([]rag.Document, len(filteredResults))
	for i, result := range filteredResults {
		docs[i] = result.Document
	}

	// Build context from retrieved documents
	contextStr := v.buildContext(filteredResults)

	// Calculate confidence based on search scores
	confidence := v.calculateConfidence(filteredResults)

	responseTime := time.Since(startTime)

	return &rag.QueryResult{
		Query:        query,
		Sources:      docs,
		Context:      contextStr,
		Confidence:   confidence,
		ResponseTime: responseTime,
		Metadata: map[string]any{
			"engine_type":     "vector_rag",
			"search_type":     config.SearchType,
			"num_results":     len(filteredResults),
			"avg_score":       confidence,
			"reranking_used":  v.config.EnableReranking,
			"score_threshold": config.ScoreThreshold,
			"filters_applied": config.Filter != nil,
		},
	}, nil
}

// AddDocuments adds documents to the vector store
func (v *VectorRAGEngine) AddDocuments(ctx context.Context, docs []rag.Document) error {
	startTime := time.Now()

	// Process documents: split into chunks if needed
	processedDocs := make([]rag.Document, 0)
	splitter := splitter.NewSimpleTextSplitter(v.config.ChunkSize, v.config.ChunkOverlap)

	for _, doc := range docs {
		// Split document into chunks
		chunks := splitter.SplitDocuments([]rag.Document{doc})
		processedDocs = append(processedDocs, chunks...)
	}

	// Generate embeddings for documents
	for i := range processedDocs {
		embedding, err := v.embedder.EmbedDocument(ctx, processedDocs[i].Content)
		if err != nil {
			return fmt.Errorf("failed to embed document %s: %w", processedDocs[i].ID, err)
		}
		processedDocs[i].Embedding = embedding
	}

	// Add documents to vector store
	if err := v.vectorStore.Add(ctx, processedDocs); err != nil {
		return fmt.Errorf("failed to add documents to vector store: %w", err)
	}

	// Update metrics
	v.metrics.IndexingLatency = time.Since(startTime)
	v.metrics.TotalDocuments += int64(len(docs))

	return nil
}

// DeleteDocument removes documents from the vector store
func (v *VectorRAGEngine) DeleteDocument(ctx context.Context, docID string) error {
	return v.vectorStore.Delete(ctx, []string{docID})
}

// UpdateDocument updates documents in the vector store
func (v *VectorRAGEngine) UpdateDocument(ctx context.Context, doc rag.Document) error {
	// Generate new embedding for the document
	embedding, err := v.embedder.EmbedDocument(ctx, doc.Content)
	if err != nil {
		return fmt.Errorf("failed to embed document %s: %w", doc.ID, err)
	}
	doc.Embedding = embedding

	return v.vectorStore.Update(ctx, []rag.Document{doc})
}

// SimilaritySearch performs similarity search without generation
func (v *VectorRAGEngine) SimilaritySearch(ctx context.Context, query string, k int) ([]rag.Document, error) {
	searchResults, err := v.SimilaritySearchWithScores(ctx, query, k)
	if err != nil {
		return nil, err
	}

	docs := make([]rag.Document, len(searchResults))
	for i, result := range searchResults {
		docs[i] = result.Document
	}

	return docs, nil
}

// SimilaritySearchWithScores performs similarity search with scores
func (v *VectorRAGEngine) SimilaritySearchWithScores(ctx context.Context, query string, k int) ([]rag.DocumentSearchResult, error) {
	queryEmbedding := v.embedQuery(ctx, query)
	return v.vectorStore.Search(ctx, queryEmbedding, k)
}

// embedQuery embeds a query using the configured embedder
func (v *VectorRAGEngine) embedQuery(ctx context.Context, query string) []float32 {
	embedding, err := v.embedder.EmbedDocument(ctx, query)
	if err != nil {
		// Return empty embedding if embedding fails
		return make([]float32, v.embedder.GetDimension())
	}
	return embedding
}

// buildContext builds context string from search results
func (v *VectorRAGEngine) buildContext(results []rag.DocumentSearchResult) string {
	if len(results) == 0 {
		return ""
	}

	contextStr := ""
	for i, result := range results {
		doc := result.Document

		contextStr += fmt.Sprintf("Document %d (Score: %.4f):\n", i+1, result.Score)

		// Add key metadata if available
		if doc.Metadata != nil {
			if title, ok := doc.Metadata["title"]; ok {
				contextStr += fmt.Sprintf("Title: %v\n", title)
			}
			if source, ok := doc.Metadata["source"]; ok {
				contextStr += fmt.Sprintf("Source: %v\n", source)
			}
			if url, ok := doc.Metadata["url"]; ok {
				contextStr += fmt.Sprintf("URL: %v\n", url)
			}
		}

		contextStr += fmt.Sprintf("Content: %s\n\n", doc.Content)
	}

	return contextStr
}

// calculateConfidence calculates average confidence from search results
func (v *VectorRAGEngine) calculateConfidence(results []rag.DocumentSearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, result := range results {
		totalScore += result.Score
	}

	return totalScore / float64(len(results))
}

// rerankResults reranks search results using the configured reranker
func (v *VectorRAGEngine) rerankResults(ctx context.Context, query string, results []rag.DocumentSearchResult) ([]rag.DocumentSearchResult, error) {
	// This is a placeholder for reranking
	// In a real implementation, this would use a reranking model or algorithm
	return results, nil
}

// applyMMR applies Maximal Marginal Relevance to search results
func (v *VectorRAGEngine) applyMMR(results []rag.DocumentSearchResult, k int) []rag.DocumentSearchResult {
	if len(results) <= k {
		return results
	}

	// Simple MMR implementation
	selected := make([]rag.DocumentSearchResult, 0)
	selected = append(selected, results[0]) // Always select the highest scoring result

	candidates := results[1:]

	for len(selected) < k && len(candidates) > 0 {
		// Find the candidate with highest MMR score
		bestIdx := 0
		bestScore := 0.0

		for i, candidate := range candidates {
			// Calculate relevance score
			relevance := candidate.Score

			// Calculate maximal similarity to already selected documents
			maxSimilarity := 0.0
			for _, selectedDoc := range selected {
				similarity := v.calculateSimilarity(candidate.Document, selectedDoc.Document)
				if similarity > maxSimilarity {
					maxSimilarity = similarity
				}
			}

			// MMR score: λ * relevance - (1-λ) * maxSimilarity
			lambda := 0.5 // Balance between relevance and diversity
			mmrScore := lambda*relevance - (1-lambda)*maxSimilarity

			if mmrScore > bestScore {
				bestScore = mmrScore
				bestIdx = i
			}
		}

		// Add the best candidate to selected results
		selected = append(selected, candidates[bestIdx])
		// Remove from candidates
		candidates = append(candidates[:bestIdx], candidates[bestIdx+1:]...)
	}

	return selected
}

// calculateSimilarity calculates similarity between two documents
func (v *VectorRAGEngine) calculateSimilarity(doc1, doc2 rag.Document) float64 {
	// Simple cosine similarity if embeddings are available
	if len(doc1.Embedding) > 0 && len(doc2.Embedding) > 0 {
		return cosineSimilarity(doc1.Embedding, doc2.Embedding)
	}

	// Fallback to Jaccard similarity on content
	return jaccardSimilarity(doc1.Content, doc2.Content)
}

// cosineSimilarity calculates cosine similarity between two embeddings
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32
	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return float64(dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))))
}

// jaccardSimilarity calculates Jaccard similarity between two texts
func jaccardSimilarity(a, b string) float64 {
	setA := make(map[string]bool)
	setB := make(map[string]bool)

	// Create sets of words
	wordsA := strings.Fields(strings.ToLower(a))
	wordsB := strings.Fields(strings.ToLower(b))

	for _, word := range wordsA {
		setA[word] = true
	}

	for _, word := range wordsB {
		setB[word] = true
	}

	// Calculate Jaccard similarity
	intersection := 0
	for word := range setA {
		if setB[word] {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// GetVectorStore returns the underlying vector store for advanced operations
func (v *VectorRAGEngine) GetVectorStore() rag.VectorStore {
	return v.vectorStore
}

// GetMetrics returns the current metrics
func (v *VectorRAGEngine) GetMetrics() *rag.Metrics {
	return v.metrics
}

// GetStats returns vector store statistics
func (v *VectorRAGEngine) GetStats(ctx context.Context) (*rag.VectorStoreStats, error) {
	return v.vectorStore.GetStats(ctx)
}

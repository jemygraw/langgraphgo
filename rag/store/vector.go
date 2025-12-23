package store

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/smallnest/langgraphgo/rag"
)

// InMemoryVectorStore is a simple in-memory vector store implementation
type InMemoryVectorStore struct {
	documents  []rag.Document
	embeddings [][]float32
	embedder   rag.Embedder
}

// NewInMemoryVectorStore creates a new InMemoryVectorStore
func NewInMemoryVectorStore(embedder rag.Embedder) *InMemoryVectorStore {
	return &InMemoryVectorStore{
		documents:  make([]rag.Document, 0),
		embeddings: make([][]float32, 0),
		embedder:   embedder,
	}
}

// AddWithEmbedding adds a document to the in-memory vector store with an explicit embedding
func (s *InMemoryVectorStore) AddWithEmbedding(ctx context.Context, doc rag.Document, embedding []float32) error {
	s.documents = append(s.documents, doc)
	s.embeddings = append(s.embeddings, embedding)
	return nil
}

// Add adds multiple documents to the in-memory vector store
func (s *InMemoryVectorStore) Add(ctx context.Context, documents []rag.Document) error {
	for _, doc := range documents {
		embedding := doc.Embedding
		if len(embedding) == 0 {
			if s.embedder == nil {
				return fmt.Errorf("no embedder configured and document has no embedding")
			}
			var err error
			embedding, err = s.embedder.EmbedDocument(ctx, doc.Content)
			if err != nil {
				return fmt.Errorf("failed to embed document: %w", err)
			}
		}
		s.documents = append(s.documents, doc)
		s.embeddings = append(s.embeddings, embedding)
	}
	return nil
}

// AddBatch adds multiple documents with explicit embeddings
func (s *InMemoryVectorStore) AddBatch(ctx context.Context, documents []rag.Document, embeddings [][]float32) error {
	if len(documents) != len(embeddings) {
		return fmt.Errorf("documents and embeddings must have same length")
	}

	s.documents = append(s.documents, documents...)
	s.embeddings = append(s.embeddings, embeddings...)
	return nil
}

// Search performs similarity search
func (s *InMemoryVectorStore) Search(ctx context.Context, queryEmbedding []float32, k int) ([]rag.DocumentSearchResult, error) {
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive")
	}

	if len(s.documents) == 0 {
		return []rag.DocumentSearchResult{}, nil
	}

	// Calculate similarities
	type docScore struct {
		index int
		score float64
	}

	scores := make([]docScore, len(s.documents))
	for i, docEmb := range s.embeddings {
		similarity := cosineSimilarity32(queryEmbedding, docEmb)
		scores[i] = docScore{index: i, score: similarity}
	}

	// Sort by similarity score (descending)
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if k > len(scores) {
		k = len(scores)
	}

	results := make([]rag.DocumentSearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = rag.DocumentSearchResult{
			Document: s.documents[scores[i].index],
			Score:    float64(scores[i].score),
		}
	}

	return results, nil
}

// SearchWithFilter performs similarity search with filters
func (s *InMemoryVectorStore) SearchWithFilter(ctx context.Context, queryEmbedding []float32, k int, filter map[string]any) ([]rag.DocumentSearchResult, error) {
	if k <= 0 {
		return nil, fmt.Errorf("k must be positive")
	}

	// Filter documents first
	var filteredDocs []rag.Document
	var filteredEmbeddings [][]float32

	for i, doc := range s.documents {
		if s.matchesFilter(doc, filter) {
			filteredDocs = append(filteredDocs, doc)
			filteredEmbeddings = append(filteredEmbeddings, s.embeddings[i])
		}
	}

	if len(filteredDocs) == 0 {
		return []rag.DocumentSearchResult{}, nil
	}

	// Calculate similarities
	type docScore struct {
		index int
		score float64
	}

	scores := make([]docScore, len(filteredDocs))
	for i, docEmb := range filteredEmbeddings {
		similarity := cosineSimilarity32(queryEmbedding, docEmb)
		scores[i] = docScore{index: i, score: similarity}
	}

	// Sort by similarity score (descending)
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	if k > len(scores) {
		k = len(scores)
	}

	results := make([]rag.DocumentSearchResult, k)
	for i := 0; i < k; i++ {
		results[i] = rag.DocumentSearchResult{
			Document: filteredDocs[scores[i].index],
			Score:    float64(scores[i].score),
		}
	}

	return results, nil
}

// Delete removes a document by ID
func (s *InMemoryVectorStore) Delete(ctx context.Context, ids []string) error {
	idMap := make(map[string]bool)
	for _, id := range ids {
		idMap[id] = true
	}

	var newDocs []rag.Document
	var newEmbeddings [][]float32

	for i, doc := range s.documents {
		if !idMap[doc.ID] {
			newDocs = append(newDocs, doc)
			newEmbeddings = append(newEmbeddings, s.embeddings[i])
		}
	}

	s.documents = newDocs
	s.embeddings = newEmbeddings
	return nil
}

// UpdateWithEmbedding updates a document and its embedding
func (s *InMemoryVectorStore) UpdateWithEmbedding(ctx context.Context, doc rag.Document, embedding []float32) error {
	for i, existingDoc := range s.documents {
		if existingDoc.ID == doc.ID {
			s.documents[i] = doc
			s.embeddings[i] = embedding
			return nil
		}
	}
	return fmt.Errorf("document not found: %s", doc.ID)
}

// Update updates documents in the vector store
func (s *InMemoryVectorStore) Update(ctx context.Context, documents []rag.Document) error {
	for _, doc := range documents {
		embedding := doc.Embedding
		if len(embedding) == 0 {
			if s.embedder == nil {
				return fmt.Errorf("no embedder configured and document %s has no embedding", doc.ID)
			}
			var err error
			embedding, err = s.embedder.EmbedDocument(ctx, doc.Content)
			if err != nil {
				return fmt.Errorf("failed to embed document %s: %w", doc.ID, err)
			}
		}

		found := false
		for i, existingDoc := range s.documents {
			if existingDoc.ID == doc.ID {
				s.documents[i] = doc
				s.embeddings[i] = embedding
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("document not found: %s", doc.ID)
		}
	}
	return nil
}

// GetStats returns statistics about the vector store
func (s *InMemoryVectorStore) GetStats(ctx context.Context) (*rag.VectorStoreStats, error) {
	stats := &rag.VectorStoreStats{
		TotalDocuments: len(s.documents),
		TotalVectors:   len(s.embeddings),
		LastUpdated:    time.Now(),
	}

	if len(s.embeddings) > 0 {
		stats.Dimension = len(s.embeddings[0])
	}

	return stats, nil
}

// Close closes the vector store (no-op for in-memory implementation)
func (s *InMemoryVectorStore) Close() error {
	// Clear all data
	s.documents = make([]rag.Document, 0)
	s.embeddings = make([][]float32, 0)
	return nil
}

// matchesFilter checks if a document matches the given filter
func (s *InMemoryVectorStore) matchesFilter(doc rag.Document, filter map[string]any) bool {
	for key, value := range filter {
		docValue, exists := doc.Metadata[key]
		if !exists || docValue != value {
			return false
		}
	}
	return true
}

// cosineSimilarity32 calculates cosine similarity between two float32 vectors
func cosineSimilarity32(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct float64
	var normA float64
	var normB float64

	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

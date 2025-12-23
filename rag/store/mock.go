package store

import (
	"context"
	"math"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

// MockEmbedder is a simple mock embedder for testing
type MockEmbedder struct {
	Dimension int
}

// NewMockEmbedder creates a new MockEmbedder
func NewMockEmbedder(dimension int) *MockEmbedder {
	return &MockEmbedder{
		Dimension: dimension,
	}
}

// EmbedDocument generates mock embedding for a document
func (e *MockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.generateEmbedding(text), nil
}

// EmbedDocuments generates mock embeddings for documents
func (e *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = e.generateEmbedding(text)
	}
	return embeddings, nil
}

// GetDimension returns the embedding dimension
func (e *MockEmbedder) GetDimension() int {
	return e.Dimension
}

func (e *MockEmbedder) generateEmbedding(text string) []float32 {
	// Simple deterministic embedding based on text content
	embedding := make([]float32, e.Dimension)

	for i := 0; i < e.Dimension; i++ {
		var sum float64
		for j, char := range text {
			sum += float64(char) * float64(i+j+1)
		}
		embedding[i] = float32(math.Sin(sum / 1000.0))
	}

	// Normalize
	var norm float32
	for _, v := range embedding {
		norm += v * v
	}
	norm = float32(math.Sqrt(float64(norm)))

	if norm > 0 {
		for i := range embedding {
			embedding[i] /= norm
		}
	}

	return embedding
}

// SimpleReranker is a simple reranker that scores documents based on keyword matching
type SimpleReranker struct {
	// Can be extended with more sophisticated reranking logic
}

// NewSimpleReranker creates a new SimpleReranker
func NewSimpleReranker() *SimpleReranker {
	return &SimpleReranker{}
}

// Rerank reranks documents based on query relevance
func (r *SimpleReranker) Rerank(ctx context.Context, query string, documents []rag.DocumentSearchResult) ([]rag.DocumentSearchResult, error) {
	queryTerms := strings.Fields(strings.ToLower(query))

	type docScore struct {
		doc   rag.DocumentSearchResult
		score float64
	}

	scores := make([]docScore, len(documents))
	for i, docResult := range documents {
		content := strings.ToLower(docResult.Document.Content)

		// Simple scoring: count query term occurrences
		var score float64
		for _, term := range queryTerms {
			score += float64(strings.Count(content, term))
		}

		// Normalize by document length
		if len(content) > 0 {
			score = score / float64(len(content)) * 1000
		}

		// Combine with original score
		finalScore := 0.7*float64(docResult.Score) + 0.3*score

		scores[i] = docScore{doc: rag.DocumentSearchResult{
			Document: docResult.Document,
			Score:    finalScore,
			Metadata: docResult.Metadata,
		}, score: finalScore}
	}

	// Sort by score (descending)
	for i := range scores {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	results := make([]rag.DocumentSearchResult, len(scores))
	for i, s := range scores {
		results[i] = s.doc
	}

	return results, nil
}

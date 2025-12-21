package retriever

import (
	"context"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

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
		finalScore := 0.7*docResult.Score + 0.3*score

		scores[i] = docScore{doc: rag.DocumentSearchResult{
			Document: docResult.Document,
			Score:    finalScore,
			Metadata: docResult.Metadata,
		}, score: finalScore}
	}

	// Sort by score (descending)
	for i := 0; i < len(scores); i++ {
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

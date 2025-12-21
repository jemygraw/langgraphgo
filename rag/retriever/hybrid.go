package retriever

import (
	"context"
	"fmt"

	"github.com/smallnest/langgraphgo/rag"
)

// HybridRetriever combines multiple retrieval strategies
type HybridRetriever struct {
	retrievers []rag.Retriever
	weights    []float64
	config     rag.RetrievalConfig
}

// NewHybridRetriever creates a new hybrid retriever that combines multiple retrievers
func NewHybridRetriever(retrievers []rag.Retriever, weights []float64, config rag.RetrievalConfig) *HybridRetriever {
	if len(weights) == 0 {
		// Use equal weights if none provided
		weights = make([]float64, len(retrievers))
		for i := range weights {
			weights[i] = 1.0
		}
	}

	if len(weights) != len(retrievers) {
		// Adjust weights to match number of retrievers
		newWeights := make([]float64, len(retrievers))
		for i := range newWeights {
			if i < len(weights) {
				newWeights[i] = weights[i]
			} else {
				newWeights[i] = 1.0
			}
		}
		weights = newWeights
	}

	return &HybridRetriever{
		retrievers: retrievers,
		weights:    weights,
		config:     config,
	}
}

// Retrieve retrieves documents using all configured retrievers and combines results
func (h *HybridRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return h.RetrieveWithK(ctx, query, h.config.K)
}

// RetrieveWithK retrieves exactly k documents using hybrid strategy
func (h *HybridRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	config := h.config
	config.K = k
	results, err := h.RetrieveWithConfig(ctx, query, &config)
	if err != nil {
		return nil, err
	}

	docs := make([]rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// RetrieveWithConfig retrieves documents with custom configuration using hybrid strategy
func (h *HybridRetriever) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	if config == nil {
		config = &h.config
	}

	// Collect results from all retrievers
	allResults := make([][]rag.DocumentSearchResult, len(h.retrievers))

	for i, retriever := range h.retrievers {
		results, err := retriever.RetrieveWithConfig(ctx, query, config)
		if err != nil {
			// Continue with other retrievers if one fails
			allResults[i] = []rag.DocumentSearchResult{}
		} else {
			allResults[i] = results
		}
	}

	// Combine and score results
	combinedResults := h.combineResults(allResults)

	// Filter by score threshold
	if config.ScoreThreshold > 0 {
		filtered := make([]rag.DocumentSearchResult, 0)
		for _, result := range combinedResults {
			if result.Score >= config.ScoreThreshold {
				filtered = append(filtered, result)
			}
		}
		combinedResults = filtered
	}

	// Limit to K results
	if len(combinedResults) > config.K {
		combinedResults = combinedResults[:config.K]
	}

	return combinedResults, nil
}

// combineResults combines results from multiple retrievers using weighted scoring
func (h *HybridRetriever) combineResults(allResults [][]rag.DocumentSearchResult) []rag.DocumentSearchResult {
	// Create a map to track documents and their scores
	documentScores := make(map[string]*CombinedDocumentScore)

	// Process results from each retriever
	for retrieverIdx, results := range allResults {
		weight := h.weights[retrieverIdx]

		for _, result := range results {
			docID := result.Document.ID

			if existing, found := documentScores[docID]; found {
				// Update existing document score
				existing.TotalScore += float64(result.Score) * weight
				existing.RetrieverCount++
				existing.Sources = append(existing.Sources, fmt.Sprintf("retriever_%d", retrieverIdx))
			} else {
				// Add new document
				documentScores[docID] = &CombinedDocumentScore{
					Document:       result.Document,
					TotalScore:     float64(result.Score) * weight,
					RetrieverCount: 1,
					Sources:        []string{fmt.Sprintf("retriever_%d", retrieverIdx)},
					Metadata:       result.Metadata,
				}
			}
		}
	}

	// Convert map back to slice and calculate final scores
	combinedResults := make([]rag.DocumentSearchResult, 0, len(documentScores))
	for _, combined := range documentScores {
		// Calculate final score as weighted average
		finalScore := combined.TotalScore / float64(combined.RetrieverCount)

		// Boost score if document comes from multiple retrievers
		if combined.RetrieverCount > 1 {
			finalScore *= 1.1 // 10% boost for multi-source documents
		}

		// Cap score at 1.0
		if finalScore > 1.0 {
			finalScore = 1.0
		}

		result := rag.DocumentSearchResult{
			Document: combined.Document,
			Score:    finalScore,
			Metadata: map[string]any{
				"retriever_count":   combined.RetrieverCount,
				"sources":           combined.Sources,
				"original_metadata": combined.Metadata,
			},
		}

		combinedResults = append(combinedResults, result)
	}

	// Sort results by score (descending)
	h.sortResults(combinedResults)

	return combinedResults
}

// CombinedDocumentScore tracks score information for a document from multiple retrievers
type CombinedDocumentScore struct {
	Document       rag.Document
	TotalScore     float64
	RetrieverCount int
	Sources        []string
	Metadata       map[string]any
}

// sortResults sorts results by score in descending order
func (h *HybridRetriever) sortResults(results []rag.DocumentSearchResult) {
	// Simple bubble sort - in practice, you'd use a more efficient sorting algorithm
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
}

// GetRetrieverCount returns the number of retrievers being used
func (h *HybridRetriever) GetRetrieverCount() int {
	return len(h.retrievers)
}

// GetWeights returns the weights being used for each retriever
func (h *HybridRetriever) GetWeights() []float64 {
	weights := make([]float64, len(h.weights))
	copy(weights, h.weights)
	return weights
}

// SetWeights updates the weights for each retriever
func (h *HybridRetriever) SetWeights(weights []float64) error {
	if len(weights) != len(h.retrievers) {
		return fmt.Errorf("number of weights (%d) must match number of retrievers (%d)",
			len(weights), len(h.retrievers))
	}

	h.weights = make([]float64, len(weights))
	copy(h.weights, weights)
	return nil
}

// AddRetriever adds a new retriever to the hybrid strategy
func (h *HybridRetriever) AddRetriever(retriever rag.Retriever, weight float64) {
	h.retrievers = append(h.retrievers, retriever)
	h.weights = append(h.weights, weight)
}

// RemoveRetriever removes a retriever by index
func (h *HybridRetriever) RemoveRetriever(index int) error {
	if index < 0 || index >= len(h.retrievers) {
		return fmt.Errorf("index %d out of range", index)
	}

	h.retrievers = append(h.retrievers[:index], h.retrievers[index+1:]...)
	h.weights = append(h.weights[:index], h.weights[index+1:]...)
	return nil
}

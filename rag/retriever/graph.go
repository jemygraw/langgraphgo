package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

// GraphRetriever implements document retrieval using knowledge graphs
type GraphRetriever struct {
	knowledgeGraph rag.KnowledgeGraph
	embedder       rag.Embedder
	config         rag.RetrievalConfig
}

// NewGraphRetriever creates a new graph retriever
func NewGraphRetriever(knowledgeGraph rag.KnowledgeGraph, embedder rag.Embedder, config rag.RetrievalConfig) *GraphRetriever {
	if config.K == 0 {
		config.K = 4
	}
	if config.ScoreThreshold == 0 {
		config.ScoreThreshold = 0.3
	}
	if config.SearchType == "" {
		config.SearchType = "graph"
	}

	return &GraphRetriever{
		knowledgeGraph: knowledgeGraph,
		embedder:       embedder,
		config:         config,
	}
}

// Retrieve retrieves documents based on a query using the knowledge graph
func (r *GraphRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return r.RetrieveWithK(ctx, query, r.config.K)
}

// RetrieveWithK retrieves exactly k documents
func (r *GraphRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	config := r.config
	config.K = k
	results, err := r.RetrieveWithConfig(ctx, query, &config)
	if err != nil {
		return nil, err
	}

	docs := make([]rag.Document, len(results))
	for i, result := range results {
		docs[i] = result.Document
	}

	return docs, nil
}

// RetrieveWithConfig retrieves documents with custom configuration
func (r *GraphRetriever) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	if config == nil {
		config = &r.config
	}

	// Extract entities from the query
	entities, err := r.extractEntitiesFromQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities from query: %w", err)
	}

	// If no entities found, perform simple similarity search on entity names
	if len(entities) == 0 {
		return r.performEntitySimilaritySearch(ctx, query, config)
	}

	// Build graph query based on extracted entities
	graphQuery := &rag.GraphQuery{
		MaxDepth: 3, // Default depth for entity traversal
	}

	// Use the first entity as the starting point
	graphQuery.StartEntity = entities[0].ID
	graphQuery.EntityType = entities[0].Type

	// Add filters from config
	if config.Filter != nil {
		graphQuery.Filters = config.Filter
	}

	// Perform graph query
	graphResult, err := r.knowledgeGraph.Query(ctx, graphQuery)
	if err != nil {
		return nil, fmt.Errorf("graph query failed: %w", err)
	}

	// Convert graph results to document search results
	results := r.graphResultsToSearchResults(graphResult, entities)

	// Apply score threshold filter
	if config.ScoreThreshold > 0 {
		filtered := make([]rag.DocumentSearchResult, 0)
		for _, result := range results {
			if result.Score >= config.ScoreThreshold {
				filtered = append(filtered, result)
			}
		}
		results = filtered
	}

	// Limit results to K
	if len(results) > config.K {
		results = results[:config.K]
	}

	return results, nil
}

// extractEntitiesFromQuery extracts entities from the query string
func (r *GraphRetriever) extractEntitiesFromQuery(ctx context.Context, query string) ([]*rag.Entity, error) {
	// This is a simplified entity extraction
	// In a real implementation, you'd use NLP models or external services

	entities := make([]*rag.Entity, 0)

	// Look for potential entities based on patterns
	// This is a placeholder - actual implementation would be more sophisticated
	words := r.extractWords(query)

	for _, word := range words {
		// Skip very short words or common words
		if len(word) < 3 || r.isCommonWord(word) {
			continue
		}

		// Try to find this entity in the knowledge graph
		entity, err := r.knowledgeGraph.GetEntity(ctx, word)
		if err == nil && entity != nil {
			entities = append(entities, entity)
		}
	}

	return entities, nil
}

// performEntitySimilaritySearch performs similarity search on entity names
func (r *GraphRetriever) performEntitySimilaritySearch(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	// This is a fallback method when no entities are extracted from the query
	// It performs similarity search on entity names/descriptions

	results := make([]rag.DocumentSearchResult, 0)

	// Get all entities from the knowledge graph
	// Note: In a real implementation, you'd have a method to list or search entities
	// For now, we'll return an empty result set
	// This would need to be implemented based on the specific knowledge graph interface

	return results, nil
}

// graphResultsToSearchResults converts graph query results to document search results
func (r *GraphRetriever) graphResultsToSearchResults(graphResult *rag.GraphQueryResult, queryEntities []*rag.Entity) []rag.DocumentSearchResult {
	results := make([]rag.DocumentSearchResult, 0)

	// Create documents from entities
	for _, entity := range graphResult.Entities {
		content := r.entityToDocumentContent(entity)

		doc := rag.Document{
			ID:      entity.ID,
			Content: content,
			Metadata: map[string]any{
				"entity_type": entity.Type,
				"entity_name": entity.Name,
				"properties":  entity.Properties,
				"source":      "knowledge_graph",
				"created_at":  entity.CreatedAt,
			},
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		}

		// Calculate score based on entity relevance to query
		score := r.calculateEntityScore(entity, queryEntities)

		results = append(results, rag.DocumentSearchResult{
			Document: doc,
			Score:    score,
			Metadata: map[string]any{
				"entity_match": true,
				"entity_type":  entity.Type,
			},
		})
	}

	// Create documents from relationships
	for _, relationship := range graphResult.Relationships {
		content := r.relationshipToDocumentContent(relationship)

		doc := rag.Document{
			ID:      relationship.ID,
			Content: content,
			Metadata: map[string]any{
				"relationship_type": relationship.Type,
				"source_entity":     relationship.Source,
				"target_entity":     relationship.Target,
				"source":            "knowledge_graph",
				"confidence":        relationship.Confidence,
			},
			CreatedAt: relationship.CreatedAt,
		}

		// Use relationship confidence as score
		score := relationship.Confidence

		results = append(results, rag.DocumentSearchResult{
			Document: doc,
			Score:    score,
			Metadata: map[string]any{
				"relationship_match": true,
				"relationship_type":  relationship.Type,
			},
		})
	}

	return results
}

// entityToDocumentContent converts an entity to document content
func (r *GraphRetriever) entityToDocumentContent(entity *rag.Entity) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Entity: %s\nType: %s\n", entity.Name, entity.Type))

	if entity.Properties != nil {
		if description, ok := entity.Properties["description"]; ok {
			content.WriteString(fmt.Sprintf("Description: %v\n", description))
		}

		// Add other relevant properties
		for key, value := range entity.Properties {
			if key != "description" {
				content.WriteString(fmt.Sprintf("%s: %v\n", key, value))
			}
		}
	}

	return content.String()
}

// relationshipToDocumentContent converts a relationship to document content
func (r *GraphRetriever) relationshipToDocumentContent(relationship *rag.Relationship) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Relationship: %s -> %s\nType: %s\n",
		relationship.Source, relationship.Target, relationship.Type))

	if relationship.Confidence > 0 {
		content.WriteString(fmt.Sprintf("Confidence: %.2f\n", relationship.Confidence))
	}

	if relationship.Properties != nil {
		for key, value := range relationship.Properties {
			content.WriteString(fmt.Sprintf("%s: %v\n", key, value))
		}
	}

	return content.String()
}

// calculateEntityScore calculates relevance score for an entity
func (r *GraphRetriever) calculateEntityScore(entity *rag.Entity, queryEntities []*rag.Entity) float64 {
	// Base score
	score := 0.5

	// Boost score if entity matches query entities
	for _, queryEntity := range queryEntities {
		if entity.ID == queryEntity.ID || entity.Name == queryEntity.Name {
			score += 0.5
		}
		// Boost if entities are of the same type
		if entity.Type == queryEntity.Type {
			score += 0.2
		}
	}

	// Cap score at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score
}

// extractWords extracts words from text
func (r *GraphRetriever) extractWords(text string) []string {
	words := make([]string, 0)
	current := ""

	for _, char := range text {
		if isAlphaNumeric(char) {
			current += string(char)
		} else {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// isCommonWord checks if a word is a common stop word
func (r *GraphRetriever) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "have": true, "has": true, "had": true, "do": true, "does": true,
		"did": true, "will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "can": true, "this": true, "that": true, "these": true,
		"those": true, "i": true, "you": true, "he": true, "she": true, "it": true,
		"we": true, "they": true, "what": true, "where": true, "when": true, "why": true,
		"how": true, "who": true, "which": true, "whose": true, "whom": true,
	}

	lowerWord := strings.ToLower(word)
	return commonWords[lowerWord]
}

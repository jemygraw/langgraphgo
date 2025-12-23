package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/smallnest/langgraphgo/rag"
)

// GraphRAGEngine implements GraphRAG functionality with knowledge graphs
type GraphRAGEngine struct {
	config         rag.GraphRAGConfig
	knowledgeGraph rag.KnowledgeGraph
	embedder       rag.Embedder
	llm            rag.LLMInterface
	baseEngine     *rag.BaseEngine
	metrics        *rag.Metrics
}

// NewGraphRAGEngine creates a new GraphRAG engine
func NewGraphRAGEngine(config rag.GraphRAGConfig, llm rag.LLMInterface, embedder rag.Embedder, kg rag.KnowledgeGraph) (*GraphRAGEngine, error) {
	if kg == nil {
		return nil, fmt.Errorf("knowledge graph is required")
	}

	// Set default extraction prompt if not provided
	if config.ExtractionPrompt == "" {
		config.ExtractionPrompt = DefaultExtractionPrompt
	}

	// Set default entity types if not provided
	if len(config.EntityTypes) == 0 {
		config.EntityTypes = DefaultEntityTypes
	}

	// Set default max depth if not provided
	if config.MaxDepth == 0 {
		config.MaxDepth = 3
	}

	baseEngine := rag.NewBaseEngine(nil, embedder, &rag.Config{
		GraphRAG: &config,
	})

	return &GraphRAGEngine{
		config:         config,
		knowledgeGraph: kg,
		embedder:       embedder,
		llm:            llm,
		baseEngine:     baseEngine,
		metrics:        &rag.Metrics{},
	}, nil
}

// Query performs a GraphRAG query
func (g *GraphRAGEngine) Query(ctx context.Context, query string) (*rag.QueryResult, error) {
	return g.QueryWithConfig(ctx, query, &rag.RetrievalConfig{
		K:              5,
		ScoreThreshold: 0.3,
		SearchType:     "graph",
		IncludeScores:  true,
	})
}

// QueryWithConfig performs a GraphRAG query with custom configuration
func (g *GraphRAGEngine) QueryWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) (*rag.QueryResult, error) {
	startTime := time.Now()

	// Extract entities from the query
	queryEntities, err := g.extractEntities(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to extract entities from query: %w", err)
	}

	// Build graph query
	graphQuery := rag.GraphQuery{
		Limit:   config.K,
		Filters: config.Filter,
	}

	// Add extracted entities to the query
	if len(queryEntities) > 0 {
		graphQuery.EntityTypes = []string{queryEntities[0].Type}
	}

	// Perform graph search
	graphResult, err := g.knowledgeGraph.Query(ctx, &graphQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to perform graph search: %w", err)
	}

	// Convert graph results to documents
	docs := g.graphResultsToDocuments(graphResult)

	// If no entities were found, fall back to entity search
	if len(docs) == 0 && len(queryEntities) > 0 {
		docs, err = g.entityBasedSearch(ctx, queryEntities, config.K)
		if err != nil {
			return nil, fmt.Errorf("failed entity-based search: %w", err)
		}
	}

	// Build context from graph results
	contextStr := g.buildGraphContext(graphResult, queryEntities)

	// Calculate confidence based on entity matches and relationships
	confidence := g.calculateGraphConfidence(graphResult, queryEntities)

	responseTime := time.Since(startTime)

	return &rag.QueryResult{
		Query:        query,
		Sources:      docs,
		Context:      contextStr,
		Confidence:   confidence,
		ResponseTime: responseTime,
		Metadata: map[string]any{
			"engine_type":     "graph_rag",
			"entities_found":  len(graphResult.Entities),
			"relationships":   len(graphResult.Relationships),
			"paths_found":     len(graphResult.Paths),
			"graph_query":     graphQuery,
			"extraction_time": responseTime,
		},
	}, nil
}

// AddDocuments adds documents to the knowledge graph
func (g *GraphRAGEngine) AddDocuments(ctx context.Context, docs []rag.Document) error {
	startTime := time.Now()

	for _, doc := range docs {
		// Extract entities from the document
		entities, err := g.extractEntities(ctx, doc.Content)
		if err != nil {
			return fmt.Errorf("failed to extract entities from document %s: %w", doc.ID, err)
		}

		// Extract relationships between entities
		relationships, err := g.extractRelationships(ctx, doc.Content, entities)
		if err != nil {
			return fmt.Errorf("failed to extract relationships from document %s: %w", doc.ID, err)
		}

		// Add entities to the knowledge graph
		for _, entity := range entities {
			if err := g.knowledgeGraph.AddEntity(ctx, entity); err != nil {
				return fmt.Errorf("failed to add entity %s: %w", entity.ID, err)
			}
		}

		// Add relationships to the knowledge graph
		for _, rel := range relationships {
			if err := g.knowledgeGraph.AddRelationship(ctx, rel); err != nil {
				return fmt.Errorf("failed to add relationship %s: %w", rel.ID, err)
			}
		}
	}

	g.metrics.IndexingLatency = time.Since(startTime)
	g.metrics.TotalDocuments += int64(len(docs))

	return nil
}

// DeleteDocument removes entities and relationships associated with a document
func (g *GraphRAGEngine) DeleteDocument(ctx context.Context, docID string) error {
	// This would require tracking which entities/relationships belong to which documents
	// For now, this is a placeholder implementation
	return fmt.Errorf("document deletion not implemented for GraphRAG engine")
}

// UpdateDocument updates a document in the knowledge graph
func (g *GraphRAGEngine) UpdateDocument(ctx context.Context, doc rag.Document) error {
	// Delete old entities and relationships, then add new ones
	if err := g.DeleteDocument(ctx, doc.ID); err != nil {
		return err
	}
	return g.AddDocuments(ctx, []rag.Document{doc})
}

// SimilaritySearch performs entity-based similarity search
func (g *GraphRAGEngine) SimilaritySearch(ctx context.Context, query string, k int) ([]rag.Document, error) {
	queryEntities, err := g.extractEntities(ctx, query)
	if err != nil {
		return nil, err
	}

	return g.entityBasedSearch(ctx, queryEntities, k)
}

// SimilaritySearchWithScores performs entity-based similarity search with scores
func (g *GraphRAGEngine) SimilaritySearchWithScores(ctx context.Context, query string, k int) ([]rag.DocumentSearchResult, error) {
	docs, err := g.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}

	results := make([]rag.DocumentSearchResult, len(docs))
	for i, doc := range docs {
		results[i] = rag.DocumentSearchResult{
			Document: doc,
			Score:    1.0, // GraphRAG doesn't provide traditional similarity scores
		}
	}

	return results, nil
}

// extractEntities extracts entities from text using the LLM
func (g *GraphRAGEngine) extractEntities(ctx context.Context, text string) ([]*rag.Entity, error) {
	prompt := fmt.Sprintf(g.config.ExtractionPrompt, text, strings.Join(g.config.EntityTypes, ", "))

	response, err := g.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var extractionResult EntityExtractionResult
	if err := json.Unmarshal([]byte(response), &extractionResult); err != nil {
		// Try to extract entities manually if JSON parsing fails
		return g.manualEntityExtraction(ctx, text), nil
	}

	// Convert extracted entities to Entity structs
	entities := make([]*rag.Entity, len(extractionResult.Entities))
	for i, extracted := range extractionResult.Entities {
		entity := &rag.Entity{
			ID:         extracted.Name,
			Type:       extracted.Type,
			Name:       extracted.Name,
			Properties: extracted.Properties,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		entities[i] = entity
	}

	return entities, nil
}

// extractRelationships extracts relationships between entities using the LLM
func (g *GraphRAGEngine) extractRelationships(ctx context.Context, text string, entities []*rag.Entity) ([]*rag.Relationship, error) {
	if len(entities) < 2 {
		return nil, nil
	}

	// Create a prompt for relationship extraction
	entityList := make([]string, len(entities))
	for i, entity := range entities {
		entityList[i] = fmt.Sprintf("%s (%s)", entity.Name, entity.Type)
	}

	prompt := fmt.Sprintf(RelationshipExtractionPrompt, text, strings.Join(entityList, ", "))

	response, err := g.llm.Generate(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var extractionResult RelationshipExtractionResult
	if err := json.Unmarshal([]byte(response), &extractionResult); err != nil {
		return g.manualRelationshipExtraction(ctx, text, entities), nil
	}

	// Convert extracted relationships to Relationship structs
	relationships := make([]*rag.Relationship, len(extractionResult.Relationships))
	for i, extracted := range extractionResult.Relationships {
		relationships[i] = &rag.Relationship{
			ID:         fmt.Sprintf("%s_%s_%s", extracted.Source, extracted.Type, extracted.Target),
			Source:     extracted.Source,
			Target:     extracted.Target,
			Type:       extracted.Type,
			Properties: extracted.Properties,
			CreatedAt:  time.Now(),
		}
	}

	return relationships, nil
}

// entityBasedSearch performs search based on entities
func (g *GraphRAGEngine) entityBasedSearch(ctx context.Context, entities []*rag.Entity, k int) ([]rag.Document, error) {
	if len(entities) == 0 {
		return []rag.Document{}, nil
	}

	// Use the first entity as the starting point for graph traversal
	relatedEntities, err := g.knowledgeGraph.GetRelatedEntities(ctx, entities[0].ID, 1)
	if err != nil {
		return nil, err
	}

	// Convert related entities to documents
	docs := make([]rag.Document, 0, len(relatedEntities))
	count := 0

	for _, entity := range relatedEntities {
		if count >= k {
			break
		}

		// Create a document from the entity
		content := fmt.Sprintf("Entity: %s\nType: %s\nDescription: %v",
			entity.Name, entity.Type, entity.Properties["description"])

		doc := rag.Document{
			ID:      entity.ID,
			Content: content,
			Metadata: map[string]any{
				"entity_type": entity.Type,
				"properties":  entity.Properties,
				"source":      "knowledge_graph",
			},
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		}

		docs = append(docs, doc)
		count++
	}

	return docs, nil
}

// graphResultsToDocuments converts graph query results to documents
func (g *GraphRAGEngine) graphResultsToDocuments(result *rag.GraphQueryResult) []rag.Document {
	docs := make([]rag.Document, 0, len(result.Entities))

	for _, entity := range result.Entities {
		content := fmt.Sprintf("Entity: %s\nType: %s\n", entity.Name, entity.Type)

		if entity.Properties != nil {
			if desc, ok := entity.Properties["description"]; ok {
				content += fmt.Sprintf("Description: %v\n", desc)
			}
		}

		doc := rag.Document{
			ID:      entity.ID,
			Content: content,
			Metadata: map[string]any{
				"entity_type": entity.Type,
				"properties":  entity.Properties,
				"source":      "knowledge_graph",
			},
			CreatedAt: entity.CreatedAt,
			UpdatedAt: entity.UpdatedAt,
		}

		docs = append(docs, doc)
	}

	return docs
}

// buildGraphContext builds context string from graph results
func (g *GraphRAGEngine) buildGraphContext(result *rag.GraphQueryResult, queryEntities []*rag.Entity) string {
	if len(result.Entities) == 0 {
		return "No relevant entities found in the knowledge graph."
	}

	var contextStr strings.Builder
	contextStr.WriteString("Knowledge Graph Information:\n\n")

	// Add entities
	contextStr.WriteString("Relevant Entities:\n")
	for _, entity := range result.Entities {
		contextStr.WriteString(fmt.Sprintf("- %s (%s): %v\n", entity.Name, entity.Type, entity.Properties))
	}

	// Add relationships
	if len(result.Relationships) > 0 {
		contextStr.WriteString("\nRelationships:\n")
		for _, rel := range result.Relationships {
			contextStr.WriteString(fmt.Sprintf("- %s -> %s (%s)\n",
				rel.Source, rel.Target, rel.Type))
		}
	}

	// Add paths
	if len(result.Paths) > 0 {
		contextStr.WriteString("\nEntity Paths:\n")
		for i, path := range result.Paths {
			pathStr := make([]string, len(path))
			for j, entity := range path {
				pathStr[j] = fmt.Sprintf("%s(%s)", entity.Name, entity.Type)
			}
			contextStr.WriteString(fmt.Sprintf("Path %d: %s\n", i+1, strings.Join(pathStr, " -> ")))
		}
	}

	return contextStr.String()
}

// calculateGraphConfidence calculates confidence based on graph results
func (g *GraphRAGEngine) calculateGraphConfidence(result *rag.GraphQueryResult, queryEntities []*rag.Entity) float64 {
	if len(result.Entities) == 0 {
		return 0.0
	}

	// Base confidence from number of entities found
	entityConfidence := float64(len(result.Entities)) / 10.0
	if entityConfidence > 1.0 {
		entityConfidence = 1.0
	}

	// Boost confidence if query entities were matched
	if len(queryEntities) > 0 {
		matchedEntities := 0
		for _, queryEntity := range queryEntities {
			for _, foundEntity := range result.Entities {
				if queryEntity.ID == foundEntity.ID || queryEntity.Name == foundEntity.Name {
					matchedEntities++
					break
				}
			}
		}
		entityConfidence += float64(matchedEntities) / float64(len(queryEntities)) * 0.3
	}

	// Consider relationships count (since store.Relationship doesn't have confidence)
	relConfidence := 0.0
	if len(result.Relationships) > 0 {
		relConfidence = float64(len(result.Relationships)) * 0.1 // Give some weight for having relationships
	}

	totalConfidence := entityConfidence + relConfidence
	if totalConfidence > 1.0 {
		totalConfidence = 1.0
	}

	return totalConfidence
}

// manualEntityExtraction provides a fallback for entity extraction
func (g *GraphRAGEngine) manualEntityExtraction(ctx context.Context, text string) []*rag.Entity {
	// Simple keyword-based entity extraction as fallback
	// In a real implementation, this would use more sophisticated NLP
	entities := make([]*rag.Entity, 0)

	// Look for capitalized words (potential entities)
	words := strings.FieldsSeq(text)
	for word := range words {
		if len(word) > 2 && unicode.IsUpper(rune(word[0])) {
			entity := &rag.Entity{
				ID:   word,
				Type: "UNKNOWN",
				Name: word,
				Properties: map[string]any{
					"description": fmt.Sprintf("Entity extracted from text: %s", word),
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			entities = append(entities, entity)
		}
	}

	return entities
}

// manualRelationshipExtraction provides a fallback for relationship extraction
func (g *GraphRAGEngine) manualRelationshipExtraction(ctx context.Context, text string, entities []*rag.Entity) []*rag.Relationship {
	// Simple co-occurrence based relationship extraction as fallback
	relationships := make([]*rag.Relationship, 0)

	// If entities appear close together in text, assume a relationship
	for i, entity1 := range entities {
		for j, entity2 := range entities {
			if i >= j {
				continue
			}
			relationship := &rag.Relationship{
				ID:         fmt.Sprintf("%s_related_to_%s", entity1.ID, entity2.ID),
				Source:     entity1.ID,
				Target:     entity2.ID,
				Type:       "RELATED_TO",
				Properties: map[string]any{},
				CreatedAt:  time.Now(),
			}
			relationships = append(relationships, relationship)
		}
	}

	return relationships
}

// GetKnowledgeGraph returns the underlying knowledge graph for advanced operations
func (g *GraphRAGEngine) GetKnowledgeGraph() rag.KnowledgeGraph {
	return g.knowledgeGraph
}

// GetMetrics returns the current metrics
func (g *GraphRAGEngine) GetMetrics() *rag.Metrics {
	return g.metrics
}

// Constants for default prompts and entity types
const (
	DefaultExtractionPrompt = `
Extract entities from the following text. Focus on these entity types: %s.
Return a JSON response with this structure:
{
  "entities": [
    {
      "name": "entity_name",
      "type": "entity_type",
      "description": "brief description",
      "properties": {}
    }
  ]
}

Text: %s
`

	RelationshipExtractionPrompt = `
Extract relationships between the following entities from this text.
Consider relationships like: works_with, located_in, created_by, part_of, related_to, etc.
Return a JSON response with this structure:
{
  "relationships": [
    {
      "source": "entity1_name",
      "target": "entity2_name",
      "type": "relationship_type",
      "properties": {},
      "confidence": 0.9
    }
  ]
}

Text: %s
Entities: %s
`
)

// DefaultEntityTypes contains commonly used entity types
var DefaultEntityTypes = []string{
	"PERSON",
	"ORGANIZATION",
	"LOCATION",
	"DATE",
	"PRODUCT",
	"EVENT",
	"CONCEPT",
	"TECHNOLOGY",
}

// Supporting structs for JSON parsing
type EntityExtractionResult struct {
	Entities []ExtractedEntity `json:"entities"`
}

type ExtractedEntity struct {
	Name        string         `json:"name"`
	Type        string         `json:"type"`
	Description string         `json:"description"`
	Properties  map[string]any `json:"properties"`
}

type RelationshipExtractionResult struct {
	Relationships []ExtractedRelationship `json:"relationships"`
}

type ExtractedRelationship struct {
	Source     string         `json:"source"`
	Target     string         `json:"target"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Confidence float64        `json:"confidence"`
}

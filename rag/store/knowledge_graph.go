package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

// NewKnowledgeGraph creates a new knowledge graph based on the database URL
func NewKnowledgeGraph(databaseURL string) (rag.KnowledgeGraph, error) {
	if strings.HasPrefix(databaseURL, "memory://") {
		return &MemoryGraph{
			entities:      make(map[string]rag.Entity),
			relationships: make(map[string]rag.Relationship),
			entityIndex:   make(map[string][]string),
		}, nil
	}

	if strings.HasPrefix(databaseURL, "falkordb://") {
		return NewFalkorDBGraph(databaseURL)
	}

	// Placeholder for other database types
	return nil, fmt.Errorf("only memory:// and falkordb:// URLs are currently supported")
}

// MemoryGraph implements an in-memory knowledge graph
type MemoryGraph struct {
	entities      map[string]rag.Entity
	relationships map[string]rag.Relationship
	entityIndex   map[string][]string
}

// AddEntity adds an entity to the memory graph
func (m *MemoryGraph) AddEntity(ctx context.Context, entity *rag.Entity) error {
	m.entities[entity.ID] = *entity

	// Update type index
	if _, exists := m.entityIndex[entity.Type]; !exists {
		m.entityIndex[entity.Type] = make([]string, 0)
	}
	m.entityIndex[entity.Type] = append(m.entityIndex[entity.Type], entity.ID)

	return nil
}

// AddRelationship adds a relationship to the memory graph
func (m *MemoryGraph) AddRelationship(ctx context.Context, rel *rag.Relationship) error {
	m.relationships[rel.ID] = *rel
	return nil
}

// Query performs a graph query
func (m *MemoryGraph) Query(ctx context.Context, query *rag.GraphQuery) (*rag.GraphQueryResult, error) {
	result := &rag.GraphQueryResult{
		Entities:      make([]*rag.Entity, 0),
		Relationships: make([]*rag.Relationship, 0),
		Paths:         make([][]*rag.Entity, 0),
		Metadata:      make(map[string]any),
	}

	// Filter by entity types
	if len(query.EntityTypes) > 0 {
		for _, entityType := range query.EntityTypes {
			if entityIDs, exists := m.entityIndex[entityType]; exists {
				for _, id := range entityIDs {
					if entity, exists := m.entities[id]; exists {
						e := entity
						result.Entities = append(result.Entities, &e)
					}
				}
			}
		}
	}

	// Filter by relationship types
	// Note: GraphQuery in rag/types.go has Relationships []string, checking implementation
	if len(query.Relationships) > 0 {
		for _, relType := range query.Relationships {
			for _, rel := range m.relationships {
				if rel.Type == relType {
					r := rel
					result.Relationships = append(result.Relationships, &r)
				}
			}
		}
	}

	// Apply limit
	if query.Limit > 0 && len(result.Entities) > query.Limit {
		result.Entities = result.Entities[:query.Limit]
	}

	return result, nil
}

// GetEntity retrieves an entity by ID
func (m *MemoryGraph) GetEntity(ctx context.Context, id string) (*rag.Entity, error) {
	entity, exists := m.entities[id]
	if !exists {
		return nil, fmt.Errorf("entity not found: %s", id)
	}
	return &entity, nil
}

// GetRelationship retrieves a relationship by ID
func (m *MemoryGraph) GetRelationship(ctx context.Context, id string) (*rag.Relationship, error) {
	rel, exists := m.relationships[id]
	if !exists {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}
	return &rel, nil
}

// GetRelatedEntities finds entities related to a given entity
func (m *MemoryGraph) GetRelatedEntities(ctx context.Context, entityID string, maxDepth int) ([]*rag.Entity, error) {
	related := make([]*rag.Entity, 0)
	visited := make(map[string]bool)

	// Simple implementation for depth 1
	// For maxDepth > 1, would need BFS

	// Find relationships connected to this entity
	for _, rel := range m.relationships {
		// Note: relationshipType filter not in signature anymore, maxDepth added

		if rel.Source == entityID {
			if !visited[rel.Target] {
				visited[rel.Target] = true
				if entity, exists := m.entities[rel.Target]; exists {
					e := entity
					related = append(related, &e)
				}
			}
		} else if rel.Target == entityID {
			if !visited[rel.Source] {
				visited[rel.Source] = true
				if entity, exists := m.entities[rel.Source]; exists {
					e := entity
					related = append(related, &e)
				}
			}
		}
	}

	return related, nil
}

// DeleteEntity removes an entity from the memory graph
func (m *MemoryGraph) DeleteEntity(ctx context.Context, id string) error {
	delete(m.entities, id)

	// Remove from type index
	for entityType, entityIDs := range m.entityIndex {
		for i, entityID := range entityIDs {
			if entityID == id {
				m.entityIndex[entityType] = append(entityIDs[:i], entityIDs[i+1:]...)
				break
			}
		}
		if len(m.entityIndex[entityType]) == 0 {
			delete(m.entityIndex, entityType)
		}
	}

	return nil
}

// DeleteRelationship removes a relationship from the memory graph
func (m *MemoryGraph) DeleteRelationship(ctx context.Context, id string) error {
	delete(m.relationships, id)
	return nil
}

// UpdateEntity updates an entity in the memory graph
func (m *MemoryGraph) UpdateEntity(ctx context.Context, entity *rag.Entity) error {
	if _, exists := m.entities[entity.ID]; !exists {
		return fmt.Errorf("entity not found: %s", entity.ID)
	}

	// Update type index if type changed
	oldEntity, exists := m.entities[entity.ID]
	if exists && oldEntity.Type != entity.Type {
		// Remove from old type index
		for i, entityID := range m.entityIndex[oldEntity.Type] {
			if entityID == entity.ID {
				m.entityIndex[oldEntity.Type] = append(m.entityIndex[oldEntity.Type][:i], m.entityIndex[oldEntity.Type][i+1:]...)
				break
			}
		}

		// Add to new type index
		if _, exists := m.entityIndex[entity.Type]; !exists {
			m.entityIndex[entity.Type] = make([]string, 0)
		}
		m.entityIndex[entity.Type] = append(m.entityIndex[entity.Type], entity.ID)
	}

	m.entities[entity.ID] = *entity
	return nil
}

// UpdateRelationship updates a relationship in the memory graph
func (m *MemoryGraph) UpdateRelationship(ctx context.Context, rel *rag.Relationship) error {
	if _, exists := m.relationships[rel.ID]; !exists {
		return fmt.Errorf("relationship not found: %s", rel.ID)
	}

	m.relationships[rel.ID] = *rel
	return nil
}

// Close closes the memory graph (no-op for in-memory implementation)
func (m *MemoryGraph) Close() error {
	// Clear all data
	m.entities = make(map[string]rag.Entity)
	m.relationships = make(map[string]rag.Relationship)
	m.entityIndex = make(map[string][]string)
	return nil
}

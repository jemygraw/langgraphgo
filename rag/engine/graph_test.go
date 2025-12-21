package engine

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

type mockKG struct {
	entities []*rag.Entity
}

func (m *mockKG) Query(ctx context.Context, q *rag.GraphQuery) (*rag.GraphQueryResult, error) {
	return &rag.GraphQueryResult{Entities: m.entities}, nil
}
func (m *mockKG) AddEntity(ctx context.Context, e *rag.Entity) error             { return nil }
func (m *mockKG) AddRelationship(ctx context.Context, r *rag.Relationship) error { return nil }
func (m *mockKG) GetRelatedEntities(ctx context.Context, id string, d int) ([]*rag.Entity, error) {
	return m.entities, nil
}
func (m *mockKG) GetEntity(ctx context.Context, id string) (*rag.Entity, error) {
	if len(m.entities) > 0 && m.entities[0].ID == id {
		return m.entities[0], nil
	}
	return nil, nil
}

func TestGraphRAGEngine(t *testing.T) {
	ctx := context.Background()
	llm := &mockLLM{}
	kg := &mockKG{entities: []*rag.Entity{{ID: "e1", Name: "e1", Type: "person"}}}
	embedder := &mockEmbedder{}

	e, err := NewGraphRAGEngine(rag.GraphRAGConfig{}, llm, embedder, kg)
	assert.NoError(t, err)
	assert.NotNil(t, e)

	t.Run("Query", func(t *testing.T) {
		res, err := e.Query(ctx, "e1")
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		docs, err := e.SimilaritySearch(ctx, "e1", 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, docs)
	})

	t.Run("AddDocuments", func(t *testing.T) {
		docs := []rag.Document{{ID: "d1", Content: "e1 knows e2"}}
		err := e.AddDocuments(ctx, docs)
		assert.NoError(t, err)
	})

	t.Run("Context and Confidence", func(t *testing.T) {
		qr := &rag.GraphQueryResult{
			Entities:      []*rag.Entity{{Name: "e1", Type: "p"}},
			Relationships: []*rag.Relationship{{Source: "e1", Target: "e2", Type: "k"}},
		}
		ctxStr := e.buildGraphContext(qr, nil)
		assert.NotEmpty(t, ctxStr)

		conf := e.calculateGraphConfidence(qr, nil)
		assert.Greater(t, conf, 0.0)
	})
}

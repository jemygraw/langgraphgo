package retriever

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
	scores := make([]float64, len(m.entities))
	for i := range scores {
		scores[i] = 0.9
	}
	return &rag.GraphQueryResult{Entities: m.entities, Scores: scores}, nil
}
func (m *mockKG) AddEntity(ctx context.Context, e *rag.Entity) error             { return nil }
func (m *mockKG) AddRelationship(ctx context.Context, r *rag.Relationship) error { return nil }
func (m *mockKG) GetRelatedEntities(ctx context.Context, id string, d int) ([]*rag.Entity, error) {
	return nil, nil
}
func (m *mockKG) GetEntity(ctx context.Context, id string) (*rag.Entity, error) {
	for _, e := range m.entities {
		if e.ID == id || e.Name == id {
			return e, nil
		}
	}
	return nil, nil
}

func TestGraphRetriever(t *testing.T) {
	ctx := context.Background()
	kg := &mockKG{entities: []*rag.Entity{{ID: "e1", Name: "entity1", Type: "person"}}}
	embedder := &mockEmbedder{}

	r := NewGraphRetriever(kg, embedder, rag.RetrievalConfig{K: 1})
	assert.NotNil(t, r)

	t.Run("Retrieve", func(t *testing.T) {
		// Use a name that will be matched as an entity ID in extractEntitiesFromQuery
		docs, err := r.Retrieve(ctx, "entity1")
		assert.NoError(t, err)
		assert.NotEmpty(t, docs)
	})
}

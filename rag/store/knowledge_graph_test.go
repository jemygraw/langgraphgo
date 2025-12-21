package store

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestInMemoryKnowledgeGraph(t *testing.T) {
	ctx := context.Background()
	kgInterface, err := NewKnowledgeGraph("memory://")
	assert.NoError(t, err)
	kg := kgInterface.(*MemoryGraph)
	assert.NotNil(t, kg)

	t.Run("Add and Get Entity", func(t *testing.T) {
		e := &rag.Entity{ID: "e1", Name: "entity1", Type: "person"}
		err := kg.AddEntity(ctx, e)
		assert.NoError(t, err)

		res, err := kg.GetEntity(ctx, "e1")
		assert.NoError(t, err)
		assert.Equal(t, "entity1", res.Name)
	})

	t.Run("Add Relationship", func(t *testing.T) {
		r := &rag.Relationship{ID: "r1", Source: "e1", Target: "e2", Type: "knows"}
		err := kg.AddRelationship(ctx, r)
		assert.NoError(t, err)

		rel, err := kg.GetRelationship(ctx, "r1")
		assert.NoError(t, err)
		assert.Equal(t, "knows", rel.Type)
	})

	t.Run("Related Entities", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "e2", Name: "entity2"})
		related, err := kg.GetRelatedEntities(ctx, "e1", 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, related)
	})

	t.Run("Delete and Update", func(t *testing.T) {
		err := kg.DeleteEntity(ctx, "e1")
		assert.NoError(t, err)
		_, err = kg.GetEntity(ctx, "e1")
		assert.Error(t, err)
	})

	t.Run("Query", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "e3", Type: "type1"})
		res, err := kg.Query(ctx, &rag.GraphQuery{EntityTypes: []string{"type1"}})
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Entities)
	})

	t.Run("Query with Filters", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "e5", Type: "T1"})
		kg.AddRelationship(ctx, &rag.Relationship{ID: "r3", Type: "R1"})

		q := &rag.GraphQuery{
			EntityTypes:   []string{"T1"},
			Relationships: []string{"R1"},
		}
		res, err := kg.Query(ctx, q)
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Entities)
		assert.NotEmpty(t, res.Relationships)
	})

	t.Run("Related Entities Bi-directional", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "source"})
		kg.AddEntity(ctx, &rag.Entity{ID: "target"})
		kg.AddRelationship(ctx, &rag.Relationship{ID: "rel", Source: "source", Target: "target"})

		rel1, _ := kg.GetRelatedEntities(ctx, "source", 1)
		assert.Len(t, rel1, 1)

		rel2, _ := kg.GetRelatedEntities(ctx, "target", 1)
		assert.Len(t, rel2, 1)
	})

	t.Run("Update and Delete Relationship", func(t *testing.T) {
		kg.AddRelationship(ctx, &rag.Relationship{ID: "r4", Type: "orig"})
		kg.UpdateRelationship(ctx, &rag.Relationship{ID: "r4", Type: "upd"})
		r, _ := kg.GetRelationship(ctx, "r4")
		assert.Equal(t, "upd", r.Type)

		assert.NoError(t, kg.DeleteRelationship(ctx, "r4"))
		_, err := kg.GetRelationship(ctx, "r4")
		assert.Error(t, err)
	})

	t.Run("Query Rel Type", func(t *testing.T) {
		kg.AddRelationship(ctx, &rag.Relationship{ID: "r5", Type: "RT1"})
		res, _ := kg.Query(ctx, &rag.GraphQuery{Relationships: []string{"RT1"}})
		assert.NotEmpty(t, res.Relationships)
	})

	t.Run("Update Entity Type", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "e_upd", Name: "orig", Type: "T1"})
		kg.UpdateEntity(ctx, &rag.Entity{ID: "e_upd", Name: "upd", Type: "T2"})
		e, _ := kg.GetEntity(ctx, "e_upd")
		assert.Equal(t, "T2", e.Type)
	})

	t.Run("Query Limit", func(t *testing.T) {
		kg.AddEntity(ctx, &rag.Entity{ID: "l1", Type: "L"})
		kg.AddEntity(ctx, &rag.Entity{ID: "l2", Type: "L"})
		res, _ := kg.Query(ctx, &rag.GraphQuery{EntityTypes: []string{"L"}, Limit: 1})
		assert.Len(t, res.Entities, 1)
	})

	t.Run("Close", func(t *testing.T) {
		assert.NoError(t, kg.Close())
	})
}

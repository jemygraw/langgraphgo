package store

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

type mockEmbedder struct {
	dim int
}

func (m *mockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	res := make([]float32, m.dim)
	for i := 0; i < m.dim; i++ {
		res[i] = 0.1
	}
	return res, nil
}

func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i := range texts {
		emb, _ := m.EmbedDocument(ctx, texts[i])
		res[i] = emb
	}
	return res, nil
}

func (m *mockEmbedder) GetDimension() int {
	return m.dim
}

func TestInMemoryVectorStore(t *testing.T) {
	ctx := context.Background()
	embedder := &mockEmbedder{dim: 3}
	s := NewInMemoryVectorStore(embedder)

	t.Run("Add and Search", func(t *testing.T) {
		docs := []rag.Document{
			{ID: "1", Content: "hello", Embedding: []float32{1, 0, 0}},
			{ID: "2", Content: "world", Embedding: []float32{0, 1, 0}},
		}
		err := s.Add(ctx, docs)
		assert.NoError(t, err)

		// Search for something close to "hello"
		results, err := s.Search(ctx, []float32{1, 0.1, 0}, 1)
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "1", results[0].Document.ID)
		assert.Greater(t, results[0].Score, 0.9)
	})

	t.Run("Search with Filter", func(t *testing.T) {
		docs := []rag.Document{
			{ID: "3", Content: "filtered", Embedding: []float32{0, 0, 1}, Metadata: map[string]any{"type": "special"}},
		}
		s.Add(ctx, docs)

		results, err := s.SearchWithFilter(ctx, []float32{0, 0, 1}, 10, map[string]any{"type": "special"})
		assert.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "3", results[0].Document.ID)

		results, err = s.SearchWithFilter(ctx, []float32{0, 0, 1}, 10, map[string]any{"type": "none"})
		assert.NoError(t, err)
		assert.Len(t, results, 0)
	})

	t.Run("Update and Delete", func(t *testing.T) {
		doc := rag.Document{ID: "1", Content: "updated", Embedding: []float32{1, 1, 1}}
		err := s.Update(ctx, []rag.Document{doc})
		assert.NoError(t, err)

		stats, _ := s.GetStats(ctx)
		countBefore := stats.TotalDocuments

		err = s.Delete(ctx, []string{"1"})
		assert.NoError(t, err)

		stats, _ = s.GetStats(ctx)
		assert.Equal(t, countBefore-1, stats.TotalDocuments)
	})

	t.Run("AddBatch", func(t *testing.T) {
		docs := []rag.Document{{ID: "4", Content: "batch1"}}
		embs := [][]float32{{0.5, 0.5, 0.5}}
		err := s.AddBatch(ctx, docs, embs)
		assert.NoError(t, err)

		stats, _ := s.GetStats(ctx)
		assert.GreaterOrEqual(t, stats.TotalDocuments, 1)
	})

	t.Run("Update without Embedding", func(t *testing.T) {
		doc := rag.Document{ID: "4", Content: "updated batch1"}
		err := s.Update(ctx, []rag.Document{doc})
		assert.NoError(t, err)

		results, _ := s.Search(ctx, []float32{0.5, 0.5, 0.5}, 1)
		assert.Equal(t, "updated batch1", results[0].Document.Content)
	})

	t.Run("Add without embedding", func(t *testing.T) {
		doc := rag.Document{ID: "5", Content: "no emb"}
		err := s.Add(ctx, []rag.Document{doc})
		assert.NoError(t, err)

		stats, _ := s.GetStats(ctx)
		assert.GreaterOrEqual(t, stats.TotalVectors, 1)
	})

	t.Run("Update with embedding", func(t *testing.T) {
		doc := rag.Document{ID: "5", Content: "updated with emb"}
		err := s.UpdateWithEmbedding(ctx, doc, []float32{0.9, 0.9, 0.9})
		assert.NoError(t, err)
	})

	t.Run("Delete", func(t *testing.T) {
		err := s.Delete(ctx, []string{"4"})
		assert.NoError(t, err)
	})

	t.Run("Matches Filter", func(t *testing.T) {
		doc := rag.Document{Metadata: map[string]any{"key": "val"}}
		assert.True(t, s.matchesFilter(doc, map[string]any{"key": "val"}))
		assert.False(t, s.matchesFilter(doc, map[string]any{"key": "wrong"}))
		assert.False(t, s.matchesFilter(doc, map[string]any{"missing": "any"}))
	})
}

func TestCosineSimilarity32(t *testing.T) {
	v1 := []float32{1, 0}
	v2 := []float32{1, 0}
	assert.InDelta(t, 1.0, cosineSimilarity32(v1, v2), 1e-6)

	v3 := []float32{0, 1}
	assert.InDelta(t, 0.0, cosineSimilarity32(v1, v3), 1e-6)

	assert.Equal(t, 0.0, cosineSimilarity32([]float32{1}, []float32{1, 2}))
	assert.Equal(t, 0.0, cosineSimilarity32([]float32{0}, []float32{0}))
}

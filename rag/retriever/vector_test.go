package retriever

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

type mockVectorStore struct {
	docs []rag.Document
}

func (m *mockVectorStore) Add(ctx context.Context, documents []rag.Document) error {
	m.docs = append(m.docs, documents...)
	return nil
}

func (m *mockVectorStore) Search(ctx context.Context, query []float32, k int) ([]rag.DocumentSearchResult, error) {
	var results []rag.DocumentSearchResult
	for i := 0; i < len(m.docs) && i < k; i++ {
		results = append(results, rag.DocumentSearchResult{
			Document: m.docs[i],
			Score:    1.0 - float64(i)*0.1,
		})
	}
	return results, nil
}

func (m *mockVectorStore) SearchWithFilter(ctx context.Context, query []float32, k int, filter map[string]any) ([]rag.DocumentSearchResult, error) {
	return m.Search(ctx, query, k)
}

func (m *mockVectorStore) Delete(ctx context.Context, ids []string) error             { return nil }
func (m *mockVectorStore) Update(ctx context.Context, documents []rag.Document) error { return nil }
func (m *mockVectorStore) GetStats(ctx context.Context) (*rag.VectorStoreStats, error) {
	return nil, nil
}

func TestVectorRetriever(t *testing.T) {
	ctx := context.Background()
	store := &mockVectorStore{
		docs: []rag.Document{
			{ID: "doc1", Content: "content 1"},
			{ID: "doc2", Content: "content 2"},
		},
	}
	embedder := &mockEmbedder{}

	r := NewVectorRetriever(store, embedder, rag.RetrievalConfig{K: 2})

	t.Run("Basic Retrieve", func(t *testing.T) {
		docs, err := r.Retrieve(ctx, "test query")
		assert.NoError(t, err)
		assert.Len(t, docs, 2)
		assert.Equal(t, "doc1", docs[0].ID)
	})

	t.Run("Retrieve with Score Threshold", func(t *testing.T) {
		rLow := NewVectorRetriever(store, embedder, rag.RetrievalConfig{K: 2, ScoreThreshold: 0.95})
		docs, err := rLow.Retrieve(ctx, "test query")
		assert.NoError(t, err)
		assert.Len(t, docs, 1) // Only doc1 has score 1.0 >= 0.95
	})

	t.Run("Retrieve with MMR", func(t *testing.T) {
		rMMR := NewVectorRetriever(store, embedder, rag.RetrievalConfig{K: 2, SearchType: "mmr"})
		docs, err := rMMR.Retrieve(ctx, "test query")
		assert.NoError(t, err)
		assert.NotEmpty(t, docs)
	})

	t.Run("Retrieve with Diversity", func(t *testing.T) {
		rDiv := NewVectorRetriever(store, embedder, rag.RetrievalConfig{K: 2, SearchType: "diversity"})
		docs, err := rDiv.Retrieve(ctx, "test query")
		assert.NoError(t, err)
		assert.NotEmpty(t, docs)
	})

	t.Run("VectorStoreRetriever", func(t *testing.T) {
		vsr := NewVectorStoreRetriever(store, embedder, 2)
		docs, err := vsr.Retrieve(ctx, "test query")
		assert.NoError(t, err)
		assert.Len(t, docs, 2)

		res, err := vsr.RetrieveWithConfig(ctx, "test", &rag.RetrievalConfig{K: 1})
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})
}

func TestContentSimilarity(t *testing.T) {
	s1 := "hello world"
	s2 := "hello there"
	sim := contentSimilarity(s1, s2)
	assert.Greater(t, sim, 0.0)
	assert.Less(t, sim, 1.0)
}

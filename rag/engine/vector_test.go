package engine

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestVectorRAGEngine(t *testing.T) {
	ctx := context.Background()
	llm := &mockLLM{}
	store := &mockVectorStore{docs: []rag.Document{{Content: "c1"}}}
	embedder := &mockEmbedder{}

	e, err := NewVectorRAGEngine(llm, embedder, store, 1)
	assert.NoError(t, err)
	assert.NotNil(t, e)

	t.Run("Query", func(t *testing.T) {
		res, err := e.Query(ctx, "test")
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Context)
	})

	t.Run("Operations", func(t *testing.T) {
		err := e.AddDocuments(ctx, []rag.Document{{Content: "new"}})
		assert.NoError(t, err)
		assert.NoError(t, e.DeleteDocument(ctx, "1"))
		assert.NoError(t, e.UpdateDocument(ctx, rag.Document{}))
	})

	t.Run("Similarity Search", func(t *testing.T) {
		docs, err := e.SimilaritySearch(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)
	})

	t.Run("QueryWithConfig", func(t *testing.T) {
		config := &rag.RetrievalConfig{
			K:          1,
			SearchType: "mmr",
		}
		res, err := e.QueryWithConfig(ctx, "test", config)
		assert.NoError(t, err)
		assert.Len(t, res.Sources, 1)
	})

	t.Run("Query with Reranking", func(t *testing.T) {
		config := rag.VectorRAGConfig{
			EnableReranking: true,
			RetrieverConfig: rag.RetrievalConfig{K: 1},
		}
		e2, _ := NewVectorRAGEngineWithConfig(llm, embedder, store, config)
		res, err := e2.Query(ctx, "test")
		assert.NoError(t, err)
		assert.NotNil(t, res)
	})

	t.Run("Calculate Similarity", func(t *testing.T) {
		d1 := rag.Document{Content: "word1 word2"}
		d2 := rag.Document{Content: "word1 word3"}
		sim := e.calculateSimilarity(d1, d2)
		assert.Greater(t, sim, 0.0)
	})
}

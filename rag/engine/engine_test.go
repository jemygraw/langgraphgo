package engine

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestBaseEngine(t *testing.T) {
	ctx := context.Background()
	retriever := &mockRetriever{docs: []rag.Document{{ID: "1", Content: "c1"}}}
	embedder := &mockEmbedder{}

	e := rag.NewBaseEngine(retriever, embedder, nil)
	assert.NotNil(t, e)

	t.Run("Query", func(t *testing.T) {
		res, err := e.Query(ctx, "test")
		assert.NoError(t, err)
		assert.NotEmpty(t, res.Context)
	})

	t.Run("SimilaritySearch", func(t *testing.T) {
		docs, err := e.SimilaritySearch(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)

		res, err := e.SimilaritySearchWithScores(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})

	t.Run("Engine Operations", func(t *testing.T) {
		assert.Error(t, e.AddDocuments(ctx, nil))
		assert.Error(t, e.DeleteDocument(ctx, "1"))
		assert.Error(t, e.UpdateDocument(ctx, rag.Document{}))
	})

	t.Run("Stats", func(t *testing.T) {
		assert.NotNil(t, e.GetMetrics())
		e.ResetMetrics()
	})
}

func TestCompositeEngine(t *testing.T) {
	ctx := context.Background()
	retriever := &mockRetriever{docs: []rag.Document{{ID: "1", Content: "c1"}}}
	embedder := &mockEmbedder{}
	engine1 := rag.NewBaseEngine(retriever, embedder, nil)

	comp := rag.NewCompositeEngine([]rag.Engine{engine1}, nil)

	t.Run("Composite Query", func(t *testing.T) {
		res, err := comp.Query(ctx, "test")
		assert.NoError(t, err)
		assert.Len(t, res.Sources, 1)

		res2, _ := comp.QueryWithConfig(ctx, "test", nil)
		assert.Len(t, res2.Sources, 1)
	})

	t.Run("Composite Operations", func(t *testing.T) {
		// BaseEngine returns errors for these, so Composite should too
		assert.Error(t, comp.AddDocuments(ctx, []rag.Document{{}}))
		assert.Error(t, comp.DeleteDocument(ctx, "1"))
		assert.Error(t, comp.UpdateDocument(ctx, rag.Document{}))
	})

	t.Run("Composite Search", func(t *testing.T) {
		docs, err := comp.SimilaritySearch(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)

		res, err := comp.SimilaritySearchWithScores(ctx, "test", 1)
		assert.NoError(t, err)
		assert.Len(t, res, 1)
	})
}

func TestAggregators(t *testing.T) {
	res1 := &rag.QueryResult{
		Confidence: 0.5,
		Sources:    []rag.Document{{ID: "1"}},
		Metadata:   make(map[string]any),
	}
	res2 := &rag.QueryResult{
		Confidence: 0.8,
		Sources:    []rag.Document{{ID: "2"}},
		Metadata:   make(map[string]any),
	}

	t.Run("DefaultAggregator", func(t *testing.T) {
		agg := rag.DefaultAggregator([]*rag.QueryResult{res1, res2})
		assert.Equal(t, 0.8, agg.Confidence)
		assert.Len(t, agg.Sources, 2)
	})

	t.Run("WeightedAggregator", func(t *testing.T) {
		wAgg := rag.WeightedAggregator([]float64{1.0, 0.1})([]*rag.QueryResult{res1, res2})
		assert.Equal(t, 0.5, wAgg.Confidence)
	})
}

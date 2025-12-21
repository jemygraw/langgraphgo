package retriever

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestHybridRetriever(t *testing.T) {
	ctx := context.Background()
	r1 := &mockRetriever{docs: []rag.Document{{ID: "1", Content: "r1"}}}
	r2 := &mockRetriever{docs: []rag.Document{{ID: "2", Content: "r2"}}}

	h := NewHybridRetriever([]rag.Retriever{r1, r2}, []float64{0.7, 0.3}, rag.RetrievalConfig{K: 2})
	assert.NotNil(t, h)

	t.Run("Hybrid Retrieve", func(t *testing.T) {
		docs, err := h.Retrieve(ctx, "test")
		assert.NoError(t, err)
		assert.Len(t, docs, 2)
	})

	t.Run("Hybrid Weights", func(t *testing.T) {
		h.SetWeights([]float64{0.5, 0.5})
		assert.Equal(t, []float64{0.5, 0.5}, h.GetWeights())
	})

	t.Run("Retriever Management", func(t *testing.T) {
		assert.Equal(t, 2, h.GetRetrieverCount())
		r3 := &mockRetriever{}
		h.AddRetriever(r3, 0.1)
		assert.Equal(t, 3, h.GetRetrieverCount())
		h.RemoveRetriever(2)
		assert.Equal(t, 2, h.GetRetrieverCount())
	})
}

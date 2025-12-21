package retriever

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestSimpleReranker(t *testing.T) {
	ctx := context.Background()
	r := NewSimpleReranker()
	assert.NotNil(t, r)

	docs := []rag.DocumentSearchResult{
		{Document: rag.Document{Content: "match"}, Score: 0.5},
		{Document: rag.Document{Content: "no match"}, Score: 0.1},
	}

	t.Run("Rerank with exact match", func(t *testing.T) {
		res, err := r.Rerank(ctx, "match", docs)
		assert.NoError(t, err)
		assert.Greater(t, res[0].Score, 0.5)
	})
}

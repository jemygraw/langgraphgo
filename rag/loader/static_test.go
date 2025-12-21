package loader

import (
	"context"
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestStaticDocumentLoader(t *testing.T) {
	ctx := context.Background()
	docs := []rag.Document{
		{ID: "1", Content: "static 1"},
		{ID: "2", Content: "static 2"},
	}

	loader := NewStaticDocumentLoader(docs)

	t.Run("Basic Load", func(t *testing.T) {
		loaded, err := loader.Load(ctx)
		assert.NoError(t, err)
		assert.Equal(t, docs, loaded)
	})

	t.Run("Load with Metadata", func(t *testing.T) {
		loaded, err := loader.LoadWithMetadata(ctx, map[string]any{"extra": "meta"})
		assert.NoError(t, err)
		assert.Len(t, loaded, 2)
		assert.Equal(t, "meta", loaded[0].Metadata["extra"])
	})
}

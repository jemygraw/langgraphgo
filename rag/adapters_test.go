package rag

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

type mockLCEmbedder struct{}

func (m *mockLCEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	res := make([][]float32, len(texts))
	for i := range texts {
		res[i] = []float32{0.1, 0.2}
	}
	return res, nil
}

func (m *mockLCEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

type mockLCLoader struct{}

func (m *mockLCLoader) Load(ctx context.Context) ([]schema.Document, error) {
	return []schema.Document{{PageContent: "lc content", Metadata: map[string]any{"source": "lc"}}}, nil
}
func (m *mockLCLoader) LoadAndSplit(ctx context.Context, s textsplitter.TextSplitter) ([]schema.Document, error) {
	return m.Load(ctx)
}

func TestLangChainAdapters(t *testing.T) {
	ctx := context.Background()

	t.Run("LangChainDocumentLoader", func(t *testing.T) {
		lcLoader := &mockLCLoader{}
		adapter := NewLangChainDocumentLoader(lcLoader)
		docs, err := adapter.Load(ctx)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)
		assert.Equal(t, "lc content", docs[0].Content)

		docs2, _ := adapter.LoadWithMetadata(ctx, map[string]any{"a": "b"})
		assert.Equal(t, "b", docs2[0].Metadata["a"])

		docs3, _ := adapter.LoadAndSplit(ctx, nil)
		assert.NotEmpty(t, docs3)
	})

	t.Run("LangChainEmbedder", func(t *testing.T) {
		lcEmb := &mockLCEmbedder{}
		adapter := NewLangChainEmbedder(lcEmb)

		emb, err := adapter.EmbedDocument(ctx, "test")
		assert.NoError(t, err)
		assert.Equal(t, []float32{0.1, 0.2}, emb)

		embs, err := adapter.EmbedDocuments(ctx, []string{"test"})
		assert.NoError(t, err)
		assert.Equal(t, [][]float32{{0.1, 0.2}}, embs)

		assert.Equal(t, 2, adapter.GetDimension())
	})

	t.Run("Conversion functions", func(t *testing.T) {
		schemaDocs := []schema.Document{
			{PageContent: "content", Metadata: map[string]any{"source": "src1"}},
		}
		docs := convertSchemaDocuments(schemaDocs)
		assert.Len(t, docs, 1)
		assert.Equal(t, "content", docs[0].Content)
		assert.Equal(t, "src1", docs[0].ID)
	})

	t.Run("LangChainTextSplitter", func(t *testing.T) {
		lcSplitter := NewLangChainTextSplitter(nil)
		text := "para1\n\npara2"
		splits := lcSplitter.SplitText(text)
		assert.Len(t, splits, 2)

		docs := lcSplitter.SplitDocuments([]Document{{Content: text}})
		assert.Len(t, docs, 2)

		joined := lcSplitter.JoinText([]string{"a", "b"})
		assert.Equal(t, "a b", joined)
	})

	t.Run("LangChainEmbedder Dimension", func(t *testing.T) {
		lcEmb := &mockLCEmbedder{}
		adapter := NewLangChainEmbedder(lcEmb)
		assert.Equal(t, 2, adapter.GetDimension())
	})

	t.Run("LangChainRetriever", func(t *testing.T) {
		// We can't easily mock vectorstores.VectorStore due to its complexity and
		// generic return types, but we can test the adapter logic where possible.
		adapter := NewLangChainRetriever(nil, 3)
		assert.NotNil(t, adapter)
		assert.Equal(t, 3, adapter.topK)
	})

	t.Run("LangChainVectorStore", func(t *testing.T) {
		adapter := NewLangChainVectorStore(nil)
		assert.NotNil(t, adapter)

		stats, _ := adapter.GetStats(ctx)
		assert.NotNil(t, stats)
	})
}

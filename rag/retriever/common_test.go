package retriever

import (
	"context"

	"github.com/smallnest/langgraphgo/rag"
)

type mockEmbedder struct{}

func (m *mockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}
func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2}}, nil
}
func (m *mockEmbedder) GetDimension() int { return 2 }

type mockRetriever struct {
	docs []rag.Document
}

func (m *mockRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return m.docs, nil
}

func (m *mockRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]rag.Document, error) {
	return m.docs, nil
}

func (m *mockRetriever) RetrieveWithConfig(ctx context.Context, query string, config *rag.RetrievalConfig) ([]rag.DocumentSearchResult, error) {
	res := make([]rag.DocumentSearchResult, len(m.docs))
	for i, d := range m.docs {
		res[i] = rag.DocumentSearchResult{Document: d, Score: 0.9}
	}
	return res, nil
}

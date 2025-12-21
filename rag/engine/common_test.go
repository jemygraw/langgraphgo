package engine

import (
	"context"

	"github.com/smallnest/langgraphgo/rag"
)

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

type mockEmbedder struct{}

func (m *mockEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1}, nil
}
func (m *mockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{{0.1}}, nil
}
func (m *mockEmbedder) GetDimension() int { return 1 }

type mockLLM struct{}

func (m *mockLLM) Generate(ctx context.Context, prompt string) (string, error) {
	return `{"entities": [{"name": "e1", "type": "person"}]}`, nil
}
func (m *mockLLM) GenerateWithConfig(ctx context.Context, prompt string, config map[string]any) (string, error) {
	return `{"entities": [{"name": "e1", "type": "person"}]}`, nil
}
func (m *mockLLM) GenerateWithSystem(ctx context.Context, system, prompt string) (string, error) {
	return `{"entities": [{"name": "e1", "type": "person"}]}`, nil
}

type mockVectorStore struct {
	docs []rag.Document
}

func (m *mockVectorStore) Add(ctx context.Context, docs []rag.Document) error { return nil }
func (m *mockVectorStore) Search(ctx context.Context, q []float32, k int) ([]rag.DocumentSearchResult, error) {
	res := make([]rag.DocumentSearchResult, len(m.docs))
	for i, d := range m.docs {
		res[i] = rag.DocumentSearchResult{Document: d, Score: 0.9}
	}
	return res, nil
}
func (m *mockVectorStore) SearchWithFilter(ctx context.Context, q []float32, k int, f map[string]any) ([]rag.DocumentSearchResult, error) {
	return m.Search(ctx, q, k)
}
func (m *mockVectorStore) Delete(ctx context.Context, ids []string) error        { return nil }
func (m *mockVectorStore) Update(ctx context.Context, docs []rag.Document) error { return nil }
func (m *mockVectorStore) GetStats(ctx context.Context) (*rag.VectorStoreStats, error) {
	return &rag.VectorStoreStats{}, nil
}
func (m *mockVectorStore) Close() error { return nil }

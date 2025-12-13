package prebuilt

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
)

// MockLangChainLoader is a mock implementation of langchaingo/documentloaders.Loader
type MockLangChainLoader struct {
	documents []schema.Document
	err       error
}

func NewMockLangChainLoader(documents []schema.Document, err error) *MockLangChainLoader {
	return &MockLangChainLoader{
		documents: documents,
		err:       err,
	}
}

func (m *MockLangChainLoader) Load(ctx context.Context) ([]schema.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.documents, nil
}

func (m *MockLangChainLoader) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	if m.err != nil {
		return nil, m.err
	}

	docs, err := m.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Use the splitter to split documents
	var allChunks []schema.Document
	for _, doc := range docs {
		chunks, err := splitter.SplitText(doc.PageContent)
		if err != nil {
			return nil, err
		}

		for _, chunk := range chunks {
			allChunks = append(allChunks, schema.Document{
				PageContent: chunk,
				Metadata:    doc.Metadata,
			})
		}
	}

	return allChunks, nil
}

// MockLangChainTextSplitter is a mock implementation of langchaingo/textsplitter.TextSplitter
type MockLangChainTextSplitter struct {
	chunks []string
	err    error
}

func NewMockLangChainTextSplitter(chunks []string, err error) *MockLangChainTextSplitter {
	return &MockLangChainTextSplitter{
		chunks: chunks,
		err:    err,
	}
}

func (m *MockLangChainTextSplitter) SplitText(text string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.chunks, nil
}

func (m *MockLangChainTextSplitter) SplitDocuments(documents []schema.Document) ([]schema.Document, error) {
	if m.err != nil {
		return nil, m.err
	}

	var allChunks []schema.Document
	for _, doc := range documents {
		chunks, err := m.SplitText(doc.PageContent)
		if err != nil {
			return nil, err
		}

		for _, chunk := range chunks {
			allChunks = append(allChunks, schema.Document{
				PageContent: chunk,
				Metadata:    doc.Metadata,
			})
		}
	}

	return allChunks, nil
}

// MockLangChainEmbedder is a mock implementation of langchaingo/embeddings.Embedder that returns float32
type MockLangChainEmbedder struct {
	dimension int
	err       error
}

func NewMockLangChainEmbedder(dimension int) *MockLangChainEmbedder {
	return &MockLangChainEmbedder{
		dimension: dimension,
	}
}

func (m *MockLangChainEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if m.err != nil {
		return nil, m.err
	}

	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		embeddings[i] = m.generateEmbedding32(text)
	}
	return embeddings, nil
}

func (m *MockLangChainEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.generateEmbedding32(text), nil
}

func (m *MockLangChainEmbedder) generateEmbedding32(text string) []float32 {
	embedding := make([]float32, m.dimension)

	for i := 0; i < m.dimension; i++ {
		var sum float64
		for j, char := range text {
			sum += float64(char) * float64(i+j+1)
		}
		embedding[i] = float32(sum / 1000.0)
	}

	return embedding
}

// MockLangChainVectorStoreWithErrors is a mock vector store that can return errors
type MockLangChainVectorStoreWithErrors struct {
	err error
}

func (m *MockLangChainVectorStoreWithErrors) AddDocuments(ctx context.Context, docs []schema.Document, options ...vectorstores.Option) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []string{"id1", "id2"}, nil
}

func (m *MockLangChainVectorStoreWithErrors) SimilaritySearch(ctx context.Context, query string, numDocuments int, options ...vectorstores.Option) ([]schema.Document, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []schema.Document{
		{PageContent: "Result 1", Score: 0.9},
		{PageContent: "Result 2", Score: 0.8},
	}, nil
}

// TestLangChainDocumentLoader tests the LangChainDocumentLoader adapter
func TestLangChainDocumentLoader_Load(t *testing.T) {
	ctx := context.Background()

	// Create mock documents
	mockDocs := []schema.Document{
		{
			PageContent: "Test document 1",
			Metadata:    map[string]any{"source": "test1.txt"},
		},
		{
			PageContent: "Test document 2",
			Metadata:    map[string]any{"source": "test2.txt"},
		},
	}

	// Create mock loader
	mockLoader := NewMockLangChainLoader(mockDocs, nil)

	// Create adapter
	adapter := NewLangChainDocumentLoader(mockLoader)

	// Test loading
	docs, err := adapter.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(docs))
	assert.Equal(t, "Test document 1", docs[0].PageContent)
	assert.Equal(t, "Test document 2", docs[1].PageContent)
}

func TestLangChainDocumentLoader_Load_Error(t *testing.T) {
	ctx := context.Background()

	// Create mock loader with error
	mockLoader := NewMockLangChainLoader(nil, errors.New("load error"))
	adapter := NewLangChainDocumentLoader(mockLoader)

	// Test loading with error
	docs, err := adapter.Load(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "load error")
	assert.Nil(t, docs)
}

func TestLangChainDocumentLoader_LoadAndSplit(t *testing.T) {
	ctx := context.Background()

	// Create mock documents
	mockDocs := []schema.Document{
		{
			PageContent: "This is a test document to be split",
			Metadata:    map[string]any{"source": "test.txt"},
		},
	}

	// Create mock loader and splitter
	mockLoader := NewMockLangChainLoader(mockDocs, nil)
	mockSplitter := NewMockLangChainTextSplitter([]string{"This is a test", "document to be split"}, nil)

	// Create adapter
	adapter := NewLangChainDocumentLoader(mockLoader)

	// Test loading and splitting
	docs, err := adapter.LoadAndSplit(ctx, mockSplitter)
	require.NoError(t, err)
	assert.Equal(t, 2, len(docs))
	assert.Equal(t, "This is a test", docs[0].PageContent)
	assert.Equal(t, "document to be split", docs[1].PageContent)
	assert.Equal(t, "test.txt", docs[0].Metadata["source"])
}

func TestLangChainDocumentLoader_LoadAndSplit_Error(t *testing.T) {
	ctx := context.Background()

	// Create mock loader with error
	mockLoader := NewMockLangChainLoader(nil, errors.New("split error"))
	adapter := NewLangChainDocumentLoader(mockLoader)

	// Create mock splitter
	mockSplitter := NewMockLangChainTextSplitter([]string{"test"}, nil)

	// Test loading and splitting with error
	docs, err := adapter.LoadAndSplit(ctx, mockSplitter)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "split error")
	assert.Nil(t, docs)
}

func TestLangChainDocumentLoader_Conversions(t *testing.T) {
	// Test with score
	mockDocs := []schema.Document{
		{
			PageContent: "Document with score",
			Metadata:    map[string]any{"source": "test.txt"},
			Score:       0.85,
		},
	}

	mockLoader := NewMockLangChainLoader(mockDocs, nil)
	adapter := NewLangChainDocumentLoader(mockLoader)

	ctx := context.Background()
	docs, err := adapter.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, len(docs))
	assert.Equal(t, "Document with score", docs[0].PageContent)
	assert.Equal(t, float32(0.85), docs[0].Metadata["score"])
}

// TestLangChainTextSplitter tests the LangChainTextSplitter adapter
func TestLangChainTextSplitter_SplitDocuments(t *testing.T) {
	// Create mock splitter
	mockSplitter := NewMockLangChainTextSplitter([]string{"Part 1", "Part 2"}, nil)
	adapter := NewLangChainTextSplitter(mockSplitter)

	// Create input documents
	inputDocs := []Document{
		{PageContent: "Document 1", Metadata: map[string]any{"source": "doc1"}},
		{PageContent: "Document 2", Metadata: map[string]any{"source": "doc2"}},
	}

	// Test splitting documents
	docs, err := adapter.SplitDocuments(inputDocs)
	require.NoError(t, err)
	assert.Equal(t, 4, len(docs)) // 2 chunks per document
	assert.Equal(t, "Part 1", docs[0].PageContent)
	assert.Equal(t, "Part 2", docs[1].PageContent)
	assert.Equal(t, "doc1", docs[0].Metadata["source"])
	assert.Equal(t, 0, docs[0].Metadata["chunk_index"])
	assert.Equal(t, 2, docs[0].Metadata["total_chunks"])
	assert.Equal(t, 1, docs[1].Metadata["chunk_index"])
	assert.Equal(t, 2, docs[1].Metadata["total_chunks"])
}

func TestLangChainTextSplitter_SplitDocuments_Error(t *testing.T) {
	// Create mock splitter with error
	mockSplitter := NewMockLangChainTextSplitter(nil, errors.New("split error"))
	adapter := NewLangChainTextSplitter(mockSplitter)

	// Create input documents
	inputDocs := []Document{
		{PageContent: "Document 1", Metadata: map[string]any{"source": "doc1"}},
	}

	// Test splitting with error
	docs, err := adapter.SplitDocuments(inputDocs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "split error")
	assert.Nil(t, docs)
}

func TestLangChainTextSplitter_EmptyDocuments(t *testing.T) {
	mockSplitter := NewMockLangChainTextSplitter([]string{}, nil)
	adapter := NewLangChainTextSplitter(mockSplitter)

	// Test with empty documents
	docs, err := adapter.SplitDocuments([]Document{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(docs))

	// Test with empty text
	docs, err = adapter.SplitDocuments([]Document{{PageContent: "", Metadata: nil}})
	require.NoError(t, err)
	assert.Equal(t, 0, len(docs))
}

// TestLangChainEmbedder tests the LangChainEmbedder adapter
func TestLangChainEmbedder_EmbedDocuments(t *testing.T) {
	ctx := context.Background()

	// Create mock embedder with 3-dimensional embeddings
	var mockEmbedder embeddings.Embedder = NewMockLangChainEmbedder(3)
	adapter := NewLangChainEmbedder(mockEmbedder)

	// Test embedding documents
	texts := []string{"Text 1", "Text 2", "Text 3"}
	embeddings, err := adapter.EmbedDocuments(ctx, texts)
	require.NoError(t, err)
	assert.Equal(t, 3, len(embeddings))
	assert.Equal(t, 3, len(embeddings[0])) // 3-dimensional

	// Verify embeddings are different
	assert.NotEqual(t, embeddings[0], embeddings[1])
	assert.NotEqual(t, embeddings[1], embeddings[2])
}

func TestLangChainEmbedder_EmbedQuery(t *testing.T) {
	ctx := context.Background()

	// Create mock embedder
	var mockEmbedder embeddings.Embedder = NewMockLangChainEmbedder(5)
	adapter := NewLangChainEmbedder(mockEmbedder)

	// Test embedding query
	text := "Query text"
	embedding, err := adapter.EmbedQuery(ctx, text)
	require.NoError(t, err)
	assert.Equal(t, 5, len(embedding)) // 5-dimensional
}

func TestLangChainEmbedder_EmptyTexts(t *testing.T) {
	ctx := context.Background()

	var mockEmbedder embeddings.Embedder = NewMockLangChainEmbedder(3)
	adapter := NewLangChainEmbedder(mockEmbedder)

	// Test with empty texts
	embeddings, err := adapter.EmbedDocuments(ctx, []string{})
	require.NoError(t, err)
	assert.Equal(t, 0, len(embeddings))

	// Test with empty string text
	embedding, err := adapter.EmbedQuery(ctx, "")
	require.NoError(t, err)
	assert.Equal(t, 3, len(embedding))
}

func TestLangChainEmbedder_Float32ToFloat64Conversion(t *testing.T) {
	ctx := context.Background()

	// Create mock embedder that returns known float32 values
	mockEmbedder := &MockLangChainEmbedder{dimension: 2}
	adapter := NewLangChainEmbedder(mockEmbedder)

	// Test conversion from float32 to float64
	texts := []string{"test"}
	embeddings, err := adapter.EmbedDocuments(ctx, texts)
	require.NoError(t, err)
	assert.Equal(t, 2, len(embeddings[0]))

	// Verify values are converted to float64
	for _, emb := range embeddings {
		for _, val := range emb {
			assert.IsType(t, float64(0), val)
		}
	}
}

// TestLangChainVectorStore_PrivateFields tests accessing private fields
func TestLangChainVectorStore_PrivateFields(t *testing.T) {
	mockStore := &MockLangChainVectorStore{}
	adapter := NewLangChainVectorStore(mockStore)

	// Simply verify the adapter is created successfully
	assert.NotNil(t, adapter)

	// Verify the adapter works with the store
	ctx := context.Background()
	docs := []Document{{PageContent: "test"}}
	err := adapter.AddDocuments(ctx, docs, [][]float64{{0.1}})
	assert.NoError(t, err)
}

func TestLangChainVectorStore_AddDocuments_Error(t *testing.T) {
	mockStore := &MockLangChainVectorStoreWithErrors{err: errors.New("add error")}
	adapter := NewLangChainVectorStore(mockStore)

	ctx := context.Background()
	documents := []Document{{PageContent: "Test"}}
	embeddings := [][]float64{{0.1, 0.2}}

	err := adapter.AddDocuments(ctx, documents, embeddings)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "add error")
}

// TestLangChainVectorStore_ErrorCases tests additional error scenarios
func TestLangChainVectorStore_ErrorCases(t *testing.T) {
	tests := []struct {
		name       string
		documents  []Document
		embeddings [][]float64
	}{
		{
			name:       "nil documents",
			documents:  nil,
			embeddings: [][]float64{},
		},
		{
			name:       "empty documents",
			documents:  []Document{},
			embeddings: [][]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &MockLangChainVectorStore{}
			adapter := NewLangChainVectorStore(mockStore)
			ctx := context.Background()

			// This should not panic
			err := adapter.AddDocuments(ctx, tt.documents, tt.embeddings)
			assert.NoError(t, err)
		})
	}
}

// TestLangChainAdapter_Integration tests the complete adapter workflow
func TestLangChainAdapter_Integration(t *testing.T) {
	ctx := context.Background()

	// Create components
	mockLoader := NewMockLangChainLoader([]schema.Document{
		{PageContent: "LangGraph is a library", Metadata: map[string]any{"source": "doc1"}},
		{PageContent: "Go is efficient", Metadata: map[string]any{"source": "doc2"}},
	}, nil)

	mockSplitter := NewMockLangChainTextSplitter([]string{"LangGraph", "is", "a", "library"}, nil)
	var mockEmbedder embeddings.Embedder = NewMockLangChainEmbedder(4)
	mockStore := &MockLangChainVectorStore{}

	// Create adapters
	loaderAdapter := NewLangChainDocumentLoader(mockLoader)
	splitterAdapter := NewLangChainTextSplitter(mockSplitter)
	embedderAdapter := NewLangChainEmbedder(mockEmbedder)
	storeAdapter := NewLangChainVectorStore(mockStore)

	// Test the workflow
	// 1. Load documents
	docs, err := loaderAdapter.Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, len(docs))

	// 2. Split documents
	splitDocs, err := splitterAdapter.SplitDocuments([]Document{docs[0]})
	require.NoError(t, err)
	assert.Greater(t, len(splitDocs), 0)

	// 3. Embed documents
	texts := make([]string, len(splitDocs))
	for i, doc := range splitDocs {
		texts[i] = doc.PageContent
	}
	embeddings, err := embedderAdapter.EmbedDocuments(ctx, texts)
	require.NoError(t, err)
	assert.Equal(t, len(splitDocs), len(embeddings))

	// 4. Store in vector store
	err = storeAdapter.AddDocuments(ctx, splitDocs, embeddings)
	require.NoError(t, err)

	// 5. Search
	searchResults, err := storeAdapter.SimilaritySearch(ctx, "library", 2)
	require.NoError(t, err)
	assert.Equal(t, 2, len(searchResults))
}

// TestDocumentConversion tests the conversion helper functions
func TestDocumentConversion(t *testing.T) {
	// Test convertSchemaDocuments
	schemaDocs := []schema.Document{
		{
			PageContent: "Test content",
			Metadata:    map[string]any{"source": "test"},
			Score:       0.75,
		},
	}

	docs := convertSchemaDocuments(schemaDocs)
	require.Equal(t, 1, len(docs))
	assert.Equal(t, "Test content", docs[0].PageContent)
	assert.Equal(t, "test", docs[0].Metadata["source"])
	assert.Equal(t, float32(0.75), docs[0].Metadata["score"])

	// Test convertToSchemaDocuments
	convertedBack := convertToSchemaDocuments(docs)
	require.Equal(t, 1, len(convertedBack))
	assert.Equal(t, "Test content", convertedBack[0].PageContent)
	assert.Equal(t, "test", convertedBack[0].Metadata["source"])
	assert.Equal(t, float32(0.75), convertedBack[0].Score)
}
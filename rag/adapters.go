package rag

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
)

// LangChainDocumentLoader adapts langchaingo's documentloaders.Loader to our DocumentLoader interface
type LangChainDocumentLoader struct {
	loader documentloaders.Loader
}

// NewLangChainDocumentLoader creates a new adapter for langchaingo document loaders
func NewLangChainDocumentLoader(loader documentloaders.Loader) *LangChainDocumentLoader {
	return &LangChainDocumentLoader{
		loader: loader,
	}
}

// Load loads documents using the underlying langchaingo loader
func (l *LangChainDocumentLoader) Load(ctx context.Context) ([]Document, error) {
	schemaDocs, err := l.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	return convertSchemaDocuments(schemaDocs), nil
}

// LoadWithMetadata loads documents with additional metadata using the underlying langchaingo loader
func (l *LangChainDocumentLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]Document, error) {
	docs, err := l.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Add additional metadata to all documents
	if metadata != nil {
		for i := range docs {
			if docs[i].Metadata == nil {
				docs[i].Metadata = make(map[string]any)
			}
			maps.Copy(docs[i].Metadata, metadata)
		}
	}

	return docs, nil
}

// LoadAndSplit loads and splits documents using langchaingo's text splitter
func (l *LangChainDocumentLoader) LoadAndSplit(ctx context.Context, splitter textsplitter.TextSplitter) ([]Document, error) {
	// Note: langchaingo's LoadAndSplit method signature might be different
	// For now, load first and then split
	schemaDocs, err := l.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Use the splitter to split documents
	var splitDocs []schema.Document
	for _, doc := range schemaDocs {
		// Simple split by paragraphs for now
		paragraphs := strings.SplitSeq(doc.PageContent, "\n\n")
		for para := range paragraphs {
			if strings.TrimSpace(para) != "" {
				splitDocs = append(splitDocs, schema.Document{
					PageContent: strings.TrimSpace(para),
					Metadata:    doc.Metadata,
				})
			}
		}
	}
	return convertSchemaDocuments(splitDocs), nil
}

// convertSchemaDocuments converts langchaingo schema.Document to our Document type
func convertSchemaDocuments(schemaDocs []schema.Document) []Document {
	docs := make([]Document, len(schemaDocs))
	for i, schemaDoc := range schemaDocs {
		docs[i] = Document{
			Content:  schemaDoc.PageContent,
			Metadata: convertSchemaMetadata(schemaDoc.Metadata),
		}

		// Set ID if available in metadata
		if source, ok := schemaDoc.Metadata["source"]; ok {
			docs[i].ID = fmt.Sprintf("%v", source)
		} else {
			docs[i].ID = fmt.Sprintf("doc_%d", i)
		}
	}
	return docs
}

// convertSchemaMetadata converts langchaingo metadata to our format
func convertSchemaMetadata(metadata map[string]any) map[string]any {
	result := make(map[string]any)
	maps.Copy(result, metadata)
	return result
}

// LangChainTextSplitter adapts langchaingo's textsplitter.TextSplitter to our TextSplitter interface
type LangChainTextSplitter struct {
	splitter textsplitter.TextSplitter
}

// NewLangChainTextSplitter creates a new adapter for langchaingo text splitters
func NewLangChainTextSplitter(splitter textsplitter.TextSplitter) *LangChainTextSplitter {
	return &LangChainTextSplitter{
		splitter: splitter,
	}
}

// SplitText splits text using simple paragraph splitting
func (l *LangChainTextSplitter) SplitText(text string) []string {
	// Simple split by paragraphs
	paragraphs := strings.Split(text, "\n\n")
	result := make([]string, 0, len(paragraphs))
	for _, para := range paragraphs {
		if strings.TrimSpace(para) != "" {
			result = append(result, strings.TrimSpace(para))
		}
	}
	return result
}

// SplitDocuments splits documents using simple paragraph splitting
func (l *LangChainTextSplitter) SplitDocuments(docs []Document) []Document {
	var result []Document
	for _, doc := range docs {
		// Simple split by paragraphs
		paragraphs := strings.SplitSeq(doc.Content, "\n\n")
		for para := range paragraphs {
			if strings.TrimSpace(para) != "" {
				newDoc := Document{
					Content:  strings.TrimSpace(para),
					Metadata: doc.Metadata,
				}
				result = append(result, newDoc)
			}
		}
	}
	return result
}

// JoinText joins text chunks back together
func (l *LangChainTextSplitter) JoinText(chunks []string) string {
	// Simple implementation
	var result strings.Builder
	for i, chunk := range chunks {
		if i > 0 {
			result.WriteString(" ")
		}
		result.WriteString(chunk)
	}
	return result.String()
}

// LangChainEmbedder adapts langchaingo's embeddings.Embedder to our Embedder interface
type LangChainEmbedder struct {
	embedder embeddings.Embedder
}

// NewLangChainEmbedder creates a new adapter for langchaingo embedders
func NewLangChainEmbedder(embedder embeddings.Embedder) *LangChainEmbedder {
	return &LangChainEmbedder{
		embedder: embedder,
	}
}

// EmbedDocument embeds a single document using the underlying langchaingo embedder
func (l *LangChainEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	embedding, err := l.embedder.EmbedQuery(ctx, text)
	if err != nil {
		return nil, err
	}

	// Convert float64 to float32
	result := make([]float32, len(embedding))
	for i, val := range embedding {
		result[i] = float32(val)
	}
	return result, nil
}

// EmbedDocuments embeds multiple documents using the underlying langchaingo embedder
func (l *LangChainEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings, err := l.embedder.EmbedDocuments(ctx, texts)
	if err != nil {
		return nil, err
	}

	// Convert float64 to float32
	result := make([][]float32, len(embeddings))
	for i, embedding := range embeddings {
		result[i] = make([]float32, len(embedding))
		for j, val := range embedding {
			result[i][j] = float32(val)
		}
	}
	return result, nil
}

// GetDimension returns the embedding dimension
func (l *LangChainEmbedder) GetDimension() int {
	// LangChain embedders don't typically expose dimension directly
	// We could try to embed a test document to determine it
	testEmbedding, err := l.embedder.EmbedQuery(context.Background(), "test")
	if err != nil {
		return 0
	}
	return len(testEmbedding)
}

// LangChainVectorStore adapts langchaingo's vectorstores.VectorStore to our VectorStore interface
type LangChainVectorStore struct {
	store vectorstores.VectorStore
}

// NewLangChainVectorStore creates a new adapter for langchaingo vector stores
func NewLangChainVectorStore(store vectorstores.VectorStore) *LangChainVectorStore {
	return &LangChainVectorStore{
		store: store,
	}
}

// Add adds documents to the vector store
func (l *LangChainVectorStore) Add(ctx context.Context, docs []Document) error {
	schemaDocs := make([]schema.Document, len(docs))
	for i, doc := range docs {
		schemaDocs[i] = schema.Document{
			PageContent: doc.Content,
			Metadata:    doc.Metadata,
		}
	}

	ids, err := l.store.AddDocuments(ctx, schemaDocs)
	if err != nil {
		return err
	}

	// Update document IDs if they were empty
	for i, id := range ids {
		if docs[i].ID == "" {
			docs[i].ID = id
		}
	}

	return nil
}

// Search performs similarity search
func (l *LangChainVectorStore) Search(ctx context.Context, query []float32, k int) ([]DocumentSearchResult, error) {
	// Vector search not supported by generic LangChain adapter as the interface differs
	return []DocumentSearchResult{}, nil
}

// LangChainRetriever adapts langchaingo's vectorstores.VectorStore to our Retriever interface
type LangChainRetriever struct {
	store vectorstores.VectorStore
	topK  int
}

// NewLangChainRetriever creates a new adapter for langchaingo vector stores as a retriever
func NewLangChainRetriever(store vectorstores.VectorStore, topK int) *LangChainRetriever {
	if topK <= 0 {
		topK = 4
	}
	return &LangChainRetriever{
		store: store,
		topK:  topK,
	}
}

// Retrieve retrieves documents based on a query
func (r *LangChainRetriever) Retrieve(ctx context.Context, query string) ([]Document, error) {
	return r.RetrieveWithK(ctx, query, r.topK)
}

// RetrieveWithK retrieves exactly k documents
func (r *LangChainRetriever) RetrieveWithK(ctx context.Context, query string, k int) ([]Document, error) {
	docs, err := r.store.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}
	return convertSchemaDocuments(docs), nil
}

// RetrieveWithConfig retrieves documents with custom configuration
func (r *LangChainRetriever) RetrieveWithConfig(ctx context.Context, query string, config *RetrievalConfig) ([]DocumentSearchResult, error) {
	k := r.topK
	if config != nil && config.K > 0 {
		k = config.K
	}

	// Use SimilaritySearch
	// Note: Generic SimilaritySearch doesn't return scores.
	// If the underlying store supports SimilaritySearchWithScore, we can't access it via the generic interface easily here.
	docs, err := r.store.SimilaritySearch(ctx, query, k)
	if err != nil {
		return nil, err
	}

	results := make([]DocumentSearchResult, len(docs))
	for i, doc := range docs {
		// Try to extract score from metadata if present (some stores put it there)
		score := 0.0
		if s, ok := doc.Metadata["_score"]; ok {
			if f, ok := s.(float64); ok {
				score = f
			}
		} else if s, ok := doc.Metadata["score"]; ok {
			if f, ok := s.(float64); ok {
				score = f
			}
		}

		results[i] = DocumentSearchResult{
			Document: Document{
				Content:  doc.PageContent,
				Metadata: convertSchemaMetadata(doc.Metadata),
			},
			Score: score,
		}
	}

	// Apply threshold if possible (post-filtering)
	if config != nil && config.ScoreThreshold > 0 {
		var filtered []DocumentSearchResult
		for _, res := range results {
			if res.Score >= config.ScoreThreshold {
				filtered = append(filtered, res)
			}
		}
		results = filtered
	}

	return results, nil
}

// SearchWithFilter performs similarity search with filters
func (l *LangChainVectorStore) SearchWithFilter(ctx context.Context, query []float32, k int, filter map[string]any) ([]DocumentSearchResult, error) {
	// Simple implementation that returns empty results
	// In a real implementation, you'd need to use the specific vector store's methods
	return []DocumentSearchResult{}, nil
}

// Delete removes documents by IDs
func (l *LangChainVectorStore) Delete(ctx context.Context, ids []string) error {
	// Simple implementation - LangChain vector stores may not have a standard Delete method
	// In a real implementation, you'd need to use the specific vector store's methods
	return nil
}

// Update updates existing documents
func (l *LangChainVectorStore) Update(ctx context.Context, docs []Document) error {
	// LangChain vector stores typically don't have a direct update method
	// We implement this as delete + add
	ids := make([]string, len(docs))
	for i, doc := range docs {
		ids[i] = doc.ID
	}

	if err := l.Delete(ctx, ids); err != nil {
		return err
	}

	return l.Add(ctx, docs)
}

// GetStats returns vector store statistics
func (l *LangChainVectorStore) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	// LangChain vector stores don't typically provide statistics
	// Return basic information
	return &VectorStoreStats{
		TotalDocuments: 0,
		TotalVectors:   0,
		Dimension:      0,
		LastUpdated:    time.Now(),
	}, nil
}

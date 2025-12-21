package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/tmc/langchaingo/llms"
)

// RAGState represents the state flowing through a RAG pipeline
type RAGState struct {
	Query              string
	Documents          []RAGDocument
	RetrievedDocuments []RAGDocument
	RankedDocuments    []DocumentSearchResult
	Context            string
	Answer             string
	Citations          []string
	Metadata           map[string]any
}

// PipelineConfig configures a RAG pipeline
type PipelineConfig struct {
	// Retrieval configuration
	TopK           int     // Number of documents to retrieve
	ScoreThreshold float64 // Minimum relevance score
	UseReranking   bool    // Whether to use reranking
	UseFallback    bool    // Whether to use fallback search

	// Generation configuration
	SystemPrompt     string
	IncludeCitations bool
	MaxTokens        int
	Temperature      float64

	// Components
	Loader      RAGDocumentLoader
	Splitter    RAGTextSplitter
	Embedder    Embedder
	VectorStore VectorStore
	Retriever   Retriever
	Reranker    Reranker
	LLM         llms.Model
}

// DefaultPipelineConfig returns a default RAG configuration
func DefaultPipelineConfig() *PipelineConfig {
	return &PipelineConfig{
		TopK:             4,
		ScoreThreshold:   0.7,
		UseReranking:     false,
		UseFallback:      false,
		SystemPrompt:     "You are a helpful assistant. Answer the question based on the provided context. If you cannot answer based on the context, say so.",
		IncludeCitations: true,
		MaxTokens:        1000,
		Temperature:      0.0,
	}
}

// RAGPipeline represents a complete RAG pipeline
type RAGPipeline struct {
	config *PipelineConfig
	graph  *graph.StateGraph
}

// NewRAGPipeline creates a new RAG pipeline with the given configuration
func NewRAGPipeline(config *PipelineConfig) *RAGPipeline {
	if config == nil {
		config = DefaultPipelineConfig()
	}

	g := graph.NewStateGraph()
	g.SetSchema(&ragStateSchema{})

	return &RAGPipeline{
		config: config,
		graph:  g,
	}
}

type ragStateSchema struct{}

func (s *ragStateSchema) Init() any {
	return RAGState{
		Metadata: make(map[string]any),
	}
}

func (s *ragStateSchema) Update(current, new any) (any, error) {
	currState, ok := current.(RAGState)
	if !ok {
		return new, nil
	}
	newState, ok := new.(RAGState)
	if !ok {
		return current, nil
	}

	// Simple overwrite for now, but ensure we don't lose data
	if newState.Query != "" {
		currState.Query = newState.Query
	}
	if newState.Context != "" {
		currState.Context = newState.Context
	}
	if newState.Answer != "" {
		currState.Answer = newState.Answer
	}
	if len(newState.Documents) > 0 {
		currState.Documents = newState.Documents
	}
	if len(newState.RetrievedDocuments) > 0 {
		currState.RetrievedDocuments = newState.RetrievedDocuments
	}
	if len(newState.RankedDocuments) > 0 {
		currState.RankedDocuments = newState.RankedDocuments
	}
	if len(newState.Citations) > 0 {
		currState.Citations = newState.Citations
	}
	if newState.Metadata != nil {
		if currState.Metadata == nil {
			currState.Metadata = make(map[string]any)
		}
		for k, v := range newState.Metadata {
			currState.Metadata[k] = v
		}
	}

	return currState, nil
}

// BuildBasicRAG builds a basic RAG pipeline: Retrieve -> Generate
func (p *RAGPipeline) BuildBasicRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for basic RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for basic RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "generate")
	p.graph.AddEdge("generate", graph.END)

	return nil
}

// BuildAdvancedRAG builds an advanced RAG pipeline: Retrieve -> Rerank -> Generate
func (p *RAGPipeline) BuildAdvancedRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for advanced RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for advanced RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node if enabled
	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node if enabled
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline
	p.graph.SetEntryPoint("retrieve")

	if p.config.UseReranking && p.config.Reranker != nil {
		p.graph.AddEdge("retrieve", "rerank")
		p.graph.AddEdge("rerank", "generate")
	} else {
		p.graph.AddEdge("retrieve", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// BuildConditionalRAG builds a RAG pipeline with conditional routing based on relevance
func (p *RAGPipeline) BuildConditionalRAG() error {
	if p.config.Retriever == nil {
		return fmt.Errorf("retriever is required for conditional RAG")
	}
	if p.config.LLM == nil {
		return fmt.Errorf("LLM is required for conditional RAG")
	}

	// Add retrieval node
	p.graph.AddNode("retrieve", "Document retrieval node", p.retrieveNode)

	// Add reranking node
	p.graph.AddNode("rerank", "Document reranking node", p.rerankNode)

	// Add fallback search node if enabled
	if p.config.UseFallback {
		p.graph.AddNode("fallback_search", "Fallback search node", p.fallbackSearchNode)
	}

	// Add generation node
	p.graph.AddNode("generate", "Answer generation node", p.generateNode)

	// Add citation formatting node
	if p.config.IncludeCitations {
		p.graph.AddNode("format_citations", "Citation formatting node", p.formatCitationsNode)
	}

	// Build pipeline with conditional routing
	p.graph.SetEntryPoint("retrieve")
	p.graph.AddEdge("retrieve", "rerank")

	// Conditional edge based on relevance score
	p.graph.AddConditionalEdge("rerank", func(ctx context.Context, state any) string {
		ragState := state.(RAGState)
		if len(ragState.RankedDocuments) > 0 && ragState.RankedDocuments[0].Score >= p.config.ScoreThreshold {
			return "generate"
		}
		if p.config.UseFallback {
			return "fallback_search"
		}
		return "generate"
	})

	if p.config.UseFallback {
		p.graph.AddEdge("fallback_search", "generate")
	}

	if p.config.IncludeCitations {
		p.graph.AddEdge("generate", "format_citations")
		p.graph.AddEdge("format_citations", graph.END)
	} else {
		p.graph.AddEdge("generate", graph.END)
	}

	return nil
}

// Compile compiles the RAG pipeline into a runnable graph
func (p *RAGPipeline) Compile() (*graph.StateRunnable, error) {
	return p.graph.Compile()
}

// GetGraph returns the underlying graph for visualization
func (p *RAGPipeline) GetGraph() *graph.StateGraph {
	return p.graph
}

// Node implementations

func (p *RAGPipeline) retrieveNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	docs, err := p.config.Retriever.Retrieve(ctx, ragState.Query)
	if err != nil {
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	ragState.RetrievedDocuments = convertToRAGDocuments(docs)
	ragState.Documents = convertToRAGDocuments(docs)

	return ragState, nil
}

func (p *RAGPipeline) rerankNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	if p.config.Reranker == nil {
		// If no reranker, just assign scores based on order
		rankedDocs := make([]DocumentSearchResult, len(ragState.RetrievedDocuments))
		for i, doc := range ragState.RetrievedDocuments {
			rankedDocs[i] = DocumentSearchResult{
				Document: doc.Document(),
				Score:    1.0 - float64(i)*0.1, // Simple decreasing score
			}
		}
		ragState.RankedDocuments = rankedDocs
		return ragState, nil
	}

	// Convert to DocumentSearchResult for reranking
	searchResults := make([]DocumentSearchResult, len(ragState.RetrievedDocuments))
	for i, doc := range ragState.RetrievedDocuments {
		searchResults[i] = DocumentSearchResult{
			Document: doc.Document(),
			Score:    1.0 - float64(i)*0.1,
		}
	}

	rerankedResults, err := p.config.Reranker.Rerank(ctx, ragState.Query, searchResults)
	if err != nil {
		return nil, fmt.Errorf("reranking failed: %w", err)
	}

	ragState.RankedDocuments = rerankedResults

	// Update documents with reranked order
	docs := make([]RAGDocument, len(rerankedResults))
	for i, rd := range rerankedResults {
		docs[i] = DocumentFromRAGDocument(rd.Document)
	}
	ragState.Documents = docs

	return ragState, nil
}

func (p *RAGPipeline) fallbackSearchNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	// Placeholder for fallback search (e.g., web search)
	// In a real implementation, this would call an external search API
	ragState.Metadata = map[string]any{
		"fallback_used": true,
	}

	return ragState, nil
}

func (p *RAGPipeline) generateNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)
	// fmt.Printf("DEBUG generateNode: state.Documents len = %d\n", len(ragState.Documents))

	// Build context from retrieved documents
	var contextParts []string
	for i, doc := range ragState.Documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		contextParts = append(contextParts, fmt.Sprintf("[%d] Source: %s\nContent: %s", i+1, source, doc.Content))
	}
	ragState.Context = strings.Join(contextParts, "\n\n")

	// Build prompt
	prompt := fmt.Sprintf("Context:\n%s\n\nQuestion: %s\n\nAnswer:", ragState.Context, ragState.Query)

	messages := []llms.MessageContent{
		llms.TextParts("system", p.config.SystemPrompt),
		llms.TextParts("human", prompt),
	}

	// Generate answer
	response, err := p.config.LLM.GenerateContent(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	if len(response.Choices) > 0 {
		ragState.Answer = response.Choices[0].Content
	}

	return ragState, nil
}

func (p *RAGPipeline) formatCitationsNode(ctx context.Context, state any) (any, error) {
	ragState := state.(RAGState)

	// Extract citations from documents
	citations := make([]string, len(ragState.Documents))
	for i, doc := range ragState.Documents {
		source := "Unknown"
		if s, ok := doc.Metadata["source"]; ok {
			source = fmt.Sprintf("%v", s)
		}
		citations[i] = fmt.Sprintf("[%d] %s", i+1, source)
	}
	ragState.Citations = citations

	return ragState, nil
}

// RAGDocument represents a document with content and metadata (for pipeline compatibility)
type RAGDocument struct {
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// ConvertToDocument converts RAGDocument to Document
func (d RAGDocument) Document() Document {
	return Document{
		Content:   d.Content,
		Metadata:  d.Metadata,
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt,
	}
}

// DocumentFromRAGDocument converts Document to RAGDocument
func DocumentFromRAGDocument(doc Document) RAGDocument {
	return RAGDocument{
		Content:   doc.Content,
		Metadata:  doc.Metadata,
		CreatedAt: doc.CreatedAt,
		UpdatedAt: doc.UpdatedAt,
	}
}

// RAGDocumentLoader represents a document loader for RAG pipelines
type RAGDocumentLoader interface {
	Load(ctx context.Context) ([]RAGDocument, error)
}

// RAGTextSplitter represents a text splitter for RAG pipelines
type RAGTextSplitter interface {
	SplitDocuments(documents []RAGDocument) ([]RAGDocument, error)
}

// convertToRAGDocuments converts Document to RAGDocument
func convertToRAGDocuments(docs []Document) []RAGDocument {
	result := make([]RAGDocument, len(docs))
	for i, doc := range docs {
		result[i] = DocumentFromRAGDocument(doc)
	}
	return result
}

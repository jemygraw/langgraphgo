package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Create sample documents
	documents := []rag.Document{
		{
			Content: "LangGraph is a library for building stateful, multi-actor applications with LLMs. " +
				"It extends LangChain Expression Language with the ability to coordinate multiple chains " +
				"across multiple steps of computation in a cyclic manner.",
			Metadata: map[string]any{
				"source": "langgraph_intro.txt",
			},
		},
		{
			Content: "Go, also known as Golang, is a statically typed, compiled programming language designed at Google. " +
				"It is syntactically similar to C, but with memory safety, garbage collection, structural typing, " +
				"and CSP-style concurrency.",
			Metadata: map[string]any{
				"source": "golang_intro.txt",
			},
		},
	}

	// Create embedder and vector store
	// Note: We are using internal implementations from the rag package for demonstration.
	// In a real app, you might use a specific vector DB adapter.
	// Since we don't have access to the mock implementations in 'rag' package (they are likely in test files or internal),
	// we will define simple ones here or assume 'rag' exports some basics.

	// Check if rag package exports these. based on pipeline.go, it uses interfaces.
	// We need to implement a simple InMemoryVectorStore and Embedder for this example to run if they are not exported.
	// Let's assume for this example we are demonstrating the Graph structure primarily.

	// BUT, looking at the previous examples, they used `prebuilt.NewMockEmbedder`.
	// Since I cannot find `prebuilt.NewMockEmbedder` in my file listing, I will define a minimal one here.

	embedder := &MockEmbedder{}
	vectorStore := NewInMemoryVectorStore(embedder)

	// Add documents
	// We need embeddings.
	texts := make([]string, len(documents))
	for i, doc := range documents {
		texts[i] = doc.Content
	}
	embeddings, _ := embedder.EmbedDocuments(ctx, texts)
	vectorStore.AddDocuments(ctx, documents, embeddings)

	retriever := &VectorStoreRetriever{store: vectorStore, topK: 3}

	// Create the Graph
	g := graph.NewStateGraph()

	// Define the state
	// We will use rag.RAGState but we might need to extend it or just use it.
	// rag.RAGState has: Query, Documents, RetrievedDocuments, RankedDocuments, Context, Answer, Citations, Metadata

	// Node: Query Rewrite
	g.AddNode("rewrite_query", "Rewrite user query for better retrieval", func(ctx context.Context, state any) (any, error) {
		s := state.(rag.RAGState)
		originalQuery := s.Query

		// Simple mock rewrite logic (in production, use LLM)
		// e.g. "What is LangGraph?" -> "LangGraph library features multi-actor applications"
		rewrittenQuery := fmt.Sprintf("%s (refined for search)", originalQuery)

		// Use LLM to actually rewrite
		prompt := fmt.Sprintf("Rewrite the following query to be more specific and search-friendly. Return ONLY the rewritten query text, without any explanations or quotes.\nOriginal Query: %s", originalQuery)
		resp, err := llm.Call(ctx, prompt)
		if err == nil {
			rewrittenQuery = strings.TrimSpace(resp)
			rewrittenQuery = strings.Trim(rewrittenQuery, "\"") // Remove quotes if present
		}

		fmt.Printf("Original Query: %s\nRewritten Query: %s\n", originalQuery, rewrittenQuery)

		s.Query = rewrittenQuery // Update query for retrieval
		// Store original query in metadata if needed
		if s.Metadata == nil {
			s.Metadata = make(map[string]any)
		}
		s.Metadata["original_query"] = originalQuery

		return s, nil
	})

	// Node: Retrieve (using rag pipeline's logic style)
	g.AddNode("retrieve", "Retrieve documents", func(ctx context.Context, state any) (any, error) {
		s := state.(rag.RAGState)
		docs, err := retriever.Retrieve(ctx, s.Query)
		if err != nil {
			return nil, err
		}

		// Convert []rag.Document to []rag.RAGDocument
		ragDocs := make([]rag.RAGDocument, len(docs))
		for i, d := range docs {
			ragDocs[i] = rag.DocumentFromRAGDocument(d)
		}
		s.RetrievedDocuments = ragDocs
		s.Documents = ragDocs // Set as active docs
		return s, nil
	})

	// Node: Generate
	g.AddNode("generate", "Generate Answer", func(ctx context.Context, state any) (any, error) {
		s := state.(rag.RAGState)

		// Combine context
		var contextParts []string
		for _, doc := range s.Documents {
			contextParts = append(contextParts, doc.Content)
		}
		s.Context = strings.Join(contextParts, "\n\n")

		prompt := fmt.Sprintf("Context:\n%s\n\nQuestion (Original): %v\n\nAnswer:", s.Context, s.Metadata["original_query"])

		resp, err := llm.Call(ctx, prompt)
		if err != nil {
			return nil, err
		}
		s.Answer = resp
		return s, nil
	})

	// Edges
	g.SetEntryPoint("rewrite_query")
	g.AddEdge("rewrite_query", "retrieve")
	g.AddEdge("retrieve", "generate")
	g.AddEdge("generate", graph.END)

	// Compile
	app, err := g.Compile()
	if err != nil {
		log.Fatal(err)
	}

	// Run
	inputs := rag.RAGState{
		Query: "tell me about langgraph",
	}

	fmt.Println("=== Running Query Rewrite RAG ===")
	res, err := app.Invoke(ctx, inputs)
	if err != nil {
		log.Fatal(err)
	}

	finalState := res.(rag.RAGState)
	fmt.Printf("\nFinal Answer:\n%s\n", finalState.Answer)
}

// --- Minimal Mock Implementations for standalone example ---

type MockEmbedder struct{}

func (m *MockEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	// Return dummy embeddings
	return make([][]float32, len(texts)), nil
}
func (m *MockEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return []float32{}, nil
}

type InMemoryVectorStore struct {
	docs []rag.Document
}

func NewInMemoryVectorStore(embedder *MockEmbedder) *InMemoryVectorStore {
	return &InMemoryVectorStore{}
}
func (s *InMemoryVectorStore) AddDocuments(ctx context.Context, docs []rag.Document, embeddings [][]float32) error {
	s.docs = append(s.docs, docs...)
	return nil
}
func (s *InMemoryVectorStore) SimilaritySearch(ctx context.Context, query string, k int) ([]rag.Document, error) {
	// Return all for mock
	if len(s.docs) > k {
		return s.docs[:k], nil
	}
	return s.docs, nil
}

type VectorStoreRetriever struct {
	store *InMemoryVectorStore
	topK  int
}

func (r *VectorStoreRetriever) Retrieve(ctx context.Context, query string) ([]rag.Document, error) {
	return r.store.SimilaritySearch(ctx, query, r.topK)
}

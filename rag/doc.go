// RAG (Retrieval-Augmented Generation) Package
//
// The rag package provides comprehensive RAG (Retrieval-Augmented Generation) capabilities
// for the LangGraph Go framework. It integrates various RAG approaches including traditional
// vector-based retrieval and advanced GraphRAG techniques.
//
// # Features
//
//   - Vector-based RAG: Traditional retrieval using vector similarity
//   - GraphRAG: Knowledge graph-based retrieval for enhanced context understanding
//   - Multiple Embedding Models: Support for OpenAI, local models, and more
//   - Flexible Document Processing: Various document loaders and splitters
//   - Hybrid Search: Combine vector and graph-based retrieval
//   - Integration Ready: Seamless integration with LangGraph agents
//
// # Quick Start
//
// Basic vector RAG:
//
//	import (
//		"context"
//		"github.com/smallnest/langgraphgo/rag/engine"
//		"github.com/tmc/langchaingo/embeddings/openai"
//		"github.com/tmc/langchaingo/vectorstores/pgvector"
//	)
//
//	func main() {
//		llm := initLLM()
//		embedder, _ := openai.NewEmbedder()
//		store, _ := pgvector.New(ctx, pgvector.WithEmbedder(embedder))
//
//		ragEngine, _ := engine.NewVectorRAGEngine(llm, embedder, store, 5)
//
//		result, err := ragEngine.Query(ctx, "What is quantum computing?")
//	}
//
// GraphRAG integration:
//
//	import (
//		"context"
//		"github.com/smallnest/langgraphgo/rag/engine"
//	)
//
//	func main() {
//		graphRAG, _ := engine.NewGraphRAGEngine(engine.GraphRAGConfig{
//			DatabaseURL: "redis://localhost:6379",
//			ModelProvider: "openai",
//			EmbeddingModel: "text-embedding-3-small",
//		}, llm, embedder, kg)
//
//		// Extract and store knowledge graph
//		err := graphRAG.AddDocuments(ctx, documents)
//
//		// Query using graph-enhanced retrieval
//		response, err := graphRAG.Query(ctx, "Who directed the Matrix?")
//	}
//
// # Architecture
//
// The rag package consists of several key components:
//
// # Core Components
//
// rag/engine.go
// Main RAG engine interfaces and base implementations
//
//	type Engine interface {
//		Query(ctx context.Context, query string) (*QueryResult, error)
//		AddDocuments(ctx context.Context, docs []Document) error
//		SimilaritySearch(ctx context.Context, query string, k int) ([]Document, error)
//	}
//
// rag/engine/vector.go
// Traditional vector-based RAG implementation
//
//	vectorEngine, _ := engine.NewVectorRAGEngine(llm, embedder, vectorStore, k)
//
// rag/engine/graph.go
// GraphRAG implementation with knowledge graph extraction
//
//	graphEngine, _ := engine.NewGraphRAGEngine(config, llm, embedder, kg)
//
// # Document Processing
//
// rag/types.go
// Core document and entity types
//
//	type Document struct {
//		ID       string
//		Content  string
//		Metadata map[string]any
//	}
//
// rag/loader/
// Various document loaders (text, static, etc.)
//
//	loader := loader.NewTextLoader("document.txt")
//	docs, err := loader.Load(ctx)
//
// rag/splitter/
// Text splitting strategies
//
//	splitter := splitter.NewRecursiveCharacterTextSplitter(
//		splitter.WithChunkSize(1000),
//		splitter.WithChunkOverlap(200),
//	)
//
// # Retrieval Strategies
//
// rag/retriever/
// Various retrieval implementations
//
//	vectorRetriever := retriever.NewVectorRetriever(vectorStore, embedder, 5)
//	graphRetriever := retriever.NewGraphRetriever(knowledgeGraph, 5)
//	hybridRetriever := retriever.NewHybridRetriever([]Retriever{r1, r2}, weights, config)
//
// # Integration with LangGraph
//
// The rag package integrates seamlessly with LangGraph agents:
//
//	// Create a RAG pipeline
//	pipeline := rag.NewRAGPipeline(config)
//	runnable, _ := pipeline.Compile()
//	result, _ := runnable.Invoke(ctx, rag.RAGState{Query: "..."})
//
// # Configuration
//
// The rag package supports various configuration options:
//
//	type Config struct {
//		VectorRAG *VectorRAGConfig `json:"vector_rag,omitempty"`
//		GraphRAG  *GraphRAGConfig  `json:"graph_rag,omitempty"`
//	}
//
// # Supported Data Sources
//
//   - Local files (TXT, MD, etc.)
//   - Static documents
//   - Web pages and websites (via adapters)
//
// # Supported Vector Stores
//
//   - pgvector (via adapter)
//   - Redis (via adapter)
//   - Mock/In-memory store for testing
//
// # GraphRAG Features
//
//   - Automatic entity extraction
//   - Relationship detection
//   - Multi-hop reasoning
//   - Context-aware retrieval
package rag // import "github.com/smallnest/langgraphgo/rag"

package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/chroma"
)

// This example demonstrates using Chroma vector database with LangGraphGo
// Chroma is an open-source embedding database
func main() {
	ctx := context.Background()

	fmt.Println("=== RAG with Chroma VectorStore Example ===")

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Initialize embeddings
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}

	// Prepare sample documents about Go programming
	textContent := `Go is a statically typed, compiled programming language designed at Google.
It is syntactically similar to C, but with memory safety, garbage collection, structural typing,
and CSP-style concurrency. Go was designed by Robert Griesemer, Rob Pike, and Ken Thompson.

Key features of Go include:
- Fast compilation times
- Built-in concurrency with goroutines and channels
- Simple and clean syntax
- Strong standard library
- Efficient garbage collection
- Cross-platform support

Go is particularly well-suited for:
- Web servers and APIs
- Cloud and network services
- Command-line tools
- DevOps and site reliability automation
- Distributed systems

Popular Go frameworks and libraries:
- Gin and Echo for web development
- gRPC for RPC communication
- Cobra for CLI applications
- GORM for database ORM
- Kubernetes and Docker are written in Go`

	// Load and split documents
	fmt.Println("Loading and splitting documents...")
	textReader := strings.NewReader(textContent)
	textLoader := documentloaders.NewText(textReader)
	loader := rag.NewLangChainDocumentLoader(textLoader)

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(250),
		textsplitter.WithChunkOverlap(50),
	)

	chunks, err := loader.LoadAndSplit(ctx, splitter)
	if err != nil {
		log.Fatalf("Failed to load and split: %v", err)
	}

	fmt.Printf("Split into %d chunks\n\n", len(chunks))

	// Create Chroma vector store
	// Note: This requires a running Chroma server
	// Start with: docker run -p 8000:8000 chromadb/chroma
	fmt.Println("Connecting to Chroma vector database...")

	chromaStore, err := chroma.New(
		chroma.WithChromaURL("http://localhost:8000"),
		chroma.WithEmbedder(embedder),
		chroma.WithDistanceFunction("cosine"),
		chroma.WithNameSpace("langgraphgo_example"),
	)
	if err != nil {
		log.Fatalf("Failed to create Chroma store: %v", err)
	}

	// Wrap with our adapter
	vectorStore := rag.NewLangChainVectorStore(chromaStore)

	// Add documents to Chroma
	fmt.Println("Adding documents to Chroma...")

	// LangChain adapter/store handles embeddings
	err = vectorStore.Add(ctx, chunks)
	if err != nil {
		log.Fatalf("Failed to add documents to Chroma: %v", err)
	}

	fmt.Println("Documents successfully added to Chroma")

	// Build RAG pipeline
	fmt.Println("Building RAG pipeline...")

	// Use LangChain retriever adapter
	retriever := rag.NewLangChainRetriever(chromaStore, 3)

	config := rag.DefaultPipelineConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.TopK = 3
	config.SystemPrompt = "You are a helpful AI assistant. Answer questions about Go programming based on the provided context."
	config.IncludeCitations = true

	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildAdvancedRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	fmt.Println("Pipeline ready!")

	// Visualize the pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("=== Pipeline Visualization ===")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries
	queries := []string{
		"What is Go programming language?",
		"What are the key features of Go?",
		"What is Go good for?",
		"What popular software is written in Go?",
	}

	for i, query := range queries {
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("Query %d: %s\n", i+1, query)
		fmt.Println(strings.Repeat("-", 80))

		result, err := runnable.Invoke(ctx, rag.RAGState{
			Query: query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(rag.RAGState)

		fmt.Printf("\nRetrieved %d documents from Chroma:\n", len(finalState.Documents))
		for j, doc := range finalState.Documents {
			fmt.Printf("  [%d] %s\n", j+1, truncate(doc.Content, 100))
		}

		fmt.Printf("\nAnswer:\n%s\n", finalState.Answer)

		if len(finalState.Citations) > 0 {
			fmt.Println("\nCitations:")
			for _, citation := range finalState.Citations {
				fmt.Printf("  %s\n", citation)
			}
		}
		fmt.Println()
	}

	// Demonstrate similarity search with scores
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Similarity Search with Scores")
	fmt.Println(strings.Repeat("-", 80))

	searchQuery := "concurrency in Go"

	results, err := retriever.RetrieveWithConfig(ctx, searchQuery, &rag.RetrievalConfig{K: 5})
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Query: %s\n", searchQuery)
		fmt.Printf("Found %d results:\n\n", len(results))
		for i, result := range results {
			fmt.Printf("[%d] Score: %.4f\n", i+1, result.Score)
			fmt.Printf("    Content: %s\n\n", truncate(result.Document.Content, 150))
		}
	}

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("\nNote: This example requires a running Chroma server.")
	fmt.Println("Start Chroma with: docker run -p 8000:8000 chromadb/chroma")
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

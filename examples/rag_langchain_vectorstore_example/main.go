package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/store"
	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/weaviate"
)

func main() {
	ctx := context.Background()

	fmt.Println("=== RAG with LangChain VectorStores Example ===")

	// Initialize LLM
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Initialize OpenAI embeddings
	embedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("Failed to create embedder: %v", err)
	}
	ragEmbedder := rag.NewLangChainEmbedder(embedder)

	// Example 1: Using LangChain In-Memory VectorStore
	fmt.Println("Example 1: In-Memory VectorStore with LangChain Integration")
	fmt.Println(strings.Repeat("-", 80))

	// Prepare sample documents
	textContent := `LangGraph is a library for building stateful, multi-actor applications with LLMs.
It extends LangChain Expression Language with the ability to coordinate multiple chains
across multiple steps of computation in a cyclic manner. LangGraph is particularly useful
for building complex agent workflows and multi-agent systems.

Key features of LangGraph include:
- Stateful graph-based workflows
- Support for cycles and conditional edges
- Built-in checkpointing for persistence
- Human-in-the-loop capabilities
- Integration with LangChain components

LangGraph supports multiple checkpoint backends including:
- PostgreSQL for production deployments
- SQLite for local development
- Redis for distributed systems
- In-memory for testing

The library provides prebuilt components like:
- RAG (Retrieval-Augmented Generation) pipelines
- ReAct agents for tool-using workflows
- Supervisor patterns for multi-agent coordination
- Tool executors for function calling`

	// Load and split documents
	textReader := strings.NewReader(textContent)
	textLoader := documentloaders.NewText(textReader)
	loader := rag.NewLangChainDocumentLoader(textLoader)

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(200),
		textsplitter.WithChunkOverlap(50),
	)

	chunks, err := loader.LoadAndSplit(ctx, splitter)
	if err != nil {
		log.Fatalf("Failed to load and split: %v", err)
	}

	fmt.Printf("Split into %d chunks\n", len(chunks))

	// Create LangChain in-memory vector store
	// Note: We'll use a simple in-memory store from langchaingo
	// For production, you would use Weaviate, Pinecone, Chroma, etc.

	// Since langchaingo's in-memory store might not be directly available,
	// we'll demonstrate with our wrapper approach
	inMemStore := store.NewInMemoryVectorStore(ragEmbedder)

	// Add documents to vector store
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Content
	}

	embeddings, err := ragEmbedder.EmbedDocuments(ctx, texts)
	if err != nil {
		log.Fatalf("Failed to generate embeddings: %v", err)
	}

	err = inMemStore.AddBatch(ctx, chunks, embeddings)
	if err != nil {
		log.Fatalf("Failed to add documents: %v", err)
	}

	fmt.Println("Documents added to vector store successfully")
	fmt.Println()

	// Example 2: Build RAG Pipeline with LangChain VectorStore
	fmt.Println("Example 2: RAG Pipeline with LangChain VectorStore")
	fmt.Println(strings.Repeat("-", 80))

	// Create retriever
	retriever := retriever.NewVectorStoreRetriever(inMemStore, ragEmbedder, 3)

	// Configure RAG pipeline
	config := rag.DefaultPipelineConfig()
	config.Retriever = retriever
	config.LLM = llm
	config.TopK = 3
	config.SystemPrompt = "You are a helpful AI assistant. Answer questions based on the provided context about LangGraph."
	config.IncludeCitations = true

	// Build advanced RAG pipeline
	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildAdvancedRAG()
	if err != nil {
		log.Fatalf("Failed to build RAG pipeline: %v", err)
	}

	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("Failed to compile pipeline: %v", err)
	}

	// Visualize pipeline
	exporter := graph.NewExporter(pipeline.GetGraph())
	fmt.Println("Pipeline Visualization:")
	fmt.Println(exporter.DrawMermaid())
	fmt.Println()

	// Test queries
	queries := []string{
		"What is LangGraph?",
		"What checkpoint backends does LangGraph support?",
		"What prebuilt components does LangGraph provide?",
	}

	for i, query := range queries {
		fmt.Printf("Query %d: %s\n", i+1, query)

		result, err := runnable.Invoke(ctx, rag.RAGState{
			Query: query,
		})
		if err != nil {
			log.Printf("Failed to process query: %v", err)
			continue
		}

		finalState := result.(rag.RAGState)

		fmt.Printf("Retrieved %d documents:\n", len(finalState.Documents))
		for j, doc := range finalState.Documents {
			fmt.Printf("  [%d] %s\n", j+1, truncate(doc.Content, 100))
		}

		fmt.Printf("\nAnswer: %s\n", finalState.Answer)

		if len(finalState.Citations) > 0 {
			fmt.Println("\nCitations:")
			for _, citation := range finalState.Citations {
				fmt.Printf("  %s\n", citation)
			}
		}

		fmt.Println(strings.Repeat("-", 80))
		fmt.Println()
	}

	// Example 3: Using LangChain VectorStore Wrapper (if external store available)
	fmt.Println("Example 3: External VectorStore Integration (Optional)")
	fmt.Println(strings.Repeat("-", 80))

	// Check if Weaviate is available
	weaviateURL := os.Getenv("WEAVIATE_URL")
	if weaviateURL != "" {
		fmt.Printf("Connecting to Weaviate at %s\n", weaviateURL)

		// Create Weaviate store
		weaviateStore, err := weaviate.New(
			weaviate.WithScheme("http"),
			weaviate.WithHost(weaviateURL),
			weaviate.WithEmbedder(embedder),
		)
		if err != nil {
			log.Printf("Failed to create Weaviate store: %v", err)
		} else {
			// Wrap with our adapter
			wrappedStore := rag.NewLangChainVectorStore(weaviateStore)

			// Add documents
			// LangChain adapter handles embedding implicitly if embedder was configured in weaviate
			err = wrappedStore.Add(ctx, chunks)
			if err != nil {
				log.Printf("Failed to add documents to Weaviate: %v", err)
			} else {
				fmt.Println("Successfully added documents to Weaviate")

				// Search
				// Need to embed query first for generic Search
				q := "What is LangGraph?"
				emb, _ := ragEmbedder.EmbedDocument(ctx, q)
				results, err := wrappedStore.Search(ctx, emb, 3)
				if err != nil {
					log.Printf("Search failed: %v", err)
				} else {
					fmt.Printf("Found %d results from Weaviate\n", len(results))
					for i, doc := range results {
						fmt.Printf("  [%d] %s\n", i+1, truncate(doc.Document.Content, 80))
					}
				}
			}
		}
	} else {
		fmt.Println("WEAVIATE_URL not set, skipping external vector store example")
		fmt.Println("To use Weaviate, set WEAVIATE_URL environment variable")
		fmt.Println("Example: export WEAVIATE_URL=localhost:8080")
	}
	fmt.Println()

	// Example 4: Similarity Search with Scores
	fmt.Println("Example 4: Similarity Search with Scores")
	fmt.Println(strings.Repeat("-", 80))

	query := "checkpointing and persistence"

	// Use retriever to search with scores
	results, err := retriever.RetrieveWithConfig(ctx, query, &rag.RetrievalConfig{K: 5, IncludeScores: true})
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Query: %s\n", query)
		fmt.Printf("Found %d results:\n\n", len(results))
		for i, result := range results {
			fmt.Printf("[%d] Score: %.4f\n", i+1, result.Score)
			fmt.Printf("    Content: %s\n", truncate(result.Document.Content, 120))
			fmt.Println()
		}
	}

	fmt.Println("\n=== Example Complete ===")
}

func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

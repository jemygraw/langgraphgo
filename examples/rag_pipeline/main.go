package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/smallnest/langgraphgo/rag/loader"
	"github.com/smallnest/langgraphgo/rag/retriever"
	"github.com/smallnest/langgraphgo/rag/splitter"
	"github.com/smallnest/langgraphgo/rag/store"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms/openai"
)

func main() {
	ctx := context.Background()

	// 1. Initialize LLM and Embedder
	// Make sure OPENAI_API_KEY is set in your environment
	llm, err := openai.New()
	if err != nil {
		log.Fatalf("failed to create llm: %v", err)
	}

	openaiEmbedder, err := embeddings.NewEmbedder(llm)
	if err != nil {
		log.Fatalf("failed to create embedder: %v", err)
	}
	// Wrap langchaingo embedder with our adapter
	embedder := rag.NewLangChainEmbedder(openaiEmbedder)

	// 2. Initialize Components
	// In-memory vector store
	vectorStore := store.NewInMemoryVectorStore(embedder)

	// 3. Build Knowledge Base (Ingestion)
	fmt.Println("=== Ingesting Documents ===")
	dataDir := "./data"
	err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".txt" {
			fmt.Printf("Processing %s...\n", path)

			// Load
			l := loader.NewTextLoader(path)
			docs, err := l.Load(ctx)
			if err != nil {
				return err
			}

			// Split
			s := splitter.NewRecursiveCharacterTextSplitter(
				splitter.WithChunkSize(500),
				splitter.WithChunkOverlap(50),
			)
			chunks := s.SplitDocuments(docs)

			// Store (Embeds automatically if vectorStore has embedder)
			err = vectorStore.Add(ctx, chunks)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("failed to ingest documents: %v", err)
	}
	fmt.Println("Ingestion complete.")
	fmt.Println()

	// 4. Set up Q&A Pipeline
	// Create retriever
	r := retriever.NewVectorRetriever(vectorStore, embedder, rag.RetrievalConfig{
		K:              3,
		ScoreThreshold: 0.5,
	})

	// Configure pipeline
	config := rag.DefaultPipelineConfig()
	config.LLM = llm
	config.Retriever = r
	config.IncludeCitations = true

	pipeline := rag.NewRAGPipeline(config)
	err = pipeline.BuildBasicRAG()
	if err != nil {
		log.Fatalf("failed to build pipeline: %v", err)
	}

	runnable, err := pipeline.Compile()
	if err != nil {
		log.Fatalf("failed to compile pipeline: %v", err)
	}

	// 5. Intelligent Q&A
	fmt.Println("=== Intelligent Q&A ===")
	query := "What is LangGraphGo and what are its main features?"
	fmt.Printf("Query: %s\n", query)

	result, err := runnable.Invoke(ctx, rag.RAGState{
		Query: query,
	})
	if err != nil {
		log.Fatalf("failed to invoke pipeline: %v", err)
	}

	finalState := result.(rag.RAGState)
	fmt.Printf("\nAnswer:\n%s\n", finalState.Answer)

	if len(finalState.Citations) > 0 {
		fmt.Println("\nCitations:")
		for _, citation := range finalState.Citations {
			fmt.Printf("- %s\n", citation)
		}
	}
}

// Ensure rag.Embedder is satisfied by LangChainEmbedder
var _ rag.Embedder = (*rag.LangChainEmbedder)(nil)

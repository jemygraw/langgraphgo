package splitter

import (
	"maps"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

// SimpleTextSplitter splits text into chunks of a given size
type SimpleTextSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string
}

// NewSimpleTextSplitter creates a new SimpleTextSplitter
func NewSimpleTextSplitter(chunkSize, chunkOverlap int) rag.TextSplitter {
	return &SimpleTextSplitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separator:    "\n\n",
	}
}

// SplitText splits text into chunks
func (s *SimpleTextSplitter) SplitText(text string) []string {
	return s.splitText(text)
}

// SplitDocuments splits documents into smaller chunks
func (s *SimpleTextSplitter) SplitDocuments(documents []rag.Document) []rag.Document {
	var result []rag.Document

	for _, doc := range documents {
		chunks := s.splitText(doc.Content)
		for i, chunk := range chunks {
			newDoc := rag.Document{
				ID:        doc.ID,
				Content:   chunk,
				Metadata:  make(map[string]any),
				CreatedAt: doc.CreatedAt,
				UpdatedAt: doc.UpdatedAt,
			}

			// Copy metadata
			maps.Copy(newDoc.Metadata, doc.Metadata)

			// Add chunk metadata
			newDoc.Metadata["chunk_index"] = i
			newDoc.Metadata["total_chunks"] = len(chunks)

			result = append(result, newDoc)
		}
	}

	return result
}

// JoinText joins text chunks back together
func (s *SimpleTextSplitter) JoinText(chunks []string) string {
	if len(chunks) == 0 {
		return ""
	}
	if len(chunks) == 1 {
		return chunks[0]
	}

	// Simple join - reconstruct with minimal duplication
	return strings.Join(chunks, " ")
}

func (s *SimpleTextSplitter) splitText(text string) []string {
	if len(text) <= s.ChunkSize {
		return []string{text}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + s.ChunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at a separator
		if end < len(text) {
			lastSep := strings.LastIndex(text[start:end], s.Separator)
			if lastSep > 0 {
				end = start + lastSep + len(s.Separator)
			}
		}

		chunks = append(chunks, strings.TrimSpace(text[start:end]))

		nextStart := end - s.ChunkOverlap
		if nextStart <= start {
			// If overlap would cause us to get stuck or move backwards (because the chunk was small),
			// just move forward to the end of the current chunk.
			nextStart = end
		}

		start = max(nextStart, 0)
	}

	return chunks
}

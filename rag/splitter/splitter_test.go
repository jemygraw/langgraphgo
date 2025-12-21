package splitter

import (
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestRecursiveCharacterTextSplitter(t *testing.T) {
	t.Run("Basic splitting", func(t *testing.T) {
		s := NewRecursiveCharacterTextSplitter(
			WithChunkSize(10),
			WithChunkOverlap(0),
		)
		text := "1234567890abcdefghij"
		chunks := s.SplitText(text)
		assert.Len(t, chunks, 2)
		assert.Equal(t, "1234567890", chunks[0])
		assert.Equal(t, "abcdefghij", chunks[1])
	})

	t.Run("Split with separators", func(t *testing.T) {
		s := NewRecursiveCharacterTextSplitter(
			WithChunkSize(10),
			WithChunkOverlap(0),
			WithSeparators([]string{"\n"}),
		)
		text := "part1\npart2\npart3"
		chunks := s.SplitText(text)
		assert.Len(t, chunks, 3)
		assert.Equal(t, "part1", chunks[0])
		assert.Equal(t, "part2", chunks[1])
		assert.Equal(t, "part3", chunks[2])
	})

	t.Run("Split documents", func(t *testing.T) {
		s := NewRecursiveCharacterTextSplitter(
			WithChunkSize(10),
			WithChunkOverlap(2),
		)
		doc := rag.Document{
			ID:       "doc1",
			Content:  "123456789012345",
			Metadata: map[string]any{"key": "val"},
		}
		chunks := s.SplitDocuments([]rag.Document{doc})

		assert.NotEmpty(t, chunks)
		for i, chunk := range chunks {
			assert.Equal(t, "doc1", chunk.Metadata["parent_id"])
			assert.Equal(t, i, chunk.Metadata["chunk_index"])
			assert.Equal(t, len(chunks), chunk.Metadata["chunk_total"])
		}
	})
}

func TestCharacterTextSplitter(t *testing.T) {
	s := NewCharacterTextSplitter(
		WithCharacterSeparator("|"),
		WithCharacterChunkSize(5),
		WithCharacterChunkOverlap(0),
	)
	text := "abc|def|ghi"
	chunks := s.SplitText(text)
	assert.Len(t, chunks, 3)
	assert.Equal(t, "abc", chunks[0])
	assert.Equal(t, "def", chunks[1])

	joined := s.JoinText(chunks)
	assert.Equal(t, "abc|def|ghi", joined)
}

func TestTokenTextSplitter(t *testing.T) {
	s := NewTokenTextSplitter(5, 0, nil)
	text := "one two three four five six seven eight"
	chunks := s.SplitText(text)
	assert.Len(t, chunks, 2)
	assert.Equal(t, "one two three four five", chunks[0])

	doc := rag.Document{ID: "tok1", Content: text}
	docChunks := s.SplitDocuments([]rag.Document{doc})
	assert.Len(t, docChunks, 2)
}

func TestRecursiveCharacterJoin(t *testing.T) {
	s := NewRecursiveCharacterTextSplitter(WithChunkOverlap(0))
	joined := s.JoinText([]string{"a", "b"})
	assert.Equal(t, "a b", joined)
}

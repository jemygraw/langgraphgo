package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTextLoader(t *testing.T) {
	ctx := context.Background()
	content := "Line 1\nLine 2\nLine 3"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	t.Run("Basic Load", func(t *testing.T) {
		loader := NewTextLoader(tmpFile)
		docs, err := loader.Load(ctx)
		assert.NoError(t, err)
		assert.Len(t, docs, 1)
		assert.Equal(t, content, docs[0].Content)
		assert.Equal(t, tmpFile, docs[0].Metadata["source"])
	})

	t.Run("Load with Metadata", func(t *testing.T) {
		loader := NewTextLoader(tmpFile, WithMetadata(map[string]any{"author": "test"}))
		docs, err := loader.Load(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test", docs[0].Metadata["author"])
	})
}

func TestTextByLinesLoader(t *testing.T) {
	ctx := context.Background()
	content := "Line 1\n\nLine 2\n  \nLine 3"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_lines.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	loader := NewTextByLinesLoader(tmpFile, nil)
	docs, err := loader.Load(ctx)
	assert.NoError(t, err)
	assert.Len(t, docs, 3) // Empty lines should be skipped
	assert.Equal(t, "Line 1", docs[0].Content)
	assert.Equal(t, "Line 2", docs[1].Content)
	assert.Equal(t, "Line 3", docs[2].Content)
}

func TestTextByChaptersLoader(t *testing.T) {
	ctx := context.Background()
	content := "Chapter 1\nContent 1\nChapter 2\nContent 2"
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_chapters.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	assert.NoError(t, err)

	loader := NewTextByChaptersLoader(tmpFile, WithChapterPattern("Chapter"))
	docs, err := loader.Load(ctx)
	assert.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Contains(t, docs[0].Content, "Chapter 1")
	assert.Contains(t, docs[1].Content, "Chapter 2")
}

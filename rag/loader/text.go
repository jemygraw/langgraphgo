package loader

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"strings"

	"github.com/smallnest/langgraphgo/rag"
)

// TextLoader loads documents from text files
type TextLoader struct {
	filePath      string
	encoding      string
	metadata      map[string]any
	lineSeparator string
}

// TextLoaderOption configures the TextLoader
type TextLoaderOption func(*TextLoader)

// WithEncoding sets the text encoding
func WithEncoding(encoding string) TextLoaderOption {
	return func(l *TextLoader) {
		l.encoding = encoding
	}
}

// WithMetadata sets additional metadata for loaded documents
func WithMetadata(metadata map[string]any) TextLoaderOption {
	return func(l *TextLoader) {
		maps.Copy(l.metadata, metadata)
	}
}

// WithLineSeparator sets the line separator
func WithLineSeparator(separator string) TextLoaderOption {
	return func(l *TextLoader) {
		l.lineSeparator = separator
	}
}

// NewTextLoader creates a new TextLoader
func NewTextLoader(filePath string, opts ...TextLoaderOption) rag.DocumentLoader {
	l := &TextLoader{
		filePath:      filePath,
		encoding:      "utf-8",
		metadata:      make(map[string]any),
		lineSeparator: "\n",
	}

	// Add default metadata
	l.metadata["source"] = filePath
	l.metadata["type"] = "text"

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Load loads documents from the text file
func (l *TextLoader) Load(ctx context.Context) ([]rag.Document, error) {
	return l.LoadWithMetadata(ctx, l.metadata)
}

// LoadWithMetadata loads documents with additional metadata
func (l *TextLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]rag.Document, error) {
	// Combine default metadata with provided metadata
	combinedMetadata := make(map[string]any)
	maps.Copy(combinedMetadata, l.metadata)
	maps.Copy(combinedMetadata, metadata)

	// Open the file
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", l.filePath, err)
	}
	defer file.Close()

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", l.filePath, err)
	}

	// Create document
	doc := rag.Document{
		ID:       l.generateDocumentID(),
		Content:  string(content),
		Metadata: combinedMetadata,
	}

	return []rag.Document{doc}, nil
}

// generateDocumentID generates a unique document ID
func (l *TextLoader) generateDocumentID() string {
	return fmt.Sprintf("text_%s", l.filePath)
}

// TextByLinesLoader loads documents splitting by lines
type TextByLinesLoader struct {
	filePath string
	metadata map[string]any
}

// NewTextByLinesLoader creates a new TextByLinesLoader
func NewTextByLinesLoader(filePath string, metadata map[string]any) rag.DocumentLoader {
	if metadata == nil {
		metadata = make(map[string]any)
	}

	metadata["source"] = filePath
	metadata["type"] = "text_lines"

	return &TextByLinesLoader{
		filePath: filePath,
		metadata: metadata,
	}
}

// Load loads documents from the text file, splitting by lines
func (l *TextByLinesLoader) Load(ctx context.Context) ([]rag.Document, error) {
	return l.LoadWithMetadata(ctx, l.metadata)
}

// LoadWithMetadata loads documents with additional metadata, splitting by lines
func (l *TextByLinesLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]rag.Document, error) {
	// Combine metadata
	combinedMetadata := make(map[string]any)
	maps.Copy(combinedMetadata, l.metadata)
	maps.Copy(combinedMetadata, metadata)

	// Open the file
	file, err := os.Open(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", l.filePath, err)
	}
	defer file.Close()

	// Read line by line
	var documents []rag.Document
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // Skip empty lines
		}

		lineMetadata := make(map[string]any)
		maps.Copy(lineMetadata, combinedMetadata)
		lineMetadata["line_number"] = lineNumber

		doc := rag.Document{
			ID:       fmt.Sprintf("%s_line_%d", l.filePath, lineNumber),
			Content:  line,
			Metadata: lineMetadata,
		}

		documents = append(documents, doc)
		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", l.filePath, err)
	}

	return documents, nil
}

// TextByParagraphsLoader loads documents splitting by paragraphs
type TextByParagraphsLoader struct {
	filePath        string
	metadata        map[string]any
	paragraphMarker string
}

// TextByParagraphsLoaderOption configures the TextByParagraphsLoader
type TextByParagraphsLoaderOption func(*TextByParagraphsLoader)

// WithParagraphMarker sets the paragraph marker
func WithParagraphMarker(marker string) TextByParagraphsLoaderOption {
	return func(l *TextByParagraphsLoader) {
		l.paragraphMarker = marker
	}
}

// NewTextByParagraphsLoader creates a new TextByParagraphsLoader
func NewTextByParagraphsLoader(filePath string, opts ...TextByParagraphsLoaderOption) rag.DocumentLoader {
	l := &TextByParagraphsLoader{
		filePath:        filePath,
		metadata:        make(map[string]any),
		paragraphMarker: "\n\n",
	}

	l.metadata["source"] = filePath
	l.metadata["type"] = "text_paragraphs"

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Load loads documents from the text file, splitting by paragraphs
func (l *TextByParagraphsLoader) Load(ctx context.Context) ([]rag.Document, error) {
	return l.LoadWithMetadata(ctx, l.metadata)
}

// LoadWithMetadata loads documents with additional metadata, splitting by paragraphs
func (l *TextByParagraphsLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]rag.Document, error) {
	// Combine metadata
	combinedMetadata := make(map[string]any)
	maps.Copy(combinedMetadata, l.metadata)
	maps.Copy(combinedMetadata, metadata)

	// Read the entire file content
	content, err := os.ReadFile(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", l.filePath, err)
	}

	// Split by paragraphs
	paragraphs := strings.Split(string(content), l.paragraphMarker)
	var documents []rag.Document

	for i, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue // Skip empty paragraphs
		}

		paragraphMetadata := make(map[string]any)
		maps.Copy(paragraphMetadata, combinedMetadata)
		paragraphMetadata["paragraph_number"] = i

		doc := rag.Document{
			ID:       fmt.Sprintf("%s_paragraph_%d", l.filePath, i),
			Content:  paragraph,
			Metadata: paragraphMetadata,
		}

		documents = append(documents, doc)
	}

	return documents, nil
}

// TextByChaptersLoader loads documents splitting by chapters
type TextByChaptersLoader struct {
	filePath       string
	metadata       map[string]any
	chapterPattern string
}

// TextByChaptersLoaderOption configures the TextByChaptersLoader
type TextByChaptersLoaderOption func(*TextByChaptersLoader)

// WithChapterPattern sets the pattern that identifies chapters
func WithChapterPattern(pattern string) TextByChaptersLoaderOption {
	return func(l *TextByChaptersLoader) {
		l.chapterPattern = pattern
	}
}

// NewTextByChaptersLoader creates a new TextByChaptersLoader
func NewTextByChaptersLoader(filePath string, opts ...TextByChaptersLoaderOption) rag.DocumentLoader {
	l := &TextByChaptersLoader{
		filePath:       filePath,
		metadata:       make(map[string]any),
		chapterPattern: "Chapter",
	}

	l.metadata["source"] = filePath
	l.metadata["type"] = "text_chapters"

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// Load loads documents from the text file, splitting by chapters
func (l *TextByChaptersLoader) Load(ctx context.Context) ([]rag.Document, error) {
	return l.LoadWithMetadata(ctx, l.metadata)
}

// LoadWithMetadata loads documents with additional metadata, splitting by chapters
func (l *TextByChaptersLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]rag.Document, error) {
	// Combine metadata
	combinedMetadata := make(map[string]any)
	maps.Copy(combinedMetadata, l.metadata)
	maps.Copy(combinedMetadata, metadata)

	// Read the entire file content
	content, err := os.ReadFile(l.filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", l.filePath, err)
	}

	// Split by lines to find chapters
	lines := strings.Split(string(content), "\n")
	var documents []rag.Document
	var currentChapter strings.Builder
	chapterNumber := 1
	var chapterTitle string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Check if this line starts a new chapter
		if strings.Contains(trimmedLine, l.chapterPattern) {
			// Save the previous chapter if it exists
			if currentChapter.Len() > 0 {
				chapterContent := strings.TrimSpace(currentChapter.String())
				if chapterContent != "" {
					chapterMetadata := make(map[string]any)
					maps.Copy(chapterMetadata, combinedMetadata)
					chapterMetadata["chapter_number"] = chapterNumber
					chapterMetadata["chapter_title"] = chapterTitle

					doc := rag.Document{
						ID:       fmt.Sprintf("%s_chapter_%d", l.filePath, chapterNumber),
						Content:  chapterContent,
						Metadata: chapterMetadata,
					}

					documents = append(documents, doc)
				}
			}

			// Start a new chapter
			currentChapter.Reset()
			currentChapter.WriteString(line + "\n")
			chapterTitle = trimmedLine
			chapterNumber++
		} else {
			currentChapter.WriteString(line + "\n")
		}
	}

	// Don't forget the last chapter
	if currentChapter.Len() > 0 {
		chapterContent := strings.TrimSpace(currentChapter.String())
		if chapterContent != "" {
			chapterMetadata := make(map[string]any)
			maps.Copy(chapterMetadata, combinedMetadata)
			chapterMetadata["chapter_number"] = chapterNumber
			chapterMetadata["chapter_title"] = chapterTitle

			doc := rag.Document{
				ID:       fmt.Sprintf("%s_chapter_%d", l.filePath, chapterNumber),
				Content:  chapterContent,
				Metadata: chapterMetadata,
			}

			documents = append(documents, doc)
		}
	}

	return documents, nil
}

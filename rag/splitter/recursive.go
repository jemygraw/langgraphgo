package splitter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/smallnest/langgraphgo/rag"
)

// RecursiveCharacterTextSplitter recursively splits text while keeping related pieces together
type RecursiveCharacterTextSplitter struct {
	separators   []string
	chunkSize    int
	chunkOverlap int
	lengthFunc   func(string) int
}

// RecursiveCharacterTextSplitterOption configures the RecursiveCharacterTextSplitter
type RecursiveCharacterTextSplitterOption func(*RecursiveCharacterTextSplitter)

// WithChunkSize sets the chunk size for the splitter
func WithChunkSize(size int) RecursiveCharacterTextSplitterOption {
	return func(s *RecursiveCharacterTextSplitter) {
		s.chunkSize = size
	}
}

// WithChunkOverlap sets the chunk overlap for the splitter
func WithChunkOverlap(overlap int) RecursiveCharacterTextSplitterOption {
	return func(s *RecursiveCharacterTextSplitter) {
		s.chunkOverlap = overlap
	}
}

// WithSeparators sets the custom separators for the splitter
func WithSeparators(separators []string) RecursiveCharacterTextSplitterOption {
	return func(s *RecursiveCharacterTextSplitter) {
		s.separators = separators
	}
}

// WithLengthFunction sets a custom length function
func WithLengthFunction(fn func(string) int) RecursiveCharacterTextSplitterOption {
	return func(s *RecursiveCharacterTextSplitter) {
		s.lengthFunc = fn
	}
}

// NewRecursiveCharacterTextSplitter creates a new RecursiveCharacterTextSplitter
func NewRecursiveCharacterTextSplitter(opts ...RecursiveCharacterTextSplitterOption) rag.TextSplitter {
	s := &RecursiveCharacterTextSplitter{
		separators:   []string{"\n\n", "\n", " ", ""},
		chunkSize:    1000,
		chunkOverlap: 200,
		lengthFunc:   func(s string) int { return len(s) },
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SplitText splits text into chunks
func (s *RecursiveCharacterTextSplitter) SplitText(text string) []string {
	return s.splitTextRecursive(text, s.separators)
}

// SplitDocuments splits documents into chunks
func (s *RecursiveCharacterTextSplitter) SplitDocuments(docs []rag.Document) []rag.Document {
	chunks := make([]rag.Document, 0)

	for _, doc := range docs {
		textChunks := s.SplitText(doc.Content)

		for i, chunk := range textChunks {
			// Create metadata for the chunk
			metadata := make(map[string]any)
			for k, v := range doc.Metadata {
				metadata[k] = v
			}

			// Add chunk-specific metadata
			metadata["chunk_index"] = i
			metadata["chunk_total"] = len(textChunks)
			metadata["parent_id"] = doc.ID

			chunkDoc := rag.Document{
				ID:        fmt.Sprintf("%s_chunk_%d", doc.ID, i),
				Content:   chunk,
				Metadata:  metadata,
				CreatedAt: doc.CreatedAt,
				UpdatedAt: doc.UpdatedAt,
			}

			chunks = append(chunks, chunkDoc)
		}
	}

	return chunks
}

// JoinText joins text chunks back together
func (s *RecursiveCharacterTextSplitter) JoinText(chunks []string) string {
	if s.chunkOverlap == 0 {
		return strings.Join(chunks, " ")
	}

	// Join chunks with overlap consideration
	result := chunks[0]
	for i := 1; i < len(chunks); i++ {
		// Remove overlap from the beginning of current chunk
		chunk := chunks[i]
		if len(chunk) > s.chunkOverlap {
			chunk = chunk[s.chunkOverlap:]
		}
		result += " " + chunk
	}

	return result
}

// splitTextRecursive recursively splits text using the provided separators
func (s *RecursiveCharacterTextSplitter) splitTextRecursive(text string, separators []string) []string {
	if s.lengthFunc(text) <= s.chunkSize {
		return []string{text}
	}

	if len(separators) == 0 {
		// No more separators, split by character
		return s.splitByCharacter(text)
	}

	separator := separators[0]
	remainingSeparators := separators[1:]

	splits := s.splitTextHelper(text, separator)

	// Filter out empty splits
	var finalSplits []string
	for _, split := range splits {
		if strings.TrimSpace(split) != "" {
			finalSplits = append(finalSplits, split)
		}
	}

	// Now further split the splits that are too large
	var goodSplits []string
	for _, split := range finalSplits {
		if s.lengthFunc(split) <= s.chunkSize {
			goodSplits = append(goodSplits, split)
		} else {
			// If split is still too large, recursively split with next separator
			otherSplits := s.splitTextRecursive(split, remainingSeparators)
			goodSplits = append(goodSplits, otherSplits...)
		}
	}

	return s.mergeSplits(goodSplits)
}

// splitTextHelper splits text by a separator
func (s *RecursiveCharacterTextSplitter) splitTextHelper(text, separator string) []string {
	if separator == "" {
		return s.splitByCharacter(text)
	}

	return strings.Split(text, separator)
}

// splitByCharacter splits text by character
func (s *RecursiveCharacterTextSplitter) splitByCharacter(text string) []string {
	var splits []string

	for i := 0; i < len(text); i += s.chunkSize - s.chunkOverlap {
		end := i + s.chunkSize
		if end > len(text) {
			end = len(text)
		}

		splits = append(splits, text[i:end])
	}

	return splits
}

// mergeSplits merges splits together to respect chunk size and overlap
func (s *RecursiveCharacterTextSplitter) mergeSplits(splits []string) []string {
	var merged []string
	var current string

	for _, split := range splits {
		// If current is empty, start with this split
		if current == "" {
			current = split
			continue
		}

		// Check if adding this split would exceed chunk size
		proposed := current + "\n\n" + split
		if s.lengthFunc(proposed) <= s.chunkSize {
			current = proposed
		} else {
			// Add current to merged and start new with split
			merged = append(merged, current)
			current = split
		}
	}

	// Add the last chunk
	if current != "" {
		merged = append(merged, current)
	}

	// Apply overlap
	if s.chunkOverlap > 0 && len(merged) > 1 {
		merged = s.applyOverlap(merged)
	}

	return merged
}

// applyOverlap applies overlap between consecutive chunks
func (s *RecursiveCharacterTextSplitter) applyOverlap(chunks []string) []string {
	var overlapped []string

	for i, chunk := range chunks {
		if i == 0 {
			overlapped = append(overlapped, chunk)
			continue
		}

		prevChunk := chunks[i-1]
		overlap := s.findOverlap(prevChunk, chunk)

		if overlap != "" {
			// Remove overlap from current chunk
			chunk = strings.TrimPrefix(chunk, overlap)
			chunk = strings.TrimSpace(chunk)
		}

		overlapped = append(overlapped, chunk)
	}

	return overlapped
}

// findOverlap finds the maximum overlap between the end of text1 and start of text2
func (s *RecursiveCharacterTextSplitter) findOverlap(text1, text2 string) string {
	maxOverlap := min(s.chunkOverlap, len(text1), len(text2))

	for overlap := maxOverlap; overlap > 0; overlap-- {
		text1End := text1[len(text1)-overlap:]
		text2Start := text2[:overlap]

		// Normalize whitespace for comparison
		text1End = strings.TrimSpace(text1End)
		text2Start = strings.TrimSpace(text2Start)

		if text1End == text2Start {
			return text2[:overlap]
		}
	}

	return ""
}

// min returns the minimum of multiple values
func min(values ...int) int {
	if len(values) == 0 {
		return 0
	}

	minValue := values[0]
	for _, value := range values[1:] {
		if value < minValue {
			minValue = value
		}
	}

	return minValue
}

// CharacterTextSplitter splits text by character count
type CharacterTextSplitter struct {
	separator    string
	chunkSize    int
	chunkOverlap int
	lengthFunc   func(string) int
}

// CharacterTextSplitterOption configures the CharacterTextSplitter
type CharacterTextSplitterOption func(*CharacterTextSplitter)

// WithCharacterSeparator sets the separator for character splitter
func WithCharacterSeparator(separator string) CharacterTextSplitterOption {
	return func(s *CharacterTextSplitter) {
		s.separator = separator
	}
}

// WithCharacterChunkSize sets the chunk size for character splitter
func WithCharacterChunkSize(size int) CharacterTextSplitterOption {
	return func(s *CharacterTextSplitter) {
		s.chunkSize = size
	}
}

// WithCharacterChunkOverlap sets the chunk overlap for character splitter
func WithCharacterChunkOverlap(overlap int) CharacterTextSplitterOption {
	return func(s *CharacterTextSplitter) {
		s.chunkOverlap = overlap
	}
}

// NewCharacterTextSplitter creates a new CharacterTextSplitter
func NewCharacterTextSplitter(opts ...CharacterTextSplitterOption) rag.TextSplitter {
	s := &CharacterTextSplitter{
		separator:    "\n",
		chunkSize:    1000,
		chunkOverlap: 200,
		lengthFunc:   func(s string) int { return len(s) },
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// SplitText splits text into chunks by separator or character
func (s *CharacterTextSplitter) SplitText(text string) []string {
	if s.separator != "" {
		return s.splitBySeparator(text)
	}
	return s.splitByCharacterCount(text)
}

// SplitDocuments splits documents into chunks
func (s *CharacterTextSplitter) SplitDocuments(docs []rag.Document) []rag.Document {
	chunks := make([]rag.Document, 0)

	for _, doc := range docs {
		textChunks := s.SplitText(doc.Content)

		for i, chunk := range textChunks {
			metadata := make(map[string]any)
			for k, v := range doc.Metadata {
				metadata[k] = v
			}

			metadata["chunk_index"] = i
			metadata["chunk_total"] = len(textChunks)
			metadata["parent_id"] = doc.ID

			chunkDoc := rag.Document{
				ID:        fmt.Sprintf("%s_chunk_%d", doc.ID, i),
				Content:   chunk,
				Metadata:  metadata,
				CreatedAt: doc.CreatedAt,
				UpdatedAt: doc.UpdatedAt,
			}

			chunks = append(chunks, chunkDoc)
		}
	}

	return chunks
}

// JoinText joins text chunks back together
func (s *CharacterTextSplitter) JoinText(chunks []string) string {
	if s.separator != "" {
		return strings.Join(chunks, s.separator)
	}
	return strings.Join(chunks, "")
}

// splitBySeparator splits text by separator
func (s *CharacterTextSplitter) splitBySeparator(text string) []string {
	if s.separator == "" {
		return s.splitByCharacterCount(text)
	}

	splits := strings.Split(text, s.separator)
	var chunks []string
	var current string

	for _, split := range splits {
		if s.lengthFunc(current)+s.lengthFunc(split)+len(s.separator) <= s.chunkSize {
			if current != "" {
				current += s.separator + split
			} else {
				current = split
			}
		} else {
			if current != "" {
				chunks = append(chunks, current)
			}
			current = split
		}
	}

	if current != "" {
		chunks = append(chunks, current)
	}

	return chunks
}

// splitByCharacterCount splits text by character count
func (s *CharacterTextSplitter) splitByCharacterCount(text string) []string {
	var chunks []string

	for i := 0; i < len(text); i += s.chunkSize - s.chunkOverlap {
		end := i + s.chunkSize
		if end > len(text) {
			end = len(text)
		}

		chunks = append(chunks, text[i:end])
	}

	return chunks
}

// TokenTextSplitter splits text by token count
type TokenTextSplitter struct {
	chunkSize    int
	chunkOverlap int
	tokenizer    Tokenizer
}

// Tokenizer interface for different tokenization strategies
type Tokenizer interface {
	Encode(text string) []string
	Decode(tokens []string) string
}

// DefaultTokenizer is a simple word-based tokenizer
type DefaultTokenizer struct{}

// Encode tokenizes text into words
func (t *DefaultTokenizer) Encode(text string) []string {
	words := []string{}
	current := ""

	for _, char := range text {
		if unicode.IsSpace(rune(char)) {
			if current != "" {
				words = append(words, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		words = append(words, current)
	}

	return words
}

// Decode detokenizes words back to text
func (t *DefaultTokenizer) Decode(tokens []string) string {
	return strings.Join(tokens, " ")
}

// NewTokenTextSplitter creates a new TokenTextSplitter
func NewTokenTextSplitter(chunkSize, chunkOverlap int, tokenizer Tokenizer) rag.TextSplitter {
	if tokenizer == nil {
		tokenizer = &DefaultTokenizer{}
	}

	return &TokenTextSplitter{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
		tokenizer:    tokenizer,
	}
}

// SplitText splits text into chunks by token count
func (s *TokenTextSplitter) SplitText(text string) []string {
	tokens := s.tokenizer.Encode(text)

	if len(tokens) <= s.chunkSize {
		return []string{text}
	}

	var chunks []string

	for i := 0; i < len(tokens); i += s.chunkSize - s.chunkOverlap {
		end := i + s.chunkSize
		if end > len(tokens) {
			end = len(tokens)
		}

		chunkTokens := tokens[i:end]
		chunks = append(chunks, s.tokenizer.Decode(chunkTokens))
	}

	return chunks
}

// SplitDocuments splits documents into chunks
func (s *TokenTextSplitter) SplitDocuments(docs []rag.Document) []rag.Document {
	chunks := make([]rag.Document, 0)

	for _, doc := range docs {
		textChunks := s.SplitText(doc.Content)

		for i, chunk := range textChunks {
			metadata := make(map[string]any)
			for k, v := range doc.Metadata {
				metadata[k] = v
			}

			metadata["chunk_index"] = i
			metadata["chunk_total"] = len(textChunks)
			metadata["parent_id"] = doc.ID

			chunkDoc := rag.Document{
				ID:        fmt.Sprintf("%s_chunk_%d", doc.ID, i),
				Content:   chunk,
				Metadata:  metadata,
				CreatedAt: doc.CreatedAt,
				UpdatedAt: doc.UpdatedAt,
			}

			chunks = append(chunks, chunkDoc)
		}
	}

	return chunks
}

// JoinText joins text chunks back together
func (s *TokenTextSplitter) JoinText(chunks []string) string {
	return strings.Join(chunks, " ")
}

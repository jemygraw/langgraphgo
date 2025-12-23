package loader

import (
	"context"
	"maps"

	"github.com/smallnest/langgraphgo/rag"
)

// StaticDocumentLoader loads documents from a static list
type StaticDocumentLoader struct {
	Documents []rag.Document
}

// NewStaticDocumentLoader creates a new StaticDocumentLoader
func NewStaticDocumentLoader(documents []rag.Document) *StaticDocumentLoader {
	return &StaticDocumentLoader{
		Documents: documents,
	}
}

// Load returns the static list of documents
func (l *StaticDocumentLoader) Load(ctx context.Context) ([]rag.Document, error) {
	return l.Documents, nil
}

// LoadWithMetadata returns the static list of documents with additional metadata
func (l *StaticDocumentLoader) LoadWithMetadata(ctx context.Context, metadata map[string]any) ([]rag.Document, error) {
	if metadata == nil {
		return l.Documents, nil
	}

	// Copy documents and add metadata
	docs := make([]rag.Document, len(l.Documents))
	for i, doc := range l.Documents {
		newDoc := doc
		if newDoc.Metadata == nil {
			newDoc.Metadata = make(map[string]any)
		}
		maps.Copy(newDoc.Metadata, metadata)
		docs[i] = newDoc
	}

	return docs, nil
}

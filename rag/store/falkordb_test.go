package store

import (
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestFalkorDBGraph(t *testing.T) {
	// We can't easily test real FalkorDB connection without a running instance.
	// But we can test NewFalkorDBGraph returns error for invalid URL.

	t.Run("Invalid URL", func(t *testing.T) {
		g, err := NewFalkorDBGraph("invalid://")
		assert.Error(t, err)
		assert.Nil(t, g)
	})

	// Test the helper functions if they are exported or accessible
	// Most are private, let's see if we can trigger some paths via NewKnowledgeGraph

	t.Run("NewKnowledgeGraph FalkorDB", func(t *testing.T) {
		g, err := NewKnowledgeGraph("falkordb://localhost:6379/graph")
		// It might fail to connect but should reach the factory logic
		if err == nil {
			assert.NotNil(t, g)
		}
	})

	t.Run("Sanitize Label", func(t *testing.T) {
		assert.Equal(t, "Person", sanitizeLabel("Person"))
		assert.Equal(t, "Person_Age", sanitizeLabel("Person Age"))
	})

	t.Run("Props ToString", func(t *testing.T) {
		props := map[string]any{"name": "test", "age": 30, "ok": true}
		s := propsToString(props)
		assert.Contains(t, s, "name")
		assert.Contains(t, s, "age")
		assert.Contains(t, s, "ok")
	})

	t.Run("Internal Helpers", func(t *testing.T) {
		assert.Equal(t, "\"test\"", quoteString("test"))
		assert.Equal(t, 123, quoteString(123))

		rs := randomString(10)
		assert.Len(t, rs, 10)

		n := &Node{Alias: "a", Label: "L", Properties: map[string]any{"p": 1}}
		assert.Contains(t, n.String(), "a:L")

		e := &Edge{Source: n, Destination: n, Relation: "T"}
		assert.Contains(t, e.String(), "-[:T]->")
	})

	t.Run("Entity To Map", func(t *testing.T) {
		e := &rag.Entity{ID: "1", Name: "n", Type: "T", Properties: map[string]any{"p": "v"}}
		m := entityToMap(e)
		assert.Equal(t, "n", m["name"])
		assert.Equal(t, "T", m["type"])
		assert.Equal(t, "v", m["p"])
	})

	t.Run("Relationship To Map", func(t *testing.T) {
		r := &rag.Relationship{ID: "1", Source: "s", Target: "t", Type: "T", Properties: map[string]any{"p": "v"}}
		m := relationshipToMap(r)
		assert.Equal(t, "T", m["type"])
		assert.Equal(t, "v", m["p"])
	})
}

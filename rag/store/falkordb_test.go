package store

import (
	"testing"

	"github.com/smallnest/langgraphgo/rag"
	"github.com/stretchr/testify/assert"
)

func TestNewFalkorDBGraph(t *testing.T) {
	t.Run("Valid connection string with custom graph name", func(t *testing.T) {
		g, err := NewFalkorDBGraph("falkordb://localhost:6379/custom_graph")
		assert.NoError(t, err)
		assert.NotNil(t, g)
		fg := g.(*FalkorDBGraph)
		assert.Equal(t, "custom_graph", fg.graphName)
		assert.NotNil(t, fg.client)
		fg.Close()
	})

	t.Run("Valid connection string with default graph name", func(t *testing.T) {
		g, err := NewFalkorDBGraph("falkordb://localhost:6379")
		assert.NoError(t, err)
		assert.NotNil(t, g)
		fg := g.(*FalkorDBGraph)
		assert.Equal(t, "rag", fg.graphName) // Default graph name
		fg.Close()
	})

	t.Run("Invalid URL", func(t *testing.T) {
		g, err := NewFalkorDBGraph("://invalid")
		assert.Error(t, err)
		assert.Nil(t, g)
		assert.Contains(t, err.Error(), "invalid connection string")
	})

	t.Run("Missing host", func(t *testing.T) {
		g, err := NewFalkorDBGraph("falkordb:///graph")
		assert.Error(t, err)
		assert.Nil(t, g)
		assert.Contains(t, err.Error(), "missing host")
	})

	t.Run("NewKnowledgeGraph factory", func(t *testing.T) {
		g, err := NewKnowledgeGraph("falkordb://localhost:6379/graph")
		if err == nil {
			assert.NotNil(t, g)
			if fg, ok := g.(*FalkorDBGraph); ok {
				fg.Close()
			}
		}
	})
}

func TestSanitizeLabel(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Simple label", "Person", "Person"},
		{"Label with space", "Person Age", "Person_Age"},
		{"Label with special chars", "Person-Type@123", "Person_Type_123"},
		{"Empty label", "", "Entity"},
		{"Only special chars", "@#$%", "____"}, // Special chars become underscores, not Entity
		{"Mixed case", "MyEntity", "MyEntity"},
		{"Numbers", "Entity123", "Entity123"},
		{"Underscores preserved", "My_Entity", "My_Entity"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeLabel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPropsToString(t *testing.T) {
	t.Run("String properties", func(t *testing.T) {
		props := map[string]any{"name": "test", "age": 30}
		s := propsToString(props)
		assert.Contains(t, s, "name")
		assert.Contains(t, s, "age")
		assert.Contains(t, s, "{")
		assert.Contains(t, s, "}")
	})

	t.Run("Float32 slice embedding", func(t *testing.T) {
		props := map[string]any{
			"name":      "entity",
			"embedding": []float32{0.1, 0.2, 0.3},
		}
		s := propsToString(props)
		assert.Contains(t, s, "embedding")
		assert.Contains(t, s, "[")
		assert.Contains(t, s, "]")
	})

	t.Run("Boolean and numeric values", func(t *testing.T) {
		props := map[string]any{
			"active": true,
			"count":  42,
			"ratio":  3.14,
		}
		s := propsToString(props)
		assert.Contains(t, s, "active")
		assert.Contains(t, s, "count")
		assert.Contains(t, s, "ratio")
	})

	t.Run("Empty map", func(t *testing.T) {
		props := map[string]any{}
		s := propsToString(props)
		assert.Equal(t, "{}", s)
	})
}

func TestEntityToMap(t *testing.T) {
	t.Run("Entity with all fields", func(t *testing.T) {
		e := &rag.Entity{
			ID:         "1",
			Name:       "John",
			Type:       "Person",
			Embedding:  []float32{0.1, 0.2},
			Properties: map[string]any{"age": 30},
		}
		m := entityToMap(e)
		assert.Equal(t, "John", m["name"])
		assert.Equal(t, "Person", m["type"])
		assert.Equal(t, 30, m["age"])
		assert.NotNil(t, m["embedding"])
	})

	t.Run("Entity without embedding", func(t *testing.T) {
		e := &rag.Entity{
			ID:         "2",
			Name:       "Jane",
			Type:       "Person",
			Properties: map[string]any{"city": "NYC"},
		}
		m := entityToMap(e)
		assert.Equal(t, "Jane", m["name"])
		assert.Equal(t, "Person", m["type"])
		assert.Equal(t, "NYC", m["city"])
		assert.Nil(t, m["embedding"])
	})

	t.Run("Entity with empty properties", func(t *testing.T) {
		e := &rag.Entity{
			ID:         "3",
			Name:       "Test",
			Type:       "Type",
			Properties: map[string]any{},
		}
		m := entityToMap(e)
		assert.Equal(t, "Test", m["name"])
		assert.Equal(t, "Type", m["type"])
	})
}

func TestRelationshipToMap(t *testing.T) {
	t.Run("Relationship with all fields", func(t *testing.T) {
		r := &rag.Relationship{
			ID:         "1",
			Source:     "s",
			Target:     "t",
			Type:       "KNOWS",
			Weight:     0.8,
			Confidence: 0.9,
			Properties: map[string]any{"since": 2020},
		}
		m := relationshipToMap(r)
		assert.Equal(t, "KNOWS", m["type"])
		assert.Equal(t, 0.8, m["weight"])
		assert.Equal(t, 0.9, m["confidence"])
		assert.Equal(t, 2020, m["since"])
	})

	t.Run("Relationship with empty properties", func(t *testing.T) {
		r := &rag.Relationship{
			ID:         "2",
			Source:     "a",
			Target:     "b",
			Type:       "RELATED",
			Properties: map[string]any{},
		}
		m := relationshipToMap(r)
		assert.Equal(t, "RELATED", m["type"])
		assert.Contains(t, m, "weight")
		assert.Contains(t, m, "confidence")
	})
}

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"String input", "hello", "hello"},
		{"Byte slice", []byte("world"), "world"},
		{"Integer", 123, "123"},
		{"Float", 3.14, "3.14"},
		{"Boolean", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseNode(t *testing.T) {
	t.Run("Standard format with labels and properties", func(t *testing.T) {
		// Format: [id, [labels], [[key1, val1], [key2, val2]]]
		obj := []any{
			int64(1),
			[]any{[]byte("Person")},
			[]any{
				[]any{int64(1), int64(2), "id"},
				[]any{int64(1), int64(4), "test"},
				[]any{int64(2), int64(4), "name"},
				[]any{int64(2), int64(4), "John"},
			},
		}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Equal(t, "Person", e.Type)
		assert.Equal(t, "test", e.ID)
		assert.Equal(t, "John", e.Name)
	})

	t.Run("Standard format with string labels", func(t *testing.T) {
		obj := []any{
			int64(1),
			[]any{"Company"},
			[]any{
				[]any{int64(1), int64(2), "id"},
				[]any{int64(1), int64(3), "c1"},
			},
		}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Equal(t, "Company", e.Type)
		assert.Equal(t, "c1", e.ID)
	})

	t.Run("KV format", func(t *testing.T) {
		obj := []any{
			[]any{"id", "node1"},
			[]any{"labels", []any{"Person"}},
			[]any{"properties", []any{
				[]any{"name", "Alice"},
				[]any{"id", "alice1"},
			}},
		}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Equal(t, "alice1", e.ID)
		assert.Equal(t, "Alice", e.Name)
		assert.Equal(t, "Person", e.Type)
	})

	t.Run("KV format with string labels", func(t *testing.T) {
		obj := []any{
			[]any{"id", "node2"},
			[]any{"labels", []any{"Product", "Item"}},
			[]any{"properties", []any{
				[]any{"name", "Widget"},
				[]any{"price", "9.99"},
			}},
		}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Equal(t, "node2", e.ID)
		assert.Equal(t, "Widget", e.Name)
		assert.Equal(t, "Product", e.Type) // First label
		assert.Equal(t, "9.99", e.Properties["price"])
	})

	t.Run("Invalid format", func(t *testing.T) {
		e := parseNode("not a slice")
		assert.Nil(t, e)
	})

	t.Run("Empty slice", func(t *testing.T) {
		e := parseNode([]any{})
		assert.NotNil(t, e)
	})

	t.Run("Single element slice", func(t *testing.T) {
		e := parseNode([]any{int64(1)})
		assert.NotNil(t, e)
	})

	t.Run("Two element slice without valid KV", func(t *testing.T) {
		obj := []any{int64(1), "not a slice"}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Empty(t, e.ID)
	})

	t.Run("Complex nested structure", func(t *testing.T) {
		obj := []any{
			int64(1),
			[]any{
				int64(2),
				[]any{[]byte("Label")},
				[]any{
					[]any{int64(1), int64(2), "id"},
					[]any{int64(1), int64(3), "id1"},
				},
			},
		}
		e := parseNode(obj)
		assert.NotNil(t, e)
		assert.Equal(t, "id1", e.ID)
	})
}

func TestParseNodeKV(t *testing.T) {
	t.Run("Complete KV pairs", func(t *testing.T) {
		pairs := []any{
			[]any{"id", "entity1"},
			[]any{"labels", []any{"Person"}},
			[]any{"properties", []any{
				[]any{"name", "Bob"},
				[]any{"type", "User"},
				[]any{"age", "30"},
			}},
		}
		e := parseNodeKV(pairs)
		assert.NotNil(t, e)
		assert.Equal(t, "entity1", e.ID)
		assert.Equal(t, "User", e.Type)
		assert.Equal(t, "Bob", e.Name)
		assert.Equal(t, "30", e.Properties["age"])
	})

	t.Run("Invalid pairs", func(t *testing.T) {
		pairs := []any{
			"not a pair",
			[]any{"single"},
		}
		e := parseNodeKV(pairs)
		assert.NotNil(t, e)
		assert.Empty(t, e.ID)
	})
}

func TestParseEdge(t *testing.T) {
	t.Run("Standard edge format", func(t *testing.T) {
		obj := []any{
			int64(1),
			[]byte("KNOWS"),
			int64(2),
			int64(3),
			[]any{
				[]any{"id", "rel1"},
				[]any{"weight", 0.5},
			},
		}
		rel := parseEdge(obj, "source1", "target1")
		assert.NotNil(t, rel)
		assert.Equal(t, "source1", rel.Source)
		assert.Equal(t, "target1", rel.Target)
		assert.Equal(t, "KNOWS", rel.Type)
		assert.Equal(t, "rel1", rel.ID)
	})

	t.Run("Standard edge with properties containing id", func(t *testing.T) {
		obj := []any{
			int64(1),
			[]byte("LIKES"),
			int64(2),
			int64(3),
			[]any{
				[]any{"custom_prop", "value1"},
				[]any{"id", "edge123"},
			},
		}
		rel := parseEdge(obj, "src", "dst")
		assert.NotNil(t, rel)
		assert.Equal(t, "LIKES", rel.Type)
		assert.Equal(t, "edge123", rel.ID)
		assert.Equal(t, "value1", rel.Properties["custom_prop"])
	})

	t.Run("KV edge format", func(t *testing.T) {
		obj := []any{
			[]any{"id", "edge1"},
			[]any{"type", "RELATED"},
			[]any{"properties", []any{
				[]any{"id", "edge_id1"},
				[]any{"strength", "high"},
			}},
		}
		rel := parseEdge(obj, "src", "dst")
		assert.NotNil(t, rel)
		assert.Equal(t, "src", rel.Source)
		assert.Equal(t, "dst", rel.Target)
		assert.Equal(t, "RELATED", rel.Type)
		assert.Equal(t, "edge_id1", rel.ID)
		assert.Equal(t, "high", rel.Properties["strength"])
	})

	t.Run("KV edge format with empty properties", func(t *testing.T) {
		obj := []any{
			[]any{"id", "edge2"},
			[]any{"type", "CONNECTS"},
			[]any{"properties", []any{}},
		}
		rel := parseEdge(obj, "a", "b")
		assert.NotNil(t, rel)
		assert.Equal(t, "a", rel.Source)
		assert.Equal(t, "b", rel.Target)
		assert.Equal(t, "CONNECTS", rel.Type)
		assert.Equal(t, "edge2", rel.ID)
	})

	t.Run("KV edge format without properties key", func(t *testing.T) {
		obj := []any{
			[]any{"id", "edge3"},
			[]any{"type", "LINKS"},
			[]any{"src", "nodeA"},
			[]any{"dst", "nodeB"},
		}
		rel := parseEdge(obj, "x", "y")
		assert.NotNil(t, rel)
		assert.Equal(t, "x", rel.Source)
		assert.Equal(t, "y", rel.Target)
		assert.Equal(t, "LINKS", rel.Type)
	})

	t.Run("Invalid format", func(t *testing.T) {
		rel := parseEdge("not a slice", "src", "dst")
		assert.Nil(t, rel)
	})

	t.Run("Short slice", func(t *testing.T) {
		obj := []any{int64(1), []byte("TYPE")}
		rel := parseEdge(obj, "s", "t")
		assert.Nil(t, rel)
	})

	t.Run("Three element slice", func(t *testing.T) {
		obj := []any{int64(1), []byte("TEST"), int64(3)}
		rel := parseEdge(obj, "a", "b")
		// Three elements: first is id, second is type (byte), third is something
		// The code returns a relationship with type set
		assert.NotNil(t, rel)
		assert.Equal(t, "TEST", rel.Type)
		assert.Equal(t, "a", rel.Source)
		assert.Equal(t, "b", rel.Target)
	})

	t.Run("String type", func(t *testing.T) {
		obj := []any{
			int64(1),
			"WORKS_WITH",
			int64(2),
			int64(3),
			[]any{},
		}
		rel := parseEdge(obj, "a", "b")
		assert.NotNil(t, rel)
		assert.Equal(t, "WORKS_WITH", rel.Type)
	})

	t.Run("String type with non-empty properties", func(t *testing.T) {
		obj := []any{
			int64(1),
			"MARRIED_TO",
			int64(2),
			int64(3),
			[]any{
				[]any{"since", "2020"},
				[]any{"id", "rel_custom_id"},
			},
		}
		rel := parseEdge(obj, "p1", "p2")
		assert.NotNil(t, rel)
		assert.Equal(t, "MARRIED_TO", rel.Type)
		assert.Equal(t, "rel_custom_id", rel.ID)
		assert.Equal(t, "2020", rel.Properties["since"])
	})

	t.Run("Empty slice", func(t *testing.T) {
		rel := parseEdge([]any{}, "src", "dst")
		assert.Nil(t, rel)
	})

	t.Run("KV edge format with src key", func(t *testing.T) {
		obj := []any{
			[]any{"id", "edge_src"},
			[]any{"src", "node_src"},
			[]any{"dst", "node_dst"},
		}
		rel := parseEdge(obj, "x", "y")
		assert.NotNil(t, rel)
		assert.Equal(t, "edge_src", rel.ID)
		// src and dst keys are recognized but don't override the parameters
		assert.Equal(t, "x", rel.Source)
		assert.Equal(t, "y", rel.Target)
	})

	t.Run("KV edge format with src and dst override", func(t *testing.T) {
		obj := []any{
			[]any{"type", "CONTAINS"},
			[]any{"src", "actual_source"},
			[]any{"dst", "actual_target"},
		}
		rel := parseEdge(obj, "param_source", "param_target")
		assert.NotNil(t, rel)
		assert.Equal(t, "CONTAINS", rel.Type)
		// The src/dst in KV don't override parameters in current implementation
		assert.Equal(t, "param_source", rel.Source)
		assert.Equal(t, "param_target", rel.Target)
	})

	t.Run("Edge with byte type value in properties", func(t *testing.T) {
		obj := []any{
			int64(1),
			[]byte("CONNECTED"),
			int64(2),
			int64(3),
			[]any{
				[]any{"id", "edge_bytes"},
				[]any{"note", []byte("note_value")},
			},
		}
		rel := parseEdge(obj, "s", "t")
		assert.NotNil(t, rel)
		assert.Equal(t, "CONNECTED", rel.Type)
		assert.Equal(t, "edge_bytes", rel.ID)
		assert.Equal(t, "note_value", rel.Properties["note"])
	})

	t.Run("Edge with weight in properties", func(t *testing.T) {
		obj := []any{
			int64(1),
			"RELATES",
			int64(2),
			int64(3),
			[]any{
				[]any{"weight", "0.7"},
				[]any{"id", "rel_weight"},
			},
		}
		rel := parseEdge(obj, "a", "b")
		assert.NotNil(t, rel)
		assert.Equal(t, "RELATES", rel.Type)
		assert.Equal(t, "rel_weight", rel.ID)
		assert.Equal(t, float64(0), rel.Weight) // weight is set to 0
	})

	t.Run("KV edge with weight in nested properties", func(t *testing.T) {
		obj := []any{
			[]any{"id", "edge_kvp"},
			[]any{"type", "WEIGHTED"},
			[]any{"properties", []any{
				[]any{"weight", "0.9"},
				[]any{"id", "prop_id"},
			}},
		}
		rel := parseEdge(obj, "src", "dst")
		assert.NotNil(t, rel)
		assert.Equal(t, "prop_id", rel.ID) // properties id overrides top-level id
		assert.Equal(t, "WEIGHTED", rel.Type)
		assert.Equal(t, float64(0), rel.Weight)
	})

	t.Run("KV edge with non-KV first element", func(t *testing.T) {
		obj := []any{
			"not a KV pair",
			[]any{"type", "SOME_TYPE"},
			[]any{"src", "source"},
		}
		rel := parseEdge(obj, "s", "t")
		assert.NotNil(t, rel)
		// First element is not a KV pair with special keys, so it falls through to standard parsing
		// But it's also not a []any, so we don't match the KV format
	})
}

func TestParseFalkorDBProperties(t *testing.T) {
	t.Run("Even number of properties", func(t *testing.T) {
		props := []any{
			[]any{int64(1), int64(2), "name"},
			[]any{int64(2), int64(4), "John"},
			[]any{int64(3), int64(4), "type"},
			[]any{int64(4), int64(6), "Person"},
		}
		e := &rag.Entity{Properties: make(map[string]any)}
		parseFalkorDBProperties(props, e)
		assert.Equal(t, "John", e.Name)
		assert.Equal(t, "Person", e.Type)
	})

	t.Run("Odd number of properties", func(t *testing.T) {
		props := []any{
			[]any{int64(1), int64(2), "id"},
			[]any{int64(2), int64(5), "test1"},
			[]any{int64(3), int64(3), "age"},
		}
		e := &rag.Entity{Properties: make(map[string]any)}
		parseFalkorDBProperties(props, e)
		assert.Equal(t, "test1", e.ID)
	})

	t.Run("Custom properties", func(t *testing.T) {
		props := []any{
			[]any{int64(1), int64(3), "city"},
			[]any{int64(2), int64(3), "NYC"},
			[]any{int64(3), int64(3), "country"},
			[]any{int64(4), int64(3), "USA"},
		}
		e := &rag.Entity{Properties: make(map[string]any)}
		parseFalkorDBProperties(props, e)
		assert.Equal(t, "NYC", e.Properties["city"])
		assert.Equal(t, "USA", e.Properties["country"])
	})
}

func TestExtractStringFromFalkorDBFormat(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "Three element array [id, len, str]",
			input:    []any{int64(1), int64(5), "hello"},
			expected: "hello",
		},
		{
			name:     "Three element array with bytes",
			input:    []any{int64(1), int64(5), []byte("world")},
			expected: "world",
		},
		{
			name:     "Two element array [id, str]",
			input:    []any{int64(1), "test"},
			expected: "test",
		},
		{
			name:     "Two element array with bytes",
			input:    []any{int64(1), []byte("test2")},
			expected: "test2",
		},
		{
			name:     "Direct string",
			input:    "direct",
			expected: "direct",
		},
		{
			name:     "Direct bytes",
			input:    []byte("bytes"),
			expected: "bytes",
		},
		{
			name:     "Empty array",
			input:    []any{},
			expected: "",
		},
		{
			name:     "Single element array",
			input:    []any{int64(1)},
			expected: "",
		},
		{
			name:     "Unsupported type",
			input:    123,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractStringFromFalkorDBFormat(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFalkorDBClose(t *testing.T) {
	t.Run("Close with valid client", func(t *testing.T) {
		fg, err := NewFalkorDBGraph("falkordb://localhost:6379/test")
		assert.NoError(t, err)
		assert.NotNil(t, fg)

		// Type assert to FalkorDBGraph
		graph := fg.(*FalkorDBGraph)
		err = graph.Close()
		// Close might fail if Redis is not running, but should not panic
		assert.NoError(t, err)
	})

	t.Run("Close with nil client", func(t *testing.T) {
		fg := &FalkorDBGraph{client: nil}
		err := fg.Close()
		assert.NoError(t, err)
	})
}

func TestInternalHelpers(t *testing.T) {
	t.Run("quoteString with empty string", func(t *testing.T) {
		assert.Equal(t, "\"\"", quoteString(""))
	})

	t.Run("quoteString with plain string", func(t *testing.T) {
		assert.Equal(t, "\"test\"", quoteString("test"))
	})

	t.Run("quoteString with quoted string", func(t *testing.T) {
		assert.Equal(t, "\"already\"", quoteString("\"already\""))
	})

	t.Run("quoteString with single quotes", func(t *testing.T) {
		result := quoteString("it's")
		assert.Equal(t, "\"it\\'s\"", result)
	})

	t.Run("quoteString with non-string types", func(t *testing.T) {
		assert.Equal(t, 123, quoteString(123))
		assert.Equal(t, true, quoteString(true))
		assert.Equal(t, 3.14, quoteString(3.14))
	})

	t.Run("randomString", func(t *testing.T) {
		rs := randomString(10)
		assert.Len(t, rs, 10)

		// Different calls should produce different strings
		rs2 := randomString(10)
		assert.NotEqual(t, rs, rs2)

		// Verify it's only letters
		for _, c := range rs {
			assert.True(t, (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'))
		}
	})

	t.Run("Node String with all fields", func(t *testing.T) {
		n := &Node{Alias: "a", Label: "Person", Properties: map[string]any{"name": "John", "age": 30}}
		s := n.String()
		assert.Contains(t, s, "a:Person")
		assert.Contains(t, s, "name")
		assert.Contains(t, s, "age")
	})

	t.Run("Node String with only alias", func(t *testing.T) {
		n := &Node{Alias: "x"}
		s := n.String()
		assert.Equal(t, "(x)", s)
	})

	t.Run("Node String with only label", func(t *testing.T) {
		n := &Node{Label: "Type"}
		s := n.String()
		assert.Equal(t, "(:Type)", s)
	})

	t.Run("Edge String", func(t *testing.T) {
		n1 := &Node{Alias: "a"}
		n2 := &Node{Alias: "b"}
		e := &Edge{Source: n1, Destination: n2, Relation: "KNOWS"}
		s := e.String()
		assert.Contains(t, s, "-[:KNOWS]->")
	})

	t.Run("Edge String with properties", func(t *testing.T) {
		n1 := &Node{Alias: "src"}
		n2 := &Node{Alias: "dst"}
		e := &Edge{Source: n1, Destination: n2, Relation: "LIKES", Properties: map[string]any{"weight": 0.5}}
		s := e.String()
		assert.Contains(t, s, "-[:LIKES{") // Properties are inside braces
		assert.Contains(t, s, "weight")
	})

	t.Run("Edge String without relation", func(t *testing.T) {
		n1 := &Node{Alias: "a"}
		n2 := &Node{Alias: "b"}
		e := &Edge{Source: n1, Destination: n2}
		s := e.String()
		assert.Contains(t, s, "-[")
		assert.Contains(t, s, "]->")
	})
}

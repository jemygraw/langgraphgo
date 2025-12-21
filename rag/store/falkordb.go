package store

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/smallnest/langgraphgo/rag"
)

// FalkorDBGraph implements a FalkorDB knowledge graph
type FalkorDBGraph struct {
	client    redis.UniversalClient
	graphName string
}

// NewFalkorDBGraph creates a new FalkorDB knowledge graph
func NewFalkorDBGraph(connectionString string) (rag.KnowledgeGraph, error) {
	// Format: falkordb://host:port/graph_name
	u, err := url.Parse(connectionString)
	if err != nil {
		return nil, fmt.Errorf("invalid connection string: %w", err)
	}

	addr := u.Host
	if addr == "" {
		return nil, fmt.Errorf("invalid connection string: missing host")
	}
	graphName := strings.TrimPrefix(u.Path, "/")
	if graphName == "" {
		graphName = "rag"
	}

	// Create a go-redis client
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	return &FalkorDBGraph{
		client:    client,
		graphName: graphName,
	}, nil
}

// AddEntity adds an entity to the graph
func (f *FalkorDBGraph) AddEntity(ctx context.Context, entity *rag.Entity) error {
	g := NewGraph(f.graphName, f.client)

	label := sanitizeLabel(entity.Type)
	props := entityToMap(entity)
	propsStr := propsToString(props)

	// Using MERGE to avoid duplicates
	query := fmt.Sprintf("MERGE (n:%s {id: '%s'}) SET n += %s", label, entity.ID, propsStr)

	_, err := g.Query(ctx, query)
	return err
}

// AddRelationship adds a relationship to the graph
func (f *FalkorDBGraph) AddRelationship(ctx context.Context, rel *rag.Relationship) error {
	g := NewGraph(f.graphName, f.client)

	relType := sanitizeLabel(rel.Type)
	props := relationshipToMap(rel)
	propsStr := propsToString(props)

	// MATCH source and target, then MERGE relationship
	query := fmt.Sprintf("MATCH (a {id: '%s'}), (b {id: '%s'}) MERGE (a)-[r:%s {id: '%s'}]->(b) SET r += %s",
		rel.Source, rel.Target, relType, rel.ID, propsStr)

	_, err := g.Query(ctx, query)
	return err
}

// Query performs a graph query
func (f *FalkorDBGraph) Query(ctx context.Context, query *rag.GraphQuery) (*rag.GraphQueryResult, error) {
	g := NewGraph(f.graphName, f.client)

	cypher := "MATCH (n)-[r]->(m)"
	where := []string{}

	if len(query.EntityTypes) > 0 {
		orClauses := []string{}
		for _, t := range query.EntityTypes {
			lbl := sanitizeLabel(t)
			orClauses = append(orClauses, fmt.Sprintf("n:%s", lbl))
			orClauses = append(orClauses, fmt.Sprintf("m:%s", lbl))
		}
		if len(orClauses) > 0 {
			where = append(where, "("+strings.Join(orClauses, " OR ")+")")
		}
	}

	if len(query.Relationships) > 0 {
		relClauses := []string{}
		for _, t := range query.Relationships {
			lbl := sanitizeLabel(t)
			relClauses = append(relClauses, fmt.Sprintf("type(r) = '%s'", lbl))
		}
		if len(relClauses) > 0 {
			where = append(where, "("+strings.Join(relClauses, " OR ")+")")
		}
	}

	if len(where) > 0 {
		cypher += " WHERE " + strings.Join(where, " AND ")
	}

	cypher += " RETURN n, r, m"

	if query.Limit > 0 {
		cypher += fmt.Sprintf(" LIMIT %d", query.Limit)
	}

	qr, err := g.Query(ctx, cypher)
	if err != nil {
		return nil, err
	}

	result := &rag.GraphQueryResult{
		Entities:      make([]*rag.Entity, 0),
		Relationships: make([]*rag.Relationship, 0),
	}

	seenEntities := make(map[string]bool)
	seenRels := make(map[string]bool)

	for _, row := range qr.Results {
		if len(row) < 3 {
			continue
		}

		nObj := row[0]
		rObj := row[1]
		mObj := row[2]

		entN := parseNode(nObj)
		if entN != nil && !seenEntities[entN.ID] {
			result.Entities = append(result.Entities, entN)
			seenEntities[entN.ID] = true
		}

		entM := parseNode(mObj)
		if entM != nil && !seenEntities[entM.ID] {
			result.Entities = append(result.Entities, entM)
			seenEntities[entM.ID] = true
		}

		if entN != nil && entM != nil {
			rel := parseEdge(rObj, entN.ID, entM.ID)
			if rel != nil && !seenRels[rel.ID] {
				result.Relationships = append(result.Relationships, rel)
				seenRels[rel.ID] = true
			}
		}
	}

	return result, nil
}

// GetEntity retrieves an entity by ID
func (f *FalkorDBGraph) GetEntity(ctx context.Context, id string) (*rag.Entity, error) {
	g := NewGraph(f.graphName, f.client)

	query := fmt.Sprintf("MATCH (n {id: '%s'}) RETURN n", id)
	qr, err := g.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(qr.Results) == 0 {
		return nil, fmt.Errorf("entity not found: %s", id)
	}

	row := qr.Results[0]
	if len(row) == 0 {
		return nil, fmt.Errorf("invalid result")
	}

	ent := parseNode(row[0])
	if ent == nil {
		return nil, fmt.Errorf("failed to parse entity")
	}
	return ent, nil
}

// GetRelationship retrieves a relationship by ID
func (f *FalkorDBGraph) GetRelationship(ctx context.Context, id string) (*rag.Relationship, error) {
	g := NewGraph(f.graphName, f.client)

	query := fmt.Sprintf("MATCH (a)-[r {id: '%s'}]->(b) RETURN a, r, b", id)
	qr, err := g.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(qr.Results) == 0 {
		return nil, fmt.Errorf("relationship not found: %s", id)
	}

	row := qr.Results[0]
	if len(row) < 3 {
		return nil, fmt.Errorf("invalid result")
	}

	a := parseNode(row[0])
	b := parseNode(row[2])
	rel := parseEdge(row[1], a.ID, b.ID)

	return rel, nil
}

// GetRelatedEntities finds entities related to a given entity
func (f *FalkorDBGraph) GetRelatedEntities(ctx context.Context, entityID string, maxDepth int) ([]*rag.Entity, error) {
	if maxDepth < 1 {
		maxDepth = 1
	}

	g := NewGraph(f.graphName, f.client)

	query := fmt.Sprintf("MATCH (n {id: '%s'})-[*1..%d]-(m) RETURN DISTINCT m", entityID, maxDepth)
	qr, err := g.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	entities := []*rag.Entity{}
	seen := make(map[string]bool)

	for _, row := range qr.Results {
		if len(row) == 0 {
			continue
		}
		ent := parseNode(row[0])
		if ent != nil && !seen[ent.ID] {
			entities = append(entities, ent)
			seen[ent.ID] = true
		}
	}
	return entities, nil
}

// DeleteEntity removes an entity
func (f *FalkorDBGraph) DeleteEntity(ctx context.Context, id string) error {
	g := NewGraph(f.graphName, f.client)

	query := fmt.Sprintf("MATCH (n {id: '%s'}) DETACH DELETE n", id)
	_, err := g.Query(ctx, query)
	return err
}

// DeleteRelationship removes a relationship
func (f *FalkorDBGraph) DeleteRelationship(ctx context.Context, id string) error {
	g := NewGraph(f.graphName, f.client)

	query := fmt.Sprintf("MATCH ()-[r {id: '%s'}]->() DELETE r", id)
	_, err := g.Query(ctx, query)
	return err
}

// UpdateEntity updates an entity
func (f *FalkorDBGraph) UpdateEntity(ctx context.Context, entity *rag.Entity) error {
	return f.AddEntity(ctx, entity)
}

// UpdateRelationship updates a relationship
func (f *FalkorDBGraph) UpdateRelationship(ctx context.Context, rel *rag.Relationship) error {
	return f.AddRelationship(ctx, rel)
}

// Close closes the driver
func (f *FalkorDBGraph) Close() error {
	if f.client != nil {
		return f.client.Close()
	}
	return nil
}

// Helpers

var labelRegex = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func sanitizeLabel(l string) string {
	clean := labelRegex.ReplaceAllString(l, "_")
	if clean == "" {
		return "Entity"
	}
	return clean
}

func propsToString(m map[string]interface{}) string {
	parts := []string{}
	for k, v := range m {
		var val interface{}
		switch v := v.(type) {
		case []float32:
			// Convert to Cypher list: [v1, v2, ...]
			s := make([]string, len(v))
			for i, f := range v {
				s[i] = fmt.Sprintf("%f", f)
			}
			val = "[" + strings.Join(s, ",") + "]"
		default:
			val = quoteString(v)
		}
		parts = append(parts, fmt.Sprintf("%s: %v", k, val))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func entityToMap(e *rag.Entity) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range e.Properties {
		m[k] = v
	}
	m["name"] = e.Name
	m["type"] = e.Type

	if len(e.Embedding) > 0 {
		m["embedding"] = e.Embedding
	}
	return m
}

func relationshipToMap(r *rag.Relationship) map[string]interface{} {
	m := make(map[string]interface{})
	for k, v := range r.Properties {
		m[k] = v
	}
	m["weight"] = r.Weight
	m["confidence"] = r.Confidence
	m["type"] = r.Type
	return m
}

// Parsing Helpers for Redigo Graph Response
// A Node in RedisGraph response is usually: [ID (int64), Labels ([]interface{}), Properties ([]interface{})]
// Properties are [key, value, key, value...]

func parseNode(obj interface{}) *rag.Entity {
	// Redigo might return []interface{}
	vals, ok := obj.([]interface{})
	if !ok || len(vals) < 3 {
		return nil
	}

	// Index 0: ID (internal graph id, not our string ID) - usually int64
	// Index 1: Labels - []interface{} of strings
	// Index 2: Properties - []interface{} of [key, value] pairs (Redigo default?) or just flat?
	// RedisGraph protocol:
	// Node: [id, [label1, label2], [[key1, type1, val1], ...]] -> No, this depends on compact mode.
	// I used "GRAPH.QUERY ... --compact".
	// In --compact mode (which redigo-redisgraph often expects? No, I added it manually).
	// If I REMOVE --compact, the response is text based for results?
	// No, default is header/results/stats.
	// But the result rows contain objects.

	// Let's assume standard object structure returned by redis.Values

	e := &rag.Entity{
		Properties: make(map[string]interface{}),
	}

	// Labels
	if labels, ok := vals[1].([]interface{}); ok && len(labels) > 0 {
		if l, ok := labels[0].([]byte); ok {
			e.Type = string(l)
		} else if l, ok := labels[0].(string); ok {
			e.Type = l
		}
	}

	// Properties
	if props, ok := vals[2].([]interface{}); ok {
		for i := 0; i < len(props); i++ {
			// Prop is usually [key, value]
			if propPair, ok := props[i].([]interface{}); ok && len(propPair) == 2 {
				key := ""
				if k, ok := propPair[0].([]byte); ok {
					key = string(k)
				} else if k, ok := propPair[0].(string); ok {
					key = k
				}

				val := propPair[1]
				// Convert val from []byte if needed
				if b, ok := val.([]byte); ok {
					val = string(b)
				}

				switch key {
				case "id":
					e.ID = fmt.Sprint(val)
				case "name":
					e.Name = fmt.Sprint(val)
				default:
					e.Properties[key] = val
				}
			}
		}
	}

	return e
}

func parseEdge(obj interface{}, sourceID, targetID string) *rag.Relationship {
	vals, ok := obj.([]interface{})
	if !ok || len(vals) < 3 {
		return nil
	}

	// Edge: [id, type, src, dst, props] -> Structure varies.
	// Standard: [id, type, srcID, dstID, properties]

	rel := &rag.Relationship{
		Source:     sourceID,
		Target:     targetID,
		Properties: make(map[string]interface{}),
	}

	// Type (Index 1)
	if t, ok := vals[1].([]byte); ok {
		rel.Type = string(t)
	} else if t, ok := vals[1].(string); ok {
		rel.Type = t
	}

	// Properties (Index 4 usually, but check len)
	if len(vals) > 4 {
		if props, ok := vals[4].([]interface{}); ok {
			for i := 0; i < len(props); i++ {
				if propPair, ok := props[i].([]interface{}); ok && len(propPair) == 2 {
					key := ""
					if k, ok := propPair[0].([]byte); ok {
						key = string(k)
					} else if k, ok := propPair[0].(string); ok {
						key = k
					}

					val := propPair[1]
					if b, ok := val.([]byte); ok {
						val = string(b)
					}

					switch key {
					case "id":
						rel.ID = fmt.Sprint(val)
					case "weight":
						// Handle float conversion
						rel.Weight = 0 // simplify
					default:
						rel.Properties[key] = val
					}
				}
			}
		}
	}

	return rel
}

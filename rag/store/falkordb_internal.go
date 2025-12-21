package store

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
	"github.com/redis/go-redis/v9"
)

func quoteString(i interface{}) interface{} {
	switch x := i.(type) {
	case string:
		if len(x) == 0 {
			return "\"\""
		}
		if x[0] != '"' {
			x = "\"" + x
		}
		if x[len(x)-1] != '"' {
			x += "\""
		}
		return x
	default:
		return i
	}
}

func randomString(n int) string {
	const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	output := make([]byte, n)
	randomness := make([]byte, n)
	_, err := rand.Read(randomness)
	if err != nil {
		panic(err)
	}
	l := len(letterBytes)
	for pos := range output {
		random := uint8(randomness[pos])
		randomPos := random % uint8(l)
		output[pos] = letterBytes[randomPos]
	}
	return string(output)
}

// Node represents a node within a graph.
type Node struct {
	ID         string
	Alias      string
	Label      string
	Properties map[string]interface{}
}

func (n *Node) String() string {
	s := "("
	if n.Alias != "" {
		s += n.Alias
	}
	if n.Label != "" {
		s += ":" + n.Label
	}
	if len(n.Properties) > 0 {
		p := ""
		for k, v := range n.Properties {
			p += fmt.Sprintf("%s:%v,", k, quoteString(v))
		}
		p = p[:len(p)-1]
		s += "{" + p + "}"
	}
	s += ")"
	return s
}

// Edge represents an edge connecting two nodes in the graph.
type Edge struct {
	Source      *Node
	Destination *Node
	Relation    string
	Properties  map[string]interface{}
}

func (e *Edge) String() string {
	s := "(" + e.Source.Alias + ")"
	s += "-["
	if e.Relation != "" {
		s += ":" + e.Relation
	}
	if len(e.Properties) > 0 {
		p := ""
		for k, v := range e.Properties {
			p += fmt.Sprintf("%s:%s,", k, quoteString(v))
		}
		p = p[:len(p)-1]
		s += "{" + p + "}"
	}
	s += "]->"
	s += "(" + e.Destination.Alias + ")"
	return s
}

// Graph represents a graph, which is a collection of nodes and edges.
type Graph struct {
	Name  string
	Nodes map[string]*Node
	Edges []*Edge
	Conn  redis.UniversalClient
}

// NewGraph creates a new graph (helper constructor).
func NewGraph(name string, conn redis.UniversalClient) Graph {
	return Graph{
		Name:  name,
		Nodes: make(map[string]*Node),
		Conn:  conn,
	}
}

// AddNode adds a node to the graph structure (for Commit usage).
func (g *Graph) AddNode(n *Node) error {
	if n.Alias == "" {
		n.Alias = randomString(10)
	}
	g.Nodes[n.Alias] = n
	return nil
}

// AddEdge adds an edge to the graph structure (for Commit usage).
func (g *Graph) AddEdge(e *Edge) error {
	if e.Source == nil || e.Destination == nil {
		return fmt.Errorf("AddEdge: both source and destination nodes should be defined")
	}
	if _, ok := g.Nodes[e.Source.Alias]; !ok {
		return fmt.Errorf("AddEdge: source node neeeds to be added to the graph first")
	}
	if _, ok := g.Nodes[e.Destination.Alias]; !ok {
		return fmt.Errorf("AddEdge: destination node neeeds to be added to the graph first")
	}
	g.Edges = append(g.Edges, e)
	return nil
}

// Commit creates the entire graph (using CREATE).
func (g *Graph) Commit(ctx context.Context) (QueryResult, error) {
	q := "CREATE "
	for _, n := range g.Nodes {
		q += fmt.Sprintf("%s,", n)
	}
	for _, e := range g.Edges {
		q += fmt.Sprintf("%s,", e)
	}
	q = q[:len(q)-1]
	return g.Query(ctx, q)
}

// QueryResult represents the results of a query.
type QueryResult struct {
	Header     []string
	Results    [][]interface{}
	Statistics []string
}

// Query executes a query against the graph.
func (g *Graph) Query(ctx context.Context, q string) (QueryResult, error) {
	qr := QueryResult{}

	// go-redis Do returns a Cmd which can be used to get the result
	res, err := g.Conn.Do(ctx, "GRAPH.QUERY", g.Name, q, "--compact").Result()
	if err != nil {
		return qr, err
	}

	r, ok := res.([]interface{})
	if !ok {
		return qr, fmt.Errorf("unexpected response type: %T", res)
	}

	if len(r) == 3 {
		// Header
		if header, ok := r[0].([]interface{}); ok {
			qr.Header = make([]string, len(header))
			for i, h := range header {
				qr.Header[i] = fmt.Sprint(h)
			}
		}

		// Results
		if rows, ok := r[1].([]interface{}); ok {
			qr.Results = make([][]interface{}, len(rows))
			for i, row := range rows {
				if rVals, ok := row.([]interface{}); ok {
					qr.Results[i] = rVals
				}
			}
		}

		// Stats
		if stats, ok := r[2].([]interface{}); ok {
			qr.Statistics = make([]string, len(stats))
			for i, s := range stats {
				qr.Statistics[i] = fmt.Sprint(s)
			}
		}

	} else if len(r) == 2 {
		// Results
		if rows, ok := r[0].([]interface{}); ok {
			qr.Results = make([][]interface{}, len(rows))
			for i, row := range rows {
				if rVals, ok := row.([]interface{}); ok {
					qr.Results[i] = rVals
				}
			}
		}

		// Stats
		if stats, ok := r[1].([]interface{}); ok {
			qr.Statistics = make([]string, len(stats))
			for i, s := range stats {
				qr.Statistics[i] = fmt.Sprint(s)
			}
		}
	} else {
		return qr, fmt.Errorf("unexpected response length: %d", len(r))
	}

	return qr, nil
}

func (g *Graph) Delete(ctx context.Context) error {
	return g.Conn.Do(ctx, "GRAPH.DELETE", g.Name).Err()
}

func (qr *QueryResult) PrettyPrint() {
	if len(qr.Results) > 0 {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAutoFormatHeaders(false)
		if len(qr.Header) > 0 {
			table.SetHeader(qr.Header)
		}

		for _, row := range qr.Results {
			sRow := make([]string, len(row))
			for i, v := range row {
				sRow[i] = fmt.Sprint(v)
			}
			table.Append(sRow)
		}
		table.Render()
	}

	for _, stat := range qr.Statistics {
		fmt.Fprintf(os.Stdout, "\n%s", stat)
	}
	fmt.Fprintf(os.Stdout, "\n")
}

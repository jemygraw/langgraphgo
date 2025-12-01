package graph

import "fmt"

// NodeInterrupt is returned when a node requests an interrupt (e.g. waiting for human input).
type NodeInterrupt struct {
	// Node is the name of the node that triggered the interrupt
	Node string
	// Value is the data/query provided by the interrupt
	Value interface{}
}

func (e *NodeInterrupt) Error() string {
	return fmt.Sprintf("interrupt at node %s: %v", e.Node, e.Value)
}

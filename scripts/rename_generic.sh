#!/bin/bash
# Comprehensive rename script for generic implementation
# This script performs the full breaking change rename

set -e

# Get the project root directory
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "=== Starting Generic Rename ==="
echo "Project root: $PROJECT_ROOT"

# Step 1: Rename the original untyped types to Untyped versions
echo "Step 1: Renaming untyped types..."

# First, rename state_graph.go types
sed -i '' \
  -e 's/type StateGraph struct/type StateGraphUntyped struct/g' \
  -e 's/type StateRunnable struct/type StateRunnableUntyped struct/g' \
  -e 's/func NewStateGraph()/func NewStateGraphUntyped()/g' \
  graph/state_graph.go

# Step 2: Rename StateGraphTyped to StateGraph in state_graph_typed.go
echo "Step 2: Renaming StateGraphTyped to StateGraph..."

sed -i '' \
  -e 's/type StateGraphTyped\[S any\] struct/type StateGraph[S any] struct/g' \
  -e 's/type NodeTyped\[S any\] struct/type TypedNode[S any] struct/g' \
  -e 's/type StateMergerTyped\[S any\]/type StateMerger[S any]/g' \
  -e 's/type StateRunnableTyped\[S any\] struct/type StateRunnable[S any] struct/g' \
  graph/state_graph_typed.go

# Fix the generic Node references - use TypedNode to avoid conflict with untyped Node
sed -i '' \
  -e 's/nodes map\[string\]Node\[S\]/nodes map[string]TypedNode[S]/g' \
  -e 's/Node\[S\]{/TypedNode[S]{/g' \
  -e 's/StateMerger\[S\] func/StateMerger[S] func/g' \
  graph/state_graph_typed.go

# Fix method receivers and function calls
sed -i '' \
  -e 's/func (g \*StateGraphTyped\[S\])/func (g *StateGraph[S])/g' \
  -e 's/func (r \*StateRunnableTyped\[S\])/func (r *StateRunnable[S])/g' \
  -e 's/NewStateGraphTyped/NewStateGraph/g' \
  -e 's/\*StateGraphTyped\[S\]/*StateGraph[S]/g' \
  -e 's/\*StateRunnableTyped\[S\]/*StateRunnable[S]/g' \
  -e 's/StateRunnableTyped\[S\]/StateRunnable[S]/g' \
  graph/state_graph_typed.go

# Step 3: Update listeners_typed.go
echo "Step 3: Updating listeners_typed.go..."
sed -i '' \
  -e 's/NodeTyped/TypedNode/g' \
  -e 's/StateGraphTyped/StateGraph/g' \
  -e 's/StateRunnableTyped/StateRunnable/g' \
  graph/listeners_typed.go

# Step 4: Update all prebuilt generic files
echo "Step 4: Updating prebuilt generic files..."
for file in prebuilt/*_generic.go prebuilt/react_agent_typed.go prebuilt/supervisor_typed.go; do
  if [ -f "$file" ]; then
    sed -i '' \
      -e 's/StateGraphTyped/StateGraph/g' \
      -e 's/StateRunnableTyped/StateRunnable/g' \
      -e 's/NewStateGraphTyped/NewStateGraph/g' \
      "$file"
  fi
done

# Step 5: Update examples
echo "Step 5: Updating examples..."
for file in examples/generic_state_graph*/main.go examples/generic_state_graph*/listenable_example.go; do
  if [ -f "$file" ]; then
    sed -i '' \
      -e 's/StateGraphTyped/StateGraph/g' \
      -e 's/StateRunnableTyped/StateRunnable/g' \
      -e 's/NewStateGraphTyped/NewStateGraph/g' \
      "$file"
  fi
done

# Step 6: Update doc.go files
echo "Step 6: updating documentation..."
sed -i '' \
  -e 's/StateGraphTyped/StateGraph/g' \
  -e 's/StateRunnableTyped/StateRunnable/g' \
  doc.go graph/doc.go

echo "=== Rename Complete ==="
echo "Please run 'go build ./...' to verify the changes"

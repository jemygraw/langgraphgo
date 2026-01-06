package graph

import (
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// SchemaTestState is used for schema tests
type SchemaTestState struct {
	Count   int
	Name    string
	Numbers []int
	Logs    []string
}

// StructSchema Tests

func TestNewStructSchema(t *testing.T) {
	t.Run("Create schema with merge function", func(t *testing.T) {
		mergeFunc := func(current, new SchemaTestState) (SchemaTestState, error) {
			current.Count += new.Count
			return current, nil
		}
		schema := NewStructSchema(SchemaTestState{Count: 0}, mergeFunc)
		assert.NotNil(t, schema)
		assert.Equal(t, 0, schema.InitialValue.Count)
		assert.NotNil(t, schema.MergeFunc)
	})

	t.Run("Create schema with default merge", func(t *testing.T) {
		schema := NewStructSchema(SchemaTestState{Count: 5}, nil)
		assert.NotNil(t, schema)
		assert.Equal(t, 5, schema.InitialValue.Count)
		assert.NotNil(t, schema.MergeFunc)
	})
}

func TestStructSchema_Init(t *testing.T) {
	schema := NewStructSchema(SchemaTestState{Count: 10, Name: "test"}, nil)
	initial := schema.Init()
	assert.Equal(t, 10, initial.Count)
	assert.Equal(t, "test", initial.Name)
}

func TestStructSchema_Update(t *testing.T) {
	t.Run("Update with custom merge function", func(t *testing.T) {
		mergeFunc := func(current, new SchemaTestState) (SchemaTestState, error) {
			current.Count += new.Count
			if new.Name != "" {
				current.Name = new.Name
			}
			return current, nil
		}
		schema := NewStructSchema(SchemaTestState{}, mergeFunc)

		current := SchemaTestState{Count: 5, Name: "old"}
		new := SchemaTestState{Count: 3, Name: "new"}

		result, err := schema.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 8, result.Count)
		assert.Equal(t, "new", result.Name)
	})

	t.Run("Update with nil merge function uses default", func(t *testing.T) {
		schema := &StructSchema[SchemaTestState]{
			InitialValue: SchemaTestState{},
			MergeFunc:    nil,
		}

		current := SchemaTestState{Count: 5, Name: "old"}
		new := SchemaTestState{Count: 0, Name: "new"} // Count is zero, won't update

		result, err := schema.Update(current, new)
		assert.NoError(t, err)
		// When MergeFunc is nil, Update returns the new value directly
		assert.Equal(t, 0, result.Count)    // New value (0)
		assert.Equal(t, "new", result.Name) // New value
	})
}

func TestDefaultStructMerge(t *testing.T) {
	t.Run("Merge non-zero fields", func(t *testing.T) {
		current := SchemaTestState{Count: 5, Name: "old"}
		new := SchemaTestState{Count: 10, Name: ""} // Name is zero

		result, err := DefaultStructMerge(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 10, result.Count)
		assert.Equal(t, "old", result.Name) // Keep old name
	})

	t.Run("Merge with slices", func(t *testing.T) {
		current := SchemaTestState{Numbers: []int{1, 2}}
		new := SchemaTestState{Numbers: []int{3, 4}}

		result, err := DefaultStructMerge(current, new)
		assert.NoError(t, err)
		assert.Equal(t, []int{3, 4}, result.Numbers) // Overwrites
	})

	t.Run("Merge with zero values", func(t *testing.T) {
		current := SchemaTestState{Count: 5, Name: "test"}
		new := SchemaTestState{Count: 0, Name: ""}

		result, err := DefaultStructMerge(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 5, result.Count)
		assert.Equal(t, "test", result.Name)
	})

	t.Run("Non-struct type returns new", func(t *testing.T) {
		current := 5
		new := 10

		result, err := DefaultStructMerge(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 10, result)
	})
}

func TestOverwriteStructMerge(t *testing.T) {
	current := SchemaTestState{Count: 5, Name: "old", Numbers: []int{1}}
	new := SchemaTestState{Count: 10, Name: "new", Numbers: []int{2, 3}}

	result, err := OverwriteStructMerge(current, new)
	assert.NoError(t, err)
	assert.Equal(t, new, result)
	assert.Equal(t, 10, result.Count)
	assert.Equal(t, "new", result.Name)
	assert.Equal(t, []int{2, 3}, result.Numbers)
}

// FieldMerger Tests

func TestNewFieldMerger(t *testing.T) {
	fm := NewFieldMerger(SchemaTestState{Count: 0})
	assert.NotNil(t, fm)
	assert.Equal(t, 0, fm.InitialValue.Count)
	assert.NotNil(t, fm.FieldMergeFns)
}

func TestFieldMerger_Init(t *testing.T) {
	fm := NewFieldMerger(SchemaTestState{Count: 10, Name: "test"})
	initial := fm.Init()
	assert.Equal(t, 10, initial.Count)
	assert.Equal(t, "test", initial.Name)
}

func TestFieldMerger_RegisterFieldMerge(t *testing.T) {
	fm := NewFieldMerger(SchemaTestState{})
	fm.RegisterFieldMerge("Count", SumIntMerge)
	fm.RegisterFieldMerge("Name", OverwriteMerge)

	assert.Contains(t, fm.FieldMergeFns, "Count")
	assert.Contains(t, fm.FieldMergeFns, "Name")
}

func TestFieldMerger_Update(t *testing.T) {
	t.Run("Update with custom field mergers", func(t *testing.T) {
		fm := NewFieldMerger(SchemaTestState{})
		fm.RegisterFieldMerge("Count", SumIntMerge)
		fm.RegisterFieldMerge("Name", KeepCurrentMerge) // Keep current name

		current := SchemaTestState{Count: 5, Name: "original"}
		new := SchemaTestState{Count: 3, Name: "new"}

		result, err := fm.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 8, result.Count)         // Sum
		assert.Equal(t, "original", result.Name) // Keep current
	})

	t.Run("Update with default behavior for unregistered fields", func(t *testing.T) {
		fm := NewFieldMerger(SchemaTestState{})
		// No field mergers registered

		current := SchemaTestState{Count: 5, Name: "old"}
		new := SchemaTestState{Count: 0, Name: "new"} // Count is zero

		result, err := fm.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 5, result.Count) // Keep old (zero doesn't overwrite)
		assert.Equal(t, "new", result.Name)
	})

	t.Run("Update with slice merge", func(t *testing.T) {
		fm := NewFieldMerger(SchemaTestState{})
		fm.RegisterFieldMerge("Numbers", AppendSliceMerge)
		fm.RegisterFieldMerge("Logs", AppendSliceMerge)

		current := SchemaTestState{
			Numbers: []int{1, 2},
			Logs:    []string{"a"},
		}
		new := SchemaTestState{
			Numbers: []int{3, 4},
			Logs:    []string{"b"},
		}

		result, err := fm.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3, 4}, result.Numbers)
		assert.Equal(t, []string{"a", "b"}, result.Logs)
	})

	t.Run("Update non-struct returns error", func(t *testing.T) {
		fm := NewFieldMerger(0) // Not a struct

		current := 5
		newVal := 10

		_, err := fm.Update(current, newVal)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only works with struct")
	})
}

// Merge Helper Tests

func TestAppendSliceMerge(t *testing.T) {
	t.Run("Append slices", func(t *testing.T) {
		current := []int{1, 2}
		new := []int{3, 4}

		result := AppendSliceMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.True(t, result.IsValid())
		resultSlice := result.Interface().([]int)
		assert.Equal(t, []int{1, 2, 3, 4}, resultSlice)
	})

	t.Run("Non-slice values return new", func(t *testing.T) {
		current := 5
		new := 10

		result := AppendSliceMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})
}

func TestSumIntMerge(t *testing.T) {
	t.Run("Sum integers", func(t *testing.T) {
		current := 5
		new := 3

		result := SumIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.True(t, result.IsValid())
		assert.Equal(t, int64(8), result.Int())
	})

	t.Run("Non-int values return new", func(t *testing.T) {
		current := "hello"
		new := "world"

		result := SumIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})
}

func TestOverwriteMerge(t *testing.T) {
	current := "old"
	new := "new"

	result := OverwriteMerge(
		reflect.ValueOf(current),
		reflect.ValueOf(new),
	)

	assert.Equal(t, reflect.ValueOf(new), result)
}

func TestKeepCurrentMerge(t *testing.T) {
	current := "old"
	new := "new"

	result := KeepCurrentMerge(
		reflect.ValueOf(current),
		reflect.ValueOf(new),
	)

	assert.Equal(t, reflect.ValueOf(current), result)
}

func TestMaxIntMerge(t *testing.T) {
	t.Run("Max of two integers - current larger", func(t *testing.T) {
		current := 10
		new := 5

		result := MaxIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(current), result)
	})

	t.Run("Max of two integers - new larger", func(t *testing.T) {
		current := 5
		new := 10

		result := MaxIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})

	t.Run("Non-int values return new", func(t *testing.T) {
		current := "hello"
		new := "world"

		result := MaxIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})
}

func TestMinIntMerge(t *testing.T) {
	t.Run("Min of two integers - current smaller", func(t *testing.T) {
		current := 3
		new := 7

		result := MinIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(current), result)
	})

	t.Run("Min of two integers - new smaller", func(t *testing.T) {
		current := 7
		new := 3

		result := MinIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})

	t.Run("Non-int values return new", func(t *testing.T) {
		current := "hello"
		new := "world"

		result := MinIntMerge(
			reflect.ValueOf(current),
			reflect.ValueOf(new),
		)

		assert.Equal(t, reflect.ValueOf(new), result)
	})
}

// MapSchema Tests

func TestNewMapSchema(t *testing.T) {
	schema := NewMapSchema()
	assert.NotNil(t, schema)
	assert.NotNil(t, schema.Reducers)
}

func TestMapSchema_RegisterReducer(t *testing.T) {
	schema := NewMapSchema()
	schema.RegisterReducer("key1", OverwriteReducer)
	schema.RegisterReducer("key2", AppendReducer)

	assert.Contains(t, schema.Reducers, "key1")
	assert.Contains(t, schema.Reducers, "key2")
}

func TestMapSchema_Init(t *testing.T) {
	schema := NewMapSchema()
	initial := schema.Init()
	assert.NotNil(t, initial)
	assert.NotNil(t, initial)
	assert.Empty(t, initial) // Empty map
}

func TestMapSchema_Update(t *testing.T) {
	t.Run("Update with nil current creates new map", func(t *testing.T) {
		schema := NewMapSchema()
		new := map[string]any{"key": "value"}

		result, err := schema.Update(nil, new)
		assert.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("Update with reducer", func(t *testing.T) {
		schema := NewMapSchema()
		schema.RegisterReducer("count", func(current, new any) (any, error) {
			currInt := current.(int)
			newInt := new.(int)
			return currInt + newInt, nil
		})

		current := map[string]any{"count": 5}
		new := map[string]any{"count": 3}

		result, err := schema.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, 8, result["count"])
	})

	t.Run("Update without reducer overwrites", func(t *testing.T) {
		schema := NewMapSchema()
		// No reducer for "name"

		current := map[string]any{"name": "old"}
		new := map[string]any{"name": "new"}

		result, err := schema.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, "new", result["name"])
	})

	t.Run("Update doesn't mutate original", func(t *testing.T) {
		schema := NewMapSchema()
		current := map[string]any{"key": "original"}
		new := map[string]any{"key": "updated"}

		result, err := schema.Update(current, new)
		assert.NoError(t, err)
		assert.Equal(t, "original", current["key"]) // Original unchanged
		assert.Equal(t, "updated", result["key"])
	})

	t.Run("Reducer error propagates", func(t *testing.T) {
		schema := NewMapSchema()
		schema.RegisterReducer("key", func(current, new any) (any, error) {
			return nil, assert.AnError
		})

		_, err := schema.Update(map[string]any{}, map[string]any{"key": "value"})
		assert.Error(t, err)
	})
}

// Reducer Tests

func TestOverwriteReducer(t *testing.T) {
	current := "old"
	new := "new"

	result, err := OverwriteReducer(current, new)
	assert.NoError(t, err)
	assert.Equal(t, "new", result)
}

func TestAppendReducer(t *testing.T) {
	t.Run("Append slice to slice", func(t *testing.T) {
		current := []int{1, 2}
		new := []int{3, 4}

		result, err := AppendReducer(current, new)
		assert.NoError(t, err)
		resultSlice := result.([]int)
		assert.Equal(t, []int{1, 2, 3, 4}, resultSlice)
	})

	t.Run("Append single element to slice", func(t *testing.T) {
		current := []int{1, 2}
		new := 3

		result, err := AppendReducer(current, new)
		assert.NoError(t, err)
		resultSlice := result.([]int)
		assert.Equal(t, []int{1, 2, 3}, resultSlice)
	})

	t.Run("Start new slice from single element", func(t *testing.T) {
		var current []int = nil
		new := 42

		result, err := AppendReducer(current, new)
		assert.NoError(t, err)
		resultSlice := result.([]int)
		assert.Equal(t, []int{42}, resultSlice)
	})

	t.Run("Start new slice from slice", func(t *testing.T) {
		var current []int = nil
		new := []int{1, 2, 3}

		result, err := AppendReducer(current, new)
		assert.NoError(t, err)
		resultSlice := result.([]int)
		assert.Equal(t, []int{1, 2, 3}, resultSlice)
	})

	t.Run("Append slice to slice with different types", func(t *testing.T) {
		current := []string{"a", "b"}
		new := []int{1, 2}

		result, err := AppendReducer(current, new)
		assert.NoError(t, err)
		resultSlice := result.([]any)
		assert.Equal(t, []any{"a", "b", 1, 2}, resultSlice)
	})

	t.Run("Non-slice current returns error", func(t *testing.T) {
		current := "not a slice"
		new := []int{1}

		_, err := AppendReducer(current, new)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a slice")
	})
}

// Integration Tests

func TestStateGraph_Schema(t *testing.T) {
	g := NewStateGraph[map[string]any]()

	schema := NewMapSchema()
	schema.RegisterReducer("messages", AppendReducer)
	g.SetSchema(schema)

	g.AddNode("A", "A", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{
			"messages": []string{"A"},
		}, nil
	})

	g.AddNode("B", "B", func(ctx context.Context, state map[string]any) (map[string]any, error) {
		return map[string]any{
			"messages": []string{"B"},
		}, nil
	})

	g.SetEntryPoint("A")
	g.AddEdge("A", "B")
	g.AddEdge("B", END)

	runnable, err := g.Compile()
	assert.NoError(t, err)

	initialState := map[string]any{
		"messages": []string{"start"},
	}

	result, err := runnable.Invoke(context.Background(), initialState)
	assert.NoError(t, err)

	assert.Equal(t, []string{"start", "A", "B"}, result["messages"])
}

func TestMapSchema_Update_Integration(t *testing.T) {
	schema := NewMapSchema()
	schema.RegisterReducer("messages", AppendReducer)

	initialState := map[string]any{
		"messages": []string{"hello"},
		"count":    1,
	}

	// Update 1: Append message
	update1 := map[string]any{
		"messages": []string{"world"},
	}

	newState1, err := schema.Update(initialState, update1)
	assert.NoError(t, err)

	state1 := newState1
	assert.Equal(t, []string{"hello", "world"}, state1["messages"])
	assert.Equal(t, 1, state1["count"])

	// Update 2: Overwrite count
	update2 := map[string]any{
		"count": 2,
	}

	newState2, err := schema.Update(state1, update2)
	assert.NoError(t, err)

	state2 := newState2
	assert.Equal(t, []string{"hello", "world"}, state2["messages"])
	assert.Equal(t, 2, state2["count"])

	// Update 3: Append single element (if supported by AppendReducer logic, currently it supports slice or element)
	// Let's test appending a single string
	update3 := map[string]any{
		"messages": "!",
	}

	newState3, err := schema.Update(state2, update3)
	assert.NoError(t, err)

	state3 := newState3
	assert.Equal(t, []string{"hello", "world", "!"}, state3["messages"])
}

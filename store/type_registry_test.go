package store

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test types for type registry
type TestState struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type AnotherState struct {
	ID    int    `json:"id"`
	Value string `json:"value"`
}

type PointerState struct {
	Field1 string
	Field2 int
}

type CustomState struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// newTestRegistry creates a new isolated registry for testing
func newTestRegistry() *TypeRegistry {
	return &TypeRegistry{
		typeNameToType:    make(map[string]reflect.Type),
		typeToName:        make(map[reflect.Type]string),
		typeCreators:      make(map[string]func() any),
		jsonMarshallers:   make(map[reflect.Type]func(any) ([]byte, error)),
		jsonUnmarshallers: make(map[reflect.Type]func([]byte, any) (any, error)),
	}
}

func TestTypeRegistry_RegisterType(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Register struct type", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[TestState](), "TestState")
		assert.NoError(t, err)

		// Verify we can retrieve it
		typ, ok := registry.GetTypeByName("TestState")
		assert.True(t, ok)
		assert.Equal(t, "TestState", typ.Name())
	})

	t.Run("Register pointer to struct", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[*PointerState](), "PointerState")
		assert.NoError(t, err)

		// Verify we can retrieve it
		typ, ok := registry.GetTypeByName("PointerState")
		assert.True(t, ok)
		assert.True(t, typ.Kind() == reflect.Ptr)
	})

	t.Run("Register non-struct type should fail", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[string](), "StringType")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a struct")
	})

	t.Run("Register pointer to non-struct should fail", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[*int](), "IntPtr")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be a struct")
	})

	t.Run("Register same type with different name should fail", func(t *testing.T) {
		// Use a fresh registry for this test
		registry2 := newTestRegistry()
		err1 := registry2.RegisterTypeInternal(reflect.TypeFor[TestState](), "TestState_First")
		assert.NoError(t, err1)

		err2 := registry2.RegisterTypeInternal(reflect.TypeFor[TestState](), "TestState_Second")
		assert.Error(t, err2)
		assert.Contains(t, err2.Error(), "already registered")
	})

	t.Run("Register same type with same name should succeed", func(t *testing.T) {
		err1 := registry.RegisterTypeInternal(reflect.TypeFor[AnotherState](), "AnotherState")
		assert.NoError(t, err1)

		err2 := registry.RegisterTypeInternal(reflect.TypeFor[AnotherState](), "AnotherState")
		assert.NoError(t, err2)
	})
}

func TestTypeRegistry_GetTypeName(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Get existing type name", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState1")

		name, ok := registry.GetTypeName(typ)
		assert.True(t, ok)
		assert.Equal(t, "TestState1", name)
	})

	t.Run("Get non-existing type name", func(t *testing.T) {
		typ := reflect.TypeOf(struct{ Field int }{})
		_, ok := registry.GetTypeName(typ)
		assert.False(t, ok)
	})
}

func TestTypeRegistry_GetTypeByName(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Get existing type", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState2")

		retrievedType, ok := registry.GetTypeByName("TestState2")
		assert.True(t, ok)
		assert.Equal(t, reflect.TypeFor[TestState](), retrievedType)
	})

	t.Run("Get non-existing type", func(t *testing.T) {
		_, ok := registry.GetTypeByName("NonExistentType")
		assert.False(t, ok)
	})
}

func TestTypeRegistry_CreateInstance(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Create instance of registered type", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState3")

		instance, err := registry.CreateInstance("TestState3")
		assert.NoError(t, err)
		assert.NotNil(t, instance)

		// Verify it's the correct type
		_, ok := instance.(TestState)
		assert.True(t, ok)
	})

	t.Run("Create instance of non-registered type", func(t *testing.T) {
		_, err := registry.CreateInstance("NonExistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestTypeRegistry_Marshal(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Marshal registered type", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState4")

		state := TestState{Name: "test", Count: 42}
		data, err := registry.Marshal(state)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify wrapped structure
		var wrapped map[string]json.RawMessage
		err = json.Unmarshal(data, &wrapped)
		assert.NoError(t, err)
		assert.Contains(t, wrapped, "_type")
		assert.Contains(t, wrapped, "_value")

		var typeName string
		json.Unmarshal(wrapped["_type"], &typeName)
		assert.Equal(t, "TestState4", typeName)
	})

	t.Run("Marshal unregistered type", func(t *testing.T) {
		state := struct{ Field int }{Field: 123}
		data, err := registry.Marshal(state)
		assert.NoError(t, err)

		// Should be plain JSON without type wrapper
		var result map[string]any
		err = json.Unmarshal(data, &result)
		assert.NoError(t, err)
		assert.Equal(t, 123.0, result["Field"])
		assert.NotContains(t, result, "_type")
	})

	t.Run("Marshal nil", func(t *testing.T) {
		data, err := registry.Marshal(nil)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		var result any
		err = json.Unmarshal(data, &result)
		assert.NoError(t, err)
		assert.Nil(t, result)
	})
}

func TestTypeRegistry_Unmarshal(t *testing.T) {
	t.Run("Unmarshal registered type", func(t *testing.T) {
		registry := newTestRegistry()
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState5")

		// Create wrapped data
		wrapped := map[string]any{
			"_type": "TestState5",
			"_value": map[string]any{
				"name":  "test",
				"count": 99,
			},
		}
		jsonData, _ := json.Marshal(wrapped)

		result, err := registry.Unmarshal(jsonData)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify type and content
		state, ok := result.(TestState)
		assert.True(t, ok)
		assert.Equal(t, "test", state.Name)
		assert.Equal(t, 99, state.Count)
	})

	t.Run("Unmarshal unknown type", func(t *testing.T) {
		registry := newTestRegistry()
		wrapped := map[string]any{
			"_type":  "UnknownType",
			"_value": map[string]any{},
		}
		jsonData, _ := json.Marshal(wrapped)

		_, err := registry.Unmarshal(jsonData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown type")
	})

	t.Run("Unmarshal non-wrapped data", func(t *testing.T) {
		registry := newTestRegistry()
		data := []byte(`{"field": "value"}`)
		result, err := registry.Unmarshal(data)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		var resultMap map[string]any
		err = json.Unmarshal(data, &resultMap)
		assert.NoError(t, err)
		assert.Equal(t, "value", resultMap["field"])
	})

	t.Run("Unmarshal wrapped data with missing _value", func(t *testing.T) {
		registry := newTestRegistry()
		// Register the type first so we can test missing _value
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState6")

		wrapped := map[string]any{
			"_type": "TestState6",
		}
		jsonData, _ := json.Marshal(wrapped)

		_, err := registry.Unmarshal(jsonData)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing _value")
	})
}

func TestTypeRegistry_CustomSerialization(t *testing.T) {
	registry := newTestRegistry()

	t.Run("Register with custom serialization", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()

		marshalFunc := func(v any) ([]byte, error) {
			state := v.(TestState)
			custom := map[string]any{
				"custom_name":  state.Name,
				"custom_count": state.Count,
			}
			return json.Marshal(custom)
		}

		unmarshalFunc := func(data []byte) (any, error) {
			var custom map[string]any
			if err := json.Unmarshal(data, &custom); err != nil {
				return nil, err
			}
			return TestState{
				Name:  custom["custom_name"].(string),
				Count: int(custom["custom_count"].(float64)),
			}, nil
		}

		err := registry.RegisterTypeWithCustomSerializationInternal(typ, "CustomState", marshalFunc, unmarshalFunc)
		assert.NoError(t, err)

		// Test marshal
		state := TestState{Name: "custom", Count: 100}
		data, err := registry.Marshal(state)
		assert.NoError(t, err)

		var wrapped map[string]json.RawMessage
		json.Unmarshal(data, &wrapped)

		var value map[string]any
		json.Unmarshal(wrapped["_value"], &value)
		assert.Equal(t, "custom", value["custom_name"])
		assert.Equal(t, 100.0, value["custom_count"])

		// Test unmarshal
		result, err := registry.Unmarshal(data)
		assert.NoError(t, err)
		resultState, ok := result.(TestState)
		assert.True(t, ok)
		assert.Equal(t, "custom", resultState.Name)
		assert.Equal(t, 100, resultState.Count)
	})
}

func TestTypeRegistry_StateMarshalerUnmarshaler(t *testing.T) {
	t.Run("StateMarshaler creates correct function", func(t *testing.T) {
		// Use a fresh registry
		registry := newTestRegistry()

		// First register the type
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState7")

		// Then create the marshaler
		marshaler := registry.StateMarshaler()
		assert.NotNil(t, marshaler)

		state := TestState{Name: "marshaler", Count: 1}
		data, err := marshaler(state)
		assert.NoError(t, err)
		assert.NotNil(t, data)

		// Verify wrapped structure
		var wrapped map[string]json.RawMessage
		err = json.Unmarshal(data, &wrapped)
		assert.NoError(t, err)
		assert.Contains(t, wrapped, "_type")
		assert.Contains(t, wrapped, "_value")

		var typeName string
		json.Unmarshal(wrapped["_type"], &typeName)
		assert.Equal(t, "TestState7", typeName)
	})

	t.Run("StateUnmarshaler creates correct function", func(t *testing.T) {
		// Use a fresh registry
		registry := newTestRegistry()

		// First register the type
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState8")

		// Then create the unmarshaler
		unmarshaler := registry.StateUnmarshaler()
		assert.NotNil(t, unmarshaler)

		wrapped := map[string]any{
			"_type":  "TestState8",
			"_value": map[string]any{"name": "unmarshaler", "count": float64(2)},
		}
		jsonData, _ := json.Marshal(wrapped)

		result, err := unmarshaler(jsonData)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultState, ok := result.(TestState)
		assert.True(t, ok)
		assert.Equal(t, "unmarshaler", resultState.Name)
		assert.Equal(t, 2, resultState.Count)
	})
}

func TestCheckpointData(t *testing.T) {
	// Use global registry for CheckpointData tests since that's what it uses
	registry := GlobalTypeRegistry()

	t.Run("NewCheckpointData with registered type", func(t *testing.T) {
		typ := reflect.TypeFor[TestState]()
		registry.RegisterTypeInternal(typ, "TestState9")

		state := TestState{Name: "checkpoint", Count: 50}
		cd, err := NewCheckpointData(state)
		assert.NoError(t, err)
		assert.NotNil(t, cd)
		assert.Equal(t, "TestState9", cd.TypeName)
		assert.NotNil(t, cd.Data)

		// Verify we can convert back
		value, err := cd.ToValue()
		assert.NoError(t, err)
		resultState, ok := value.(TestState)
		assert.True(t, ok)
		assert.Equal(t, "checkpoint", resultState.Name)
		assert.Equal(t, 50, resultState.Count)
	})

	t.Run("NewCheckpointData with unregistered type", func(t *testing.T) {
		state := struct{ Field string }{Field: "test"}
		cd, err := NewCheckpointData(state)
		assert.NoError(t, err)
		assert.NotNil(t, cd)
		assert.Equal(t, "", cd.TypeName) // No type name
		assert.NotNil(t, cd.Data)
	})

	t.Run("NewCheckpointData with nil", func(t *testing.T) {
		cd, err := NewCheckpointData(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cd)
		assert.Equal(t, "", cd.TypeName)
		assert.Equal(t, 0, len(cd.Data))
	})

	t.Run("CheckpointData ToValue with unknown type", func(t *testing.T) {
		cd := &CheckpointData{
			TypeName: "UnknownType",
			Data:     json.RawMessage(`{"field": "value"}`),
		}

		_, err := cd.ToValue()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown type")
	})

	t.Run("CheckpointData ToValue without type info", func(t *testing.T) {
		data, _ := json.Marshal(map[string]any{"field": "value"})
		cd := &CheckpointData{
			Data: json.RawMessage(data),
		}

		result, err := cd.ToValue()
		assert.NoError(t, err)
		assert.NotNil(t, result)

		resultMap, ok := result.(map[string]any)
		assert.True(t, ok)
		assert.Equal(t, "value", resultMap["field"])
	})

	t.Run("CheckpointData ToValue empty data", func(t *testing.T) {
		cd := &CheckpointData{}
		result, err := cd.ToValue()
		assert.NoError(t, err)
		assert.Nil(t, result)
	})

	t.Run("CheckpointData ToValue with custom unmarshaler", func(t *testing.T) {
		typ := reflect.TypeFor[CustomState]()

		unmarshalFunc := func(data []byte) (any, error) {
			var custom map[string]any
			if err := json.Unmarshal(data, &custom); err != nil {
				return nil, err
			}
			return CustomState{
				Name:  "custom_" + custom["name"].(string),
				Count: 999,
			}, nil
		}

		registry.RegisterTypeWithCustomSerializationInternal(
			typ, "CustomUnmarshalState",
			nil,
			unmarshalFunc,
		)

		stateData, _ := json.Marshal(CustomState{Name: "test", Count: 1})
		cd := &CheckpointData{
			TypeName: "CustomUnmarshalState",
			Data:     json.RawMessage(stateData),
		}

		result, err := cd.ToValue()
		assert.NoError(t, err)
		resultState, ok := result.(CustomState)
		assert.True(t, ok)
		assert.Equal(t, "custom_test", resultState.Name)
		assert.Equal(t, 999, resultState.Count)
	})
}

func TestGlobalTypeRegistry(t *testing.T) {
	t.Run("GlobalTypeRegistry returns singleton", func(t *testing.T) {
		r1 := GlobalTypeRegistry()
		r2 := GlobalTypeRegistry()
		assert.Same(t, r1, r2)
	})
}

func TestRegisterTypeWithValue(t *testing.T) {
	// Use local registry to avoid conflicts
	registry := newTestRegistry()

	t.Run("RegisterTypeWithValue with struct value", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[TestState](), "TestState10")
		assert.NoError(t, err)

		// Verify it's registered
		_, ok := registry.GetTypeByName("TestState10")
		assert.True(t, ok)
	})

	t.Run("RegisterTypeWithValue with pointer", func(t *testing.T) {
		err := registry.RegisterTypeInternal(reflect.TypeFor[*PointerState](), "PointerState2")
		assert.NoError(t, err)

		// Verify it's registered
		typ, ok := registry.GetTypeByName("PointerState2")
		assert.True(t, ok)
		assert.True(t, typ.Kind() == reflect.Ptr)
	})
}

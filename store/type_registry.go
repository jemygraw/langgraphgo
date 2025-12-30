package store

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
)

// TypeRegistry manages type information for generic state serialization/deserialization.
// It allows state types to register themselves for proper checkpointing.
type TypeRegistry struct {
	mu                sync.RWMutex
	typeNameToType    map[string]reflect.Type
	typeToName        map[reflect.Type]string
	typeCreators      map[string]func() any
	jsonMarshallers   map[reflect.Type]func(any) ([]byte, error)
	jsonUnmarshallers map[reflect.Type]func([]byte, any) (any, error)
}

// globalTypeRegistry is the singleton instance of TypeRegistry
var globalTypeRegistry = &TypeRegistry{
	typeNameToType:     make(map[string]reflect.Type),
	typeToName:         make(map[reflect.Type]string),
	typeCreators:       make(map[string]func() any),
	jsonMarshallers:    make(map[reflect.Type]func(any) ([]byte, error)),
	jsonUnmarshallers: make(map[reflect.Type]func([]byte, any) (any, error)),
}

// GlobalTypeRegistry returns the global type registry instance
func GlobalTypeRegistry() *TypeRegistry {
	return globalTypeRegistry
}

// RegisterType registers a reflect.Type with the registry for serialization/deserialization.
// Use RegisterTypeWithValue for a more convenient API with generics.
//
// Example usage:
//
//	var state MyState
//	RegisterType(reflect.TypeOf(state), "MyState")
func RegisterType(t reflect.Type, typeName string) error {
	return globalTypeRegistry.RegisterTypeInternal(t, typeName)
}

// RegisterTypeInternal registers a type with the registry.
func (r *TypeRegistry) RegisterTypeInternal(t reflect.Type, typeName string) error {
	// Only allow struct types (or pointers to structs)
	if t.Kind() != reflect.Struct {
		if t.Kind() == reflect.Ptr {
			elem := t.Elem()
			if elem.Kind() != reflect.Struct {
				return fmt.Errorf("type %s must be a struct or pointer to struct", t)
			}
		} else {
			return fmt.Errorf("type %s must be a struct", t)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if type already registered with different name
	if existingName, ok := r.typeToName[t]; ok && existingName != typeName {
		return fmt.Errorf("type %v already registered as %s", t, existingName)
	}

	r.typeNameToType[typeName] = t
	r.typeToName[t] = typeName
	r.typeCreators[typeName] = func() any {
		return reflect.New(t).Elem().Interface()
	}

	return nil
}

// RegisterTypeWithValue is a convenience function that uses reflection from a value.
// This is the recommended way to register types.
//
// Example usage:
//
//	var state MyState
//	RegisterTypeWithValue(state, "MyState")
func RegisterTypeWithValue(value any, typeName string) error {
	return globalTypeRegistry.RegisterTypeInternal(reflect.TypeOf(value), typeName)
}

// RegisterTypeWithCustomSerialization registers a type with custom JSON marshaling/unmarshaling.
//
// Example usage:
//
//	var state MyState
//	RegisterTypeWithCustomSerialization(
//		reflect.TypeOf(state),
//		"MyState",
//		func(v any) ([]byte, error) { ... },
//		func(data []byte) (any, error) { ... },
//	)
func RegisterTypeWithCustomSerialization(
	t reflect.Type,
	typeName string,
	marshalFunc func(any) ([]byte, error),
	unmarshalFunc func([]byte) (any, error),
) error {
	return globalTypeRegistry.RegisterTypeWithCustomSerializationInternal(t, typeName, marshalFunc, unmarshalFunc)
}

// RegisterTypeWithCustomSerializationInternal registers a type with custom serialization.
func (r *TypeRegistry) RegisterTypeWithCustomSerializationInternal(
	t reflect.Type,
	typeName string,
	marshalFunc func(any) ([]byte, error),
	unmarshalFunc func([]byte) (any, error),
) error {
	if err := r.RegisterTypeInternal(t, typeName); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.jsonMarshallers[t] = marshalFunc
	r.jsonUnmarshallers[t] = func(data []byte, _ any) (any, error) {
		return unmarshalFunc(data)
	}

	return nil
}

// GetTypeByName returns the reflect.Type for a registered type name.
func (r *TypeRegistry) GetTypeByName(typeName string) (reflect.Type, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.typeNameToType[typeName]
	return t, ok
}

// GetTypeName returns the registered name for a type.
func (r *TypeRegistry) GetTypeName(t reflect.Type) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.typeToName[t]
	return name, ok
}

// CreateInstance creates a new instance of a registered type by name.
func (r *TypeRegistry) CreateInstance(typeName string) (any, error) {
	r.mu.RLock()
	creator, ok := r.typeCreators[typeName]
	r.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("type %s not registered", typeName)
	}

	instance := creator()
	return instance, nil
}

// MarshalJSON marshals a value to JSON with type information.
func (r *TypeRegistry) MarshalJSON(value any) ([]byte, error) {
	if value == nil {
		return json.Marshal(nil)
	}

	t := reflect.TypeOf(value)

	// Get type name
	typeName, ok := r.GetTypeName(t)
	if !ok {
		// Type not registered, try standard JSON marshaling
		return json.Marshal(value)
	}

	r.mu.RLock()
	marshalFunc, hasCustomMarshaler := r.jsonMarshallers[t]
	r.mu.RUnlock()

	var jsonData []byte
	var err error

	if hasCustomMarshaler {
		jsonData, err = marshalFunc(value)
	} else {
		jsonData, err = json.Marshal(value)
	}

	if err != nil {
		return nil, err
	}

	// Wrap with type information
	wrapped := map[string]any{
		"_type": typeName,
		"_value": json.RawMessage(jsonData),
	}

	return json.Marshal(wrapped)
}

// UnmarshalJSON unmarshals JSON with type information.
func (r *TypeRegistry) UnmarshalJSON(data []byte) (any, error) {
	// First, try to unmarshal as wrapped type
	var wrapped map[string]json.RawMessage
	if err := json.Unmarshal(data, &wrapped); err != nil {
		// Not a wrapped object, return as-is
		var result any
		if err := json.Unmarshal(data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Check if this is a typed wrapper
	if typeBytes, ok := wrapped["_type"]; ok {
		var typeName string
		if err := json.Unmarshal(typeBytes, &typeName); err != nil {
			return nil, fmt.Errorf("failed to unmarshal type name: %w", err)
		}

		// Get the type
		t, ok := r.GetTypeByName(typeName)
		if !ok {
			return nil, fmt.Errorf("unknown type: %s", typeName)
		}

		// Create instance
		instance, err := r.CreateInstance(typeName)
		if err != nil {
			return nil, err
		}

		r.mu.RLock()
		unmarshalFunc, hasCustomUnmarshaler := r.jsonUnmarshallers[t]
		r.mu.RUnlock()

		if hasCustomUnmarshaler {
			// Use custom unmarshaler
			valueBytes, ok := wrapped["_value"]
			if !ok {
				return nil, fmt.Errorf("missing _value in wrapped data")
			}
			return unmarshalFunc(valueBytes, instance)
		}

		// Use standard JSON unmarshaling
		valueBytes, ok := wrapped["_value"]
		if !ok {
			return nil, fmt.Errorf("missing _value in wrapped data")
		}

		if err := json.Unmarshal(valueBytes, instance); err != nil {
			return nil, fmt.Errorf("failed to unmarshal value: %w", err)
		}

		return instance, nil
	}

	// Not a typed wrapper, return as-is
	var result any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// MarshalState is a helper for marshaling checkpoint states
type MarshalFunc func(any) ([]byte, error)

// UnmarshalState is a helper for unmarshaling checkpoint states
type UnmarshalFunc func([]byte) (any, error)

// StateMarshaler creates a marshal function for the given registry
func (r *TypeRegistry) StateMarshaler() MarshalFunc {
	return func(state any) ([]byte, error) {
		return r.MarshalJSON(state)
	}
}

// StateUnmarshaler creates an unmarshal function for the given registry
func (r *TypeRegistry) StateUnmarshaler() UnmarshalFunc {
	return func(data []byte) (any, error) {
		return r.UnmarshalJSON(data)
	}
}

// CheckpointData represents checkpoint data with type information
type CheckpointData struct {
	TypeName string          `json:"_type"`
	Data     json.RawMessage `json:"_data"`
}

// NewCheckpointData creates checkpoint data from a state value
func NewCheckpointData(state any) (*CheckpointData, error) {
	if state == nil {
		return &CheckpointData{}, nil
	}

	t := reflect.TypeOf(state)
	registry := GlobalTypeRegistry()

	typeName, ok := registry.GetTypeName(t)
	if !ok {
		// Type not registered, use standard JSON marshaling
		data, err := json.Marshal(state)
		if err != nil {
			return nil, err
		}
		return &CheckpointData{
			Data: json.RawMessage(data),
		}, nil
	}

	registry.mu.RLock()
	marshalFunc, hasCustomMarshaler := registry.jsonMarshallers[t]
	registry.mu.RUnlock()

	var jsonData []byte
	var err error

	if hasCustomMarshaler {
		jsonData, err = marshalFunc(state)
	} else {
		jsonData, err = json.Marshal(state)
	}

	if err != nil {
		return nil, err
	}

	return &CheckpointData{
		TypeName: typeName,
		Data:     json.RawMessage(jsonData),
	}, nil
}

// ToValue converts checkpoint data back to a state value
func (cd *CheckpointData) ToValue() (any, error) {
	if cd.TypeName == "" && len(cd.Data) == 0 {
		return nil, nil
	}

	registry := GlobalTypeRegistry()

	if cd.TypeName == "" {
		// No type information, try to unmarshal as-is
		var result any
		if err := json.Unmarshal(cd.Data, &result); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Get the type
	t, ok := registry.GetTypeByName(cd.TypeName)
	if !ok {
		return nil, fmt.Errorf("unknown type: %s", cd.TypeName)
	}

	// Create instance
	instance, err := registry.CreateInstance(cd.TypeName)
	if err != nil {
		return nil, err
	}

	registry.mu.RLock()
	unmarshalFunc, hasCustomUnmarshaler := registry.jsonUnmarshallers[t]
	registry.mu.RUnlock()

	if hasCustomUnmarshaler {
		return unmarshalFunc(cd.Data, instance)
	}

	// Use standard JSON unmarshaling
	if err := json.Unmarshal(cd.Data, instance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return instance, nil
}

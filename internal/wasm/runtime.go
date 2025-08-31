package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WASMRuntime manages WASM function execution
type WASMRuntime struct {
	runtime wazero.Runtime
	modules map[string]api.Module
}

// NewWASMRuntime creates a new WASM runtime
func NewWASMRuntime(ctx context.Context) *WASMRuntime {
	r := wazero.NewRuntime(ctx)

	return &WASMRuntime{
		runtime: r,
		modules: make(map[string]api.Module),
	}
}

// LoadFunction loads a WASM function from bytecode
func (w *WASMRuntime) LoadFunction(ctx context.Context, name string, wasmBytes []byte) error {
	module, err := w.runtime.Instantiate(ctx, wasmBytes)
	if err != nil {
		return fmt.Errorf("failed to instantiate WASM module %s: %w", name, err)
	}

	w.modules[name] = module
	return nil
}

// ExecuteFunction executes a WASM function
func (w *WASMRuntime) ExecuteFunction(ctx context.Context, funcName, methodName string, args ...uint64) ([]uint64, error) {
	module, exists := w.modules[funcName]
	if !exists {
		return nil, fmt.Errorf("function %s not found", funcName)
	}

	fn := module.ExportedFunction(methodName)
	if fn == nil {
		return nil, fmt.Errorf("method %s not found in function %s", methodName, funcName)
	}

	return fn.Call(ctx, args...)
}

// Close closes the WASM runtime
func (w *WASMRuntime) Close(ctx context.Context) error {
	for _, module := range w.modules {
		if err := module.Close(ctx); err != nil {
			return err
		}
	}
	return w.runtime.Close(ctx)
}

// Event represents a key event that can trigger WASM functions
type Event struct {
	Type      string // "SET", "EXPIRE", "DELETE"
	Key       string
	Value     string
	Timestamp int64
}

// EventHandler manages event-driven WASM function execution
type EventHandler struct {
	runtime  *WASMRuntime
	bindings map[string][]string // pattern -> function names
}

// NewEventHandler creates a new event handler
func NewEventHandler(runtime *WASMRuntime) *EventHandler {
	return &EventHandler{
		runtime:  runtime,
		bindings: make(map[string][]string),
	}
}

// BindFunction binds a WASM function to a key pattern for specific events
func (e *EventHandler) BindFunction(eventType, pattern, funcName string) {
	key := eventType + ":" + pattern
	e.bindings[key] = append(e.bindings[key], funcName)
}

// TriggerEvent triggers WASM functions for a key event
func (e *EventHandler) TriggerEvent(ctx context.Context, event Event) error {
	// This is a simplified pattern matching - in a real implementation,
	// you'd want proper glob pattern matching
	key := event.Type + ":" + event.Key

	if functions, exists := e.bindings[key]; exists {
		for _, funcName := range functions {
			// Execute the function with event data
			// This is simplified - real implementation would pass event data properly
			_, err := e.runtime.ExecuteFunction(ctx, funcName, "handle_event")
			if err != nil {
				return fmt.Errorf("failed to execute function %s for event %s: %w", funcName, event.Type, err)
			}
		}
	}

	return nil
}

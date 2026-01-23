// Package usertools provides a registry for custom tools with lazy loading support.
// Users can add their own tools by creating Go files in this package
// and registering them using the RegisterLazy function in an init() block.
//
// Example tool creation:
//
//	func init() {
//	    RegisterLazy(ToolDefinition{
//	        Name:        "my_tool",
//	        Description: "Description of what the tool does",
//	        Parameters: map[string]interface{}{
//	            "type": "object",
//	            "properties": map[string]interface{}{
//	                "param1": map[string]interface{}{
//	                    "type":        "string",
//	                    "description": "Description of param1",
//	                },
//	            },
//	            "required": []string{"param1"},
//	        },
//	        Loader: func() ToolHandler {
//	            // Any expensive initialization goes here
//	            // This only runs when the tool is first called
//	            return myToolHandler
//	        },
//	    })
//	}
//
//	func myToolHandler(invocation ToolInvocation) (ToolResult, error) {
//	    // Tool implementation
//	}
package usertools

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	copilot "github.com/github/copilot-sdk/go"
)

// ToolHandler is the function signature for tool handlers
type ToolHandler = copilot.ToolHandler

// ToolInvocation represents a tool invocation from the SDK
type ToolInvocation = copilot.ToolInvocation

// ToolResult represents the result of a tool execution
type ToolResult = copilot.ToolResult

// ToolDefinition defines a tool with lazy loading support
type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Loader      func() ToolHandler // Called on first use to get the handler
}

// LazyTool holds a tool definition with lazy-loaded handler and auto-unload support
type LazyTool struct {
	Definition ToolDefinition

	// Lazy loading state
	handler  ToolHandler
	loaded   bool
	mu       sync.RWMutex
	lastUsed atomic.Int64 // Unix timestamp of last use
	useCount atomic.Int64 // Number of times the tool has been used
}

// Config for the registry
var (
	// UnloadTimeout is how long a handler can be unused before being unloaded
	// Set to 0 to disable auto-unload
	UnloadTimeout = 5 * time.Minute

	// registry holds all registered tools
	registry   = make(map[string]*LazyTool)
	registryMu sync.RWMutex

	// cleanup goroutine control
	cleanupOnce    sync.Once
	cleanupStarted atomic.Bool
)

// RegisterLazy registers a tool with lazy handler loading.
// The Loader function is only called when the tool is first invoked,
// and the handler can be automatically unloaded after a period of inactivity.
func RegisterLazy(def ToolDefinition) {
	if def.Name == "" {
		panic("tool name cannot be empty")
	}
	if def.Loader == nil {
		panic(fmt.Sprintf("tool %q loader cannot be nil", def.Name))
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	registry[def.Name] = &LazyTool{
		Definition: def,
	}
}

// GetAll returns all registered tools as copilot.Tool slice with lazy wrapper handlers.
// The actual tool handlers are only loaded when first called.
func GetAll() []copilot.Tool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	// Start cleanup goroutine if not already running
	startCleanupRoutine()

	tools := make([]copilot.Tool, 0, len(registry))
	for _, t := range registry {
		tools = append(tools, copilot.Tool{
			Name:        t.Definition.Name,
			Description: t.Definition.Description,
			Parameters:  t.Definition.Parameters,
			Handler:     createLazyWrapper(t),
		})
	}
	return tools
}

// createLazyWrapper creates a handler that loads the real handler on first use
func createLazyWrapper(tool *LazyTool) ToolHandler {
	return func(invocation ToolInvocation) (ToolResult, error) {
		handler := tool.getHandler()
		if handler == nil {
			return ToolResult{
				ResultType:       "error",
				TextResultForLLM: fmt.Sprintf("Tool %q handler failed to load", tool.Definition.Name),
			}, nil
		}

		// Update usage stats
		tool.lastUsed.Store(time.Now().Unix())
		tool.useCount.Add(1)

		return handler(invocation)
	}
}

// getHandler returns the handler, loading it on first call
func (t *LazyTool) getHandler() ToolHandler {
	// Fast path: check if already loaded
	t.mu.RLock()
	if t.loaded && t.handler != nil {
		handler := t.handler
		t.mu.RUnlock()
		return handler
	}
	t.mu.RUnlock()

	// Slow path: load the handler
	t.mu.Lock()
	defer t.mu.Unlock()

	// Double-check after acquiring write lock
	if t.loaded && t.handler != nil {
		return t.handler
	}

	// Load the handler
	t.handler = t.Definition.Loader()
	t.loaded = true
	t.lastUsed.Store(time.Now().Unix())

	return t.handler
}

// unloadHandler unloads the handler to free memory
func (t *LazyTool) unloadHandler() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.handler = nil
	t.loaded = false
}

// isLoaded returns whether the handler is currently loaded
func (t *LazyTool) isLoaded() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.loaded
}

// startCleanupRoutine starts the background goroutine that unloads unused handlers
func startCleanupRoutine() {
	if UnloadTimeout <= 0 {
		return
	}

	cleanupOnce.Do(func() {
		cleanupStarted.Store(true)
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				cleanupUnusedHandlers()
			}
		}()
	})
}

// cleanupUnusedHandlers unloads handlers that haven't been used recently
func cleanupUnusedHandlers() {
	if UnloadTimeout <= 0 {
		return
	}

	registryMu.RLock()
	tools := make([]*LazyTool, 0, len(registry))
	for _, t := range registry {
		tools = append(tools, t)
	}
	registryMu.RUnlock()

	now := time.Now().Unix()
	threshold := int64(UnloadTimeout.Seconds())

	for _, tool := range tools {
		if tool.isLoaded() {
			lastUsed := tool.lastUsed.Load()
			if lastUsed > 0 && now-lastUsed > threshold {
				tool.unloadHandler()
			}
		}
	}
}

// Get returns a specific tool by name, or nil if not found
func Get(name string) *LazyTool {
	registryMu.RLock()
	defer registryMu.RUnlock()

	if t, ok := registry[name]; ok {
		return t
	}
	return nil
}

// List returns the names of all registered tools
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered tools
func Count() int {
	registryMu.RLock()
	defer registryMu.RUnlock()
	return len(registry)
}

// Stats returns usage statistics for a tool
func Stats(name string) (loaded bool, useCount int64, lastUsed time.Time) {
	tool := Get(name)
	if tool == nil {
		return false, 0, time.Time{}
	}

	loaded = tool.isLoaded()
	useCount = tool.useCount.Load()
	lastUsedUnix := tool.lastUsed.Load()
	if lastUsedUnix > 0 {
		lastUsed = time.Unix(lastUsedUnix, 0)
	}
	return
}

// ForceUnload unloads a specific tool's handler
func ForceUnload(name string) bool {
	tool := Get(name)
	if tool == nil {
		return false
	}
	tool.unloadHandler()
	return true
}

// ForceUnloadAll unloads all tool handlers
func ForceUnloadAll() {
	registryMu.RLock()
	tools := make([]*LazyTool, 0, len(registry))
	for _, t := range registry {
		tools = append(tools, t)
	}
	registryMu.RUnlock()

	for _, tool := range tools {
		tool.unloadHandler()
	}
}

// MapToStruct converts a map to a struct using JSON marshaling.
// This is a helper function for tool handlers to parse arguments.
func MapToStruct(data interface{}, target interface{}) error {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}
	if err := json.Unmarshal(jsonBytes, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target: %w", err)
	}
	return nil
}

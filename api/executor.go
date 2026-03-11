package api

import (
	"context"
	"fmt"
)

// MultiExecutor dispatches ExecuteTool calls to the registered executor for each tool name.
type MultiExecutor struct {
	executors map[string]ToolExecutor
}

// NewMultiExecutor returns an empty MultiExecutor.
func NewMultiExecutor() *MultiExecutor {
	return &MultiExecutor{executors: make(map[string]ToolExecutor)}
}

// Register associates name with the given executor.
func (m *MultiExecutor) Register(name string, exec ToolExecutor) {
	m.executors[name] = exec
}

// ExecuteTool dispatches to the executor registered for name.
// Returns an error if no executor is registered for name.
func (m *MultiExecutor) ExecuteTool(ctx context.Context, name string, input map[string]any) (string, error) {
	exec, ok := m.executors[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}
	return exec.ExecuteTool(ctx, name, input)
}

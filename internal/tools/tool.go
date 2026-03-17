package tools

import (
	"github.com/scttfrdmn/agenkit/agenkit-go/agenkit"
)

// Registry holds registered tools and exposes them to the agent router.
type Registry struct {
	tools []agenkit.Tool
}

// NewRegistry creates a new empty tool registry.
func NewRegistry() *Registry {
	return &Registry{}
}

// Register adds a tool to the registry.
func (r *Registry) Register(t agenkit.Tool) {
	r.tools = append(r.tools, t)
}

// GetAll returns all registered tools.
func (r *Registry) GetAll() []agenkit.Tool {
	result := make([]agenkit.Tool, len(r.tools))
	copy(result, r.tools)
	return result
}

// Enabled returns true if at least one tool is registered.
func (r *Registry) Enabled() bool {
	return len(r.tools) > 0
}

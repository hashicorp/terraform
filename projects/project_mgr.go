package projects

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// ProjectManager is a wrapper around Project that associates a project with
// the context it is being run in.
//
// A Project object represents the project itself, while a ProjectManager
// represents that project being used in a particular context: a specific set
// of context values, a means to access output values from upstream projects,
// and any other similar context-specific annotations that are required to
// run operations against a particular project.
type ProjectManager struct {
	project       *Project
	contextValues map[addrs.ProjectContextValue]cty.Value
}

// NewManager creates a ProjectManager object that binds the recieving project
// to a particular set of context values and other contextual context that
// will allow running operations against workspaces in the project.
func (p *Project) NewManager(contextValues map[addrs.ProjectContextValue]cty.Value) *ProjectManager {
	return &ProjectManager{
		project:       p,
		contextValues: contextValues,
	}
}

// Project returns the project that the receiving ProjectManager is managing.
func (m *ProjectManager) Project() *Project {
	return m.project
}

// ContextValues returns the configured context values for this project manager.
//
// The caller must treat the returned map as immutable, even though the Go
// type system cannot enforce that.
func (m *ProjectManager) ContextValues() map[addrs.ProjectContextValue]cty.Value {
	return m.contextValues
}

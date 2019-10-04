package projects

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/projects/projectconfigs"
)

// Project represents a single Terraform project, which is a container for
// zero or more workspaces.
type Project struct {
	config *projectconfigs.Config
}

// AllWorkspaceAddrs returns the addresses of all of the workspaces defined
// in the project.
//
// This expands each "workspace" block in the configuration to zero or more
// actual workspaces, based on the for_each expression. The result contains
// one element per workspace.
func (p *Project) AllWorkspaceAddrs() []addrs.ProjectWorkspace {
	return nil
}

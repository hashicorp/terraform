package terraform

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/tfdiags"
)

// ImportOpts are used as the configuration for Import.
type ImportOpts struct {
	// Targets are the targets to import
	Targets []*ImportTarget

	// Config is optional, and specifies a config tree that will be loaded
	// into the graph and evaluated. This is the source for provider
	// configurations.
	Config *configs.Config
}

// ImportTarget is a single resource to import.
type ImportTarget struct {
	// Addr is the address for the resource instance that the new object should
	// be imported into.
	Addr addrs.AbsResourceInstance

	// ID is the ID of the resource to import. This is resource-specific.
	ID string

	// ProviderAddr is the address of the provider that should handle the import.
	ProviderAddr addrs.AbsProviderConfig
}

// Import takes already-created external resources and brings them
// under Terraform management. Import requires the exact type, name, and ID
// of the resources to import.
//
// This operation is idempotent. If the requested resource is already
// imported, no changes are made to the state.
//
// Further, this operation also gracefully handles partial state. If during
// an import there is a failure, all previously imported resources remain
// imported.
func (c *Context) Import(opts *ImportOpts) (*State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Hold a lock since we can modify our own state here
	defer c.acquireRun("import")()

	// Copy our own state
	c.state = c.state.DeepCopy()

	// If no module is given, default to the module configured with
	// the Context.
	config := opts.Config
	if config == nil {
		config = c.config
	}

	// Initialize our graph builder
	builder := &ImportGraphBuilder{
		ImportTargets: opts.Targets,
		Config:        config,
		Components:    c.components,
		Schemas:       c.schemas,
	}

	// Build the graph!
	graph, graphDiags := builder.Build(addrs.RootModuleInstance)
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		return c.state, diags
	}

	// Walk it
	_, walkDiags := c.walk(graph, walkImport)
	diags = diags.Append(walkDiags)
	if walkDiags.HasErrors() {
		return c.state, diags
	}

	// Clean the state
	c.state.prune()

	return c.state, diags
}

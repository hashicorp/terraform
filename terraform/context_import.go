package terraform

import (
	"github.com/hashicorp/terraform/config/module"
)

// ImportOpts are used as the configuration for Import.
type ImportOpts struct {
	// Targets are the targets to import
	Targets []*ImportTarget

	// Module is optional, and specifies a config module that is loaded
	// into the graph and evaluated. The use case for this is to provide
	// provider configuration.
	Module *module.Tree
}

// ImportTarget is a single resource to import.
type ImportTarget struct {
	// Addr is the full resource address of the resource to import.
	// Example: "module.foo.aws_instance.bar"
	Addr string

	// ID is the ID of the resource to import. This is resource-specific.
	ID string
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
func (c *Context) Import(opts *ImportOpts) (*State, error) {
	// Hold a lock since we can modify our own state here
	v := c.acquireRun()
	defer c.releaseRun(v)

	// Copy our own state
	c.state = c.state.DeepCopy()

	// Get supported providers (for the graph builder)
	providers := make([]string, 0, len(c.providers))
	for k, _ := range c.providers {
		providers = append(providers, k)
	}

	// Initialize our graph builder
	builder := &ImportGraphBuilder{
		ImportTargets: opts.Targets,
		Module:        opts.Module,
		Providers:     providers,
	}

	// Build the graph!
	graph, err := builder.Build(RootModulePath)
	if err != nil {
		return c.state, err
	}

	// Walk it
	if _, err := c.walk(graph, walkImport); err != nil {
		return c.state, err
	}

	// Clean the state
	c.state.prune()

	return c.state, nil
}

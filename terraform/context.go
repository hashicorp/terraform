package terraform

import (
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/config/module"
)

// ContextOpts are the user-configurable options to create a context with
// NewContext.
type ContextOpts struct {
	Diff         *Diff
	Hooks        []Hook
	Module       *module.Tree
	Parallelism  int
	State        *State
	Providers    map[string]ResourceProviderFactory
	Provisioners map[string]ResourceProvisionerFactory
	Variables    map[string]string

	UIInput UIInput
}

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// NewContext. See the documentation for that.
type Context2 struct {
	module       *module.Tree
	providers    map[string]ResourceProviderFactory
	provisioners map[string]ResourceProvisionerFactory
	state        *State
	stateLock    sync.RWMutex
	variables    map[string]string
}

// NewContext creates a new Context structure.
//
// Once a Context is creator, the pointer values within ContextOpts
// should not be mutated in any way, since the pointers are copied, not
// the values themselves.
func NewContext2(opts *ContextOpts) *Context2 {
	return &Context2{
		module:       opts.Module,
		providers:    opts.Providers,
		provisioners: opts.Provisioners,
		state:        opts.State,
		variables:    opts.Variables,
	}
}

// GraphBuilder returns the GraphBuilder that will be used to create
// the graphs for this context.
func (c *Context2) GraphBuilder() GraphBuilder {
	// TODO test
	providers := make([]string, 0, len(c.providers))
	for k, _ := range c.providers {
		providers = append(providers, k)
	}

	provisioners := make([]string, 0, len(c.provisioners))
	for k, _ := range c.provisioners {
		provisioners = append(provisioners, k)
	}

	return &BuiltinGraphBuilder{
		Root:         c.module,
		Providers:    providers,
		Provisioners: provisioners,
		State:        c.state,
	}
}

// Refresh goes through all the resources in the state and refreshes them
// to their latest state. This will update the state that this context
// works with, along with returning it.
//
// Even in the case an error is returned, the state will be returned and
// will potentially be partially updated.
func (c *Context2) Refresh() (*State, []error) {
	if _, err := c.walk(walkRefresh); err != nil {
		var errs error
		return nil, multierror.Append(errs, err).Errors
	}

	return nil, nil
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context2) Validate() ([]string, []error) {
	var errs error

	// Validate the configuration itself
	if err := c.module.Validate(); err != nil {
		errs = multierror.Append(errs, err)
	}

	// This only needs to be done for the root module, since inter-module
	// variables are validated in the module tree.
	if config := c.module.Config(); config != nil {
		// Validate the user variables
		if err := smcUserVariables(config, c.variables); len(err) > 0 {
			errs = multierror.Append(errs, err...)
		}
	}

	// Walk
	walker, err := c.walk(walkValidate)
	if err != nil {
		return nil, multierror.Append(errs, err).Errors
	}

	// Return the result
	rerrs := multierror.Append(errs, walker.ValidationErrors...)
	return walker.ValidationWarnings, rerrs.Errors
}

func (c *Context2) walk(operation walkOperation) (*ContextGraphWalker, error) {
	// Build the graph
	graph, err := c.GraphBuilder().Build(RootModulePath)
	if err != nil {
		return nil, err
	}

	// Walk the graph
	walker := &ContextGraphWalker{Context: c, Operation: operation}
	return walker, graph.Walk(walker)
}

package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/dag"
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
	module    *module.Tree
	providers map[string]ResourceProviderFactory
	state     *State
}

// NewContext creates a new Context structure.
//
// Once a Context is creator, the pointer values within ContextOpts
// should not be mutated in any way, since the pointers are copied, not
// the values themselves.
func NewContext2(opts *ContextOpts) *Context2 {
	return &Context2{
		module:    opts.Module,
		providers: opts.Providers,
		state:     opts.State,
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

	return &BuiltinGraphBuilder{
		Root:      c.module,
		Providers: providers,
		State:     c.state,
	}
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context2) Validate() ([]string, []error) {
	var warns []string
	var errs []error

	// Validate the configuration itself
	if err := c.module.Validate(); err != nil {
		errs = append(errs, err)
	}

	evalCtx := c.evalContext()
	evalCtx.ComputeMissing = true

	// Build the graph
	graph, err := c.GraphBuilder().Build(RootModulePath)
	if err != nil {
		return nil, []error{err}
	}

	// Walk the graph
	var lock sync.Mutex
	graph.Walk(func(v dag.Vertex) {
		ev, ok := v.(GraphNodeEvalable)
		if !ok {
			return
		}

		tree := ev.EvalTree()
		if tree == nil {
			panic(fmt.Sprintf("%s (%T): nil eval tree", dag.VertexName(v), v))
		}

		_, err := Eval(tree, evalCtx)
		if err == nil {
			return
		}

		lock.Lock()
		defer lock.Unlock()

		verr, ok := err.(*EvalValidateError)
		if !ok {
			errs = append(errs, err)
			return
		}

		warns = append(warns, verr.Warnings...)
		errs = append(errs, verr.Errors...)
	})

	return warns, errs
}

func (c *Context2) evalContext() *BuiltinEvalContext {
	return &BuiltinEvalContext{
		Providers: c.providers,
	}
}

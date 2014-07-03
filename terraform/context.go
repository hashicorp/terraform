package terraform

import (
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/multierror"
)

// Context represents all the context that Terraform needs in order to
// perform operations on infrastructure. This structure is built using
// ContextOpts and NewContext. See the documentation for those.
//
// Additionally, a context can be created from a Plan using Plan.Context.
type Context struct {
	config    *config.Config
	diff      *Diff
	hooks     []Hook
	state     *State
	providers map[string]ResourceProviderFactory
	variables map[string]string
}

// ContextOpts are the user-creatable configuration structure to create
// a context with NewContext.
type ContextOpts struct {
	Config    *config.Config
	Diff      *Diff
	Hooks     []Hook
	State     *State
	Providers map[string]ResourceProviderFactory
	Variables map[string]string
}

// NewContext creates a new context.
//
// Once a context is created, the pointer values within ContextOpts should
// not be mutated in any way, since the pointers are copied, not the values
// themselves.
func NewContext(opts *ContextOpts) *Context {
	return &Context{
		config:    opts.Config,
		diff:      opts.Diff,
		hooks:     opts.Hooks,
		state:     opts.State,
		providers: opts.Providers,
		variables: opts.Variables,
	}
}

// Validate validates the configuration and returns any warnings or errors.
func (c *Context) Validate() ([]string, []error) {
	var rerr *multierror.Error

	// Validate the configuration itself
	if err := c.config.Validate(); err != nil {
		rerr = multierror.ErrorAppend(rerr, err)
	}

	// Validate the user variables
	if errs := smcUserVariables(c.config, c.variables); len(errs) > 0 {
		rerr = multierror.ErrorAppend(rerr, errs...)
	}

	var errs []error
	if rerr != nil && len(rerr.Errors) > 0 {
		errs = rerr.Errors
	}

	return nil, errs
}

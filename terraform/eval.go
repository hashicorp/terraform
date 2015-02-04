package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalContext is the interface that is given to eval nodes to execute.
type EvalContext interface {
	// InitProvider initializes the provider with the given name and
	// returns the implementation of the resource provider or an error.
	InitProvider(string) (ResourceProvider, error)

	// Provider gets the provider instance with the given name (already
	// initialized) or returns nil if the provider isn't initialized.
	Provider(string) ResourceProvider

	// Interpolate takes the given raw configuration and completes
	// the interpolations, returning the processed ResourceConfig.
	Interpolate(*config.RawConfig) (*ResourceConfig, error)
}

// EvalNode is the interface that must be implemented by graph nodes to
// evaluate/execute.
type EvalNode interface {
	// Args returns the arguments for this node as well as the list of
	// expected types. The expected types are only used for type checking
	// and not used at runtime.
	Args() ([]EvalNode, []EvalType)

	// Eval evaluates this node with the given context. The second parameter
	// are the argument values. These will match in order and 1-1 with the
	// results of the Args() return value.
	Eval(EvalContext, []interface{}) (interface{}, error)

	// Type returns the type that will be returned by this node.
	Type() EvalType
}

// GraphNodeEvalable is the interface that graph nodes must implement
// to enable valuation.
type GraphNodeEvalable interface {
	EvalTree() EvalNode
}

// MockEvalContext is a mock version of EvalContext that can be used
// for tests.
type MockEvalContext struct {
	InitProviderCalled   bool
	InitProviderName     string
	InitProviderProvider ResourceProvider
	InitProviderError    error

	ProviderCalled   bool
	ProviderName     string
	ProviderProvider ResourceProvider

	InterpolateCalled       bool
	InterpolateConfig       *config.RawConfig
	InterpolateConfigResult *ResourceConfig
	InterpolateError        error
}

func (c *MockEvalContext) InitProvider(n string) (ResourceProvider, error) {
	c.InitProviderCalled = true
	c.InitProviderName = n
	return c.InitProviderProvider, c.InitProviderError
}

func (c *MockEvalContext) Provider(n string) ResourceProvider {
	c.ProviderCalled = true
	c.ProviderName = n
	return c.ProviderProvider
}

func (c *MockEvalContext) Interpolate(
	config *config.RawConfig) (*ResourceConfig, error) {
	c.InterpolateCalled = true
	c.InterpolateConfig = config
	return c.InterpolateConfigResult, c.InterpolateError
}

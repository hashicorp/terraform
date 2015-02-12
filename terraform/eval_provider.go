package terraform

import (
	"fmt"
)

// EvalConfigProvider is an EvalNode implementation that configures
// a provider that is already initialized and retrieved.
type EvalConfigProvider struct {
	Provider string
	Config   EvalNode
}

func (n *EvalConfigProvider) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Config}, []EvalType{EvalTypeConfig}
}

func (n *EvalConfigProvider) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	config := args[0].(*ResourceConfig)

	// Get the parent configuration if there is one
	if parent := ctx.ParentProviderConfig(n.Provider); parent != nil {
		merged := config.raw.Merge(parent.raw)
		config = NewResourceConfig(merged)
	}

	return nil, ctx.ConfigureProvider(n.Provider, config)
}

func (n *EvalConfigProvider) Type() EvalType {
	return EvalTypeNull
}

// EvalInitProvider is an EvalNode implementation that initializes a provider
// and returns nothing. The provider can be retrieved again with the
// EvalGetProvider node.
type EvalInitProvider struct {
	Name string
}

func (n *EvalInitProvider) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalInitProvider) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	return ctx.InitProvider(n.Name)
}

func (n *EvalInitProvider) Type() EvalType {
	return EvalTypeNull
}

// EvalGetProvider is an EvalNode implementation that retrieves an already
// initialized provider instance for the given name.
type EvalGetProvider struct {
	Name string
}

func (n *EvalGetProvider) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalGetProvider) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	result := ctx.Provider(n.Name)
	if result == nil {
		return nil, fmt.Errorf("provider %s not initialized", n.Name)
	}

	return result, nil
}

func (n *EvalGetProvider) Type() EvalType {
	return EvalTypeResourceProvider
}

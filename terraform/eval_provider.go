package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// EvalConfigProvider is an EvalNode implementation that configures
// a provider that is already initialized and retrieved.
type EvalConfigProvider struct {
	Provider string
	Config   **ResourceConfig
}

func (n *EvalConfigProvider) Eval(ctx EvalContext) (interface{}, error) {
	cfg := *n.Config

	// If we have a configuration set, then use that
	if input := ctx.ProviderInput(n.Provider); input != nil {
		rc, err := config.NewRawConfig(input)
		if err != nil {
			return nil, err
		}

		merged := cfg.raw.Merge(rc)
		cfg = NewResourceConfig(merged)
	}

	// Get the parent configuration if there is one
	if parent := ctx.ParentProviderConfig(n.Provider); parent != nil {
		merged := cfg.raw.Merge(parent.raw)
		cfg = NewResourceConfig(merged)
	}

	return nil, ctx.ConfigureProvider(n.Provider, cfg)
}

// EvalInitProvider is an EvalNode implementation that initializes a provider
// and returns nothing. The provider can be retrieved again with the
// EvalGetProvider node.
type EvalInitProvider struct {
	Name string
}

func (n *EvalInitProvider) Eval(ctx EvalContext) (interface{}, error) {
	return ctx.InitProvider(n.Name)
}

// EvalGetProvider is an EvalNode implementation that retrieves an already
// initialized provider instance for the given name.
type EvalGetProvider struct {
	Name   string
	Output *ResourceProvider
}

func (n *EvalGetProvider) Eval(ctx EvalContext) (interface{}, error) {
	result := ctx.Provider(n.Name)
	if result == nil {
		return nil, fmt.Errorf("provider %s not initialized", n.Name)
	}

	if n.Output != nil {
		*n.Output = result
	}

	return nil, nil
}

// EvalInputProvider is an EvalNode implementation that asks for input
// for the given provider configurations.
type EvalInputProvider struct {
	Name     string
	Provider *ResourceProvider
	Config   *config.RawConfig
}

func (n *EvalInputProvider) Eval(ctx EvalContext) (interface{}, error) {
	// If we already configured this provider, then don't do this again
	if v := ctx.ProviderInput(n.Name); v != nil {
		return nil, nil
	}

	rc := NewResourceConfig(n.Config)
	rc.Config = make(map[string]interface{})

	// Wrap the input into a namespace
	input := &PrefixUIInput{
		IdPrefix:    fmt.Sprintf("provider.%s", n.Name),
		QueryPrefix: fmt.Sprintf("provider.%s.", n.Name),
		UIInput:     ctx.Input(),
	}

	// Go through each provider and capture the input necessary
	// to satisfy it.
	config, err := (*n.Provider).Input(input, rc)
	if err != nil {
		return nil, fmt.Errorf(
			"Error configuring %s: %s", n.Name, err)
	}

	if config != nil && len(config.Config) > 0 {
		// Set the configuration
		ctx.SetProviderInput(n.Name, config.Config)
	}

	return nil, nil
}

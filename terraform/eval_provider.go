package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/config"
)

// EvalSetProviderConfig sets the parent configuration for a provider
// without configuring that provider, validating it, etc.
type EvalSetProviderConfig struct {
	Provider string
	Config   **ResourceConfig
}

func (n *EvalSetProviderConfig) Eval(ctx EvalContext) (interface{}, error) {
	return nil, ctx.SetProviderConfig(n.Provider, *n.Config)
}

// EvalBuildProviderConfig outputs a *ResourceConfig that is properly
// merged with parents and inputs on top of what is configured in the file.
type EvalBuildProviderConfig struct {
	Provider string
	Config   **ResourceConfig
	Output   **ResourceConfig
}

func (n *EvalBuildProviderConfig) Eval(ctx EvalContext) (interface{}, error) {
	cfg := *n.Config

	// If we have a configuration set, then merge that in
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

	*n.Output = cfg
	return nil, nil
}

// EvalConfigProvider is an EvalNode implementation that configures
// a provider that is already initialized and retrieved.
type EvalConfigProvider struct {
	Provider string
	Config   **ResourceConfig
}

func (n *EvalConfigProvider) Eval(ctx EvalContext) (interface{}, error) {
	return nil, ctx.ConfigureProvider(n.Provider, *n.Config)
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

// EvalCloseProvider is an EvalNode implementation that closes provider
// connections that aren't needed anymore.
type EvalCloseProvider struct {
	Name string
}

func (n *EvalCloseProvider) Eval(ctx EvalContext) (interface{}, error) {
	ctx.CloseProvider(n.Name)
	return nil, nil
}

// EvalGetProvider is an EvalNode implementation that retrieves an already
// initialized provider instance for the given name.
//
// Will abort evaluation of the current node if the named provider is
// deferred, under the assumption that this node will itself be deferred
// on subsequent walks and to prevent interating with an uninitalized
// provider.
type EvalGetProvider struct {
	Name   string
	Output *ResourceProvider
}

func (n *EvalGetProvider) Eval(ctx EvalContext) (interface{}, error) {

	deferrals, lock := ctx.Deferrals()

	// Lock the deferrals to prevent concurrent modifications
	lock.Lock()
	defer lock.Unlock()

	modDeferrals := deferrals.ModuleByPath(ctx.Path())
	if modDeferrals != nil && modDeferrals.ProviderIsDeferred(n.Name) {
		// If the provider we need is deferred, abort our own
		// evaluation on this walk. We too will be deferred on the
		// *next* walk, due to depending on a deferred provider.
		log.Printf("[DEBUG] EvalGetProvider exiting early due to deferred provider %q\n", n.Name)
		return nil, EvalEarlyExitError{}
	}

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
	Config   **ResourceConfig
}

func (n *EvalInputProvider) Eval(ctx EvalContext) (interface{}, error) {
	// If we already configured this provider, then don't do this again
	if v := ctx.ProviderInput(n.Name); v != nil {
		return nil, nil
	}

	rc := *n.Config

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

	// Set the input that we received so that child modules don't attempt
	// to ask for input again.
	if config != nil && len(config.Config) > 0 {
		ctx.SetProviderInput(n.Name, config.Config)
	} else {
		ctx.SetProviderInput(n.Name, map[string]interface{}{})
	}

	return nil, nil
}

// EvalDeferComputedProvider defers the given provider if the given configuration
// contains unresolved interpolations.
//
// If the provider is deferred then this node will terminate further evaluation of
// the provider, so any subsequent EvalNodes will not be visited.
type EvalDeferComputedProvider struct {
	Name   string
	Config **ResourceConfig
}

// TODO: test
func (n *EvalDeferComputedProvider) Eval(ctx EvalContext) (interface{}, error) {
	deferrals, lock := ctx.Deferrals()

	config := *n.Config
	computed := config.ComputedKeys != nil && len(config.ComputedKeys) > 0

	if !computed {
		return true, nil
	}

	// Lock the deferrals to prevent concurrent modifications
	lock.Lock()
	defer lock.Unlock()

	modDeferrals := deferrals.ModuleByPath(ctx.Path())
	if modDeferrals == nil {
		modDeferrals = deferrals.AddModule(ctx.Path())
	}

	// We'll take the first computed key to give some context to the
	// reason message. This is arbitrary, but most providers don't have
	// lots of configuration attributes so it should usually be sufficient
	// to help the user understand what's going on.
	keyForReason := config.ComputedKeys[0]
	valueForReason, _ := config.GetRaw(keyForReason)

	reason := fmt.Sprintf("%q not yet resolvable: %s", keyForReason, valueForReason)
	modDeferrals.DeferProvider(n.Name, reason)

	log.Printf("[DEBUG] Deferring provider %q: %s\n", n.Name, reason)
	return true, EvalEarlyExitError{}
}

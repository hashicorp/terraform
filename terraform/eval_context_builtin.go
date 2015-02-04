package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	Providers map[string]ResourceProviderFactory

	providers map[string]ResourceProvider
	once      sync.Once
}

func (ctx *BuiltinEvalContext) InitProvider(n string) (ResourceProvider, error) {
	ctx.once.Do(ctx.init)

	if p := ctx.Provider(n); p != nil {
		return nil, fmt.Errorf("Provider '%s' already initialized", n)
	}

	f, ok := ctx.Providers[n]
	if !ok {
		return nil, fmt.Errorf("Provider '%s' not found", n)
	}

	return f()
}

func (ctx *BuiltinEvalContext) Provider(n string) ResourceProvider {
	ctx.once.Do(ctx.init)
	return ctx.providers[n]
}

func (ctx *BuiltinEvalContext) Interpolate(
	config *config.RawConfig) (*ResourceConfig, error) {
	// TODO: Actual interpolation, for now we just return it as-is
	return NewResourceConfig(config), nil
}

func (ctx *BuiltinEvalContext) init() {
	// We nil-check the things below because they're meant to be configured,
	// and we just default them to non-nil.
	if ctx.Providers == nil {
		ctx.Providers = make(map[string]ResourceProviderFactory)
	}

	// We always reset the things below since we only call this once and
	// they can't be initialized externally.
	ctx.providers = make(map[string]ResourceProvider)
}

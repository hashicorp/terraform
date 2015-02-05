package terraform

import (
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/lang/ast"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	Providers      map[string]ResourceProviderFactory
	ComputeMissing bool

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

	p, err := f()
	if err != nil {
		return nil, err
	}

	ctx.providers[n] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provider(n string) ResourceProvider {
	ctx.once.Do(ctx.init)
	return ctx.providers[n]
}

func (ctx *BuiltinEvalContext) Interpolate(
	cfg *config.RawConfig) (*ResourceConfig, error) {
	vs := make(map[string]ast.Variable)

	// If we don't have a config, use the blank config
	if cfg == nil {
		goto INTERPOLATE_RESULT
	}

	for n, rawV := range cfg.Variables {
		switch rawV.(type) {
		case *config.ModuleVariable:
			if ctx.ComputeMissing {
				vs[n] = ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				}
			}
		case *config.ResourceVariable:
			if ctx.ComputeMissing {
				vs[n] = ast.Variable{
					Value: config.UnknownVariableValue,
					Type:  ast.TypeString,
				}
			}
		default:
			return nil, fmt.Errorf(
				"unknown interpolation type: %#v", rawV)
		}
	}

	// Do the interpolation
	if err := cfg.Interpolate(vs); err != nil {
		return nil, err
	}

INTERPOLATE_RESULT:
	result := NewResourceConfig(cfg)
	result.interpolateForce()
	return result, nil
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

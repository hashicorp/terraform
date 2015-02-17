package terraform

import (
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	PathValue           []string
	Interpolater        *Interpolater
	Hooks               []Hook
	InputValue          UIInput
	Providers           map[string]ResourceProviderFactory
	ProviderCache       map[string]ResourceProvider
	ProviderConfigCache map[string]*ResourceConfig
	ProviderInputConfig map[string]map[string]interface{}
	ProviderLock        *sync.Mutex
	Provisioners        map[string]ResourceProvisionerFactory
	ProvisionerCache    map[string]ResourceProvisioner
	ProvisionerLock     *sync.Mutex
	DiffValue           *Diff
	DiffLock            *sync.RWMutex
	StateValue          *State
	StateLock           *sync.RWMutex

	once sync.Once
}

func (ctx *BuiltinEvalContext) Hook(fn func(Hook) (HookAction, error)) error {
	for _, h := range ctx.Hooks {
		action, err := fn(h)
		if err != nil {
			return err
		}

		switch action {
		case HookActionContinue:
			continue
		case HookActionHalt:
			// Return an early exit error to trigger an early exit
			log.Printf("[WARN] Early exit triggered by hook: %T", h)
			return EvalEarlyExitError{}
		}
	}

	return nil
}

func (ctx *BuiltinEvalContext) Input() UIInput {
	return ctx.InputValue
}

func (ctx *BuiltinEvalContext) InitProvider(n string) (ResourceProvider, error) {
	ctx.once.Do(ctx.init)

	// If we already initialized, it is an error
	if p := ctx.Provider(n); p != nil {
		return nil, fmt.Errorf("Provider '%s' already initialized", n)
	}

	// Warning: make sure to acquire these locks AFTER the call to Provider
	// above, since it also acquires locks.
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	f, ok := ctx.Providers[n]
	if !ok {
		return nil, fmt.Errorf("Provider '%s' not found", n)
	}

	p, err := f()
	if err != nil {
		return nil, err
	}

	providerPath := make([]string, len(ctx.Path())+1)
	copy(providerPath, ctx.Path())
	providerPath[len(providerPath)-1] = n

	ctx.ProviderCache[PathCacheKey(providerPath)] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provider(n string) ResourceProvider {
	ctx.once.Do(ctx.init)

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	providerPath := make([]string, len(ctx.Path())+1)
	copy(providerPath, ctx.Path())
	providerPath[len(providerPath)-1] = n

	return ctx.ProviderCache[PathCacheKey(providerPath)]
}

func (ctx *BuiltinEvalContext) ConfigureProvider(
	n string, cfg *ResourceConfig) error {
	p := ctx.Provider(n)
	if p == nil {
		return fmt.Errorf("Provider '%s' not initialized", n)
	}

	// Save the configuration
	ctx.ProviderLock.Lock()
	ctx.ProviderConfigCache[PathCacheKey(ctx.Path())] = cfg
	ctx.ProviderLock.Unlock()

	return p.Configure(cfg)
}

func (ctx *BuiltinEvalContext) ProviderInput(n string) map[string]interface{} {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	return ctx.ProviderInputConfig[n]
}

func (ctx *BuiltinEvalContext) SetProviderInput(n string, c map[string]interface{}) {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	ctx.ProviderInputConfig[n] = c
}

func (ctx *BuiltinEvalContext) ParentProviderConfig(n string) *ResourceConfig {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	path := ctx.Path()
	for i := len(path) - 1; i >= 1; i-- {
		k := PathCacheKey(path[:i])
		if v, ok := ctx.ProviderConfigCache[k]; ok {
			return v
		}
	}

	return nil
}

func (ctx *BuiltinEvalContext) InitProvisioner(
	n string) (ResourceProvisioner, error) {
	ctx.once.Do(ctx.init)

	// If we already initialized, it is an error
	if p := ctx.Provisioner(n); p != nil {
		return nil, fmt.Errorf("Provisioner '%s' already initialized", n)
	}

	// Warning: make sure to acquire these locks AFTER the call to Provisioner
	// above, since it also acquires locks.
	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	f, ok := ctx.Provisioners[n]
	if !ok {
		return nil, fmt.Errorf("Provisioner '%s' not found", n)
	}

	p, err := f()
	if err != nil {
		return nil, err
	}

	ctx.ProvisionerCache[PathCacheKey(ctx.Path())] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provisioner(n string) ResourceProvisioner {
	ctx.once.Do(ctx.init)

	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	return ctx.ProvisionerCache[PathCacheKey(ctx.Path())]
}

func (ctx *BuiltinEvalContext) Interpolate(
	cfg *config.RawConfig, r *Resource) (*ResourceConfig, error) {
	if cfg != nil {
		scope := &InterpolationScope{
			Path:     ctx.Path(),
			Resource: r,
		}
		vs, err := ctx.Interpolater.Values(scope, cfg.Variables)
		if err != nil {
			return nil, err
		}

		// Do the interpolation
		if err := cfg.Interpolate(vs); err != nil {
			return nil, err
		}
	}

	result := NewResourceConfig(cfg)
	result.interpolateForce()
	return result, nil
}

func (ctx *BuiltinEvalContext) Path() []string {
	return ctx.PathValue
}

func (ctx *BuiltinEvalContext) SetVariables(vs map[string]string) {
	for k, v := range vs {
		ctx.Interpolater.Variables[k] = v
	}
}

func (ctx *BuiltinEvalContext) Diff() (*Diff, *sync.RWMutex) {
	return ctx.DiffValue, ctx.DiffLock
}

func (ctx *BuiltinEvalContext) State() (*State, *sync.RWMutex) {
	return ctx.StateValue, ctx.StateLock
}

func (ctx *BuiltinEvalContext) init() {
	// We nil-check the things below because they're meant to be configured,
	// and we just default them to non-nil.
	if ctx.Providers == nil {
		ctx.Providers = make(map[string]ResourceProviderFactory)
	}
}

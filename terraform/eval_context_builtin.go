package terraform

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	// StopContext is the context used to track whether we're complete
	StopContext context.Context

	// PathValue is the Path that this context is operating within.
	PathValue []string

	// Interpolater setting below affect the interpolation of variables.
	//
	// The InterpolaterVars are the exact value for ${var.foo} values.
	// The map is shared between all contexts and is a mapping of
	// PATH to KEY to VALUE. Because it is shared by all contexts as well
	// as the Interpolater itself, it is protected by InterpolaterVarLock
	// which must be locked during any access to the map.
	Interpolater        *Interpolater
	InterpolaterVars    map[string]map[string]interface{}
	InterpolaterVarLock *sync.Mutex

	Components          contextComponentFactory
	Hooks               []Hook
	InputValue          UIInput
	ProviderCache       map[string]ResourceProvider
	ProviderInputConfig map[string]map[string]interface{}
	ProviderLock        *sync.Mutex
	ProvisionerCache    map[string]ResourceProvisioner
	ProvisionerLock     *sync.Mutex
	DiffValue           *Diff
	DiffLock            *sync.RWMutex
	StateValue          *State
	StateLock           *sync.RWMutex

	once sync.Once
}

func (ctx *BuiltinEvalContext) Stopped() <-chan struct{} {
	// This can happen during tests. During tests, we just block forever.
	if ctx.StopContext == nil {
		return nil
	}

	return ctx.StopContext.Done()
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

	typeName := strings.SplitN(n, ".", 2)[0]
	p, err := ctx.Components.ResourceProvider(typeName, n)
	if err != nil {
		return nil, err
	}

	ctx.ProviderCache[n] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provider(n string) ResourceProvider {
	ctx.once.Do(ctx.init)

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	return ctx.ProviderCache[n]
}

func (ctx *BuiltinEvalContext) CloseProvider(n string) error {
	ctx.once.Do(ctx.init)

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	var provider interface{}
	provider = ctx.ProviderCache[n]
	if provider != nil {
		if p, ok := provider.(ResourceProviderCloser); ok {
			delete(ctx.ProviderCache, n)
			return p.Close()
		}
	}

	return nil
}

func (ctx *BuiltinEvalContext) ConfigureProvider(
	n string, cfg *ResourceConfig) error {
	p := ctx.Provider(n)
	if p == nil {
		return fmt.Errorf("Provider '%s' not initialized", n)
	}
	return p.Configure(cfg)
}

func (ctx *BuiltinEvalContext) ProviderInput(n string) map[string]interface{} {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	// Make a copy of the path so we can safely edit it
	path := ctx.Path()
	pathCopy := make([]string, len(path)+1)
	copy(pathCopy, path)

	// Go up the tree.
	for i := len(path) - 1; i >= 0; i-- {
		pathCopy[i+1] = n
		k := PathCacheKey(pathCopy[:i+2])
		if v, ok := ctx.ProviderInputConfig[k]; ok {
			return v
		}
	}

	return nil
}

func (ctx *BuiltinEvalContext) SetProviderInput(n string, c map[string]interface{}) {
	providerPath := make([]string, len(ctx.Path())+1)
	copy(providerPath, ctx.Path())
	providerPath[len(providerPath)-1] = n

	// Save the configuration
	ctx.ProviderLock.Lock()
	ctx.ProviderInputConfig[PathCacheKey(providerPath)] = c
	ctx.ProviderLock.Unlock()
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

	provPath := make([]string, len(ctx.Path())+1)
	copy(provPath, ctx.Path())
	provPath[len(provPath)-1] = n
	key := PathCacheKey(provPath)

	p, err := ctx.Components.ResourceProvisioner(n, key)
	if err != nil {
		return nil, err
	}

	ctx.ProvisionerCache[key] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provisioner(n string) ResourceProvisioner {
	ctx.once.Do(ctx.init)

	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	provPath := make([]string, len(ctx.Path())+1)
	copy(provPath, ctx.Path())
	provPath[len(provPath)-1] = n

	return ctx.ProvisionerCache[PathCacheKey(provPath)]
}

func (ctx *BuiltinEvalContext) CloseProvisioner(n string) error {
	ctx.once.Do(ctx.init)

	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	provPath := make([]string, len(ctx.Path())+1)
	copy(provPath, ctx.Path())
	provPath[len(provPath)-1] = n

	var prov interface{}
	prov = ctx.ProvisionerCache[PathCacheKey(provPath)]
	if prov != nil {
		if p, ok := prov.(ResourceProvisionerCloser); ok {
			delete(ctx.ProvisionerCache, PathCacheKey(provPath))
			return p.Close()
		}
	}

	return nil
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

func (ctx *BuiltinEvalContext) InterpolateProvider(
	pc *config.ProviderConfig, r *Resource) (*ResourceConfig, error) {

	var cfg *config.RawConfig

	if pc != nil && pc.RawConfig != nil {
		path := pc.Path
		if len(path) == 0 {
			path = ctx.Path()
		}

		scope := &InterpolationScope{
			Path:     path,
			Resource: r,
		}

		cfg = pc.RawConfig

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

func (ctx *BuiltinEvalContext) SetVariables(n string, vs map[string]interface{}) {
	ctx.InterpolaterVarLock.Lock()
	defer ctx.InterpolaterVarLock.Unlock()

	path := make([]string, len(ctx.Path())+1)
	copy(path, ctx.Path())
	path[len(path)-1] = n
	key := PathCacheKey(path)

	vars := ctx.InterpolaterVars[key]
	if vars == nil {
		vars = make(map[string]interface{})
		ctx.InterpolaterVars[key] = vars
	}

	for k, v := range vs {
		vars[k] = v
	}
}

func (ctx *BuiltinEvalContext) Diff() (*Diff, *sync.RWMutex) {
	return ctx.DiffValue, ctx.DiffLock
}

func (ctx *BuiltinEvalContext) State() (*State, *sync.RWMutex) {
	return ctx.StateValue, ctx.StateLock
}

func (ctx *BuiltinEvalContext) init() {
}

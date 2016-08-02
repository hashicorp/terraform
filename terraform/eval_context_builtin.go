package terraform

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
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

	typeName := strings.SplitN(n, ".", 2)[0]

	f, ok := ctx.Providers[typeName]
	if !ok {
		return nil, fmt.Errorf("Provider '%s' not found", typeName)
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

func (ctx *BuiltinEvalContext) CloseProvider(n string) error {
	ctx.once.Do(ctx.init)

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	providerPath := make([]string, len(ctx.Path())+1)
	copy(providerPath, ctx.Path())
	providerPath[len(providerPath)-1] = n

	var provider interface{}
	provider = ctx.ProviderCache[PathCacheKey(providerPath)]
	if provider != nil {
		if p, ok := provider.(ResourceProviderCloser); ok {
			delete(ctx.ProviderCache, PathCacheKey(providerPath))
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

	if err := ctx.SetProviderConfig(n, cfg); err != nil {
		return nil
	}

	return p.Configure(cfg)
}

func (ctx *BuiltinEvalContext) SetProviderConfig(
	n string, cfg *ResourceConfig) error {
	providerPath := make([]string, len(ctx.Path())+1)
	copy(providerPath, ctx.Path())
	providerPath[len(providerPath)-1] = n

	// Save the configuration
	ctx.ProviderLock.Lock()
	ctx.ProviderConfigCache[PathCacheKey(providerPath)] = cfg
	ctx.ProviderLock.Unlock()

	return nil
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

func (ctx *BuiltinEvalContext) ParentProviderConfig(n string) *ResourceConfig {
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

	provPath := make([]string, len(ctx.Path())+1)
	copy(provPath, ctx.Path())
	provPath[len(provPath)-1] = n

	ctx.ProvisionerCache[PathCacheKey(provPath)] = p
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
	// We nil-check the things below because they're meant to be configured,
	// and we just default them to non-nil.
	if ctx.Providers == nil {
		ctx.Providers = make(map[string]ResourceProviderFactory)
	}
}

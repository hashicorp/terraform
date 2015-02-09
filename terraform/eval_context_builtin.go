package terraform

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/hashicorp/terraform/config"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	PathValue        []string
	Interpolater     *Interpolater
	Providers        map[string]ResourceProviderFactory
	ProviderCache    map[string]ResourceProvider
	ProviderLock     *sync.Mutex
	Provisioners     map[string]ResourceProvisionerFactory
	ProvisionerCache map[string]ResourceProvisioner
	ProvisionerLock  *sync.Mutex

	once sync.Once
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

	ctx.ProviderCache[ctx.pathCacheKey()] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provider(n string) ResourceProvider {
	ctx.once.Do(ctx.init)

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	return ctx.ProviderCache[ctx.pathCacheKey()]
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

	ctx.ProvisionerCache[ctx.pathCacheKey()] = p
	return p, nil
}

func (ctx *BuiltinEvalContext) Provisioner(n string) ResourceProvisioner {
	ctx.once.Do(ctx.init)

	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	return ctx.ProvisionerCache[ctx.pathCacheKey()]
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

func (ctx *BuiltinEvalContext) init() {
	// We nil-check the things below because they're meant to be configured,
	// and we just default them to non-nil.
	if ctx.Providers == nil {
		ctx.Providers = make(map[string]ResourceProviderFactory)
	}
}

// pathCacheKey returns a cache key for the current module path, unique to
// the module path.
//
// This is used because there is a variety of information that needs to be
// cached per-path, rather than per-context.
func (ctx *BuiltinEvalContext) pathCacheKey() string {
	// There is probably a better way to do this, but this is working for now.
	// We just create an MD5 hash of all the MD5 hashes of all the path
	// elements. This gets us the property that it is unique per ordering.
	hash := md5.New()
	for _, p := range ctx.Path() {
		single := md5.Sum([]byte(p))
		if _, err := hash.Write(single[:]); err != nil {
			panic(err)
		}
	}

	return hex.EncodeToString(hash.Sum(nil))
}

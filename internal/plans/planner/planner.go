package planner

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/opentracing/opentracing-go"
	tracelog "github.com/opentracing/opentracing-go/log"
)

type planner struct {
	opts            *Options
	config          *configs.Config
	prevRunState    *states.State
	providerFactory func(addrs.Provider) (providers.Interface, error)

	// coalescedCtx is a special context we use for operations that return
	// to more than one caller, since otherwise we'd end up arbitrarily
	// using the context of whatever caller happened to start the coalesced
	// operation.
	coalescedCtx context.Context

	// agglomerator is our orchestrator of coalesced operations, which allows
	// our various concurrent planning goroutines to fetch data from one
	// another while remaining largely decoupled from the implementation
	// details.
	agglomerator *agglomerator

	// The remaining fields are the mutable state of a planner. Access
	// these only while holding a lock on mu.

	// diags accumulates diagostics that show up during evaluation. We
	// deliver these out of band of results, rather than sending them
	// as return values as normal, because our evaluation strategy
	// causes us to potentially return the same result to multiple callers
	// but we don't want to return the same _diagnostics_ multiple times.
	diags tfdiags.Diagnostics

	// unconfiguredProviders keeps the singleton unconfigured instances of
	// each provider. These provider instances are reference-counted and
	// so will be closed once all users have called Close on their references.
	unconfiguredProviders map[addrs.Provider]*providerInstance

	// configuredProviders keeps the singleton configured instances of
	// each distinct provider configuration. These provider instances are
	// reference-counted and so will be closed once all users have called
	// Close on their references.
	//
	// The keys of configuredProviders are addrs.AbsProviderConfig.UniqueKey()
	// results.
	configuredProviders map[addrs.UniqueKey]*providerInstance

	// evaluator is the container for all of our module-specific evaluation
	// data objects, instantiated on first use.
	evaluator *evaluator

	mu sync.Mutex
}

func (p *planner) AddDiagnostics(v ...interface{}) {
	p.mu.Lock()
	p.diags = p.diags.Append(v...)
	p.mu.Unlock()
}

func (p *planner) Config() *configs.Config {
	return p.config
}

func (p *planner) PrevRunState() *states.State {
	return p.prevRunState
}

func (p *planner) TargetAddrs() []addrs.Targetable {
	return p.opts.TargetAddrs
}

func (p *planner) ResourceInConfig(addr addrs.ConfigResource) resourceInConfig {
	return resourceInConfig{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) Resource(addr addrs.AbsResource) resource {
	return resource{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) ResourceInstance(addr addrs.AbsResourceInstance) resourceInstance {
	return resourceInstance{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) InputVariable(addr addrs.AbsInputVariableInstance) inputVariable {
	return inputVariable{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) LocalValue(addr addrs.AbsLocalValue) localValue {
	return localValue{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) Module(addr addrs.Module) module {
	return module{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) ModuleCall(addr addrs.AbsModuleCall) moduleCall {
	return moduleCall{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) ModuleInstance(addr addrs.ModuleInstance) moduleInstance {
	return moduleInstance{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) Provider(addr addrs.Provider) provider {
	return provider{
		planner: p,
		addr:    addr,
	}
}

func (p *planner) PlanOptions() *Options {
	return p.opts
}

func (p *planner) Evaluator() *evaluator {
	p.mu.Lock()
	if p.evaluator == nil {
		p.evaluator = &evaluator{
			planner: p,
			modules: make(map[addrs.UniqueKey]*evaluationDataModule),
		}
	}
	ret := p.evaluator
	p.mu.Unlock()
	return ret
}

func (p *planner) ModuleInstanceExprScope(addr addrs.ModuleInstance) *lang.Scope {
	return &lang.Scope{
		Data:     p.Evaluator().DataForModuleInstance(addr),
		PureOnly: true,
		// TODO: Also BaseDir
	}
}

func (p *planner) ChildInstanceInModuleInstanceExprScope(addr addrs.ModuleInstance, repData instances.RepetitionData, self addrs.Referenceable) *lang.Scope {
	return &lang.Scope{
		Data:     p.Evaluator().DataForModuleInstance(addr).ForObjectInstance(repData),
		PureOnly: true,
		SelfAddr: self,
		// TODO: Also BaseDir
	}
}

func (p *planner) DataRequest(ctx context.Context, req dataRequest) interface{} {
	return p.agglomerator.Request(ctx, req)
}

func (p *planner) UnconfiguredProviderInstance(pdr provider) (providers.Interface, error) {
	p.mu.Lock()
	inst, err := p.unconfiguredProviderInstance(pdr)
	p.mu.Unlock()
	return inst, err
}

func (p *planner) unconfiguredProviderInstance(pdr provider) (providers.Interface, error) {
	addr := pdr.Addr()

	activeInst := p.unconfiguredProviders[addr]
	if activeInst != nil {
		log.Printf("[TRACE] New user for existing provider %s", addr)
		activeInst.refCount++
		activeInst.lifetimeSpan.LogFields(
			tracelog.Int("newRefCount", activeInst.refCount),
		)
		return activeInst, nil
	}

	log.Printf("[TRACE] Starting provider %s", addr)
	lifetimeSpan, _ := opentracing.StartSpanFromContext(p.coalescedCtx, "provider")
	lifetimeSpan.LogFields(
		tracelog.String("provider", addr.String()),
	)
	newInst, err := p.providerFactory(addr)
	if err != nil {
		lifetimeSpan.Finish()
		return nil, err
	}

	var wrapped *providerInstance
	wrapped = &providerInstance{
		Interface: newInst,
		refCount:  1,
		onClose: func() error {
			log.Printf("[TRACE] Request to close provider %s", addr)
			p.mu.Lock()
			defer p.mu.Unlock()

			wrapped.refCount--
			lifetimeSpan.LogFields(
				tracelog.Int("newRefCount", wrapped.refCount),
			)
			if wrapped.refCount == 0 {
				delete(p.unconfiguredProviders, addr)
				log.Printf("[TRACE] Closing provider %s", addr)
				err := newInst.Close()
				lifetimeSpan.Finish()
				return err
			}
			log.Printf("[TRACE] Release provider %s, but still has other users", addr)
			return nil
		},
		lifetimeSpan: lifetimeSpan,
	}
	if p.unconfiguredProviders == nil {
		p.unconfiguredProviders = make(map[addrs.Provider]*providerInstance)
	}
	p.unconfiguredProviders[addr] = wrapped
	lifetimeSpan.LogFields(
		tracelog.Int("newRefCount", wrapped.refCount),
	)

	log.Printf("[TRACE] Returning unconfigured provider instance for %s", addr)
	return wrapped, nil
}

func (p *planner) ConfiguredProviderInstance(pdrCfg providerConfig) (providers.Interface, error) {
	p.mu.Lock()
	inst, err := p.configuredProviderInstance(pdrCfg)
	p.mu.Unlock()
	return inst, err
}

func (p *planner) configuredProviderInstance(pdrCfg providerConfig) (providers.Interface, error) {
	addr := pdrCfg.Addr()
	addrKey := addr.UniqueKey()

	activeInst := p.configuredProviders[addrKey]
	if activeInst != nil {
		log.Printf("[TRACE] New user for existing provider config %s", addr)
		activeInst.refCount++
		activeInst.lifetimeSpan.LogFields(
			tracelog.Int("newRefCount", activeInst.refCount),
		)
		return activeInst, nil
	}

	log.Printf("[TRACE] Starting provider for config %s", addr)
	lifetimeSpan, _ := opentracing.StartSpanFromContext(p.coalescedCtx, "providerConfig")
	lifetimeSpan.LogFields(
		tracelog.String("providerConfig", addr.String()),
	)
	newInst, err := p.providerFactory(addr.Provider)
	if err != nil {
		lifetimeSpan.Finish()
		return nil, err
	}

	log.Printf("[TRACE] Configuring provider for %s", addr)
	diags := pdrCfg.configureProviderInstance(newInst)
	if diags.HasErrors() {
		// FIXME: We need to signal _somehow_ that the configuration failed
		// here, but without potentially duplicating the diagnostics for
		// multiple callers. For the moment this will still duplicate this
		// stubby error instead of the real diagnostic, which doesn't seem
		// great either.
		lifetimeSpan.Finish()
		return nil, fmt.Errorf("provider configuration failed")
	}

	var wrapped *providerInstance
	wrapped = &providerInstance{
		Interface: newInst,
		refCount:  1,
		onClose: func() error {
			log.Printf("[TRACE] Request to close provider config %s", addr)
			p.mu.Lock()
			defer p.mu.Unlock()

			wrapped.refCount--
			lifetimeSpan.LogFields(
				tracelog.Int("newRefCount", wrapped.refCount),
			)
			if wrapped.refCount == 0 {
				delete(p.configuredProviders, addrKey)
				log.Printf("[TRACE] Closing provider config %s", addr)
				err := newInst.Close()
				lifetimeSpan.Finish()
				return err
			}
			log.Printf("[TRACE] Release provider config %s, but still has other users", addr)
			return nil
		},
		lifetimeSpan: lifetimeSpan,
	}
	if p.configuredProviders == nil {
		p.configuredProviders = make(map[addrs.UniqueKey]*providerInstance)
	}
	p.configuredProviders[addrKey] = wrapped
	lifetimeSpan.LogFields(
		tracelog.Int("newRefCount", wrapped.refCount),
	)

	log.Printf("[TRACE] Returning configured provider instance for %s", addr)
	return wrapped, nil
}

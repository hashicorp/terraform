// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/experiments"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/namedvals"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/provisioners"
	"github.com/hashicorp/terraform/internal/refactoring"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
)

// BuiltinEvalContext is an EvalContext implementation that is used by
// Terraform by default.
type BuiltinEvalContext struct {
	// scope is the scope (module instance or set of possible module instances)
	// that this context is operating within.
	//
	// Note: this can be evalContextGlobal (i.e. nil) when visiting a graph
	// node that doesn't belong to a particular module, in which case any
	// method using it will panic.
	scope evalContextScope

	// StopContext is the context used to track whether we're complete
	StopContext context.Context

	// Evaluator is used for evaluating expressions within the scope of this
	// eval context.
	Evaluator *Evaluator

	// NamedValuesValue is where we keep the values of already-evaluated input
	// variables, local values, and output values.
	NamedValuesValue *namedvals.State

	// Plugins is a library of plugin components (providers and provisioners)
	// available for use during a graph walk.
	Plugins *contextPlugins

	// ExternalProviderConfigs are pre-configured provider instances passed
	// in by the caller, for situations like Stack components where the
	// root module isn't designed to be planned and applied in isolation and
	// instead expects to recieve certain provider configurations from the
	// stack configuration.
	ExternalProviderConfigs map[addrs.RootProviderConfig]providers.Interface

	// DeferralsValue is the object returned by [BuiltinEvalContext.Deferrals].
	DeferralsValue *deferring.Deferred

	Hooks                 []Hook
	InputValue            UIInput
	ProviderCache         map[string]providers.Interface
	ProviderFuncCache     map[string]providers.Interface
	ProviderFuncResults   *providers.FunctionResults
	ProviderInputConfig   map[string]map[string]cty.Value
	ProviderLock          *sync.Mutex
	ProvisionerCache      map[string]provisioners.Interface
	ProvisionerLock       *sync.Mutex
	ChangesValue          *plans.ChangesSync
	StateValue            *states.SyncState
	ChecksValue           *checks.State
	RefreshStateValue     *states.SyncState
	PrevRunStateValue     *states.SyncState
	InstanceExpanderValue *instances.Expander
	MoveResultsValue      refactoring.MoveResults
	OverrideValues        *mocking.Overrides
}

// BuiltinEvalContext implements EvalContext
var _ EvalContext = (*BuiltinEvalContext)(nil)

func (ctx *BuiltinEvalContext) withScope(scope evalContextScope) EvalContext {
	newCtx := *ctx
	newCtx.scope = scope
	return &newCtx
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
			return nil
		}
	}

	return nil
}

func (ctx *BuiltinEvalContext) Input() UIInput {
	return ctx.InputValue
}

func (ctx *BuiltinEvalContext) InitProvider(addr addrs.AbsProviderConfig, config *configs.Provider) (providers.Interface, error) {
	// If we already initialized, it is an error
	if p := ctx.Provider(addr); p != nil {
		return nil, fmt.Errorf("%s is already initialized", addr)
	}

	// Warning: make sure to acquire these locks AFTER the call to Provider
	// above, since it also acquires locks.
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	key := addr.String()

	if addr.Module.IsRoot() {
		rootAddr := addrs.RootProviderConfig{
			Provider: addr.Provider,
			Alias:    addr.Alias,
		}
		if external, isExternal := ctx.ExternalProviderConfigs[rootAddr]; isExternal {
			// External providers should always be pre-configured by the
			// external caller, and so we'll wrap them in a type that
			// makes operations like ConfigureProvider and Close be no-op.
			wrapped := externalProviderWrapper{external}
			ctx.ProviderCache[key] = wrapped
			return wrapped, nil
		}
	}

	p, err := ctx.Plugins.NewProviderInstance(addr.Provider)
	if err != nil {
		return nil, err
	}

	log.Printf("[TRACE] BuiltinEvalContext: Initialized %q provider for %s", addr.String(), addr)

	// The config might be nil, if there was no config block defined for this
	// provider.
	if config != nil && config.Mock {
		log.Printf("[TRACE] BuiltinEvalContext: Mocked %q provider for %s", addr.String(), addr)
		p = &providers.Mock{
			Provider: p,
			Data:     config.MockData,
		}
	}

	ctx.ProviderCache[key] = p

	return p, nil
}

func (ctx *BuiltinEvalContext) Provider(addr addrs.AbsProviderConfig) providers.Interface {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	return ctx.ProviderCache[addr.String()]
}

func (ctx *BuiltinEvalContext) ProviderSchema(addr addrs.AbsProviderConfig) (providers.ProviderSchema, error) {
	return ctx.Plugins.ProviderSchema(addr.Provider)
}

func (ctx *BuiltinEvalContext) CloseProvider(addr addrs.AbsProviderConfig) error {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	key := addr.String()
	provider := ctx.ProviderCache[key]
	if provider != nil {
		delete(ctx.ProviderCache, key)
		return provider.Close()
	}

	return nil
}

func (ctx *BuiltinEvalContext) ConfigureProvider(addr addrs.AbsProviderConfig, cfg cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if !addr.Module.Equal(ctx.Path().Module()) {
		// This indicates incorrect use of ConfigureProvider: it should be used
		// only from the module that the provider configuration belongs to.
		panic(fmt.Sprintf("%s configured by wrong module %s", addr, ctx.Path()))
	}

	p := ctx.Provider(addr)
	if p == nil {
		diags = diags.Append(fmt.Errorf("%s not initialized", addr))
		return diags
	}

	req := providers.ConfigureProviderRequest{
		TerraformVersion: version.String(),
		Config:           cfg,
	}

	resp := p.ConfigureProvider(req)
	return resp.Diagnostics
}

func (ctx *BuiltinEvalContext) ProviderInput(pc addrs.AbsProviderConfig) map[string]cty.Value {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	if !pc.Module.Equal(ctx.Path().Module()) {
		// This indicates incorrect use of InitProvider: it should be used
		// only from the module that the provider configuration belongs to.
		panic(fmt.Sprintf("%s initialized by wrong module %s", pc, ctx.Path()))
	}

	if !ctx.Path().IsRoot() {
		// Only root module provider configurations can have input.
		return nil
	}

	return ctx.ProviderInputConfig[pc.String()]
}

func (ctx *BuiltinEvalContext) SetProviderInput(pc addrs.AbsProviderConfig, c map[string]cty.Value) {
	absProvider := pc
	if !pc.Module.IsRoot() {
		// Only root module provider configurations can have input.
		log.Printf("[WARN] BuiltinEvalContext: attempt to SetProviderInput for non-root module")
		return
	}

	// Save the configuration
	ctx.ProviderLock.Lock()
	ctx.ProviderInputConfig[absProvider.String()] = c
	ctx.ProviderLock.Unlock()
}

func (ctx *BuiltinEvalContext) Provisioner(n string) (provisioners.Interface, error) {
	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	p, ok := ctx.ProvisionerCache[n]
	if !ok {
		var err error
		p, err = ctx.Plugins.NewProvisionerInstance(n)
		if err != nil {
			return nil, err
		}

		ctx.ProvisionerCache[n] = p
	}

	return p, nil
}

func (ctx *BuiltinEvalContext) ProvisionerSchema(n string) (*configschema.Block, error) {
	return ctx.Plugins.ProvisionerSchema(n)
}

func (ctx *BuiltinEvalContext) ClosePlugins() error {
	var diags tfdiags.Diagnostics
	ctx.ProvisionerLock.Lock()
	defer ctx.ProvisionerLock.Unlock()

	for name, prov := range ctx.ProvisionerCache {
		err := prov.Close()
		if err != nil {
			diags = diags.Append(fmt.Errorf("provisioner.Close %s: %s", name, err))
		}
		delete(ctx.ProvisionerCache, name)
	}

	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()
	for name, prov := range ctx.ProviderFuncCache {
		err := prov.Close()
		if err != nil {
			diags = diags.Append(fmt.Errorf("provider.Close %s: %s", name, err))
		}
		delete(ctx.ProviderFuncCache, name)
	}

	return diags.Err()
}

func (ctx *BuiltinEvalContext) EvaluateBlock(body hcl.Body, schema *configschema.Block, self addrs.Referenceable, keyData InstanceKeyEvalData) (cty.Value, hcl.Body, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	scope := ctx.EvaluationScope(self, nil, keyData)
	body, evalDiags := scope.ExpandBlock(body, schema)
	diags = diags.Append(evalDiags)
	val, evalDiags := scope.EvalBlock(body, schema)
	diags = diags.Append(evalDiags)
	return val, body, diags
}

func (ctx *BuiltinEvalContext) EvaluateExpr(expr hcl.Expression, wantType cty.Type, self addrs.Referenceable) (cty.Value, tfdiags.Diagnostics) {
	scope := ctx.EvaluationScope(self, nil, EvalDataForNoInstanceKey)
	return scope.EvalExpr(expr, wantType)
}

func (ctx *BuiltinEvalContext) EvaluateReplaceTriggeredBy(expr hcl.Expression, repData instances.RepetitionData) (*addrs.Reference, bool, tfdiags.Diagnostics) {

	// get the reference to lookup changes in the plan
	ref, diags := evalReplaceTriggeredByExpr(expr, repData)
	if diags.HasErrors() {
		return nil, false, diags
	}

	var changes []*plans.ResourceInstanceChangeSrc
	// store the address once we get it for validation
	var resourceAddr addrs.Resource

	// The reference is either a resource or resource instance
	switch sub := ref.Subject.(type) {
	case addrs.Resource:
		resourceAddr = sub
		rc := sub.Absolute(ctx.Path())
		changes = ctx.Changes().GetChangesForAbsResource(rc)
	case addrs.ResourceInstance:
		resourceAddr = sub.ContainingResource()
		rc := sub.Absolute(ctx.Path())
		change := ctx.Changes().GetResourceInstanceChange(rc, addrs.NotDeposed)
		if change != nil {
			// we'll generate an error below if there was no change
			changes = append(changes, change)
		}
	}

	// Do some validation to make sure we are expecting a change at all
	cfg := ctx.Evaluator.Config.Descendent(ctx.Path().Module())
	resCfg := cfg.Module.ResourceByAddr(resourceAddr)
	if resCfg == nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Reference to undeclared resource`,
			Detail:   fmt.Sprintf(`A resource %s has not been declared in %s`, ref.Subject, moduleDisplayAddr(ctx.Path())),
			Subject:  expr.Range().Ptr(),
		})
		return nil, false, diags
	}

	if len(changes) == 0 {
		// If the resource is valid there should always be at least one change.
		diags = diags.Append(fmt.Errorf("no change found for %s in %s", ref.Subject, moduleDisplayAddr(ctx.Path())))
		return nil, false, diags
	}

	// If we don't have a traversal beyond the resource, then we can just look
	// for any change.
	if len(ref.Remaining) == 0 {
		for _, c := range changes {
			switch c.ChangeSrc.Action {
			// Only immediate changes to the resource will trigger replacement.
			case plans.Update, plans.DeleteThenCreate, plans.CreateThenDelete:
				return ref, true, diags
			}
		}

		// no change triggered
		return nil, false, diags
	}

	// This must be an instances to have a remaining traversal, which means a
	// single change.
	change := changes[0]

	// Make sure the change is actionable. A create or delete action will have
	// a change in value, but are not valid for our purposes here.
	switch change.ChangeSrc.Action {
	case plans.Update, plans.DeleteThenCreate, plans.CreateThenDelete:
		// OK
	default:
		return nil, false, diags
	}

	// Since we have a traversal after the resource reference, we will need to
	// decode the changes, which means we need a schema.
	providerAddr := change.ProviderAddr
	schema, err := ctx.ProviderSchema(providerAddr)
	if err != nil {
		diags = diags.Append(err)
		return nil, false, diags
	}

	resAddr := change.Addr.ContainingResource().Resource
	resSchema, _ := schema.SchemaForResourceType(resAddr.Mode, resAddr.Type)
	ty := resSchema.ImpliedType()

	before, err := change.ChangeSrc.Before.Decode(ty)
	if err != nil {
		diags = diags.Append(err)
		return nil, false, diags
	}

	after, err := change.ChangeSrc.After.Decode(ty)
	if err != nil {
		diags = diags.Append(err)
		return nil, false, diags
	}

	path := traversalToPath(ref.Remaining)
	attrBefore, _ := path.Apply(before)
	attrAfter, _ := path.Apply(after)

	if attrBefore == cty.NilVal || attrAfter == cty.NilVal {
		replace := attrBefore != attrAfter
		return ref, replace, diags
	}

	replace := !attrBefore.RawEquals(attrAfter)

	return ref, replace, diags
}

func (ctx *BuiltinEvalContext) EvaluationScope(self addrs.Referenceable, source addrs.Referenceable, keyData InstanceKeyEvalData) *lang.Scope {
	switch scope := ctx.scope.(type) {
	case evalContextModuleInstance:
		data := &evaluationStateData{
			Evaluator:       ctx.Evaluator,
			ModulePath:      scope.Addr,
			InstanceKeyData: keyData,
			Operation:       ctx.Evaluator.Operation,
		}
		evalScope := ctx.Evaluator.Scope(data, self, source, ctx.evaluationExternalFunctions())

		// ctx.PathValue is the path of the module that contains whatever
		// expression the caller will be trying to evaluate, so this will
		// activate only the experiments from that particular module, to
		// be consistent with how experiment checking in the "configs"
		// package itself works. The nil check here is for robustness in
		// incompletely-mocked testing situations; mc should never be nil in
		// real situations.
		if mc := ctx.Evaluator.Config.DescendentForInstance(scope.Addr); mc != nil {
			evalScope.SetActiveExperiments(mc.Module.ActiveExperiments)
		}
		return evalScope
	case evalContextPartialExpandedModule:
		data := &evaluationPlaceholderData{
			Evaluator:      ctx.Evaluator,
			ModulePath:     scope.Addr,
			CountAvailable: keyData.CountIndex != cty.NilVal,
			EachAvailable:  keyData.EachKey != cty.NilVal,
			Operation:      ctx.Evaluator.Operation,
		}
		evalScope := ctx.Evaluator.Scope(data, self, source, ctx.evaluationExternalFunctions())
		if mc := ctx.Evaluator.Config.Descendent(scope.Addr.Module()); mc != nil {
			evalScope.SetActiveExperiments(mc.Module.ActiveExperiments)
		}
		return evalScope
	default:
		// This method is valid only for module-scoped EvalContext objects.
		panic("no evaluation scope available: not in module context")
	}

}

// evaluationExternalFunctions is a helper for method EvaluationScope which
// determines the set of external functions that should be available for
// evaluation in this EvalContext, based on declarations in the configuration.
func (ctx *BuiltinEvalContext) evaluationExternalFunctions() lang.ExternalFuncs {
	// The set of functions in scope includes the functions contributed by
	// every provider that the current module has as a requirement.
	//
	// We expose them under the local name for each provider that was selected
	// by the module author.
	ret := lang.ExternalFuncs{}

	cfg := ctx.Evaluator.Config.Descendent(ctx.scope.evalContextScopeModule())
	if cfg == nil {
		// It's weird to not have a configuration by this point, but we'll
		// tolerate it for robustness and just return no functions at all.
		return ret
	}
	if cfg.Module.ProviderRequirements == nil {
		// A module with no provider requirements can't have any
		// provider-contributed functions.
		return ret
	}

	reqs := cfg.Module.ProviderRequirements.RequiredProviders
	ret.Provider = make(map[string]map[string]function.Function, len(reqs))

	for localName, req := range reqs {
		providerAddr := req.Type
		funcDecls, err := ctx.Plugins.ProviderFunctionDecls(providerAddr)
		if err != nil {
			// If a particular provider can't return schema then we'll catch
			// it in plenty other places where it's more reasonable for us
			// to return an error, so here we'll just treat it as having
			// no functions.
			log.Printf("[WARN] Error loading schema for %s to determine its functions: %s", providerAddr, err)
			continue
		}

		ret.Provider[localName] = make(map[string]function.Function, len(funcDecls))
		funcs := ret.Provider[localName]
		for name, decl := range funcDecls {
			funcs[name] = decl.BuildFunction(providerAddr, name, ctx.ProviderFuncResults, func() (providers.Interface, error) {
				return ctx.functionProvider(providerAddr)
			})
		}
	}

	return ret
}

// functionProvider fetches a running provider instance for evaluating
// functions from the cache, or starts a new instance and adds it to the cache.
func (ctx *BuiltinEvalContext) functionProvider(addr addrs.Provider) (providers.Interface, error) {
	ctx.ProviderLock.Lock()
	defer ctx.ProviderLock.Unlock()

	p, ok := ctx.ProviderFuncCache[addr.String()]
	if ok {
		return p, nil
	}

	log.Printf("[TRACE] starting function provider instance for %s", addr)
	p, err := ctx.Plugins.NewProviderInstance(addr)
	if err == nil {
		ctx.ProviderFuncCache[addr.String()] = p
	}

	return p, err
}

func (ctx *BuiltinEvalContext) Path() addrs.ModuleInstance {
	if scope, ok := ctx.scope.(evalContextModuleInstance); ok {
		return scope.Addr
	}
	panic("not evaluating in the scope of a fully-expanded module")
}

func (ctx *BuiltinEvalContext) LanguageExperimentActive(experiment experiments.Experiment) bool {
	if ctx.Evaluator == nil || ctx.Evaluator.Config == nil {
		// Should not get here in normal code, but might get here in test code
		// if the context isn't fully populated.
		return false
	}
	scope := ctx.scope
	if scope == evalContextGlobal {
		// If we're not associated with a specific module then there can't
		// be any language experiments in play, because experiment activation
		// is module-scoped.
		return false
	}
	cfg := ctx.Evaluator.Config.Descendent(scope.evalContextScopeModule())
	if cfg == nil {
		return false
	}
	return cfg.Module.ActiveExperiments.Has(experiment)
}

func (ctx *BuiltinEvalContext) NamedValues() *namedvals.State {
	return ctx.NamedValuesValue
}

func (ctx *BuiltinEvalContext) Deferrals() *deferring.Deferred {
	return ctx.DeferralsValue
}

func (ctx *BuiltinEvalContext) Changes() *plans.ChangesSync {
	return ctx.ChangesValue
}

func (ctx *BuiltinEvalContext) State() *states.SyncState {
	return ctx.StateValue
}

func (ctx *BuiltinEvalContext) Checks() *checks.State {
	return ctx.ChecksValue
}

func (ctx *BuiltinEvalContext) RefreshState() *states.SyncState {
	return ctx.RefreshStateValue
}

func (ctx *BuiltinEvalContext) PrevRunState() *states.SyncState {
	return ctx.PrevRunStateValue
}

func (ctx *BuiltinEvalContext) InstanceExpander() *instances.Expander {
	return ctx.InstanceExpanderValue
}

func (ctx *BuiltinEvalContext) MoveResults() refactoring.MoveResults {
	return ctx.MoveResultsValue
}

func (ctx *BuiltinEvalContext) Overrides() *mocking.Overrides {
	return ctx.OverrideValues
}

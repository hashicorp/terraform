// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/lang/blocktoattr"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// policyRunOpts describes the relationships of a plan or apply walk: the
// walk relates the instances of the types of the policy plugin's
// relationship definitions, and sends each subject its related instances. A
// walk has them only when the plugin sent definitions.
type policyRunOpts struct {
	Apply bool
	// NormalPlan is set in a plan in the normal mode.
	NormalPlan bool
	Schemas    *schemarepo.Schemas

	// Relationships are the plugin's relationships, without the
	// definitions that validation skipped.
	Relationships policy.Relationships

	// AppliedChanges is set in an apply only: the applied plan's resource
	// instance changes, copied before the walk because the apply walk removes
	// changes from the working changes as it applies them.
	AppliedChanges []*plans.ResourceInstanceChange

	// OutsideConfigProviders are the addresses of the root provider
	// configurations whose value doesn't come from their configuration
	// alone: those with interactive input values, and the external providers
	// the caller configured.
	OutsideConfigProviders map[string]bool

	// Shared is the caller's run that the walk contributes to, or nil when
	// the walk relates its own instances.
	Shared *SharedPolicyRun
}

// policyOutsideConfigProviders returns the addresses of the root provider
// configurations with interactive input values or in external.
func (c *Context) policyOutsideConfigProviders(external map[addrs.RootProviderConfig]providers.Interface) map[string]bool {
	ret := make(map[string]bool)
	for addr, vals := range c.providerInputConfig {
		if len(vals) > 0 {
			ret[addr] = true
		}
	}
	for addr := range external {
		ret[addr.AbsProviderConfig().String()] = true
	}
	return ret
}

// policyRelationships returns the relationships of a walk's policy client
// validated against the walk's schemas, or those of the shared run, which its
// owner validated.
func policyRelationships(client policy.Client, shared *SharedPolicyRun, schemas *schemarepo.Schemas, locks map[addrs.Provider]*depsfile.ProviderLock) (policy.Relationships, tfdiags.Diagnostics) {
	if shared != nil {
		return shared.Relationships, nil
	}
	return ValidatePolicyRelationships(client.Relationships(), schemas.Providers, func(provider addrs.Provider) string {
		if lock := locks[provider]; lock != nil {
			return lock.Version().String()
		}
		return ""
	})
}

// planPolicyRunOpts returns the relationship options of a plan walk, or nil
// if the walk relates no instances. It fails if the policy plugin's
// relationship definitions are invalid.
func (c *Context) planPolicyRunOpts(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*policyRunOpts, tfdiags.Diagnostics) {
	if opts == nil || opts.Query || opts.PolicyClient == nil {
		return nil, nil
	}
	schemas, diags := c.Schemas(config, prevRunState)
	if diags.HasErrors() {
		return nil, diags
	}
	rels, diags := policyRelationships(opts.PolicyClient, opts.PolicyRun, schemas, opts.ProviderLocks)
	if diags.HasErrors() || len(rels.Definitions) == 0 {
		return nil, diags
	}
	return &policyRunOpts{
		NormalPlan:             opts.Mode == plans.NormalMode,
		Schemas:                schemas,
		Relationships:          rels,
		OutsideConfigProviders: c.policyOutsideConfigProviders(opts.ExternalProviders),
		Shared:                 opts.PolicyRun,
	}, diags
}

// applyPolicyRunOpts returns the relationship options of the apply walk of
// a plan, or nil if the walk relates no instances. changes are the decoded
// changes of the plan, before the walk. It fails if the policy plugin's
// relationship definitions are invalid.
func (c *Context) applyPolicyRunOpts(schemas *schemarepo.Schemas, changes []*plans.ResourceInstanceChange, opts *ApplyOpts) (*policyRunOpts, tfdiags.Diagnostics) {
	if opts == nil || opts.PolicyClient == nil {
		return nil, nil
	}
	rels, diags := policyRelationships(opts.PolicyClient, opts.PolicyRun, schemas, opts.ProviderLocks)
	if diags.HasErrors() || len(rels.Definitions) == 0 {
		return nil, diags
	}
	return &policyRunOpts{
		Apply:                  true,
		Schemas:                schemas,
		Relationships:          rels,
		AppliedChanges:         append([]*plans.ResourceInstanceChange(nil), changes...),
		OutsideConfigProviders: c.policyOutsideConfigProviders(opts.ExternalProviders),
		Shared:                 opts.PolicyRun,
	}, diags
}

// policyStateValue decodes a state object with the current schema of its
// resource type and marks its sensitive values, both those recorded in state
// and those the schema declares.
func policyStateValue(obj *states.ResourceInstanceObjectSrc, schema providers.Schema) (cty.Value, error) {
	if schema.Body == nil {
		return cty.NilVal, fmt.Errorf("no schema")
	}
	if obj.SchemaVersion != uint64(schema.Version) {
		return cty.NilVal, fmt.Errorf("the object has schema version %d, but the current schema version is %d", obj.SchemaVersion, schema.Version)
	}
	decoded, err := obj.Decode(schema)
	if err != nil {
		return cty.NilVal, err
	}
	return policyMarkSchemaSensitive(decoded.Value, schema.Body), nil
}

// policyMarkSchemaSensitive marks the values of val that the schema declares
// sensitive, keeping its existing marks.
func policyMarkSchemaSensitive(val cty.Value, block *configschema.Block) cty.Value {
	if val == cty.NilVal || block == nil {
		return val
	}
	unmarked, pvms := val.UnmarkDeepWithPaths()
	for _, path := range block.SensitivePaths(unmarked, nil) {
		pvms = append(pvms, cty.PathValueMarks{Path: path, Marks: cty.NewValueMarks(marks.Sensitive)})
	}
	return unmarked.MarkWithPaths(pvms)
}

// relationshipOriginLookup implements originLookup with the walk's schemas
// and planned changes.
type relationshipOriginLookup struct {
	schemas *Schemas
	// planned are the unmarked planned values of the walk's managed
	// resource instance changes, by address.
	planned map[string]cty.Value
	// overridden reports whether a module instance is overridden by the
	// walk's test overrides; nil without overrides.
	overridden func(addrs.ModuleInstance) bool
}

var _ originLookup = (*relationshipOriginLookup)(nil)

func (l *relationshipOriginLookup) attrType(provider addrs.Provider, resType string, path []originStep) (cty.Type, bool) {
	schema := l.schemas.ResourceTypeConfig(provider, addrs.ManagedResourceMode, resType)
	if schema.Body == nil || len(path) == 0 || path[0].name == "" {
		return cty.NilType, false
	}
	ty := schema.Body.ImpliedType()
	// The attribute steps alone find the attribute in the schema, because
	// AttributeByPath doesn't step into the elements of an attribute.
	ctyPath := make(cty.Path, 0, len(path))
	for _, step := range path {
		switch {
		case step.name != "":
			if !ty.IsObjectType() || !ty.HasAttribute(step.name) {
				return cty.NilType, false
			}
			ty = ty.AttributeType(step.name)
			ctyPath = ctyPath.GetAttr(step.name)
		case ty.IsListType():
			ty = ty.ElementType()
		case ty.IsTupleType() && step.index < int64(len(ty.TupleElementTypes())):
			ty = ty.TupleElementTypes()[step.index]
		default:
			return cty.NilType, false
		}
	}
	if !ty.IsPrimitiveType() {
		return cty.NilType, false
	}
	if attr := schema.Body.AttributeByPath(ctyPath); attr == nil || attr.WriteOnly {
		return cty.NilType, false
	}
	return ty, true
}

func (l *relationshipOriginLookup) attrComputed(provider addrs.Provider, resType string, path []string) (bool, bool) {
	schema := l.schemas.ResourceTypeConfig(provider, addrs.ManagedResourceMode, resType)
	if schema.Body == nil || len(path) == 0 {
		return false, false
	}
	ctyPath := make(cty.Path, 0, len(path))
	for _, name := range path {
		ctyPath = ctyPath.GetAttr(name)
	}
	attr := schema.Body.AttributeByPath(ctyPath)
	if attr == nil {
		return false, false
	}
	return attr.Computed, true
}

func (l *relationshipOriginLookup) plannedValue(addr addrs.AbsResourceInstance, path []originStep) (cty.Value, bool) {
	val, ok := l.planned[addr.String()]
	if !ok {
		return cty.NilVal, false
	}
	for _, step := range path {
		if !val.IsKnown() {
			return cty.NilVal, false
		}
		if val.IsNull() {
			return val, true
		}
		ty := val.Type()
		switch {
		case step.name != "":
			if !ty.IsObjectType() || !ty.HasAttribute(step.name) {
				return cty.NilVal, false
			}
			val = val.GetAttr(step.name)
		case ty.IsListType() || ty.IsTupleType():
			if step.index >= int64(val.LengthInt()) {
				return cty.NullVal(cty.DynamicPseudoType), true
			}
			val = val.Index(cty.NumberIntVal(step.index))
		default:
			return cty.NilVal, false
		}
	}
	if !val.IsKnown() {
		return cty.NilVal, false
	}
	return val, true
}

// policyOverriddenModules returns whether module instances are overridden
// by the given test overrides, or nil if there are no overrides. Overridden
// resources don't need excluding from origins: overrides only fill in their
// computed attributes, so their configured attributes still come from their
// configuration.
func policyOverriddenModules(overrides *mocking.Overrides) func(addrs.ModuleInstance) bool {
	if overrides.Empty() {
		return nil
	}
	return overrides.IsOverridden
}

// policyCollector collects the instances of a walk that relationships
// can relate.
type policyCollector struct {
	ctx EvalContext
	ps  *policySubgraph

	// keyPaths are the key paths of each type of the relationships.
	keyPaths map[policyTypeKey][][]string
	// origins is the lookup for origins in a plan; nil in an apply.
	origins *relationshipOriginLookup

	inst *PolicyInstances
}

// collectPolicyInstances collects the instances of the walk that
// relationships can relate (contract §3.4), with their origins in a plan,
// and returns them with the origin lookup of a plan. It must be called after
// the walk's changes and state are final.
func collectPolicyInstances(ctx EvalContext, ps *policySubgraph) (*PolicyInstances, *relationshipOriginLookup) {
	run := ps.run
	c := &policyCollector{
		ctx:      ctx,
		ps:       ps,
		keyPaths: policyKeyPathsByType(run.Relationships),
		inst: &PolicyInstances{
			errored:   make(map[policyTypeKey]bool),
			deferred:  make(map[policyTypeKey]map[string]*PolicyProvider),
			Providers: make(map[string]*PolicyProvider),
		},
	}

	var changes []*plans.ResourceInstanceChange
	if run.Apply {
		changes = run.AppliedChanges
	} else {
		changes = ctx.Changes().ResourceInstanceChanges()
		c.origins = &relationshipOriginLookup{
			schemas:    run.Schemas,
			planned:    make(map[string]cty.Value),
			overridden: policyOverriddenModules(ctx.Overrides()),
		}
		for _, change := range changes {
			if change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode || change.DeposedKey != states.NotDeposed || change.After == cty.NilVal {
				continue
			}
			after, _ := change.After.UnmarkDeep()
			c.origins.planned[change.Addr.String()] = after
		}
	}

	changed := make(map[string]struct{})
	for _, change := range changes {
		if change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode || change.DeposedKey != states.NotDeposed {
			continue
		}
		changed[change.Addr.String()] = struct{}{}
	}

	deferred := make(map[string]struct{})
	// deferredResources are the configuration resources with a deferred
	// change.
	deferredResources := make(map[string]struct{})
	if deferrals := ctx.Deferrals(); deferrals != nil {
		for _, d := range deferrals.GetDeferredChanges() {
			addr := d.Change.Addr
			if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			deferred[addr.String()] = struct{}{}
			deferredResources[addr.ConfigResource().String()] = struct{}{}
			key := policyTypeKey{source: d.Change.ProviderAddr.Provider.String(), typ: addr.Resource.Resource.Type}
			if _, ok := c.keyPaths[key]; !ok {
				continue
			}
			if c.inst.deferred[key] == nil {
				c.inst.deferred[key] = make(map[string]*PolicyProvider)
			}
			c.inst.deferred[key][addr.String()] = c.provider(d.Change.ProviderAddr)
		}
	}

	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	for _, change := range changes {
		if change.Addr.Resource.Resource.Mode == addrs.ManagedResourceMode {
			c.addChange(change, state)
		}
	}
	c.addState(state, changed, deferred)
	if run.NormalPlan {
		c.checkExpandedInstances(changed, deferred, deferredResources)
	}
	return c.inst, c.origins
}

// policyKeyPathsByType returns the key paths of each type of the
// relationships, which are the types of their definitions and usages.
func policyKeyPathsByType(rels policy.Relationships) map[policyTypeKey][][]string {
	ret := make(map[policyTypeKey][][]string)
	add := func(key policyTypeKey, paths ...[]string) {
		ret[key] = append(ret[key], paths...)
	}
	for _, def := range rels.Definitions {
		keyType := policyTypeKey{source: def.ProviderSource, typ: def.KeyResource}
		valueType := policyTypeKey{source: def.ProviderSource, typ: def.ValueResource}
		add(keyType)
		add(valueType)
		for _, alt := range def.Match {
			for _, key := range alt.Keys {
				add(keyType, policyKeyPathNames(key.KeyPath))
				add(valueType, policyKeyPathNames(key.ValuePath))
			}
		}
	}
	for _, usage := range rels.Usages {
		add(policyTypeKey{source: usage.ProviderSource, typ: usage.Type})
	}
	return ret
}

// provider returns the provider configuration of the walk at addr: the value
// the walk configured it with, or its static value if the walk didn't
// configure it (contract §3.5).
func (c *policyCollector) provider(addr addrs.AbsProviderConfig) *PolicyProvider {
	key := addr.String()
	if p, ok := c.inst.Providers[key]; ok {
		return p
	}
	cfg, ok := c.ps.providerConfig(key)
	if !ok {
		outside := addr.Module.IsRoot() && c.ps.run.OutsideConfigProviders[key]
		cfg = policyStaticProviderConfig(c.ctx.Config(), c.ps.run.Schemas, addr, outside)
	}
	p := &PolicyProvider{Addr: key, Config: policyKnownConfig(cfg)}
	c.inst.Providers[key] = p
	return p
}

// policyKnownConfig returns a provider configuration value without marks if
// it's wholly known, and cty.NilVal otherwise.
func policyKnownConfig(cfg cty.Value) cty.Value {
	if cfg == cty.NilVal {
		return cty.NilVal
	}
	cfg, _ = cfg.UnmarkDeep()
	if !cfg.IsWhollyKnown() {
		return cty.NilVal
	}
	return cfg
}

// policyStaticProviderConfig returns the value of a provider configuration
// decoded without an evaluation context. For a configuration that is wholly
// literal, that is the value the walk would configure the provider with.
//
// The configuration is found like attachProviderConfigs finds it, and is
// empty without a provider block, like in buildProviderConfig. It returns
// cty.NilVal if the value has parts from outside the configuration (outside
// is true: interactive input values, or the caller configured the provider),
// the module isn't in the configuration, the provider has no configuration
// schema, or decoding fails, e.g. because the configuration refers to
// anything or calls functions.
func policyStaticProviderConfig(cfg *configs.Config, schemas *schemarepo.Schemas, addr addrs.AbsProviderConfig, outside bool) cty.Value {
	if cfg == nil || schemas == nil || outside {
		return cty.NilVal
	}
	mc := cfg.Descendant(addr.Module)
	if mc == nil {
		return cty.NilVal
	}
	schema := schemas.ProviderConfig(addr.Provider)
	if schema == nil {
		return cty.NilVal
	}
	body := hcl.EmptyBody()
	localName := mc.Module.LocalNameForProvider(addr.Provider)
	for _, p := range mc.Module.ProviderConfigs {
		if p.Name == localName && p.Alias == addr.Alias {
			if p.Config != nil {
				body = p.Config
			}
			break
		}
	}
	val, diags := hcldec.Decode(blocktoattr.FixUpBlockAttrs(body, schema), schema.DecoderSpec(), &hcl.EvalContext{})
	if diags.HasErrors() {
		return cty.NilVal
	}
	return val
}

func policyKeyPathNames(path *proto.AttributePath) []string {
	names := make([]string, 0, len(path.GetSteps()))
	for _, step := range path.GetSteps() {
		names = append(names, step.GetAttributeName())
	}
	return names
}

func (c *policyCollector) schema(provider addrs.Provider, typeName string) providers.Schema {
	return c.ps.run.Schemas.ResourceTypeConfig(provider, addrs.ManagedResourceMode, typeName)
}

func (c *policyCollector) newObject(key policyTypeKey, addr addrs.AbsResourceInstance, providerAddr addrs.AbsProviderConfig) *policyObject {
	return &policyObject{
		addr:         addr.String(),
		typ:          key,
		modulePath:   addr.Module.String(),
		providerType: providerAddr.Provider.Type,
		provider:     c.provider(providerAddr),
	}
}

// encode encodes a value of an instance of the given type. If that fails,
// the type is ERROR.
func (c *policyCollector) encode(key policyTypeKey, addr addrs.AbsResourceInstance, kind string, deposed states.DeposedKey, val cty.Value) (*proto.ResourceAttributes, bool) {
	attrs, err := c.ctx.PolicyClient().ResourceAttributes(val, c.ps.valueSource(kind, deposed, addr))
	if err != nil {
		log.Printf("[WARN] policy: failed to encode %s for relationships: %s", addr, err)
		c.inst.errored[key] = true
		return nil, false
	}
	return attrs, true
}

// add adds an object whose keys are read from val.
func (c *policyCollector) add(obj *policyObject, val cty.Value) {
	obj.value, _ = val.UnmarkDeep()
	c.inst.objects = append(c.inst.objects, obj)
}

// addChange adds the objects of a change of a current or deposed object: the
// object it deletes, if any, and the object it leaves, if any (contract
// §3.4). Forgotten objects aren't added, and nor are deposed objects whose
// change is a no-op, because they are already gone.
func (c *policyCollector) addChange(change *plans.ResourceInstanceChange, state *states.State) {
	addr := change.Addr
	key := policyTypeKey{source: change.ProviderAddr.Provider.String(), typ: addr.Resource.Resource.Type}
	if _, ok := c.keyPaths[key]; !ok {
		return
	}
	deposed := change.DeposedKey != states.NotDeposed
	if change.Action == plans.Forget || (deposed && change.Action != plans.Delete) {
		return
	}
	schema := c.schema(change.ProviderAddr.Provider, addr.Resource.Resource.Type)
	if schema.Body == nil {
		log.Printf("[WARN] policy: no schema for %s in relationships", addr)
		c.inst.errored[key] = true
		return
	}
	before := policyMarkSchemaSensitive(change.Before, schema.Body)

	var op proto.Operation
	switch change.Action {
	case plans.Delete, plans.DeleteThenCreate, plans.CreateThenDelete:
		obj := c.newObject(key, addr, change.ProviderAddr)
		obj.deposed = change.DeposedKey.String()
		obj.old = change.Action != plans.Delete
		obj.op = proto.Operation_DELETE
		var ok bool
		if obj.priorAttrs, ok = c.encode(key, addr, callback.KindPrior, change.DeposedKey, before); ok {
			c.add(obj, before)
		}
		if change.Action == plans.Delete {
			return
		}
		op = proto.Operation_CREATE
	case plans.Create, plans.CreateThenForget, plans.ForgetThenCreate:
		op = proto.Operation_CREATE
	case plans.Update:
		op = proto.Operation_UPDATE
	case plans.NoOp:
		op = proto.Operation_NO_OP
	default:
		log.Printf("[WARN] policy: unexpected action %s for %s in relationships", change.Action, addr)
		c.inst.errored[key] = true
		return
	}

	obj := c.newObject(key, addr, change.ProviderAddr)
	obj.op = op
	var after cty.Value
	afterKind := callback.KindPlanned
	if c.ps.run.Apply {
		afterKind = callback.KindState
		// An applied change has the value the apply produced.
		var cur *states.ResourceInstanceObjectSrc
		if inst := state.ResourceInstance(addr); inst != nil {
			cur = inst.Current
		}
		if cur == nil {
			log.Printf("[DEBUG] policy: no object for %s after apply; its type is ERROR for relationships", addr)
			c.inst.errored[key] = true
			return
		}
		val, err := policyStateValue(cur, schema)
		if err != nil {
			log.Printf("[WARN] policy: failed to decode %s for relationships: %s", addr, err)
			c.inst.errored[key] = true
			return
		}
		after = val
	} else {
		after = policyMarkSchemaSensitive(change.After, schema.Body)
	}
	var ok bool
	if obj.attrs, ok = c.encode(key, addr, afterKind, states.NotDeposed, after); !ok {
		return
	}
	if op != proto.Operation_CREATE {
		if obj.priorAttrs, ok = c.encode(key, addr, callback.KindPrior, states.NotDeposed, before); !ok {
			return
		}
	}
	if c.origins != nil {
		configured, _ := c.ps.resourceConfigValue(addr)
		for path, ko := range originsFor(c.ctx.Config(), c.ctx.InstanceExpander(), c.origins.overridden, addr, change.Action, change.After, configured, c.keyPaths[key], c.origins, c.ps.run.Shared != nil) {
			if obj.origins == nil {
				obj.origins = make(map[string][]PolicyOrigin)
			}
			obj.origins[path] = ko.origins
			if len(ko.vars) > 0 {
				if obj.vars == nil {
					obj.vars = make(map[string][]PolicyVariableOrigin)
				}
				obj.vars[path] = ko.vars
			}
		}
	}
	c.add(obj, after)
}

// addState adds a NO_OP object for every current object in the state whose
// resource is in the configuration and that has neither a change nor a
// deferral. These are the instances the getresources callback reads at apply
// with states.ReadEachConfigResourceInstance, including instances whose key
// or module instance key no longer exists in the configuration, but one
// object per instance: that helper returns the current object once per
// object of the instance.
func (c *policyCollector) addState(state *states.State, changed, deferred map[string]struct{}) {
	cfg := c.ctx.Config()
	if cfg == nil {
		return
	}
	for _, ms := range state.Modules {
		modCfg := cfg.DescendantForInstance(ms.Addr)
		if modCfg == nil {
			continue
		}
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			key := policyTypeKey{source: rs.ProviderConfig.Provider.String(), typ: rs.Addr.Resource.Type}
			if _, ok := c.keyPaths[key]; !ok || modCfg.Module.ResourceByAddr(rs.Addr.Resource) == nil {
				continue
			}
			schema := c.schema(rs.ProviderConfig.Provider, rs.Addr.Resource.Type)
			for instKey, inst := range rs.Instances {
				if inst.Current == nil {
					continue
				}
				addr := rs.Addr.Instance(instKey)
				if _, ok := changed[addr.String()]; ok {
					continue
				}
				if _, ok := deferred[addr.String()]; ok {
					continue
				}
				val, err := policyStateValue(inst.Current, schema)
				if err != nil {
					log.Printf("[WARN] policy: failed to decode %s for relationships: %s", addr, err)
					c.inst.errored[key] = true
					continue
				}
				obj := c.newObject(key, addr, rs.ProviderConfig)
				obj.op = proto.Operation_NO_OP
				var ok bool
				if obj.attrs, ok = c.encode(key, addr, callback.KindState, states.NotDeposed, val); ok {
					c.add(obj, val)
				}
			}
		}
	}
}

// checkExpandedInstances makes a type ERROR if the instance expander knows
// an instance of a configured resource of that type that has neither a
// change nor a deferral, e.g. because planning it failed, or if a configured
// resource of that type has instance keys that aren't known and no deferred
// change.
func (c *policyCollector) checkExpandedInstances(changed, deferred, deferredResources map[string]struct{}) {
	cfg := c.ctx.Config()
	exp := c.ctx.InstanceExpander()
	if cfg == nil || exp == nil {
		return
	}
	known := exp.AllInstances()
	cfg.DeepEach(func(mc *configs.Config) {
		for _, r := range mc.Module.ManagedResources {
			provider := mc.Module.ProviderForLocalConfig(r.ProviderConfigAddr())
			key := policyTypeKey{source: provider.String(), typ: r.Type}
			if _, ok := c.keyPaths[key]; !ok || c.inst.errored[key] {
				continue
			}
			for _, modInst := range known.InstancesForModule(mc.Path, false) {
				absRes := r.Addr().Absolute(modInst)
				if !known.HasResource(absRes) {
					continue
				}
				_, keys, unknownKeys := exp.ResourceInstanceKeys(absRes)
				if _, ok := deferredResources[absRes.Config().String()]; unknownKeys && !ok {
					log.Printf("[DEBUG] policy: %s has unknown instance keys and no deferred change; its type is ERROR for relationships", absRes)
					c.inst.errored[key] = true
				}
				for _, instKey := range keys {
					addr := absRes.Instance(instKey).String()
					_, hasChange := changed[addr]
					_, isDeferred := deferred[addr]
					if !hasChange && !isDeferred {
						log.Printf("[DEBUG] policy: %s has no planned change; its type is ERROR for relationships", addr)
						c.inst.errored[key] = true
					}
				}
			}
		}
	})
}

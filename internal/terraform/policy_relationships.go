// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/callback"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// policyRunOpts describes the policy run of a plan or apply walk: the
// relationship trees whose instances the walk relates, and in an apply the
// applied changes, which say which object of an instance a value is.
type policyRunOpts struct {
	Apply bool
	// NormalPlan is set in a plan in the normal mode.
	NormalPlan bool
	Schemas    *schemarepo.Schemas

	// Relationships are the policy plugin's validated relationship trees.
	// The walk relates instances only if there are any.
	Relationships policy.Relationships

	// AppliedChanges is set in an apply only: the applied plan's resource
	// instance changes, copied before the walk because the apply walk removes
	// changes from the working changes as it applies them.
	AppliedChanges []*plans.ResourceInstanceChange
}

// relating returns true if the walk relates instances.
func (r *policyRunOpts) relating() bool {
	return r != nil && len(r.Relationships.Trees) > 0
}

// policyRelationships returns the relationship trees of a walk: those the
// caller validated, if any, or the client's, validated against the walk's
// schemas (contract §3.3).
func policyRelationships(client policy.Client, validated *policy.Relationships, schemas *schemarepo.Schemas, locks map[addrs.Provider]*depsfile.ProviderLock) (policy.Relationships, tfdiags.Diagnostics) {
	if validated != nil {
		return *validated, nil
	}
	rels := client.Relationships()
	if len(rels.Trees) == 0 {
		return rels, nil
	}
	return rels, ValidatePolicyRelationships(rels, schemas.Providers, func(provider addrs.Provider) string {
		if lock := locks[provider]; lock != nil {
			return lock.Version().String()
		}
		return ""
	})
}

// planPolicyRunOpts returns the policy run of a plan walk, or nil if the
// walk relates no instances. It fails if the policy plugin's relationship
// definitions are invalid.
func (c *Context) planPolicyRunOpts(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*policyRunOpts, tfdiags.Diagnostics) {
	if opts == nil || opts.Query || opts.PolicyClient == nil {
		return nil, nil
	}
	if opts.PolicyRelationships == nil && len(opts.PolicyClient.Relationships().Trees) == 0 {
		return nil, nil
	}
	schemas, diags := c.Schemas(config, prevRunState)
	if diags.HasErrors() {
		return nil, diags
	}
	rels, diags := policyRelationships(opts.PolicyClient, opts.PolicyRelationships, schemas, opts.ProviderLocks)
	if diags.HasErrors() || len(rels.Trees) == 0 {
		return nil, diags
	}
	return &policyRunOpts{
		NormalPlan:    opts.Mode == plans.NormalMode,
		Schemas:       schemas,
		Relationships: rels,
	}, diags
}

// applyPolicyRunOpts returns the policy run of the apply walk of a plan, or
// nil without a policy client. changes are the decoded changes of the plan,
// before the walk. It fails if the policy plugin's relationship definitions
// are invalid.
func (c *Context) applyPolicyRunOpts(schemas *schemarepo.Schemas, changes []*plans.ResourceInstanceChange, opts *ApplyOpts) (*policyRunOpts, tfdiags.Diagnostics) {
	if opts == nil || opts.PolicyClient == nil {
		return nil, nil
	}
	rels, diags := policyRelationships(opts.PolicyClient, opts.PolicyRelationships, schemas, opts.ProviderLocks)
	if diags.HasErrors() {
		return nil, diags
	}
	return &policyRunOpts{
		Apply:          true,
		Schemas:        schemas,
		Relationships:  rels,
		AppliedChanges: append([]*plans.ResourceInstanceChange(nil), changes...),
	}, diags
}

// policyStateValue decodes a state object with the current schema of its
// resource type, with the sensitive values recorded in state marked. The
// policy client also redacts those the schema declares.
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
	return decoded.Value, nil
}

// policyChangeOperation returns the operation of the object that a change
// leaves: the new object of a replace is a CREATE. ok is false for changes
// that leave no object.
func policyChangeOperation(action plans.Action) (op proto.Operation, ok bool) {
	switch action {
	case plans.Create, plans.CreateThenForget, plans.ForgetThenCreate, plans.DeleteThenCreate, plans.CreateThenDelete:
		return proto.Operation_CREATE, true
	case plans.Update:
		return proto.Operation_UPDATE, true
	case plans.NoOp:
		return proto.Operation_NO_OP, true
	default:
		return 0, false
	}
}

// policyObjectSpec says which object of a resource instance a value is.
type policyObjectSpec struct {
	addr     addrs.AbsResourceInstance
	provider addrs.AbsProviderConfig
	op       proto.Operation
	deposed  states.DeposedKey
	// old is set for the object that a replace deletes.
	old bool
}

// callbackObject returns the object of the walk's policy phase with the
// given values, cty.NilVal for those it doesn't have (contract §3.4).
func (ps *policySubgraph) callbackObject(spec policyObjectSpec, attrs, prior cty.Value, schema *configschema.Block) *callback.Object {
	addr := spec.addr.String()
	return &callback.Object{
		Generation: ps.generation,
		Address:    addr,
		DeposedKey: spec.deposed.String(),
		OldObject:  spec.old,
		Type:       spec.addr.Resource.Resource.Type,
		Metadata: &proto.PolicyEvaluateResourceRequest_ResourceMetadata{
			ProviderType:   spec.provider.Provider.Type,
			Operation:      spec.op,
			ModulePath:     spec.addr.Module.String(),
			Address:        addr,
			ProviderSource: spec.provider.Provider.String(),
			DeposedKey:     spec.deposed.String(),
			OldObject:      spec.old,
		},
		Attrs:      attrs,
		PriorAttrs: prior,
		Schema:     schema,
	}
}

// policyCollector collects the instances of a walk that relationships
// can relate.
type policyCollector struct {
	ctx EvalContext
	ps  *policySubgraph

	// keyPaths are the key paths of each type of the relationships.
	keyPaths map[policyTypeKey][][]string
	// origins resolves origins in a plan; nil in an apply.
	origins *policyOriginResolver

	r *policyRelations
}

// collectPolicyRelations collects the instances of the walk that
// relationships can relate (contract §3.4), with their origins in a plan, and
// returns their relations. It must be called after the walk's changes and
// state are final, in the walk's policy phase.
func collectPolicyRelations(ctx EvalContext, ps *policySubgraph) *policyRelations {
	run := ps.run
	c := &policyCollector{
		ctx:      ctx,
		ps:       ps,
		keyPaths: policyKeyPathsByType(run.Relationships),
		r:        newPolicyRelations(run.Relationships.Trees, !run.Apply),
	}

	var changes []*plans.ResourceInstanceChange
	if run.Apply {
		changes = run.AppliedChanges
	} else {
		changes = ctx.Changes().ResourceInstanceChanges()
		c.origins = &policyOriginResolver{
			cfg:        ctx.Config(),
			schemas:    run.Schemas,
			overridden: policyOverriddenModules(ctx.Overrides()),
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
			if c.r.deferred[key] == nil {
				c.r.deferred[key] = make(map[string]string)
			}
			c.r.deferred[key][addr.String()] = d.Change.ProviderAddr.String()
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
	c.r.finish(c.keyPaths)
	return c.r
}

// policyKeyPathsByType returns the key paths of each type of the
// relationship trees: the types of their subjects and definitions.
func policyKeyPathsByType(rels policy.Relationships) map[policyTypeKey][][]string {
	ret := make(map[policyTypeKey][][]string)
	seen := make(map[policyTypeKey]map[string]bool)
	add := func(key policyTypeKey, path *proto.AttributePath) {
		if _, ok := ret[key]; !ok {
			ret[key] = nil
			seen[key] = make(map[string]bool)
		}
		if path == nil {
			return
		}
		names := policyKeyPathNames(path)
		if joined := strings.Join(names, "."); !seen[key][joined] {
			seen[key][joined] = true
			ret[key] = append(ret[key], names)
		}
	}
	var walk func(defs []*proto.Relationship)
	walk = func(defs []*proto.Relationship) {
		for _, def := range defs {
			keyType := policyTypeKey{source: def.ProviderSource, typ: def.KeyType}
			valueType := policyTypeKey{source: def.ProviderSource, typ: def.ValueType}
			add(keyType, nil)
			add(valueType, nil)
			for _, match := range def.Matches {
				for _, key := range match.Keys {
					add(keyType, key.KeyPath)
					add(valueType, key.ValuePath)
				}
			}
			walk(def.Relationships)
		}
	}
	for _, tree := range rels.Trees {
		add(policyTypeKey{source: tree.ProviderSource, typ: tree.ResourceType}, nil)
		walk(tree.Relationships)
	}
	return ret
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

// add registers an object with the policy client, and adds it, with keys
// read from val and the given origins. If the registration fails, the
// object's type is ERROR.
func (c *policyCollector) add(key policyTypeKey, spec policyObjectSpec, attrs, prior cty.Value, schema *configschema.Block, origins map[string][]string) {
	obj := c.ps.callbackObject(spec, attrs, prior, schema)
	id, err := c.ctx.PolicyClient().RegisterObject(obj)
	if err != nil {
		log.Printf("[WARN] policy: failed to encode %s for relationships: %s", spec.addr, err)
		c.r.errored[key] = true
		return
	}
	val := attrs
	if spec.op == proto.Operation_DELETE {
		val = prior
	}
	val, _ = val.UnmarkDeep()
	c.r.add(&policyObject{
		id:       id,
		addr:     spec.addr.String(),
		typ:      key,
		op:       spec.op,
		provider: spec.provider.String(),
		value:    val,
		origins:  origins,
	})
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
		c.r.errored[key] = true
		return
	}
	before := change.Before

	switch change.Action {
	case plans.Delete, plans.DeleteThenCreate, plans.CreateThenDelete:
		spec := policyObjectSpec{
			addr:     addr,
			provider: change.ProviderAddr,
			op:       proto.Operation_DELETE,
			deposed:  change.DeposedKey,
			old:      change.Action != plans.Delete,
		}
		c.add(key, spec, cty.NilVal, before, schema.Body, nil)
		if change.Action == plans.Delete {
			return
		}
	}
	op, ok := policyChangeOperation(change.Action)
	if !ok {
		log.Printf("[WARN] policy: unexpected action %s for %s in relationships", change.Action, addr)
		c.r.errored[key] = true
		return
	}

	var after cty.Value
	if c.ps.run.Apply {
		// An applied change has the value the apply produced.
		var cur *states.ResourceInstanceObjectSrc
		if inst := state.ResourceInstance(addr); inst != nil {
			cur = inst.Current
		}
		if cur == nil {
			log.Printf("[DEBUG] policy: no object for %s after apply; its type is ERROR for relationships", addr)
			c.r.errored[key] = true
			return
		}
		val, err := policyStateValue(cur, schema)
		if err != nil {
			log.Printf("[WARN] policy: failed to decode %s for relationships: %s", addr, err)
			c.r.errored[key] = true
			return
		}
		after = val
	} else {
		after = change.After
	}
	prior := before
	if op == proto.Operation_CREATE {
		prior = cty.NilVal
	}
	var origins map[string][]string
	if c.origins != nil {
		configured, _ := c.ps.resourceConfigValue(addr)
		origins = c.origins.originsFor(addr, change.Action, change.After, configured, c.keyPaths[key])
	}
	spec := policyObjectSpec{addr: addr, provider: change.ProviderAddr, op: op}
	c.add(key, spec, after, prior, schema.Body, origins)
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
					c.r.errored[key] = true
					continue
				}
				spec := policyObjectSpec{addr: addr, provider: rs.ProviderConfig, op: proto.Operation_NO_OP}
				c.add(key, spec, val, cty.NilVal, schema.Body, nil)
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
			if _, ok := c.keyPaths[key]; !ok || c.r.errored[key] {
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
					c.r.errored[key] = true
				}
				for _, instKey := range keys {
					addr := absRes.Instance(instKey).String()
					_, hasChange := changed[addr]
					_, isDeferred := deferred[addr]
					if !hasChange && !isDeferred {
						log.Printf("[DEBUG] policy: %s has no planned change; its type is ERROR for relationships", addr)
						c.r.errored[key] = true
					}
				}
			}
		}
	})
}

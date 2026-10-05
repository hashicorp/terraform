// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// policyRunOpts describes the relationship run of a plan or apply walk. A walk
// has a run only when the policy plugin announced the relationships
// capability.
type policyRunOpts struct {
	Stage    proto.EvaluationStage
	PlanMode proto.PlanMode
	Targeted bool
	Schemas  *schemarepo.Schemas

	// AppliedChanges is set for apply runs only: the applied plan's resource
	// instance changes, copied before the walk because the apply walk removes
	// changes from the working changes as it applies them.
	AppliedChanges []*plans.ResourceInstanceChange
}

// policyRelationshipsClient returns the client as a relationships client if
// the policy plugin announced the relationships capability.
func policyRelationshipsClient(client policy.Client) (policy.RelationshipsClient, bool) {
	if client == nil {
		return nil, false
	}
	rc, ok := client.(policy.RelationshipsClient)
	if !ok || !rc.RelationshipsSupported() {
		return nil, false
	}
	return rc, true
}

func policyPlanMode(mode plans.Mode) (proto.PlanMode, bool) {
	switch mode {
	case plans.NormalMode:
		return proto.PlanMode_NORMAL_PLAN_MODE, true
	case plans.DestroyMode:
		return proto.PlanMode_DESTROY_PLAN_MODE, true
	case plans.RefreshOnlyMode:
		return proto.PlanMode_REFRESH_ONLY_PLAN_MODE, true
	default:
		return proto.PlanMode_INVALID_PLAN_MODE, false
	}
}

// planPolicyRunOpts returns the relationship run options of a plan walk, or
// nil if the walk has no relationship run.
func (c *Context) planPolicyRunOpts(config *configs.Config, prevRunState *states.State, opts *PlanOpts) *policyRunOpts {
	if opts == nil || opts.Query {
		return nil
	}
	if _, ok := policyRelationshipsClient(opts.PolicyClient); !ok {
		return nil
	}
	mode, ok := policyPlanMode(opts.Mode)
	if !ok {
		return nil
	}
	schemas, diags := c.Schemas(config, prevRunState)
	if diags.HasErrors() {
		log.Printf("[WARN] policy: not starting a relationship run because the provider schemas are unavailable: %s", diags.Err())
		return nil
	}
	return &policyRunOpts{
		Stage:    proto.EvaluationStage_PLAN_EVALUATION_STAGE,
		PlanMode: mode,
		Targeted: len(opts.Targets) > 0 || len(opts.ActionTargets) > 0,
		Schemas:  schemas,
	}
}

// applyPolicyRunOpts returns the relationship run options of the apply walk
// of the given plan, or nil if the walk has no relationship run. changes are
// the decoded changes of the plan, before the walk.
func applyPolicyRunOpts(plan *plans.Plan, schemas *schemarepo.Schemas, changes []*plans.ResourceInstanceChange, client policy.Client) *policyRunOpts {
	if _, ok := policyRelationshipsClient(client); !ok {
		return nil
	}
	mode, ok := policyPlanMode(plan.UIMode)
	if !ok {
		return nil
	}
	return &policyRunOpts{
		Stage:          proto.EvaluationStage_APPLY_EVALUATION_STAGE,
		PlanMode:       mode,
		Targeted:       len(plan.TargetAddrs) > 0 || len(plan.ActionTargetAddrs) > 0,
		Schemas:        schemas,
		AppliedChanges: append([]*plans.ResourceInstanceChange(nil), changes...),
	}
}

// startRelationshipRun begins the relationship run of the walk and reports
// its instances. It must be called after the walk's changes and state are
// final and before any subject is evaluated.
//
// It returns the diagnostics of BeginRun that the walk reports; see
// policyBeginRunDiagnostics. If they have errors, the run didn't begin: no run
// id is set, no instances are reported, and the caller must not evaluate
// policies. Failed RPCs don't fail the walk: they are logged and leave the run
// without a run id (when BeginRun fails) or with only part of its instances
// reported.
func (ps *policySubgraph) startRelationshipRun(ctx EvalContext, stopCtx context.Context) tfdiags.Diagnostics {
	client, ok := policyRelationshipsClient(ctx.PolicyClient())
	if !ok || ps.run == nil {
		return nil
	}

	runID, err := uuid.GenerateUUID()
	if err != nil {
		log.Printf("[WARN] policy: not starting a relationship run: failed to generate a run id: %s", err)
		return nil
	}

	resp, err := client.BeginRun(stopCtx, &proto.BeginRunRequest{
		RunId:           runID,
		ProviderSchemas: policyProviderSchemas(ps.run.Schemas),
		Stage:           ps.run.Stage,
		PlanMode:        ps.run.PlanMode,
		Runtime:         proto.RunRuntime_CLI_RUN_RUNTIME,
		Targeted:        ps.run.Targeted,
	})
	if err != nil {
		log.Printf("[WARN] policy: failed to begin relationship run %s: %s", runID, err)
		return nil
	}
	logPolicyRunDiagnostics("BeginRun", runID, resp.GetDiagnostics())
	diags := policyBeginRunDiagnostics(resp.GetDiagnostics())
	if diags.HasErrors() {
		log.Printf("[WARN] policy: relationship run %s did not begin: BeginRun returned errors", runID)
		return diags
	}

	ps.lock.Lock()
	ps.runID = runID
	ps.lock.Unlock()

	records, statuses, providers := collectRelationshipBatch(ctx, ps, resp.GetSpec())
	for _, req := range chunkRelationshipBatch(records, statuses, providers, runID) {
		resp, err := client.ReportInstances(stopCtx, req)
		if err != nil {
			log.Printf("[WARN] policy: failed to report instances for relationship run %s: %s", runID, err)
			return diags
		}
		logPolicyRunDiagnostics("ReportInstances", runID, resp.GetDiagnostics())
	}
	return diags
}

// finishRelationshipRun finishes the relationship run of the walk, if it has
// one.
func (ps *policySubgraph) finishRelationshipRun(ctx EvalContext, stopCtx context.Context) {
	runID := ps.RunID()
	if runID == "" {
		return
	}
	client, ok := policyRelationshipsClient(ctx.PolicyClient())
	if !ok {
		return
	}
	resp, err := client.FinishRun(stopCtx, &proto.FinishRunRequest{RunId: runID})
	if err != nil {
		log.Printf("[WARN] policy: failed to finish relationship run %s: %s", runID, err)
		return
	}
	logPolicyRunDiagnostics("FinishRun", runID, resp.GetDiagnostics())
}

// policyBeginRunDiagnostics returns the diagnostics of a BeginRun response
// that the walk reports, with their policy information.
//
// Warnings are about wrong relationship definitions that no policy uses.
// Errors are about wrong definitions that policies use, and mean that the run
// didn't begin. Both are reported once per walk.
func policyBeginRunDiagnostics(diags []*proto.Diagnostic) tfdiags.Diagnostics {
	var reported []*proto.Diagnostic
	for _, diag := range diags {
		// Diagnostics without a severity would become errors when converted,
		// so the filter uses the engine's severities.
		switch diag.GetSeverity() {
		case proto.Severity_WARNING, proto.Severity_ERROR:
			reported = append(reported, diag)
		}
	}
	return policy.DiagsFromProto(reported, nil).AsTerraformDiags()
}

func logPolicyRunDiagnostics(call, runID string, diags []*proto.Diagnostic) {
	for _, diag := range diags {
		level := "DEBUG"
		switch diag.GetSeverity() {
		case proto.Severity_ERROR, proto.Severity_WARNING:
			level = "WARN"
		}
		log.Printf("[%s] policy: %s for relationship run %s: %s: %s", level, call, runID, diag.GetSummary(), diag.GetDetail())
	}
}

// The limits of a single ReportInstances request. They are variables so that
// tests can lower them.
var (
	relationshipChunkMaxRecords = 1000
	relationshipChunkMaxBytes   = 3 << 20
)

// chunkRelationshipBatch splits a relationship batch into ReportInstances
// requests of at most relationshipChunkMaxRecords records and
// relationshipChunkMaxBytes bytes of records each; a single record over the
// byte limit is sent alone. All providers are in the first request, all
// statuses in the last, and there is always at least one request.
func chunkRelationshipBatch(records []*proto.InstanceRecord, statuses []*proto.TypeStatus, providers []*proto.ProviderInstance, runID string) []*proto.ReportInstancesRequest {
	var reqs []*proto.ReportInstancesRequest
	current := &proto.ReportInstancesRequest{RunId: runID, Providers: providers}
	currentBytes := 0
	for _, record := range records {
		size := protobuf.Size(record)
		full := len(current.Records) >= relationshipChunkMaxRecords || currentBytes+size > relationshipChunkMaxBytes
		if len(current.Records) > 0 && full {
			reqs = append(reqs, current)
			current = &proto.ReportInstancesRequest{RunId: runID}
			currentBytes = 0
		}
		current.Records = append(current.Records, record)
		currentBytes += size
	}
	current.Statuses = statuses
	return append(reqs, current)
}

// policyResourceAction returns the record action for a resource instance
// change action, or false if the action can't be the action of a managed
// resource instance record.
func policyResourceAction(action plans.Action) (proto.ResourceAction, bool) {
	switch action {
	case plans.NoOp:
		return proto.ResourceAction_NO_OP_RESOURCE_ACTION, true
	case plans.Create:
		return proto.ResourceAction_CREATE_RESOURCE_ACTION, true
	case plans.Update:
		return proto.ResourceAction_UPDATE_RESOURCE_ACTION, true
	case plans.Delete:
		return proto.ResourceAction_DELETE_RESOURCE_ACTION, true
	case plans.DeleteThenCreate:
		return proto.ResourceAction_DELETE_THEN_CREATE_RESOURCE_ACTION, true
	case plans.CreateThenDelete:
		return proto.ResourceAction_CREATE_THEN_DELETE_RESOURCE_ACTION, true
	case plans.Forget:
		return proto.ResourceAction_FORGET_RESOURCE_ACTION, true
	case plans.CreateThenForget:
		return proto.ResourceAction_CREATE_THEN_FORGET_RESOURCE_ACTION, true
	case plans.ForgetThenCreate:
		return proto.ResourceAction_FORGET_THEN_CREATE_RESOURCE_ACTION, true
	case plans.Read, plans.Open, plans.Renew, plans.Close:
		return proto.ResourceAction_INVALID_RESOURCE_ACTION, false
	default:
		return proto.ResourceAction_INVALID_RESOURCE_ACTION, false
	}
}

// policyRecordHasAttrs returns true if a record of a change with the given
// action has attrs.
func policyRecordHasAttrs(action plans.Action) bool {
	return action != plans.Delete && action != plans.Forget
}

// policyRecordHasPriorAttrs returns true if a record of a change with the
// given action has prior attrs.
func policyRecordHasPriorAttrs(action plans.Action) bool {
	switch action {
	case plans.Delete, plans.Forget, plans.DeleteThenCreate, plans.CreateThenDelete, plans.CreateThenForget, plans.ForgetThenCreate:
		return true
	default:
		return false
	}
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

type relationshipTypeKey struct {
	source   string
	typeName string
}

// relationshipCollector collects the records and type statuses of a
// relationship run, as described by the policy plugin's collection spec.
type relationshipCollector struct {
	ctx EvalContext
	ps  *policySubgraph

	types      map[relationshipTypeKey]bool
	typeOrder  []relationshipTypeKey
	incomplete map[relationshipTypeKey]bool
	deferred   map[relationshipTypeKey]map[string]struct{}
	keyPaths   map[relationshipTypeKey][][]string
	// keep are the top-level attributes kept in the values of the records
	// of a type; nil for a type whose records aren't pruned.
	keep map[relationshipTypeKey]map[string]bool

	// origins is the lookup for origins in plan runs; nil if the run has
	// no origins.
	origins *relationshipOriginLookup

	records []*proto.InstanceRecord
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

func (l *relationshipOriginLookup) attrType(provider addrs.Provider, resType string, path []string) (cty.Type, bool) {
	schema := l.schemas.ResourceTypeConfig(provider, addrs.ManagedResourceMode, resType)
	if schema.Body == nil || len(path) == 0 {
		return cty.NilType, false
	}
	ty := schema.Body.ImpliedType()
	ctyPath := make(cty.Path, 0, len(path))
	for _, name := range path {
		if !ty.IsObjectType() || !ty.HasAttribute(name) {
			return cty.NilType, false
		}
		ty = ty.AttributeType(name)
		ctyPath = ctyPath.GetAttr(name)
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

func (l *relationshipOriginLookup) plannedValue(addr addrs.AbsResourceInstance, path []string) (cty.Value, bool) {
	val, ok := l.planned[addr.String()]
	if !ok {
		return cty.NilVal, false
	}
	for _, name := range path {
		if !val.IsKnown() {
			return cty.NilVal, false
		}
		if val.IsNull() {
			return val, true
		}
		if !val.Type().IsObjectType() || !val.Type().HasAttribute(name) {
			return cty.NilVal, false
		}
		val = val.GetAttr(name)
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

// collectRelationshipBatch collects the records, type statuses and provider
// instances of the walk's relationship run for the given spec. It must be
// called after the walk's changes and state are final.
func collectRelationshipBatch(ctx EvalContext, ps *policySubgraph, spec *proto.CollectionSpec) ([]*proto.InstanceRecord, []*proto.TypeStatus, []*proto.ProviderInstance) {
	c := &relationshipCollector{
		ctx:        ctx,
		ps:         ps,
		types:      make(map[relationshipTypeKey]bool),
		incomplete: make(map[relationshipTypeKey]bool),
		deferred:   make(map[relationshipTypeKey]map[string]struct{}),
		keyPaths:   make(map[relationshipTypeKey][][]string),
		keep:       make(map[relationshipTypeKey]map[string]bool),
	}
	unpruned := make(map[relationshipTypeKey]bool)
	for _, ts := range spec.GetTypes() {
		key := relationshipTypeKey{source: ts.GetProviderSource(), typeName: ts.GetType()}
		if !c.types[key] {
			c.types[key] = true
			c.typeOrder = append(c.typeOrder, key)
		}
		for _, kp := range ts.GetKeyPaths() {
			c.keyPaths[key] = append(c.keyPaths[key], policyKeyPathNames(kp))
		}
		keep := policyKeepAttributes(ts)
		if keep == nil {
			unpruned[key] = true
			continue
		}
		if c.keep[key] == nil {
			c.keep[key] = make(map[string]bool)
		}
		for name := range keep {
			c.keep[key][name] = true
		}
	}
	for key := range unpruned {
		delete(c.keep, key)
	}

	var changes []*plans.ResourceInstanceChange
	if ps.run.Stage == proto.EvaluationStage_APPLY_EVALUATION_STAGE {
		changes = ps.run.AppliedChanges
	} else {
		changes = ctx.Changes().ResourceInstanceChanges()
	}

	changed := make(map[string]struct{})
	for _, change := range changes {
		if change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode || change.DeposedKey != states.NotDeposed {
			continue
		}
		changed[change.Addr.String()] = struct{}{}
	}

	// Origins are only computed in plan runs.
	if ps.run.Stage == proto.EvaluationStage_PLAN_EVALUATION_STAGE {
		c.origins = &relationshipOriginLookup{
			schemas:    ps.run.Schemas,
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

	deferred := make(map[string]struct{})
	if deferrals := ctx.Deferrals(); deferrals != nil {
		for _, d := range deferrals.GetDeferredChanges() {
			addr := d.Change.Addr
			if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
				continue
			}
			deferred[addr.String()] = struct{}{}
			key := relationshipTypeKey{source: d.Change.ProviderAddr.Provider.String(), typeName: addr.Resource.Resource.Type}
			if !c.types[key] {
				continue
			}
			if c.deferred[key] == nil {
				c.deferred[key] = make(map[string]struct{})
			}
			c.deferred[key][addr.String()] = struct{}{}
		}
	}

	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	for _, change := range changes {
		if change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			continue
		}
		c.addChangeRecord(change, state)
	}
	c.addStateRecords(state, changed, deferred)

	if ps.run.Stage == proto.EvaluationStage_PLAN_EVALUATION_STAGE && ps.run.PlanMode == proto.PlanMode_NORMAL_PLAN_MODE {
		c.checkExpandedInstances(changed, deferred)
	}

	sort.Slice(c.records, func(i, j int) bool {
		if c.records[i].Address != c.records[j].Address {
			return c.records[i].Address < c.records[j].Address
		}
		return c.records[i].DeposedKey < c.records[j].DeposedKey
	})
	return c.records, c.statuses(), ps.providers.all()
}

func policyKeyPathNames(path *proto.AttributePath) []string {
	names := make([]string, 0, len(path.GetSteps()))
	for _, step := range path.GetSteps() {
		names = append(names, step.GetAttributeName())
	}
	return names
}

func (c *relationshipCollector) schema(provider addrs.Provider, typeName string) providers.Schema {
	return c.ps.run.Schemas.ResourceTypeConfig(provider, addrs.ManagedResourceMode, typeName)
}

func (c *relationshipCollector) newRecord(addr addrs.AbsResourceInstance, providerAddr addrs.AbsProviderConfig) *proto.InstanceRecord {
	return &proto.InstanceRecord{
		Address:            addr.String(),
		Type:               addr.Resource.Resource.Type,
		ProviderSource:     providerAddr.Provider.String(),
		ModulePath:         addr.Module.String(),
		ProviderInstanceId: c.ps.providers.idFor(providerAddr),
	}
}

// encode encodes a value of a record of the given type, pruned to the
// attributes the type's records keep.
func (c *relationshipCollector) encode(key relationshipTypeKey, addr addrs.AbsResourceInstance, what string, val cty.Value) (*proto.ResourceAttributes, bool) {
	attrs, err := policy.EncodeResourceAttributes(policyPruneValue(val, c.keep[key]))
	if err != nil {
		log.Printf("[WARN] policy: failed to encode the %s of %s for relationship checks: %s", what, addr, err)
		c.incomplete[key] = true
		return nil, false
	}
	return attrs, true
}

// addChangeRecord adds the record of a change of a current or deposed object.
// Deposed objects only have a record when they are deleted or forgotten: a
// deposed object whose change is a no-op is already gone.
func (c *relationshipCollector) addChangeRecord(change *plans.ResourceInstanceChange, state *states.State) {
	addr := change.Addr
	key := relationshipTypeKey{source: change.ProviderAddr.Provider.String(), typeName: addr.Resource.Resource.Type}
	if !c.types[key] {
		return
	}
	deposed := change.DeposedKey != states.NotDeposed
	if deposed && change.Action == plans.NoOp {
		return
	}
	action, ok := policyResourceAction(change.Action)
	if !ok || (deposed && change.Action != plans.Delete && change.Action != plans.Forget) {
		log.Printf("[WARN] policy: unexpected action %s for %s (deposed key %q) in relationship checks", change.Action, addr, change.DeposedKey)
		c.incomplete[key] = true
		return
	}
	schema := c.schema(change.ProviderAddr.Provider, addr.Resource.Resource.Type)
	if schema.Body == nil {
		log.Printf("[WARN] policy: no schema for %s in relationship checks", addr)
		c.incomplete[key] = true
		return
	}

	rec := c.newRecord(addr, change.ProviderAddr)
	rec.Source = proto.RecordSource_PLANNED_RECORD_SOURCE
	rec.Action = action
	rec.Importing = change.Importing != nil
	rec.DeposedKey = change.DeposedKey.String()
	if change.PrevRunAddr.Resource.Resource.Type != "" && !change.PrevRunAddr.Equal(addr) {
		rec.PrevAddress = change.PrevRunAddr.String()
	}

	if policyRecordHasAttrs(change.Action) {
		after := change.After
		if c.ps.run.Stage == proto.EvaluationStage_APPLY_EVALUATION_STAGE {
			// The record of an applied change has the value the apply
			// produced.
			var obj *states.ResourceInstanceObjectSrc
			if inst := state.ResourceInstance(addr); inst != nil {
				obj = inst.Current
			}
			if obj == nil {
				log.Printf("[DEBUG] policy: no object for %s after apply; its type is incomplete for relationship checks", addr)
				c.incomplete[key] = true
				return
			}
			val, err := policyStateValue(obj, schema)
			if err != nil {
				log.Printf("[WARN] policy: failed to decode %s for relationship checks: %s", addr, err)
				c.incomplete[key] = true
				return
			}
			after = val
		} else {
			after = policyMarkSchemaSensitive(after, schema.Body)
		}
		if rec.Attrs, ok = c.encode(key, addr, "planned value", after); !ok {
			return
		}
		if c.origins != nil {
			rec.Origins = originsFor(c.ctx.Config(), c.ctx.InstanceExpander(), c.origins.overridden, addr, change.After, c.keyPaths[key], c.origins)
		}
	}
	if policyRecordHasPriorAttrs(change.Action) {
		if rec.PriorAttrs, ok = c.encode(key, addr, "prior value", policyMarkSchemaSensitive(change.Before, schema.Body)); !ok {
			return
		}
	}
	c.records = append(c.records, rec)
}

// addStateRecords adds a record for every current object in the state whose
// resource is in the configuration and that has neither a change nor a
// deferral. These are the instances the getresources callback reads at apply
// with states.ReadEachConfigResourceInstance, including instances whose key
// or module instance key no longer exists in the configuration, but one
// record per instance: that helper returns the current object once per
// object, and its selector doesn't get the instance address a record needs.
func (c *relationshipCollector) addStateRecords(state *states.State, changed, deferred map[string]struct{}) {
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
			key := relationshipTypeKey{source: rs.ProviderConfig.Provider.String(), typeName: rs.Addr.Resource.Type}
			if !c.types[key] || modCfg.Module.ResourceByAddr(rs.Addr.Resource) == nil {
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
					log.Printf("[WARN] policy: failed to decode %s for relationship checks: %s", addr, err)
					c.incomplete[key] = true
					continue
				}
				rec := c.newRecord(addr, rs.ProviderConfig)
				rec.Source = proto.RecordSource_STATE_RECORD_SOURCE
				rec.Action = proto.ResourceAction_NO_OP_RESOURCE_ACTION
				var ok bool
				if rec.Attrs, ok = c.encode(key, addr, "value", val); !ok {
					continue
				}
				c.records = append(c.records, rec)
			}
		}
	}
}

// checkExpandedInstances marks a type incomplete if the instance expander
// knows an instance of a configured resource of that type that has neither a
// change nor a deferral, e.g. because planning it failed.
func (c *relationshipCollector) checkExpandedInstances(changed, deferred map[string]struct{}) {
	cfg := c.ctx.Config()
	exp := c.ctx.InstanceExpander()
	if cfg == nil || exp == nil {
		return
	}
	known := exp.AllInstances()
	cfg.DeepEach(func(mc *configs.Config) {
		for _, r := range mc.Module.ManagedResources {
			provider := mc.Module.ProviderForLocalConfig(r.ProviderConfigAddr())
			key := relationshipTypeKey{source: provider.String(), typeName: r.Type}
			if !c.types[key] || c.incomplete[key] {
				continue
			}
			for _, modInst := range known.InstancesForModule(mc.Path, false) {
				absRes := r.Addr().Absolute(modInst)
				if !known.HasResource(absRes) {
					continue
				}
				_, keys, unknownKeys := exp.ResourceInstanceKeys(absRes)
				if unknownKeys && len(c.deferred[key]) == 0 {
					c.incomplete[key] = true
				}
				for _, instKey := range keys {
					addr := absRes.Instance(instKey).String()
					_, hasChange := changed[addr]
					_, isDeferred := deferred[addr]
					if !hasChange && !isDeferred {
						log.Printf("[DEBUG] policy: %s has no planned change; its type is incomplete for relationship checks", addr)
						c.incomplete[key] = true
					}
				}
			}
		}
	})
}

func (c *relationshipCollector) statuses() []*proto.TypeStatus {
	ret := make([]*proto.TypeStatus, 0, len(c.typeOrder))
	for _, key := range c.typeOrder {
		status := &proto.TypeStatus{
			ProviderSource: key.source,
			Type:           key.typeName,
			Completeness:   proto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS,
		}
		switch {
		case len(c.deferred[key]) > 0:
			status.Completeness = proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS
			for addr := range c.deferred[key] {
				status.DeferredAddresses = append(status.DeferredAddresses, addr)
			}
			sort.Strings(status.DeferredAddresses)
		case c.incomplete[key]:
			status.Completeness = proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS
		}
		ret = append(ret, status)
	}
	return ret
}

// policyProviderTable records the provider configurations configured during
// a walk, so the policy plugin can tell whether two resource instances are
// managed through equivalent provider configurations without seeing the
// configurations themselves.
type policyProviderTable struct {
	mu     sync.Mutex
	key    [32]byte
	byAddr map[string]*proto.ProviderInstance
	order  []*proto.ProviderInstance
}

func newPolicyProviderTable() *policyProviderTable {
	t := &policyProviderTable{
		byAddr: make(map[string]*proto.ProviderInstance),
	}
	if _, err := rand.Read(t.key[:]); err != nil {
		// crypto/rand doesn't fail on supported platforms.
		panic(err)
	}
	return t
}

// configured records the configuration of the given provider configuration.
// cfgType is the implied type of the provider's configuration schema, or
// cty.NilType if the schema isn't available.
func (t *policyProviderTable) configured(addr addrs.AbsProviderConfig, cfg cty.Value, cfgType cty.Type) {
	class := t.class(addr.Provider, cfg, cfgType)

	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.entryLocked(addr)
	entry.Known = class != nil
	entry.ConfigClass = class
}

func (t *policyProviderTable) class(provider addrs.Provider, cfg cty.Value, cfgType cty.Type) []byte {
	if cfgType == cty.NilType || cfg == cty.NilVal {
		return nil
	}
	cfg, _ = cfg.UnmarkDeep()
	if !cfg.IsWhollyKnown() {
		return nil
	}
	encoded, err := ctymsgpack.Marshal(cfg, cfgType)
	if err != nil {
		log.Printf("[WARN] policy: failed to encode the configuration of %s for relationship checks: %s", provider, err)
		return nil
	}
	mac := hmac.New(sha256.New, t.key[:])
	mac.Write([]byte(provider.String()))
	mac.Write([]byte{0})
	mac.Write(encoded)
	return mac.Sum(nil)
}

// idFor returns the id of the given provider configuration, adding an entry
// that isn't known if the provider configuration wasn't configured.
func (t *policyProviderTable) idFor(addr addrs.AbsProviderConfig) uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.entryLocked(addr).Id
}

func (t *policyProviderTable) entryLocked(addr addrs.AbsProviderConfig) *proto.ProviderInstance {
	key := addr.String()
	if entry, ok := t.byAddr[key]; ok {
		return entry
	}
	entry := &proto.ProviderInstance{
		Id:            uint32(len(t.order) + 1),
		ConfigAddress: key,
		Source:        addr.Provider.String(),
	}
	t.byAddr[key] = entry
	t.order = append(t.order, entry)
	return entry
}

// all returns copies of all entries in order of their ids.
func (t *policyProviderTable) all() []*proto.ProviderInstance {
	t.mu.Lock()
	defer t.mu.Unlock()
	ret := make([]*proto.ProviderInstance, len(t.order))
	for i, entry := range t.order {
		ret[i] = &proto.ProviderInstance{
			Id:            entry.Id,
			ConfigAddress: entry.ConfigAddress,
			Source:        entry.Source,
			ConfigClass:   append([]byte(nil), entry.ConfigClass...),
			Known:         entry.Known,
		}
	}
	return ret
}

// policyProviderSchemas returns the schemas of all providers in the given
// schemas, sorted by provider source.
func policyProviderSchemas(schemas *schemarepo.Schemas) []*proto.ProviderSchema {
	if schemas == nil {
		return nil
	}
	ret := make([]*proto.ProviderSchema, 0, len(schemas.Providers))
	for provider, schema := range schemas.Providers {
		ps := &proto.ProviderSchema{
			Type:               provider.Type,
			Source:             provider.String(),
			Resources:          policyTypeSchemas(provider, schema.ResourceTypes),
			DataSources:        policyTypeSchemas(provider, schema.DataSources),
			EphemeralResources: policyTypeSchemas(provider, schema.EphemeralResourceTypes),
		}
		for name, rs := range schema.ResourceTypes {
			if rs.Body == nil {
				continue
			}
			if paths := policyWriteOnlyPaths(rs.Body, nil); len(paths) > 0 {
				if ps.WriteOnlyPaths == nil {
					ps.WriteOnlyPaths = make(map[string]*proto.AttributePaths)
				}
				ps.WriteOnlyPaths[name] = &proto.AttributePaths{Paths: paths}
			}
		}
		ret = append(ret, ps)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Source < ret[j].Source
	})
	return ret
}

func policyTypeSchemas(provider addrs.Provider, schemas map[string]providers.Schema) map[string][]byte {
	if len(schemas) == 0 {
		return nil
	}
	ret := make(map[string][]byte, len(schemas))
	for name, schema := range schemas {
		if schema.Body == nil {
			continue
		}
		raw, err := ctyjson.MarshalType(schema.Body.ImpliedType())
		if err != nil {
			log.Printf("[WARN] policy: failed to encode the schema of %s from %s: %s", name, provider, err)
			continue
		}
		ret[name] = raw
	}
	return ret
}

// policyWriteOnlyPaths returns the paths of all write-only attributes of the
// block, at any depth, as attribute name steps sorted lexically.
func policyWriteOnlyPaths(block *configschema.Block, prefix []string) []*proto.AttributePath {
	var names [][]string
	var walkAttrs func(attrs map[string]*configschema.Attribute, prefix []string)
	var walkBlock func(block *configschema.Block, prefix []string)
	walkAttrs = func(attrs map[string]*configschema.Attribute, prefix []string) {
		for name, attr := range attrs {
			path := append(append([]string(nil), prefix...), name)
			if attr.WriteOnly {
				names = append(names, path)
			}
			if attr.NestedType != nil {
				walkAttrs(attr.NestedType.Attributes, path)
			}
		}
	}
	walkBlock = func(block *configschema.Block, prefix []string) {
		walkAttrs(block.Attributes, prefix)
		for name, nested := range block.BlockTypes {
			walkBlock(&nested.Block, append(append([]string(nil), prefix...), name))
		}
	}
	walkBlock(block, prefix)

	sort.Slice(names, func(i, j int) bool {
		return strings.Join(names[i], ".") < strings.Join(names[j], ".")
	})
	ret := make([]*proto.AttributePath, len(names))
	for i, path := range names {
		ret[i] = policyAttrPath(path)
	}
	return ret
}

func policyAttrPath(names []string) *proto.AttributePath {
	path := &proto.AttributePath{Steps: make([]*proto.AttributePath_Step, len(names))}
	for i, name := range names {
		path.Steps[i] = &proto.AttributePath_Step{
			Selector: &proto.AttributePath_Step_AttributeName{AttributeName: name},
		}
	}
	return path
}

// policyKeepAttributes returns the top-level attributes that the records of a
// type spec's type keep: the attributes policies read and the first steps of
// the key paths. It returns nil if the reads aren't complete, so the records
// keep all their attributes.
func policyKeepAttributes(spec *proto.TypeSpec) map[string]bool {
	if !spec.GetReadsComplete() {
		return nil
	}
	keep := make(map[string]bool)
	for _, name := range spec.GetReads() {
		keep[name] = true
	}
	for _, kp := range spec.GetKeyPaths() {
		if steps := kp.GetSteps(); len(steps) > 0 {
			keep[steps[0].GetAttributeName()] = true
		}
	}
	return keep
}

// policyPruneValue replaces every top-level attribute of an object value that
// isn't in keep with a null of the attribute's type, and drops the marks
// within the replaced attributes. It returns the value unchanged if keep is
// nil or if the value isn't a known, non-null object.
func policyPruneValue(val cty.Value, keep map[string]bool) cty.Value {
	if keep == nil || val == cty.NilVal {
		return val
	}
	unmarked, pvms := val.UnmarkDeepWithPaths()
	ty := unmarked.Type()
	if !unmarked.IsKnown() || unmarked.IsNull() || !ty.IsObjectType() {
		return val
	}
	attrs := make(map[string]cty.Value, len(ty.AttributeTypes()))
	for name, attrType := range ty.AttributeTypes() {
		if keep[name] {
			attrs[name] = unmarked.GetAttr(name)
		} else {
			attrs[name] = cty.NullVal(attrType)
		}
	}
	kept := make([]cty.PathValueMarks, 0, len(pvms))
	for _, pvm := range pvms {
		if len(pvm.Path) > 0 {
			if step, ok := pvm.Path[0].(cty.GetAttrStep); ok && !keep[step.Name] {
				continue
			}
		}
		kept = append(kept, pvm)
	}
	return cty.ObjectVal(attrs).MarkWithPaths(kept)
}

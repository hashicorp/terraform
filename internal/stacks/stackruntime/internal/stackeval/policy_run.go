// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"slices"
	"sort"
	"strings"
	"sync"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// policySubjectParallelism bounds the concurrent subject evaluations at the
// end of a deployment's relationship run, like the walk's parallelism bounds
// those at the end of a walk.
const policySubjectParallelism = 10

// policyRun is the relationship run of a deployment's plan or apply: one run
// with runtime STACKS that every component instance's walk contributes to.
//
// The walks evaluate no policy themselves; they report their records to this
// run and hand back their subjects. Once every component instance is done,
// finish reports the deployment-wide type statuses, evaluates every subject
// once with every policy (getresources still scoped to the subject's
// component), and finishes the run. That
// end-of-deployment batch mirrors the end-of-walk batch of a Terraform CLI
// run; streaming statuses and evaluations as types become complete is a
// later optimization.
type policyRun struct {
	main     *Main
	client   policy.RelationshipsClient
	stage    proto.EvaluationStage
	planMode proto.PlanMode

	// begin begins the run when the first component instance joins it,
	// once the walks need it; the fields below it are only set then.
	begin sync.Once
	// ctx is the operation's context, without its cancellation, for the
	// walks' ReportInstances calls.
	ctx         context.Context
	id          string
	spec        *proto.CollectionSpec
	failed      bool
	diagnostics policy.Diagnostics

	// mu serializes the ReportInstances calls and guards the fields below.
	mu sync.Mutex
	// joined and contributed count the walks that joined the run and those
	// that contributed to it. A walk that joined and didn't contribute
	// failed before its policy evaluation.
	joined, contributed int
	diagnosticsGiven    bool
	// providers are the run's provider instance ids, by config address.
	providers map[string]uint32
	// providerKey is the run's key for provider classes, shared by its
	// walks and its stack provider configurations.
	providerKey []byte
	// statuses are the merged statuses of the walks, by type.
	statuses map[policyTypeKey]*proto.TypeStatus
	// incomplete are the types incomplete for a reason outside the walks.
	incomplete map[policyTypeKey]bool
	subjects   []func(context.Context) tfdiags.Diagnostics
	// walks are the walks that joined, by component instance address.
	walks map[string]*policyWalk
	// pending are the records whose origins finish resolves before it
	// reports them.
	pending []*proto.InstanceRecord
}

// policyWalk is a component instance's walk in the run.
type policyWalk struct {
	addr stackaddrs.AbsComponentInstance
	// inputs is the component's inputs expression, or nil.
	inputs hcl.Expression
	// providers are the stack provider configuration instances of the
	// walk's root provider configurations, by the walk's (component
	// qualified) config address.
	providers map[string]*proto.ProviderInstance
	// variables and outputs are those of the walk's contribution.
	variables map[*proto.KeyOrigins][]terraform.PolicyVariableOrigin
	outputs   terraform.PolicyOutputOrigins
}

type policyTypeKey struct{ source, typeName string }

// newPolicyRun returns the deployment's relationship run, or nil if the
// policy client doesn't support relationships or the operation has none.
func newPolicyRun(m *Main) *policyRun {
	client, ok := m.PolicyClient().(policy.RelationshipsClient)
	if !ok || !client.RelationshipsSupported() {
		return nil
	}
	run := &policyRun{
		main:       m,
		client:     client,
		providers:  make(map[string]uint32),
		statuses:   make(map[policyTypeKey]*proto.TypeStatus),
		incomplete: make(map[policyTypeKey]bool),
		walks:      make(map[string]*policyWalk),
	}
	run.providerKey = make([]byte, 32)
	if _, err := rand.Read(run.providerKey); err != nil {
		return nil
	}
	switch {
	case m.Applying():
		run.stage = proto.EvaluationStage_APPLY_EVALUATION_STAGE
	case m.Planning():
		run.stage = proto.EvaluationStage_PLAN_EVALUATION_STAGE
	default:
		return nil
	}
	switch m.PlanningMode() {
	case plans.NormalMode:
		run.planMode = proto.PlanMode_NORMAL_PLAN_MODE
	case plans.DestroyMode:
		run.planMode = proto.PlanMode_DESTROY_PLAN_MODE
	case plans.RefreshOnlyMode:
		run.planMode = proto.PlanMode_REFRESH_ONLY_PLAN_MODE
	default:
		return nil
	}
	return run
}

// PolicyRun returns the deployment's relationship run, or nil if it has
// none.
func (m *Main) PolicyRun() *policyRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.policyRunInit {
		m.policyRunInit = true
		m.policyRun = newPolicyRun(m)
	}
	return m.policyRun
}

// beginRun begins the run with the schemas of every provider available to
// the deployment, so a single BeginRun covers all components.
func (r *policyRun) beginRun(ctx context.Context) {
	schemas := &schemarepo.Schemas{Providers: make(map[addrs.Provider]providers.ProviderSchema)}
	for addr := range r.main.providerFactories {
		schema, err := r.main.ProviderType(addr).Schema(ctx)
		if err != nil {
			log.Printf("[WARN] policy: the relationship run has no schema for %s: %s", addr, err)
			continue
		}
		schemas.Providers[addr] = schema
	}
	id, err := uuid.GenerateUUID()
	if err != nil {
		log.Printf("[WARN] policy: not starting a relationship run: failed to generate a run id: %s", err)
		return
	}
	resp, err := r.client.BeginRun(ctx, &proto.BeginRunRequest{
		RunId:           id,
		ProviderSchemas: terraform.PolicyProviderSchemas(schemas),
		Stage:           r.stage,
		PlanMode:        r.planMode,
		Runtime:         proto.RunRuntime_STACKS_RUN_RUNTIME,
	})
	if err != nil {
		log.Printf("[WARN] policy: failed to begin relationship run %s: %s", id, err)
		return
	}
	for _, diag := range resp.GetDiagnostics() {
		switch diag.GetSeverity() {
		case proto.Severity_WARNING, proto.Severity_ERROR:
			r.diagnostics = append(r.diagnostics, policy.DiagsFromProto([]*proto.Diagnostic{diag}, nil)...)
		}
	}
	if r.diagnostics.HasErrors() {
		r.failed = true
		return
	}
	r.id = id
	r.spec = resp.GetSpec()
}

// join returns the run for the walk of a component instance, or nil if the
// walk has none. The walk must contribute to the run or fail. inputs is the
// component's inputs expression, or nil. known are the stack provider
// configuration instances of the component's root provider configurations.
func (r *policyRun) join(ctx context.Context, addr stackaddrs.AbsComponentInstance, inputs hcl.Expression, known map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance) *terraform.SharedPolicyRun {
	if r == nil {
		return nil
	}
	r.begin.Do(func() {
		r.ctx = context.WithoutCancel(ctx)
		r.beginRun(ctx)
	})
	providers := r.stackProviders(ctx, addr, known)

	r.mu.Lock()
	defer r.mu.Unlock()
	r.joined++
	walk := &policyWalk{addr: addr, inputs: inputs, providers: providers}
	r.walks[addr.String()] = walk
	shared := &terraform.SharedPolicyRun{
		ID:            r.id,
		Spec:          r.spec,
		Failed:        r.failed,
		AddressPrefix: addr.String() + ".",
		ProviderKey:   r.providerKey,
		Contribute: func(c terraform.PolicyRunContribution) {
			r.contribute(walk, c)
		},
	}
	if !r.diagnosticsGiven {
		// One walk reports the diagnostics of BeginRun.
		r.diagnosticsGiven = true
		shared.Diagnostics = r.diagnostics
	}
	return shared
}

// contribute reports a walk's records right away and keeps its statuses and
// subjects for finish. The records with origins to resolve wait for finish
// too.
func (r *policyRun) contribute(walk *policyWalk, c terraform.PolicyRunContribution) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributed++
	r.subjects = append(r.subjects, c.Subjects...)
	walk.variables = c.Variables
	walk.outputs = c.Outputs

	// The walk's provider ids are its own; the run's are by config address,
	// which the walk qualified with its component instance. A provider
	// configuration from the stack is the stack's provider configuration
	// instance, whichever components it's passed to.
	ids := make(map[uint32]uint32, len(c.Providers))
	var providers []*proto.ProviderInstance
	for _, p := range c.Providers {
		local := p.Id
		if sp, ok := walk.providers[p.ConfigAddress]; ok {
			p.ConfigAddress = sp.ConfigAddress
			p.ConfigClass = sp.ConfigClass
			p.Known = sp.Known
		}
		id, ok := r.providers[p.ConfigAddress]
		if !ok {
			id = uint32(len(r.providers) + 1)
			r.providers[p.ConfigAddress] = id
			p.Id = id
			providers = append(providers, p)
		}
		ids[local] = id
	}
	for _, record := range c.Records {
		record.ProviderInstanceId = ids[record.ProviderInstanceId]
	}

	for _, status := range c.Statuses {
		key := policyTypeKey{status.ProviderSource, status.Type}
		for _, d := range status.Deferred {
			d.ProviderInstanceId = ids[d.ProviderInstanceId]
		}
		merged, ok := r.statuses[key]
		if !ok {
			r.statuses[key] = status
			continue
		}
		switch {
		case merged.Completeness == proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS:
		case status.Completeness == proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS:
			merged.Completeness = status.Completeness
			merged.Deferred = nil
		case status.Completeness == proto.TypeCompleteness_INCOMPLETE_DEFERRED_TYPE_COMPLETENESS:
			merged.Completeness = status.Completeness
			merged.Deferred = append(merged.Deferred, status.Deferred...)
			sort.Slice(merged.Deferred, func(i, j int) bool { return merged.Deferred[i].Address < merged.Deferred[j].Address })
		}
	}

	records := c.Records[:0:0]
	for _, record := range c.Records {
		if slices.ContainsFunc(record.Origins, func(ko *proto.KeyOrigins) bool { return c.Variables[ko] != nil }) {
			r.pending = append(r.pending, record)
			continue
		}
		records = append(records, record)
	}

	if len(records) == 0 && len(providers) == 0 {
		return
	}
	// The statuses only go out in finish, once every walk has contributed.
	r.report(records, nil, providers)
}

// stackProviders returns the stack provider configuration instances of a
// component instance's root provider configurations, by the walk's config
// address, with their classes computed like a walk computes those of its
// own provider configurations, with the run's key.
func (r *policyRun) stackProviders(ctx context.Context, component stackaddrs.AbsComponentInstance, known map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance) map[string]*proto.ProviderInstance {
	phase := PlanPhase
	if r.stage == proto.EvaluationStage_APPLY_EVALUATION_STAGE {
		phase = ApplyPhase
	}
	ret := make(map[string]*proto.ProviderInstance, len(known))
	for root, addr := range known {
		p := &proto.ProviderInstance{
			ConfigAddress: addr.String(),
			Source:        root.Provider.String(),
		}
		ret[component.String()+"."+root.AbsProviderConfig().String()] = p
		stack := r.main.Stack(ctx, addr.Stack, phase)
		if stack == nil {
			continue
		}
		config := stack.Provider(addr.Item.ProviderConfig)
		if config == nil {
			continue
		}
		insts, _ := config.Instances(ctx, phase)
		inst := insts[addr.Item.Key]
		if inst == nil {
			continue
		}
		// The address as written in the stack configuration, with the
		// provider's local name.
		local := inst.provider.config.config.LocalAddr
		p.ConfigAddress = fmt.Sprintf("provider.%s.%s", local.LocalName, local.Alias)
		if addr.Item.Key != addrs.NoKey {
			p.ConfigAddress += addr.Item.Key.String()
		}
		if !addr.Stack.IsRoot() {
			p.ConfigAddress = addr.Stack.String() + "." + p.ConfigAddress
		}
		schema, err := r.main.ProviderType(inst.ProviderType().Addr()).Schema(ctx)
		if err != nil || schema.Provider.Body == nil {
			continue
		}
		p.ConfigClass = terraform.PolicyProviderClass(r.providerKey, root.Provider, inst.ProviderArgs(ctx, phase), schema.Provider.Body.ImpliedType())
		p.Known = p.ConfigClass != nil
	}
	return ret
}

// report reports records, statuses and providers to the run. r.mu must be
// held. It reports whether it succeeded.
func (r *policyRun) report(records []*proto.InstanceRecord, statuses []*proto.TypeStatus, providers []*proto.ProviderInstance) bool {
	for _, req := range terraform.PolicyRunReports(records, statuses, providers, r.id) {
		resp, err := r.client.ReportInstances(r.ctx, req)
		if err != nil {
			log.Printf("[WARN] policy: failed to report instances for relationship run %s: %s", r.id, err)
			r.incompleteAll()
			return false
		}
		logPolicyRunDiagnostics("ReportInstances", r.id, resp.GetDiagnostics())
	}
	return true
}

// resolveVariableOrigins completes the key paths of the pending records
// whose leaves come from root module variables of their component. A
// variable whose component input refers directly to another component
// instance's output (componentOutputRef) gets the origins of that output,
// qualified with the producing component instance; any other variable keeps
// the reason VARIABLE, and the engine relates it by value. r.mu must be
// held.
func (r *policyRun) resolveVariableOrigins() {
	for _, walk := range r.walks {
		for ko, vars := range walk.variables {
			var reasons []proto.NoOriginReason
			for _, v := range vars {
				origins, more := r.resolveVariable(walk, v, 0)
				ko.Origins = append(ko.Origins, origins...)
				reasons = append(reasons, more...)
			}
			ko.Origins = policyUniqueOrigins(ko.Origins)
			ko.NoOrigin = append(ko.NoOrigin, reasons...)
			slices.Sort(ko.NoOrigin)
			ko.NoOrigin = slices.Compact(ko.NoOrigin)
		}
	}
}

// policyMaxComponentHops bounds how many component inputs and outputs the
// origin of a value is followed through.
const policyMaxComponentHops = 16

func (r *policyRun) resolveVariable(walk *policyWalk, v terraform.PolicyVariableOrigin, hops int) ([]*proto.Origin, []proto.NoOriginReason) {
	variable := []proto.NoOriginReason{proto.NoOriginReason_VARIABLE_NO_ORIGIN_REASON}
	if hops >= policyMaxComponentHops {
		return nil, variable
	}
	producerAddr, output, rest, ok := componentOutputRef(walk.inputs, v.Variable, walk.addr)
	if !ok {
		return nil, variable
	}
	producer := r.walks[producerAddr.String()]
	if producer == nil || producer.outputs == nil {
		return nil, variable
	}
	origins, reasons, vars := producer.outputs(output, rest, v)
	prefix := producerAddr.String() + "."
	for _, o := range origins {
		o.Address = prefix + o.Address
	}
	for _, next := range vars {
		moreOrigins, moreReasons := r.resolveVariable(producer, next, hops+1)
		origins = append(origins, moreOrigins...)
		reasons = append(reasons, moreReasons...)
	}
	return origins, reasons
}

// componentOutputRef returns the component instance output that the input
// variable name of a component instance refers to directly in the
// component's inputs, and the traversal into the output, if it does:
// `component.x.out`, `component.x["k"].out` or `component.x[each.key].out`,
// optionally followed by attribute and index steps. Anything else, such as a
// function call or a template, isn't a direct reference.
//
// Embedded stacks aren't followed: a reference to a stack input variable or
// to an embedded stack's output isn't a direct reference here.
func componentOutputRef(inputs hcl.Expression, name string, consumer stackaddrs.AbsComponentInstance) (stackaddrs.AbsComponentInstance, string, hcl.Traversal, bool) {
	var none stackaddrs.AbsComponentInstance
	obj, ok := inputs.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return none, "", nil, false
	}
	var value hclsyntax.Expression
	for _, item := range obj.Items {
		key, diags := item.KeyExpr.Value(nil)
		if diags.HasErrors() || !key.IsKnown() || key.IsNull() || key.Type() != cty.String {
			if kw := hcl.ExprAsKeyword(item.KeyExpr); kw != "" {
				key = cty.StringVal(kw)
			} else {
				continue
			}
		}
		if key.AsString() == name {
			value = item.ValueExpr
		}
	}

	var trav hcl.Traversal
	var instanceKey addrs.InstanceKey = addrs.NoKey
	switch expr := value.(type) {
	case *hclsyntax.ScopeTraversalExpr:
		trav = expr.Traversal
		if len(trav) > 2 {
			if idx, ok := trav[2].(hcl.TraverseIndex); ok {
				key, err := addrs.ParseInstanceKey(idx.Key)
				if err != nil {
					return none, "", nil, false
				}
				instanceKey = key
				trav = append(trav[:2:2], trav[3:]...)
			}
		}
	case *hclsyntax.RelativeTraversalExpr:
		// component.x[each.key].out
		index, ok := expr.Source.(*hclsyntax.IndexExpr)
		if !ok {
			return none, "", nil, false
		}
		coll, ok := index.Collection.(*hclsyntax.ScopeTraversalExpr)
		if !ok {
			return none, "", nil, false
		}
		key, ok := index.Key.(*hclsyntax.ScopeTraversalExpr)
		if !ok || len(key.Traversal) != 2 || key.Traversal.RootName() != "each" {
			return none, "", nil, false
		}
		if attr, ok := key.Traversal[1].(hcl.TraverseAttr); !ok || attr.Name != "key" || consumer.Item.Key == addrs.NoKey {
			return none, "", nil, false
		}
		instanceKey = consumer.Item.Key
		trav = append(slices.Clone(coll.Traversal), expr.Traversal...)
	default:
		return none, "", nil, false
	}

	if len(trav) < 3 || trav.RootName() != "component" {
		return none, "", nil, false
	}
	component, ok := trav[1].(hcl.TraverseAttr)
	if !ok {
		return none, "", nil, false
	}
	output, ok := trav[2].(hcl.TraverseAttr)
	if !ok {
		return none, "", nil, false
	}
	producer := stackaddrs.AbsComponentInstance{
		Stack: consumer.Stack,
		Item:  stackaddrs.ComponentInstance{Component: stackaddrs.Component{Name: component.Name}, Key: instanceKey},
	}
	return producer, output.Name, trav[3:], true
}

// policyUniqueOrigins sorts origins by address and path and removes
// duplicates.
func policyUniqueOrigins(origins []*proto.Origin) []*proto.Origin {
	key := func(o *proto.Origin) string { return o.Address + "#" + o.Path.String() }
	slices.SortFunc(origins, func(a, b *proto.Origin) int { return strings.Compare(key(a), key(b)) })
	return slices.CompactFunc(origins, func(a, b *proto.Origin) bool { return key(a) == key(b) })
}

// skipped notes a component instance whose walk doesn't run, because its
// plan has nothing to apply: the instances in its state aren't reported, so
// their types are incomplete.
func (r *policyRun) skipped(state *states.State) {
	if r == nil || state == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ms := range state.Modules {
		for _, rs := range ms.Resources {
			if rs.Addr.Resource.Mode == addrs.ManagedResourceMode && len(rs.Instances) > 0 {
				r.incomplete[policyTypeKey{rs.ProviderConfig.Provider.String(), rs.Addr.Resource.Type}] = true
			}
		}
	}
}

// incompleteAll marks every type of the run incomplete. r.mu must be held.
func (r *policyRun) incompleteAll() {
	for _, typ := range r.spec.GetTypes() {
		r.incomplete[policyTypeKey{typ.GetProviderSource(), typ.GetType()}] = true
	}
}

// finish reports the run's type statuses once every component instance is
// done, evaluates the relationship policies of every subject, and finishes
// the run. errored is set when the operation had errors: a component
// instance that failed before its walk didn't report its instances, so every
// type is incomplete.
func (m *Main) finishPolicyRun(ctx context.Context, errored bool) tfdiags.Diagnostics {
	m.mu.Lock()
	r := m.policyRun
	m.mu.Unlock()
	if r == nil || r.id == "" {
		return nil
	}

	r.mu.Lock()
	// Every walk has contributed, so the origins of values passed between
	// components can be resolved now.
	r.resolveVariableOrigins()
	if len(r.pending) > 0 {
		r.report(r.pending, nil, nil)
		r.pending = nil
	}
	if errored || r.joined != r.contributed {
		r.incompleteAll()
	}
	var statuses []*proto.TypeStatus
	for _, typ := range r.spec.GetTypes() {
		key := policyTypeKey{typ.GetProviderSource(), typ.GetType()}
		if slices.ContainsFunc(statuses, func(s *proto.TypeStatus) bool { return s.ProviderSource == key.source && s.Type == key.typeName }) {
			continue
		}
		status, ok := r.statuses[key]
		if !ok {
			// No walk has the type's provider: the deployment has no
			// instance of the type.
			status = &proto.TypeStatus{ProviderSource: key.source, Type: key.typeName, Completeness: proto.TypeCompleteness_COMPLETE_TYPE_COMPLETENESS}
		}
		if r.incomplete[key] {
			status.Completeness = proto.TypeCompleteness_INCOMPLETE_ERROR_TYPE_COMPLETENESS
			status.Deferred = nil
		}
		statuses = append(statuses, status)
	}
	subjects := r.subjects
	r.subjects = nil
	resp, err := r.client.ReportInstances(ctx, &proto.ReportInstancesRequest{RunId: r.id, Statuses: statuses})
	reported := err == nil
	if err != nil {
		log.Printf("[WARN] policy: failed to report the type statuses of relationship run %s: %s", r.id, err)
	} else {
		logPolicyRunDiagnostics("ReportInstances", r.id, resp.GetDiagnostics())
	}
	r.mu.Unlock()

	// Evaluate the relationship policies only now that the run holds the
	// whole deployment: the end-of-deployment counterpart of a CLI walk's
	// end-of-walk batch. Streaming the evaluations as their types become
	// complete is a later optimization.
	//
	// The subjects are evaluated even when the statuses failed to report,
	// as in a Terraform run, so the engine fails the relationship policies
	// closed instead of them giving no result.
	var diags tfdiags.Diagnostics
	if ctx.Err() == nil {
		var wg sync.WaitGroup
		var diagsMu sync.Mutex
		sem := make(chan struct{}, policySubjectParallelism)
		for _, subject := range subjects {
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				moreDiags := subject(ctx)
				diagsMu.Lock()
				diags = diags.Append(moreDiags)
				diagsMu.Unlock()
			}()
		}
		wg.Wait()
	}

	aborted := errored || ctx.Err() != nil || !reported
	finish, err := r.client.FinishRun(context.WithoutCancel(ctx), &proto.FinishRunRequest{RunId: r.id, Aborted: aborted})
	if err != nil {
		log.Printf("[WARN] policy: failed to finish relationship run %s: %s", r.id, err)
	} else {
		logPolicyRunDiagnostics("FinishRun", r.id, finish.GetDiagnostics())
	}
	return diags
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

// policyComponentInputs returns the inputs expression of a component
// instance, or nil for anything else, such as a removed component instance.
func policyComponentInputs(scope any) hcl.Expression {
	if inst, ok := scope.(*ComponentInstance); ok {
		return inst.call.config.config.Inputs
	}
	return nil
}

// policyComponentProviders returns the provider configuration expressions of
// a component instance or removed component instance, or nil.
func policyComponentProviders(scope any) map[addrs.LocalProviderConfig]hcl.Expression {
	switch inst := scope.(type) {
	case *ComponentInstance:
		return inst.call.config.config.ProviderConfigs
	case *RemovedComponentInstance:
		return inst.call.config.config.ProviderConfigs
	}
	return nil
}

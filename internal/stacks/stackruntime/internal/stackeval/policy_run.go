// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"log"
	"slices"
	"sort"
	"sync"

	"github.com/hashicorp/go-uuid"

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
// Each walk still evaluates the resource policies without relationships
// itself, during the walk, with getresources scoped to the component. The
// walks only report their records to this run. Once every component instance
// is done, finish reports the deployment-wide type statuses, evaluates the
// relationship policies of every subject, and finishes the run. That
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
	begin       sync.Once
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
	// statuses are the merged statuses of the walks, by type.
	statuses map[policyTypeKey]*proto.TypeStatus
	// incomplete are the types incomplete for a reason outside the walks.
	incomplete map[policyTypeKey]bool
	subjects   []func(context.Context) tfdiags.Diagnostics
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
// walk has none. The walk must contribute to the run or fail.
func (r *policyRun) join(ctx context.Context, addr stackaddrs.AbsComponentInstance) *terraform.SharedPolicyRun {
	if r == nil {
		return nil
	}
	r.begin.Do(func() { r.beginRun(ctx) })

	r.mu.Lock()
	defer r.mu.Unlock()
	r.joined++
	shared := &terraform.SharedPolicyRun{
		ID:            r.id,
		Spec:          r.spec,
		Failed:        r.failed,
		AddressPrefix: addr.String() + ".",
		Contribute:    r.contribute,
	}
	if !r.diagnosticsGiven {
		// One walk reports the diagnostics of BeginRun.
		r.diagnosticsGiven = true
		shared.Diagnostics = r.diagnostics
	}
	return shared
}

// contribute reports a walk's records right away and keeps its statuses and
// subjects for finish.
func (r *policyRun) contribute(c terraform.PolicyRunContribution) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributed++
	r.subjects = append(r.subjects, c.Subjects...)

	// The walk's provider ids are its own; the run's are by config address,
	// which the walk qualified with its component instance.
	ids := make(map[uint32]uint32, len(c.Providers))
	var providers []*proto.ProviderInstance
	for _, p := range c.Providers {
		local := p.Id
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

	if len(c.Records) == 0 && len(providers) == 0 {
		return
	}
	// The statuses only go out in finish, once every walk has contributed.
	for _, req := range terraform.PolicyRunReports(c.Records, nil, providers, r.id) {
		resp, err := r.client.ReportInstances(context.Background(), req)
		if err != nil {
			log.Printf("[WARN] policy: failed to report instances for relationship run %s: %s", r.id, err)
			r.incompleteAll()
			return
		}
		logPolicyRunDiagnostics("ReportInstances", r.id, resp.GetDiagnostics())
	}
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
	var diags tfdiags.Diagnostics
	if reported && ctx.Err() == nil {
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

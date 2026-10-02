// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"log"
	"sync"

	"github.com/zclconf/go-cty/cty"
	ctymsgpack "github.com/zclconf/go-cty/cty/msgpack"

	"github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
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
// final and before any subject is evaluated. It never fails the walk: errors
// are logged and leave the run without a run id (when BeginRun fails) or with
// only part of its instances reported.
func (ps *policySubgraph) startRelationshipRun(ctx EvalContext, stopCtx context.Context) {
	client, ok := policyRelationshipsClient(ctx.PolicyClient())
	if !ok || ps.run == nil {
		return
	}

	runID, err := uuid.GenerateUUID()
	if err != nil {
		log.Printf("[WARN] policy: not starting a relationship run: failed to generate a run id: %s", err)
		return
	}

	resp, err := client.BeginRun(stopCtx, &proto.BeginRunRequest{
		RunId:    runID,
		Stage:    ps.run.Stage,
		PlanMode: ps.run.PlanMode,
		Runtime:  proto.RunRuntime_CLI_RUN_RUNTIME,
		Targeted: ps.run.Targeted,
	})
	if err != nil {
		log.Printf("[WARN] policy: failed to begin relationship run %s: %s", runID, err)
		return
	}
	logPolicyRunDiagnostics("BeginRun", runID, resp.GetDiagnostics())

	ps.lock.Lock()
	ps.runID = runID
	ps.lock.Unlock()

	for _, req := range chunkRelationshipBatch(nil, nil, nil, runID) {
		resp, err := client.ReportInstances(stopCtx, req)
		if err != nil {
			log.Printf("[WARN] policy: failed to report instances for relationship run %s: %s", runID, err)
			return
		}
		logPolicyRunDiagnostics("ReportInstances", runID, resp.GetDiagnostics())
	}
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

// chunkRelationshipBatch splits a relationship batch into ReportInstances
// requests. All providers are in the first request, all statuses in the
// last, and there is always at least one request.
func chunkRelationshipBatch(records []*proto.InstanceRecord, statuses []*proto.TypeStatus, providers []*proto.ProviderInstance, runID string) []*proto.ReportInstancesRequest {
	return []*proto.ReportInstancesRequest{{
		RunId:     runID,
		Providers: providers,
		Records:   records,
		Statuses:  statuses,
	}}
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

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// SharedPolicyRun is a relationship run that the caller of a plan or apply
// owns and that spans several walks. Terraform Stacks begins one run per
// deployment and operation (runtime STACKS), and each component's walk
// contributes its instances to it.
//
// A walk with a shared run doesn't begin or finish a run. Inside the walk it
// evaluates only the resource policies without relationships, exactly as a
// walk without a run would (getresources still sees only the walk's own
// resources). Instead of evaluating the relationship policies, it hands the
// caller its records, its type statuses and one evaluation per subject; the
// caller reports the statuses and runs the evaluations once every walk has
// contributed.
type SharedPolicyRun struct {
	// ID is the id of the run, or "" if the run didn't begin because
	// BeginRun failed. Without a run the walk evaluates every policy itself,
	// like a walk whose own BeginRun failed.
	ID string

	// Spec is the collection spec that BeginRun returned.
	Spec *proto.CollectionSpec

	// Failed is set when BeginRun returned errors: the run didn't begin and,
	// like a walk whose own BeginRun returned errors, the walk evaluates no
	// policy.
	Failed bool

	// Diagnostics are the diagnostics of BeginRun that the walk reports
	// through the PolicyDiagnostics hook. The caller gives them to one walk
	// only, so they are reported once.
	Diagnostics policy.Diagnostics

	// AddressPrefix qualifies the addresses that the walk reports and
	// evaluates in the run, so that they are unique across the walks that
	// contribute to it, e.g. `component.a["k"].`. Results still reach the
	// walk's hooks with unqualified addresses.
	AddressPrefix string

	// Contribute receives the walk's contribution once its changes and state
	// are final. It's called at most once per walk, from the walk.
	Contribute func(PolicyRunContribution)
}

// PolicyRunContribution is what a walk contributes to a SharedPolicyRun.
// Addresses are qualified with the run's AddressPrefix. Provider instance
// ids are local to the walk: Providers lists the walk's providers, and the
// records and deferred instances refer to them.
type PolicyRunContribution struct {
	Records   []*proto.InstanceRecord
	Statuses  []*proto.TypeStatus
	Providers []*proto.ProviderInstance

	// Subjects evaluate the walk's resource policies with relationships, one
	// subject each, in the run, and report the results through the walk's
	// hooks. The caller calls them after it has reported every walk's records
	// and the run's type statuses, and before it finishes the run. The
	// context replaces the walk's stop context, which is gone by then.
	Subjects []func(context.Context) tfdiags.Diagnostics
}

// PolicyProviderSchemas returns the provider schemas of a BeginRun request
// for the given schemas.
func PolicyProviderSchemas(schemas *schemarepo.Schemas) []*proto.ProviderSchema {
	return policyProviderSchemas(schemas)
}

// policyEvaluationScope is the relationship run and the policy selection of
// a resource evaluation, and the prefix that qualifies the subject's address
// in the run.
type policyEvaluationScope struct {
	runID         string
	selection     proto.PolicySelection
	addressPrefix string
}

// walkEvaluationScope returns the scope of the resource evaluations inside
// the walk.
func (ps *policySubgraph) walkEvaluationScope() policyEvaluationScope {
	if ps.run != nil && ps.run.Shared != nil && ps.run.Shared.ID != "" {
		return policyEvaluationScope{selection: proto.PolicySelection_NON_RELATIONSHIP_POLICY_SELECTION}
	}
	return policyEvaluationScope{runID: ps.RunID()}
}

// contributeToSharedRun collects the walk's relationship batch and hands it,
// with an evaluation per subject, to the shared run. It must be called after
// the walk's changes and state are final.
func (ps *policySubgraph) contributeToSharedRun(ctx EvalContext) {
	shared := ps.run.Shared
	if shared.ID == "" || shared.Contribute == nil {
		return
	}
	records, statuses, providers := collectRelationshipBatch(ctx, ps, shared.Spec)
	prefix := shared.AddressPrefix
	for _, record := range records {
		record.Address = prefix + record.Address
		if record.PrevAddress != "" {
			record.PrevAddress = prefix + record.PrevAddress
		}
		for _, keyOrigins := range record.Origins {
			for _, origin := range keyOrigins.Origins {
				origin.Address = prefix + origin.Address
			}
		}
	}
	for _, status := range statuses {
		for _, deferred := range status.Deferred {
			deferred.Address = prefix + deferred.Address
		}
	}
	for _, provider := range providers {
		provider.ConfigAddress = prefix + provider.ConfigAddress
	}

	op := walkPlan
	if ps.run.Stage == proto.EvaluationStage_APPLY_EVALUATION_STAGE {
		op = walkApply
	}
	scope := policyEvaluationScope{
		runID:         shared.ID,
		selection:     proto.PolicySelection_RELATIONSHIP_POLICY_SELECTION,
		addressPrefix: prefix,
	}
	var subjects []func(context.Context) tfdiags.Diagnostics
	deferredCtx := &deferredPolicyEvalContext{EvalContext: ctx, providers: make(map[string]deferredPolicyProvider)}
	ps.lock.Lock()
	for v := range ps.graph.VerticesSeq() {
		n, ok := v.(*nodeResourcePolicy)
		if !ok {
			continue
		}
		// The walk forgets its providers when it ends, so the subject keeps
		// the ones it needs. They are the external providers of the
		// caller, which keeps them open until it has finished the run.
		key := n.ProviderAddr.String()
		if _, ok := deferredCtx.providers[key]; !ok {
			if provider := ctx.Provider(n.ProviderAddr); provider != nil {
				schema, err := ctx.ProviderSchema(n.ProviderAddr)
				if err == nil {
					deferredCtx.providers[key] = deferredPolicyProvider{provider, schema}
				}
			}
		}
		subjects = append(subjects, func(stop context.Context) tfdiags.Diagnostics {
			return n.evaluate(deferredCtx.withStop(stop), op, scope)
		})
	}
	ps.lock.Unlock()

	shared.Contribute(PolicyRunContribution{
		Records:   records,
		Statuses:  statuses,
		Providers: providers,
		Subjects:  subjects,
	})
}

// deferredPolicyEvalContext is the EvalContext of a subject evaluation that
// runs after its walk: the walk's context with a new stop context. Its
// getresources callback still reads the walk's changes and state only.
type deferredPolicyEvalContext struct {
	EvalContext
	stop      context.Context
	providers map[string]deferredPolicyProvider
}

type deferredPolicyProvider struct {
	provider providers.Interface
	schema   providers.ProviderSchema
}

func (c *deferredPolicyEvalContext) withStop(stop context.Context) *deferredPolicyEvalContext {
	ret := *c
	ret.stop = stop
	return &ret
}

func (c *deferredPolicyEvalContext) StopCtx() context.Context {
	return c.stop
}

func (c *deferredPolicyEvalContext) Provider(addr addrs.AbsProviderConfig) providers.Interface {
	return c.providers[addr.String()].provider
}

func (c *deferredPolicyEvalContext) ProviderSchema(addr addrs.AbsProviderConfig) (providers.ProviderSchema, error) {
	p, ok := c.providers[addr.String()]
	if !ok {
		return providers.ProviderSchema{}, fmt.Errorf("provider %s not initialized", addr)
	}
	return p.schema, nil
}

// PolicyRunReports splits a batch of a relationship run into ReportInstances
// requests; see chunkRelationshipBatch.
func PolicyRunReports(records []*proto.InstanceRecord, statuses []*proto.TypeStatus, providers []*proto.ProviderInstance, runID string) []*proto.ReportInstancesRequest {
	return chunkRelationshipBatch(records, statuses, providers, runID)
}

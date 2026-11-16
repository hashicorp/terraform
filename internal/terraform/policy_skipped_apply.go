// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// EvaluateSkippedApplyPolicies evaluates the policies of a plan that its
// caller doesn't apply because it isn't applyable (it has no changes), as if
// it were applied, when the policy client has relationships: like a
// Terraform CLI apply of such a plan, the walk relates every instance of the
// plan's prior state, which is the state after the skipped apply, and has
// one subject per no-op change, whose new value is the prior state's.
// Terraform Stacks skips the apply of such a component instance.
//
// There is no walk: the instances and the evaluations read the prior state
// and the plan's deferrals, and the evaluations use opts.ExternalProviders
// and report through the context's hooks.
func (c *Context) EvaluateSkippedApplyPolicies(plan *plans.Plan, config *configs.Config, opts *ApplyOpts) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	if opts == nil || opts.PolicyClient == nil || plan == nil || plan.PriorState == nil {
		return nil
	}
	if opts.PolicyRelationships == nil && len(opts.PolicyClient.Relationships().Trees) == 0 {
		return nil
	}
	schemas, moreDiags := c.Schemas(config, plan.PriorState)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return diags
	}
	changes, err := plan.Changes.Decode(schemas)
	if err != nil {
		return diags.Append(err)
	}
	plannedDeferrals, moreDiags := decodeDeferredResources(schemas, plan.DeferredResources)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}
	run, moreDiags := c.applyPolicyRunOpts(schemas, changes.Resources, opts)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() || !run.relating() {
		return diags
	}

	// Like a walk's state once its policy phase begins, the state is closed
	// to writes.
	state := plan.PriorState.DeepCopy().SyncWrapper()
	state.Close()
	deferred := deferring.NewDeferred(true)
	deferred.LoadPlannedDeferrals(plannedDeferrals)
	ctx := &skippedApplyEvalContext{
		config:    config,
		state:     state,
		deferred:  deferred,
		client:    opts.PolicyClient,
		hooks:     c.hooks,
		schemas:   schemas,
		providers: opts.ExternalProviders,
	}
	ps := newPolicySubgraphForRun(run)
	ps.generation = opts.PolicyClient.NextGeneration()
	ctx.ps = ps

	// The subjects of a CLI apply of a plan without changes: one per no-op
	// change of a current managed object, with the object's value after the
	// apply, which is its prior value.
	var subjects []*nodeResourcePolicy
	for _, change := range changes.Resources {
		if change.Action != plans.NoOp || change.DeposedKey != states.NotDeposed || change.Addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
			continue
		}
		inst := plan.PriorState.ResourceInstance(change.Addr)
		if inst == nil || inst.Current == nil {
			continue
		}
		schema := schemas.ResourceTypeConfig(change.ProviderAddr.Provider, addrs.ManagedResourceMode, change.Addr.Resource.Resource.Type)
		obj, err := inst.Current.Decode(schema)
		if err != nil {
			diags = diags.Append(err)
			continue
		}
		subjects = append(subjects, &nodeResourcePolicy{
			ResourceAddr: change.Addr,
			ProviderAddr: change.ProviderAddr,
			Before:       change.Before,
			After:        obj.Value,
			Action:       plans.NoOp,
		})
	}

	ps.relations = collectPolicyRelations(ctx, ps)
	for _, subject := range subjects {
		diags = diags.Append(subject.Execute(ctx, walkApply))
	}
	return diags
}

// skippedApplyEvalContext is the EvalContext of EvaluateSkippedApplyPolicies. It
// implements only what collecting the instances of relationships and
// evaluating a resource's policies at apply use; the other methods panic.
type skippedApplyEvalContext struct {
	EvalContext

	config    *configs.Config
	state     *states.SyncState
	deferred  *deferring.Deferred
	client    policy.Client
	hooks     []Hook
	schemas   *schemarepo.Schemas
	providers map[addrs.RootProviderConfig]providers.Interface
	ps        *policySubgraph
}

func (c *skippedApplyEvalContext) Config() *configs.Config        { return c.config }
func (c *skippedApplyEvalContext) State() *states.SyncState       { return c.state }
func (c *skippedApplyEvalContext) Deferrals() *deferring.Deferred { return c.deferred }
func (c *skippedApplyEvalContext) PolicyClient() policy.Client    { return c.client }
func (c *skippedApplyEvalContext) PolicyGraph() *policySubgraph   { return c.ps }
func (c *skippedApplyEvalContext) StopCtx() context.Context       { return context.Background() }

func (c *skippedApplyEvalContext) Hook(fn func(Hook) (HookAction, error)) error {
	for _, h := range c.hooks {
		action, err := fn(h)
		if err != nil {
			return err
		}
		if action == HookActionHalt {
			return nil
		}
	}
	return nil
}

func (c *skippedApplyEvalContext) ClientCapabilities() providers.ClientCapabilities {
	return providers.ClientCapabilities{
		DeferralAllowed:            true,
		WriteOnlyAttributesAllowed: true,
		StorePlannedPrivate:        true,
		ComputedBlocksAllowed:      true,
	}
}

// Provider returns the caller's provider for a root provider configuration;
// a skipped apply has no others.
func (c *skippedApplyEvalContext) Provider(addr addrs.AbsProviderConfig) providers.Interface {
	if !addr.Module.IsRoot() {
		return nil
	}
	return c.providers[addrs.RootProviderConfig{Provider: addr.Provider, Alias: addr.Alias}]
}

func (c *skippedApplyEvalContext) ProviderSchema(addr addrs.AbsProviderConfig) (providers.ProviderSchema, error) {
	schema, ok := c.schemas.Providers[addr.Provider]
	if !ok {
		return providers.ProviderSchema{}, fmt.Errorf("no schema for provider %s", addr)
	}
	return schema, nil
}

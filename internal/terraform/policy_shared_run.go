// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"context"
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// SharedPolicyRun relates instances across several walks, for the caller
// that owns them. Terraform Stacks has one per deployment and operation
// (contract §3.8), and each component instance's walk contributes its
// instances to it.
//
// A walk with a shared run evaluates no policy itself. It hands the caller
// its instances and one evaluation per subject; the caller relates the
// instances of every walk and runs the evaluations once every walk has
// contributed. The evaluations use the walk's context, so getresources still
// sees only the walk's own resources.
type SharedPolicyRun struct {
	// AddressPrefix qualifies the addresses that the walk contributes, so
	// that they are unique across the walks, e.g. `component.a["k"].`.
	// Results still reach the walk's hooks with unqualified addresses.
	AddressPrefix string

	// Relationships are the validated relationships of the run.
	Relationships policy.Relationships

	// Contribute receives the walk's contribution once its changes and state
	// are final. It's called at most once per walk, from the walk.
	Contribute func(PolicyRunContribution)
}

// PolicyRunContribution is what a walk contributes to a SharedPolicyRun.
// Addresses are qualified with the run's AddressPrefix. The provider
// configurations of Instances.Providers are by their unqualified address in
// the walk, and the caller can replace their values.
type PolicyRunContribution struct {
	Instances *PolicyInstances

	// Subjects evaluate the walk's resource policies, one subject each, with
	// the related members from the relations of every walk, and report the
	// results through the walk's hooks. The context replaces the walk's stop
	// context, which is gone by then.
	Subjects []func(context.Context, *PolicyRelations) tfdiags.Diagnostics

	// Outputs resolves leaves through the walk's root module outputs in a
	// plan; nil in an apply, which has no origins.
	Outputs PolicyOutputOrigins
}

// PolicyOutputOrigins returns the origins of a leaf of a value that comes
// from a root module output of a walk: the output named output, the
// traversal rest into it, and then the path of the leaf. The origins'
// addresses are the walk's own, unqualified. A leaf that comes from a root
// module variable of the walk in turn is returned in vars, for the caller to
// resolve again.
type PolicyOutputOrigins func(output string, rest hcl.Traversal, leaf PolicyVariableOrigin) (origins []PolicyOrigin, vars []PolicyVariableOrigin)

// policyOutputOrigins returns the PolicyOutputOrigins of a walk with the
// given origin lookup.
func policyOutputOrigins(cfg *configs.Config, exp *instances.Expander, lookup *relationshipOriginLookup) PolicyOutputOrigins {
	return func(output string, rest hcl.Traversal, leaf PolicyVariableOrigin) ([]PolicyOrigin, []PolicyVariableOrigin) {
		if cfg == nil || cfg.Module.Outputs[output] == nil {
			return nil, nil
		}
		e := &originEval{cfg: cfg, exp: exp, overridden: lookup.overridden, lookup: lookup, rootVariables: true}
		sym := e.steps(e.eval(cfg.Module.Outputs[output].Expr, &originScope{mod: addrs.RootModuleInstance, cfg: cfg}), rest)
		for _, step := range leaf.path {
			if step.name != "" {
				sym = e.getAttr(sym, step.name)
			} else {
				sym = e.index(sym, symLit{val: cty.NumberIntVal(step.index)})
			}
		}
		switch sym := sym.(type) {
		case symRef:
			if len(sym.path) == 0 || !e.originAllowed(sym, leaf.ty) {
				return nil, nil
			}
			return []PolicyOrigin{{Address: sym.addr.String(), path: sym.path}}, nil
		case symVar:
			return nil, []PolicyVariableOrigin{{Variable: sym.name, path: sym.path, ty: leaf.ty}}
		default:
			return nil, nil
		}
	}
}

// policyEvaluationScope is how a resource evaluation finds its related
// members, and the prefix that qualifies the subject's address.
type policyEvaluationScope struct {
	addressPrefix string
	// relations are the relations of the walk, or of the shared run; nil
	// without relationships.
	relations *PolicyRelations
}

// contributeToSharedRun hands the walk's instances, with an evaluation per
// subject, to the shared run. lookup is the walk's origin lookup in a plan,
// and nil in an apply.
func (ps *policySubgraph) contributeToSharedRun(ctx EvalContext, inst *PolicyInstances, lookup *relationshipOriginLookup) {
	shared := ps.run.Shared
	if shared.Contribute == nil {
		return
	}
	inst.prefix(shared.AddressPrefix)

	var outputs PolicyOutputOrigins
	if lookup != nil {
		outputs = policyOutputOrigins(ctx.Config(), ctx.InstanceExpander(), lookup)
	}

	op := walkPlan
	if ps.run.Apply {
		op = walkApply
	}
	var subjects []func(context.Context, *PolicyRelations) tfdiags.Diagnostics
	deferredCtx := &deferredPolicyEvalContext{EvalContext: ctx, providers: make(map[string]deferredPolicyProvider)}
	ps.lock.Lock()
	for v := range ps.graph.VerticesSeq() {
		n, ok := v.(*nodeResourcePolicy)
		if !ok {
			continue
		}
		// The walk forgets its providers when it ends, so the subject keeps
		// the ones it needs. They are the external providers of the
		// caller, which keeps them open until it has evaluated every subject.
		key := n.ProviderAddr.String()
		if _, ok := deferredCtx.providers[key]; !ok {
			if provider := ctx.Provider(n.ProviderAddr); provider != nil {
				schema, err := ctx.ProviderSchema(n.ProviderAddr)
				if err == nil {
					deferredCtx.providers[key] = deferredPolicyProvider{provider, schema}
				}
			}
		}
		subjects = append(subjects, func(stop context.Context, relations *PolicyRelations) tfdiags.Diagnostics {
			scope := policyEvaluationScope{addressPrefix: shared.AddressPrefix, relations: relations}
			return n.evaluate(deferredCtx.withStop(stop), op, scope)
		})
	}
	ps.lock.Unlock()

	shared.Contribute(PolicyRunContribution{
		Instances: inst,
		Subjects:  subjects,
		Outputs:   outputs,
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

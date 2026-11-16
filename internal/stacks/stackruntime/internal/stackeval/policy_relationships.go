// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sync"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// policySubjectParallelism bounds the concurrent subject evaluations at the
// end of a deployment, like the walk's parallelism bounds those at the end of
// a walk.
const policySubjectParallelism = 10

// policyRun relates the instances of a deployment's plan or apply across its
// component instances (contract §3.8).
//
// The walks evaluate no policy themselves; they hand the run their instances
// and their subjects. Once every component instance is done,
// finishPolicyRun relates the instances of every walk and evaluates every
// subject once, with every policy (getresources still scoped to the
// subject's component).
type policyRun struct {
	main     *Main
	rels     policy.Relationships
	planning bool
	// generation is the policy phase of the run's value sources.
	generation uint64

	// mu guards the fields below.
	mu sync.Mutex
	// joined and contributed count the walks that joined the run and those
	// that contributed to it. A walk that joined and didn't contribute
	// failed before its policy phase.
	joined, contributed int
	subjects            []func(context.Context, *terraform.PolicyRelations) tfdiags.Diagnostics
	// walks are the walks that joined, by component instance address.
	walks map[string]*policyWalk
}

// policyWalk is a component instance's walk in the run.
type policyWalk struct {
	addr stackaddrs.AbsComponentInstance
	// inputs is the component's inputs expression, or nil.
	inputs hcl.Expression
	// providers are the stack provider configuration instances of the
	// walk's root provider configurations, by the walk's address.
	providers map[string]terraform.PolicyProvider
	// instances and outputs are those of the walk's contribution.
	instances *terraform.PolicyInstances
	outputs   terraform.PolicyOutputOrigins
}

// validatePolicyRelationships validates the policy client's relationship
// definitions once for the deployment (contract §3.3), against the schemas of
// every provider the deployment can use, and prepares the deployment's
// policy run if any definition remains. It must be called before the walks;
// with errors, the operation must not walk.
func (m *Main) validatePolicyRelationships(ctx context.Context, phase EvalPhase) tfdiags.Diagnostics {
	client := m.PolicyClient()
	if client == nil {
		return nil
	}
	rels := client.Relationships()
	if len(rels.Definitions) == 0 {
		return nil
	}
	schemas := make(map[addrs.Provider]providers.ProviderSchema)
	for addr := range m.providerFactories {
		schema, err := m.ProviderType(addr).Schema(ctx)
		if err != nil {
			log.Printf("[WARN] policy: no schema for %s to validate relationships: %s", addr, err)
			continue
		}
		schemas[addr] = schema
	}
	locks := m.DependencyLocks(phase)
	rels, diags := terraform.ValidatePolicyRelationships(rels, schemas, func(provider addrs.Provider) string {
		if locks == nil {
			return ""
		}
		if lock := locks.Provider(provider); lock != nil {
			return lock.Version().String()
		}
		return ""
	})
	m.mu.Lock()
	defer m.mu.Unlock()
	if diags.HasErrors() {
		m.policyInvalid = true
		return diags
	}
	if len(rels.Definitions) == 0 {
		return diags
	}
	m.policyRun = &policyRun{
		main:       m,
		rels:       rels,
		planning:   phase == PlanPhase,
		generation: client.NextGeneration(),
		walks:      make(map[string]*policyWalk),
	}
	return diags
}

// PolicyInvalid returns true if the policy client's relationship
// definitions are invalid for the deployment.
func (m *Main) PolicyInvalid() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.policyInvalid
}

// PolicyRun returns the deployment's policy run, or nil if it has none.
func (m *Main) PolicyRun() *policyRun {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.policyRun
}

// join returns the shared run for the walk of a component instance, or nil
// if the deployment has no run. The walk must contribute to the run or fail.
// inputs is the component's inputs expression, or nil. known are the stack
// provider configuration instances of the component's root provider
// configurations.
func (r *policyRun) join(ctx context.Context, addr stackaddrs.AbsComponentInstance, inputs hcl.Expression, known map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance) *terraform.SharedPolicyRun {
	if r == nil {
		return nil
	}
	walk := &policyWalk{addr: addr, inputs: inputs, providers: r.stackProviders(ctx, known)}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.joined++
	r.walks[addr.String()] = walk
	return &terraform.SharedPolicyRun{
		AddressPrefix: addr.String() + ".",
		Relationships: r.rels,
		Generation:    r.generation,
		Contribute: func(c terraform.PolicyRunContribution) {
			r.contribute(walk, c)
		},
	}
}

// contribute keeps a walk's contribution for finishPolicyRun. A provider
// configuration from the stack is the stack's provider configuration
// instance, whichever components it's passed to.
func (r *policyRun) contribute(walk *policyWalk, c terraform.PolicyRunContribution) {
	for addr, p := range c.Instances.Providers {
		if sp, ok := walk.providers[addr]; ok {
			*p = sp
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contributed++
	r.subjects = append(r.subjects, c.Subjects...)
	walk.instances = c.Instances
	walk.outputs = c.Outputs
}

// stackProviders returns the stack provider configuration instances of a
// component instance's root provider configurations, by the walk's address,
// with their values.
func (r *policyRun) stackProviders(ctx context.Context, known map[addrs.RootProviderConfig]stackaddrs.AbsProviderConfigInstance) map[string]terraform.PolicyProvider {
	phase := PlanPhase
	if !r.planning {
		phase = ApplyPhase
	}
	ret := make(map[string]terraform.PolicyProvider, len(known))
	for root, addr := range known {
		p := terraform.PolicyProvider{Addr: addr.String()}
		walkAddr := root.AbsProviderConfig().String()
		ret[walkAddr] = p
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
		p.Addr = fmt.Sprintf("provider.%s.%s", local.LocalName, local.Alias)
		if addr.Item.Key != addrs.NoKey {
			p.Addr += addr.Item.Key.String()
		}
		if !addr.Stack.IsRoot() {
			p.Addr = addr.Stack.String() + "." + p.Addr
		}
		if args := inst.ProviderArgs(ctx, phase); args != cty.NilVal {
			if args, _ = args.UnmarkDeep(); args.IsWhollyKnown() {
				p.Config = args
			}
		}
		ret[walkAddr] = p
	}
	return ret
}

// policyMaxComponentHops bounds how many component inputs and outputs the
// origin of a value is followed through.
const policyMaxComponentHops = 16

// resolveVariable returns the origins of a leaf whose value comes from a root
// module variable of a walk's component instance (contract §3.8): if the
// component's inputs set the variable to a direct reference to another
// component instance's output, the origins of that output, qualified with the
// producing component instance. r.mu must be held.
func (r *policyRun) resolveVariable(walk *policyWalk, v terraform.PolicyVariableOrigin, hops int) []terraform.PolicyOrigin {
	if hops >= policyMaxComponentHops {
		return nil
	}
	producerAddr, output, rest, ok := componentOutputRef(walk.inputs, v.Variable, walk.addr)
	if !ok {
		return nil
	}
	producer := r.walks[producerAddr.String()]
	if producer == nil || producer.outputs == nil {
		return nil
	}
	origins, vars := producer.outputs(output, rest, v)
	prefix := producerAddr.String() + "."
	for i := range origins {
		origins[i].Address = prefix + origins[i].Address
	}
	for _, next := range vars {
		origins = append(origins, r.resolveVariable(producer, next, hops+1)...)
	}
	return origins
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

// finishPolicyRun relates the instances of every component instance once
// every one is done, and evaluates the policies of every subject. errored is
// set when the operation had errors: a component instance that failed before
// its walk didn't contribute its instances, so every hop is ERROR.
func (m *Main) finishPolicyRun(ctx context.Context, errored bool) tfdiags.Diagnostics {
	r := m.PolicyRun()
	if r == nil {
		return nil
	}

	r.mu.Lock()
	var insts []*terraform.PolicyInstances
	for _, walk := range r.walks {
		if walk.instances == nil {
			continue
		}
		// Every walk has contributed, so the origins of values passed
		// between components can be resolved now.
		walk.instances.ResolveVariables(func(v terraform.PolicyVariableOrigin) []terraform.PolicyOrigin {
			return r.resolveVariable(walk, v, 0)
		})
		insts = append(insts, walk.instances)
	}
	relations := terraform.NewPolicyRelations(r.rels, r.planning, errored || r.joined != r.contributed, insts...)
	subjects := r.subjects
	r.subjects = nil
	r.mu.Unlock()

	var diags tfdiags.Diagnostics
	if ctx.Err() != nil {
		return diags
	}
	var wg sync.WaitGroup
	var diagsMu sync.Mutex
	sem := make(chan struct{}, policySubjectParallelism)
	for _, subject := range subjects {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			moreDiags := subject(ctx, relations)
			diagsMu.Lock()
			diags = diags.Append(moreDiags)
			diagsMu.Unlock()
		}()
	}
	wg.Wait()
	return diags
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

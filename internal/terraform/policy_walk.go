// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/policy"
	"go.opentelemetry.io/otel/trace"
)

// policyWalkState owns policy evaluation data for one Terraform walk. Collection
// populates the graph, inventory, completeness information, and origins before
// evaluations read the immutable resource inventory and prepared policy values.
type policyWalkState struct {
	lock  sync.Mutex
	graph Graph

	// span carries the tracing information. We need the span itself so we can end it
	// when the policy evaluation is finished
	span trace.Span

	// resourceMap is a map of resource addresses to their policyResource.
	// typeResourceMap is a map of types to their policyResource.
	resourceMap     map[string]*policyResource
	typeResourceMap map[string][]*policyResource

	// applyChanges retains the input changes before apply consumes them
	applyChanges addrs.Map[addrs.AbsResourceInstance, *plans.ResourceInstanceChange]

	// unknownTypes contains resource types with unknown expansion and deferred instances,
	// which makes their overall total incomplete
	unknownTypes map[string]bool

	// origins is a map of an attribute to the other attribute that its expression
	// points to.
	origins map[string]globalref.ResourceAttr
}

// policyResource is the immutable record of a resource used by matching and callbacks.
type policyResource struct {
	Addr         addrs.AbsResourceInstance
	ProviderAddr addrs.AbsProviderConfig
	Value        policy.PolicyValue
	PriorValue   policy.PolicyValue
	Action       plans.Action
}

func newPolicyWalkState(trackOrigins bool) *policyWalkState {
	state := &policyWalkState{
		resourceMap:     make(map[string]*policyResource),
		typeResourceMap: make(map[string][]*policyResource),
		applyChanges:    addrs.MakeMap[addrs.AbsResourceInstance, *plans.ResourceInstanceChange](),
		unknownTypes:    make(map[string]bool),
	}
	if trackOrigins {
		state.origins = make(map[string]globalref.ResourceAttr)
	}
	return state
}

func (ps *policyWalkState) markUnknownResourceType(typ string) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.unknownTypes[typ] = true
}

func (ps *policyWalkState) hasUnknowns(resourceType string) bool {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	_, ok := ps.unknownTypes[resourceType]
	return ok
}

func (ps *policyWalkState) addResourceNode(node *nodeResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.graph.Add(node)
}

func (ps *policyWalkState) addQueryNode(node *nodeQueryResourcePolicy) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.graph.Add(node)
}

func (ps *policyWalkState) addResource(resource *policyResource) {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.resourceMap[resource.Addr.String()] = resource
	ps.typeResourceMap[resource.Addr.Resource.Resource.Type] = append(ps.typeResourceMap[resource.Addr.Resource.Resource.Type], resource)
}

func (ps *policyWalkState) evalGraph(span trace.Span) *Graph {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	ps.span = span

	finish := &nodePolicyEvalFinish{span: span}
	ps.graph.Add(finish)
	for pn := range ps.graph.VerticesSeq() {
		// Wire finish only to policy node types; all other vertices are skipped.
		switch pn.(type) {
		case *nodeResourcePolicy, *nodeQueryResourcePolicy:
		default:
			continue
		}
		// finish depends on pn, so pn runs first and finish runs after.
		ps.graph.Connect(finish, pn)
	}

	return &ps.graph
}

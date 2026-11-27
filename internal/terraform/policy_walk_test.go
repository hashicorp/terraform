// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/globalref"
	"github.com/hashicorp/terraform/internal/policy"
)

func TestPolicyWalkStateInitialization(t *testing.T) {
	graph := newPolicyWalkState(false)
	if graph.resourceMap == nil || graph.typeResourceMap == nil || graph.unknownTypes == nil || graph.applyChanges.Elems == nil {
		t.Fatal("walk indexes must be initialized without origins")
	}
	resource := &policyResource{Addr: mustResourceInstanceAddr("test_source.a")}
	graph.addResource(resource)
	if graph.resourceMap[resource.Addr.String()] != resource || graph.typeResourceMap["test_source"][0] != resource {
		t.Fatal("inventory indexes must share the same resource records")
	}
	graph.markUnknownResourceType("test_target")
	if !graph.hasUnknowns("test_target") {
		t.Fatal("walk lost its completeness state")
	}
}

func TestPolicyWalkStateOriginsAreWalkScoped(t *testing.T) {
	client := policy.NewTestMockClient(t)
	client.ConfigValue.Relationships = map[string][]policy.Relationship{"test_source": {{Name: "targets"}}}
	ctx := &Context{}
	opts := &graphWalkOpts{Config: configs.NewEmptyConfig(), PolicyClient: client}
	first := ctx.graphWalker(&Graph{}, walkPlan, opts).PolicyState
	second := ctx.graphWalker(&Graph{}, walkPlan, opts).PolicyState
	apply := ctx.graphWalker(&Graph{}, walkApply, opts).PolicyState
	if first == second || first.origins == nil || second.origins == nil || apply.origins != nil {
		t.Fatal("origins must be plan-only and isolated between walks")
	}
	source := &policyResource{Addr: mustResourceInstanceAddr("test_source.a"), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{
		"id": cty.UnknownVal(cty.String).RefineNotNull(),
	}))}
	var targets []*policyResource
	var captures sync.WaitGroup
	for _, addr := range []string{"test_target.b", "test_target.c"} {
		target := &policyResource{Addr: mustResourceInstanceAddr(addr), Value: policy.CtyToPolicyValue(cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String)}))}
		targets = append(targets, target)
		first.addResource(target)
		captures.Go(func() {
			first.recordOrigin(
				globalref.ResourceAttr{Resource: target.Addr, Attr: cty.GetAttrPath("id")},
				globalref.ResourceAttr{Resource: source.Addr, Attr: cty.GetAttrPath("id")},
			)
		})
	}
	captures.Wait()
	var comparisons sync.WaitGroup
	for _, target := range targets {
		if _, ok := second.origin(globalref.ResourceAttr{Resource: target.Addr, Attr: cty.GetAttrPath("id")}); ok || len(second.resourceMap) != 0 {
			t.Fatal("mutable policy state escaped into another walk")
		}
		comparisons.Go(func() {
			key := policy.RelationshipKey{KeyPath: cty.GetAttrPath("id"), ValuePath: cty.GetAttrPath("id")}
			if got := first.checkEquality(source, target, key); !got.RawEquals(cty.True) {
				t.Errorf("shared origin comparison = %#v, want true", got)
			}
		})
	}
	comparisons.Wait()
}

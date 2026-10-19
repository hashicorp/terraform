// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	policyproto "github.com/hashicorp/terraform/internal/policy/proto"
	"github.com/hashicorp/terraform/internal/providers"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
)

// The bucket already exists and doesn't change, so the apply skips its
// component. Like a Terraform CLI apply reports and evaluates its unchanged
// instances, the deployment's run still has the bucket's record, from the
// component's state, evaluates it, and its type is complete.
func TestApply_PolicyRelationshipsSkippedComponent(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "policy-relationships-cross-component")

	store := stacks_testing_provider.NewResourceStoreBuilder().
		AddResource("bucket", cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("bucket"),
			"value": cty.StringVal("bucket"),
		})).
		Build()
	providerFactories := map[addrs.Provider]providers.Factory{
		addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
			return stacks_testing_provider.NewProviderWithData(t, store), nil
		},
	}
	prevState := stackstate.NewStateBuilder().
		AddComponentInstance(stackstate.NewComponentInstanceBuilder(mustAbsComponentInstance("component.bucket"))).
		AddResourceInstance(stackstate.NewResourceInstanceBuilder().
			SetAddr(mustAbsResourceInstanceObject("component.bucket.testing_resource.bucket")).
			SetProviderAddr(mustDefaultRootProvider("testing")).
			SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
				Status: states.ObjectReady,
				AttrsJSON: mustMarshalJSONAttrs(map[string]any{
					"id":    "bucket",
					"value": "bucket",
				}),
			})).
		Build()

	plan := planForApplyTest(t, ctx, PlanRequest{
		PlanMode:          plans.NormalMode,
		Config:            cfg,
		PrevState:         prevState,
		ProviderFactories: providerFactories,
		DependencyLocks:   relationshipsTestLocks(),
	})
	if c := plan.GetComponent(mustAbsComponentInstance("component.bucket")); c == nil || c.PlanApplyable {
		t.Fatalf("want the bucket's component not applyable, got %#v", c)
	}

	client := newRelationshipsTestClient(t)
	got := applyAndCollectPolicyResults(t, ctx, ApplyRequest{
		Config:            cfg,
		Plan:              plan,
		ProviderFactories: providerFactories,
		DependencyLocks:   relationshipsTestLocks(),
		PolicyClient:      client,
	})

	checkDeploymentRun(t, client, policyproto.EvaluationStage_APPLY_EVALUATION_STAGE)
	checkRelationshipResults(t, got)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestApplyWithRemovedResource(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	attrs := map[string]interface{}{
		"id": "FE1D5830765C",
		"input": map[string]interface{}{
			"value": "hello",
			"type":  "string",
		},
		"output": map[string]interface{}{
			"value": nil,
			"type":  "string",
		},
		"triggers_replace": nil,
	}
	attrsJSON, err := json.Marshal(attrs)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, path.Join("empty-component", "valid-providers"))

	planReq := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},

		ForcePlanTimestamp: &fakePlanTimestamp,

		// PrevState specifies a state with a resource that is not present in
		// the current configuration. This is a common situation when a resource
		// is removed from the configuration but still exists in the state.
		PrevState: stackstate.NewStateBuilder().
			AddResourceInstance(stackstate.NewResourceInstanceBuilder().
				SetAddr(stackaddrs.AbsResourceInstanceObject{
					Component: stackaddrs.AbsComponentInstance{
						Stack: stackaddrs.RootStackInstance,
						Item: stackaddrs.ComponentInstance{
							Component: stackaddrs.Component{
								Name: "self",
							},
							Key: addrs.NoKey,
						},
					},
					Item: addrs.AbsResourceInstanceObject{
						ResourceInstance: addrs.AbsResourceInstance{
							Module: addrs.RootModuleInstance,
							Resource: addrs.ResourceInstance{
								Resource: addrs.Resource{
									Mode: addrs.ManagedResourceMode,
									Type: "terraform_data",
									Name: "main",
								},
								Key: addrs.NoKey,
							},
						},
						DeposedKey: addrs.NotDeposed,
					},
				}).
				SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
					SchemaVersion: 0,
					AttrsJSON:     attrsJSON,
					Status:        states.ObjectReady,
				}).
				SetProviderAddr(addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("terraform.io/builtin/terraform"),
				})).
			Build(),
	}

	planChangesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	planResp := PlanResponse{
		PlannedChanges: planChangesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &planReq, &planResp)
	planChanges, diags := collectPlanOutput(planChangesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, go %s", diags.ErrWithWarnings())
	}

	var raw []*anypb.Any
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		raw = append(raw, proto.Raw...)
	}

	applyReq := ApplyRequest{
		Config:  cfg,
		RawPlan: raw,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		Complete:       false,
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
	}

	if len(applyChanges) != 2 {
		t.Fatalf("expected 2 applied changes, got %d", len(applyChanges))
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr: stackaddrs.AbsComponent{
				Item: stackaddrs.Component{
					Name: "self",
				},
			},
			ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
				Item: stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{
						Name: "self",
					},
				},
			},
			OutputValues: make(map[addrs.OutputValue]cty.Value),
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: stackaddrs.AbsResourceInstanceObject{
				Component: stackaddrs.AbsComponentInstance{
					Item: stackaddrs.ComponentInstance{
						Component: stackaddrs.Component{
							Name: "self",
						},
					},
				},
				Item: addrs.AbsResourceInstanceObject{
					ResourceInstance: addrs.AbsResourceInstance{
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "terraform_data",
								Name: "main",
							},
						},
					},
				},
			},
			NewStateSrc: nil, // Deleted, so is nil.
			ProviderConfigAddr: addrs.AbsProviderConfig{
				Provider: addrs.Provider{
					Type:      "terraform",
					Namespace: "builtin",
					Hostname:  "terraform.io",
				},
			},
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		// An arbitrary sort just to make the result stable for comparison.
		return fmt.Sprintf("%T", applyChanges[i]) < fmt.Sprintf("%T", applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func collectApplyOutput(changesCh <-chan stackstate.AppliedChange, diagsCh <-chan tfdiags.Diagnostic) ([]stackstate.AppliedChange, tfdiags.Diagnostics) {
	var changes []stackstate.AppliedChange
	var diags tfdiags.Diagnostics
	for {
		select {
		case change, ok := <-changesCh:
			if !ok {
				// The plan operation is complete but we might still have
				// some buffered diagnostics to consume.
				if diagsCh != nil {
					for diag := range diagsCh {
						diags = append(diags, diag)
					}
				}
				return changes, diags
			}
			changes = append(changes, change)
		case diag, ok := <-diagsCh:
			if !ok {
				// no more diagnostics to read
				diagsCh = nil
				continue
			}
			diags = append(diags, diag)
		}
	}
}

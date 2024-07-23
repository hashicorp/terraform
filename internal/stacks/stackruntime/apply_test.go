// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/internal/stackeval"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestApplyWithRemovedResource(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, path.Join("empty-component", "valid-providers"))
	lock := depsfile.NewLocks()
	planReq := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,

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
					AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
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
					}),
					Status: states.ObjectReady,
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

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewBuiltInProvider("terraform"): func() (providers.Interface, error) {
				return terraformProvider.NewProvider(), nil
			},
		},
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
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
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyWithMovedResource(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1994-09-05T08:50:00Z")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, path.Join("state-manipulation", "moved"))

	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)

	planReq := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(stacks_testing_provider.NewResourceStoreBuilder().
					AddResource("moved", cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("moved"),
						"value": cty.StringVal("moved"),
					})).
					Build()), nil
			},
		},
		DependencyLocks: *lock,

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
									Type: "testing_resource",
									Name: "before",
								},
								Key: addrs.NoKey,
							},
						},
						DeposedKey: addrs.NotDeposed,
					},
				}).
				SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
					SchemaVersion: 0,
					AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
						"id":    "moved",
						"value": "moved",
					}),
					Status: states.ObjectReady,
				}).
				SetProviderAddr(addrs.AbsProviderConfig{
					Module:   addrs.RootModule,
					Provider: addrs.MustParseProviderSourceString("hashicorp/testing"),
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

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(stacks_testing_provider.NewResourceStoreBuilder().
					AddResource("moved", cty.ObjectVal(map[string]cty.Value{
						"id":    cty.StringVal("moved"),
						"value": cty.StringVal("moved"),
					})).
					Build()), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
	}

	expectedPreviousAddr := mustAbsResourceInstanceObject("component.self.testing_resource.before")

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
								Type: "testing_resource",
								Name: "after",
							},
						},
					},
				},
			},
			PreviousResourceInstanceObjectAddr: &expectedPreviousAddr,
			NewStateSrc: &states.ResourceInstanceObjectSrc{
				AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
					"id":    "moved",
					"value": "moved",
				}),
				Status:             states.ObjectReady,
				AttrSensitivePaths: make([]cty.Path, 0),
			},
			ProviderConfigAddr: addrs.AbsProviderConfig{
				Provider: addrs.MustParseProviderSourceString("hashicorp/testing"),
			},
			Schema: stacks_testing_provider.TestingResourceSchema,
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyWithSensitivePropagation(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, path.Join("with-single-input", "sensitive-input"))

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,

		ForcePlanTimestamp: &fakePlanTimestamp,

		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "id"}: {
				Value: cty.StringVal("bb5cf32312ec"),
			},
		},
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
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
								Type: "testing_resource",
								Name: "data",
							},
						},
					},
				},
			},
			NewStateSrc: &states.ResourceInstanceObjectSrc{
				AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
					"id":    "bb5cf32312ec",
					"value": "secret",
				}),
				AttrSensitivePaths: []cty.Path{
					cty.GetAttrPath("value"),
				},
				Status:       states.ObjectReady,
				Dependencies: make([]addrs.ConfigResource, 0),
			},
			ProviderConfigAddr: addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("testing"),
			},
			Schema: stacks_testing_provider.TestingResourceSchema,
		},
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr: stackaddrs.AbsComponent{
				Item: stackaddrs.Component{
					Name: "sensitive",
				},
			},
			ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
				Item: stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{
						Name: "sensitive",
					},
				},
			},
			OutputValues: map[addrs.OutputValue]cty.Value{
				addrs.OutputValue{Name: "out"}: cty.StringVal("secret").Mark(marks.Sensitive),
			},
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyWithCheckableObjects(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "checkable-objects")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,

		ForcePlanTimestamp: &fakePlanTimestamp,

		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "foo"}: {
				Value: cty.StringVal("bar"),
			},
		},
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}
	var wantDiags tfdiags.Diagnostics
	wantDiags = wantDiags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,

		Summary: "Check block assertion failed",
		Detail:  `value must be 'baz'`,
		Subject: &hcl.Range{
			Filename: mainBundleSourceAddrStr("checkable-objects/checkable-objects.tf"),
			Start:    hcl.Pos{Line: 32, Column: 21, Byte: 532},
			End:      hcl.Pos{Line: 32, Column: 57, Byte: 568},
		},
	})

	go Plan(ctx, &req, &resp)
	planChanges, planDiags := collectPlanOutput(changesCh, diagsCh)

	if diff := cmp.Diff(wantDiags.ForRPC(), planDiags.ForRPC()); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if diff := cmp.Diff(wantDiags.ForRPC(), applyDiags.ForRPC()); diff != "" {
		t.Errorf("wrong diagnostics\n%s", diff)
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr: stackaddrs.AbsComponent{
				Item: stackaddrs.Component{
					Name: "single",
				},
			},
			ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
				Item: stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{
						Name: "single",
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
							Name: "single",
						},
					},
				},
				Item: addrs.AbsResourceInstanceObject{
					ResourceInstance: addrs.AbsResourceInstance{
						Resource: addrs.ResourceInstance{
							Resource: addrs.Resource{
								Mode: addrs.ManagedResourceMode,
								Type: "testing_resource",
								Name: "main",
							},
						},
					},
				},
			},
			NewStateSrc: &states.ResourceInstanceObjectSrc{
				AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
					"id":    "test",
					"value": "bar",
				}),
				Status:       states.ObjectReady,
				Dependencies: make([]addrs.ConfigResource, 0),
			},
			ProviderConfigAddr: addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("testing"),
			},
			Schema: stacks_testing_provider.TestingResourceSchema,
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyWithForcePlanTimestamp(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-plantimestamp")

	forcedPlanTimestamp := "1991-08-25T20:57:08Z"
	fakePlanTimestamp, err := time.Parse(time.RFC3339, forcedPlanTimestamp)
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		ForcePlanTimestamp: &fakePlanTimestamp,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}
	// Sanity check that the plan timestamp was set correctly
	output := expectOutput(t, "plantimestamp", planChanges)
	plantimestampValue, err := output.NewValue.Decode(cty.String)
	if err != nil {
		t.Fatal(err)
	}

	if plantimestampValue.AsString() != forcedPlanTimestamp {
		t.Errorf("expected plantimestamp to be %q, got %q", forcedPlanTimestamp, plantimestampValue.AsString())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr: stackaddrs.AbsComponent{
				Item: stackaddrs.Component{
					Name: "second-self",
				},
			},
			ComponentInstanceAddr: stackaddrs.AbsComponentInstance{
				Item: stackaddrs.ComponentInstance{
					Component: stackaddrs.Component{
						Name: "second-self",
					},
				},
			},
			OutputValues: map[addrs.OutputValue]cty.Value{
				// We want to make sure the plantimestamp is set correctly
				{Name: "input"}: cty.StringVal(forcedPlanTimestamp),
				// plantimestamp should also be set for the module runtime used in the components
				{Name: "out"}: cty.StringVal(fmt.Sprintf("module-output-%s", forcedPlanTimestamp)),
			},
		},
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
			OutputValues: map[addrs.OutputValue]cty.Value{
				// We want to make sure the plantimestamp is set correctly
				{Name: "input"}: cty.StringVal(forcedPlanTimestamp),
				// plantimestamp should also be set for the module runtime used in the components
				{Name: "out"}: cty.StringVal(fmt.Sprintf("module-output-%s", forcedPlanTimestamp)),
			},
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyWithDefaultPlanTimestamp(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-plantimestamp")

	dayOfWritingThisTest := "2024-06-21T06:37:08Z"
	dayOfWritingThisTestTime, err := time.Parse(time.RFC3339, dayOfWritingThisTest)
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}
	// Sanity check that the plan timestamp was set correctly
	output := expectOutput(t, "plantimestamp", planChanges)
	plantimestampValue, err := output.NewValue.Decode(cty.String)
	if err != nil {
		t.Fatal(err)
	}

	plantimestamp, err := time.Parse(time.RFC3339, plantimestampValue.AsString())
	if err != nil {
		t.Fatal(err)
	}

	if plantimestamp.Before(dayOfWritingThisTestTime) {
		t.Errorf("expected plantimestamp to be later than %q, got %q", dayOfWritingThisTest, plantimestampValue.AsString())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", applyDiags.ErrWithWarnings())
	}

	for _, x := range applyChanges {
		if v, ok := x.(*stackstate.AppliedChangeComponentInstance); ok {
			if actualTimestampValue, ok := v.OutputValues[addrs.OutputValue{
				Name: "input",
			}]; ok {
				actualTimestamp, err := time.Parse(time.RFC3339, actualTimestampValue.AsString())
				if err != nil {
					t.Fatalf("Could not parse component output value: %q", err)
				}
				if actualTimestamp.Before(dayOfWritingThisTestTime) {
					t.Error("Timestamp is before day of writing this test, that should be incorrect.")
				}
			}

			if actualTimestampValue, ok := v.OutputValues[addrs.OutputValue{
				Name: "out",
			}]; ok {
				actualTimestamp, err := time.Parse(time.RFC3339, strings.ReplaceAll(actualTimestampValue.AsString(), "module-output-", ""))
				if err != nil {
					t.Fatalf("Could not parse component output value: %q", err)
				}
				if actualTimestamp.Before(dayOfWritingThisTestTime) {
					t.Error("Timestamp is before day of writing this test, that should be incorrect.")
				}
			}
		}
	}
}

func TestApplyWithFailedComponent(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, filepath.Join("with-single-input", "failed-parent"))

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}
	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)

	expectDiagnosticsForTest(t, applyDiags,
		// This is the expected failure, from our testing_failed_resource.
		expectDiagnostic(tfdiags.Error, "planned failure", "apply failure"))

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.parent"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_failed_resource.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
		},
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}

}

func TestApplyWithFailedProviderLinkedComponent(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, filepath.Join("with-single-input", "failed-component-to-provider"))

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}
	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)

	expectDiagnosticsForTest(t, applyDiags,
		// This is the expected failure, from our testing_failed_resource.
		expectDiagnostic(tfdiags.Error, "planned failure", "apply failure"))

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.parent"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.parent"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.parent.testing_failed_resource.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
		},
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}

}

func TestApplyWithStateManipulation(t *testing.T) {
	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)

	tcs := map[string]struct {
		state            *stackstate.State
		store            *stacks_testing_provider.ResourceStore
		inputs           map[string]cty.Value
		changes          []stackstate.AppliedChange
		counts           collections.Map[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]
		expectedWarnings []string
	}{
		"moved": {
			state: stackstate.NewStateBuilder().
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject("component.self.testing_resource.before")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "moved",
							"value": "moved",
						}),
					})).
				Build(),
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("moved", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("moved"),
					"value": cty.StringVal("moved"),
				})).
				Build(),
			changes: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.after"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "moved",
							"value": "moved",
						}),
						Status:             states.ObjectReady,
						AttrSensitivePaths: make([]cty.Path, 0),
					},
					ProviderConfigAddr:                 mustDefaultRootProvider("testing"),
					PreviousResourceInstanceObjectAddr: mustAbsResourceInstanceObjectPtr("component.self.testing_resource.before"),
					Schema:                             stacks_testing_provider.TestingResourceSchema,
				},
			},
			counts: collections.NewMap[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange](
				collections.MapElem[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]{
					K: mustAbsComponentInstance("component.self"),
					V: &hooks.ComponentInstanceChange{
						Addr: mustAbsComponentInstance("component.self"),
						Move: 1,
					},
				}),
		},
		"import": {
			state: stackstate.NewStateBuilder().Build(), // We start with an empty state for this.
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("imported", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("imported"),
					"value": cty.StringVal("imported"),
				})).
				Build(),
			inputs: map[string]cty.Value{
				"id": cty.StringVal("imported"),
			},
			changes: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "imported",
							"value": "imported",
						}),
						Status:             states.ObjectReady,
						AttrSensitivePaths: make([]cty.Path, 0),
					},
					ProviderConfigAddr: mustDefaultRootProvider("testing"),
					Schema:             stacks_testing_provider.TestingResourceSchema,
				},
			},
			counts: collections.NewMap[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange](
				collections.MapElem[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]{
					K: mustAbsComponentInstance("component.self"),
					V: &hooks.ComponentInstanceChange{
						Addr:   mustAbsComponentInstance("component.self"),
						Import: 1,
					},
				}),
		},
		"removed": {
			state: stackstate.NewStateBuilder().
				AddResourceInstance(stackstate.NewResourceInstanceBuilder().
					SetAddr(mustAbsResourceInstanceObject("component.self.testing_resource.resource")).
					SetProviderAddr(mustDefaultRootProvider("testing")).
					SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
						Status: states.ObjectReady,
						AttrsJSON: mustMarshalJSONAttrs(map[string]any{
							"id":    "removed",
							"value": "removed",
						}),
					})).
				Build(),
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("removed", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("removed"),
					"value": cty.StringVal("removed"),
				})).
				Build(),
			changes: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.self"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.resource"),
					NewStateSrc:                nil, // Deleted, so is nil.
					ProviderConfigAddr:         mustDefaultRootProvider("testing"),
				},
			},
			counts: collections.NewMap[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange](
				collections.MapElem[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]{
					K: mustAbsComponentInstance("component.self"),
					V: &hooks.ComponentInstanceChange{
						Addr:   mustAbsComponentInstance("component.self"),
						Forget: 1,
					},
				}),
			expectedWarnings: []string{"Some objects will no longer be managed by Terraform"},
		},
		"deferred": {
			store: stacks_testing_provider.NewResourceStoreBuilder().
				AddResource("self", cty.ObjectVal(map[string]cty.Value{
					"id":    cty.StringVal("deferred"),
					"value": cty.UnknownVal(cty.String),
				})).
				Build(),
			changes: []stackstate.AppliedChange{
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.deferred"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.deferred"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
				},
				&stackstate.AppliedChangeComponentInstance{
					ComponentAddr:         mustAbsComponent("component.ok"),
					ComponentInstanceAddr: mustAbsComponentInstance("component.ok"),
					OutputValues:          make(map[addrs.OutputValue]cty.Value),
				},
				&stackstate.AppliedChangeResourceInstanceObject{
					ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.ok.testing_resource.self"),
					NewStateSrc: &states.ResourceInstanceObjectSrc{
						AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
							"id":    "ok",
							"value": "ok",
						}),
						Status:             states.ObjectReady,
						AttrSensitivePaths: nil,
						Dependencies:       []addrs.ConfigResource{},
					},
					ProviderConfigAddr:                 mustDefaultRootProvider("testing"),
					PreviousResourceInstanceObjectAddr: nil,
					Schema:                             stacks_testing_provider.TestingResourceSchema,
				},
			},
			counts: collections.NewMap[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange](
				collections.MapElem[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]{
					K: mustAbsComponentInstance("component.ok"),
					V: &hooks.ComponentInstanceChange{
						Addr:  mustAbsComponentInstance("component.ok"),
						Add:   1,
						Defer: 0,
					},
				},
				collections.MapElem[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]{
					K: mustAbsComponentInstance("component.deferred"),
					V: &hooks.ComponentInstanceChange{
						Addr:  mustAbsComponentInstance("component.deferred"),
						Defer: 1,
					},
				},
			),
		},
	}

	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {

			ctx := context.Background()
			cfg := loadMainBundleConfigForTest(t, path.Join("state-manipulation", name))

			inputs := make(map[stackaddrs.InputVariable]ExternalInputValue, len(tc.inputs))
			for name, input := range tc.inputs {
				inputs[stackaddrs.InputVariable{Name: name}] = ExternalInputValue{
					Value: input,
				}
			}

			providers := map[addrs.Provider]providers.Factory{
				addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
					return stacks_testing_provider.NewProviderWithData(tc.store), nil
				},
			}

			planChangeCh := make(chan stackplan.PlannedChange)
			diagsCh := make(chan tfdiags.Diagnostic)
			planReq := PlanRequest{
				Config:             cfg,
				ProviderFactories:  providers,
				InputValues:        inputs,
				ForcePlanTimestamp: &fakePlanTimestamp,
				PrevState:          tc.state,
				DependencyLocks:    *lock,
			}
			planResp := PlanResponse{
				PlannedChanges: planChangeCh,
				Diagnostics:    diagsCh,
			}
			go Plan(ctx, &planReq, &planResp)
			planChanges, diags := collectPlanOutput(planChangeCh, diagsCh)
			reportDiagnosticsForTest(t, diags)
			if diags.HasErrors() {
				// we reported the diagnostics above, so we can just fail now
				t.FailNow()
			}
			if len(diags) > len(tc.expectedWarnings) {
				t.Fatalf("had unexpected warnings")
			}
			for i, diag := range diags {
				if diag.Description().Summary != tc.expectedWarnings[i] {
					t.Fatalf("expected diagnostic with summary %q, got %q", tc.expectedWarnings[i], diag.Description().Summary)
				}
			}

			// Check the counts during the apply for this test.
			gotCounts := collections.NewMap[stackaddrs.AbsComponentInstance, *hooks.ComponentInstanceChange]()
			ctx = ContextWithHooks(ctx, &stackeval.Hooks{
				ReportComponentInstanceApplied: func(ctx context.Context, span any, change *hooks.ComponentInstanceChange) any {
					gotCounts.Put(change.Addr, change)
					return span
				},
			})

			planLoader := stackplan.NewLoader()
			for _, change := range planChanges {
				proto, err := change.PlannedChangeProto()
				if err != nil {
					t.Fatal(err)
				}

				for _, rawMsg := range proto.Raw {
					err = planLoader.AddRaw(rawMsg)
					if err != nil {
						t.Fatal(err)
					}
				}
			}
			plan, err := planLoader.Plan()
			if err != nil {
				t.Fatal(err)
			}

			applyReq := ApplyRequest{
				Config:            cfg,
				Plan:              plan,
				ProviderFactories: providers,
				DependencyLocks:   *lock,
			}
			applyChangesCh := make(chan stackstate.AppliedChange)
			diagsCh = make(chan tfdiags.Diagnostic)
			applyResp := ApplyResponse{
				AppliedChanges: applyChangesCh,
				Diagnostics:    diagsCh,
			}

			go Apply(ctx, &applyReq, &applyResp)
			applyChanges, diags := collectApplyOutput(applyChangesCh, diagsCh)
			reportDiagnosticsForTest(t, diags)
			if diags.HasErrors() {
				// we reported the diagnostics above, so we can just fail now
				t.FailNow()
			}
			if len(diags) > 0 {
				t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
			}

			sort.SliceStable(applyChanges, func(i, j int) bool {
				return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
			})

			if diff := cmp.Diff(tc.changes, applyChanges, ctydebug.CmpOptions, cmpCollectionsSet); diff != "" {
				t.Errorf("wrong changes\n%s", diff)
			}

			wantCounts := tc.counts
			for _, elem := range wantCounts.Elems() {
				// First, make sure everything we wanted is present.
				if !gotCounts.HasKey(elem.K) {
					t.Errorf("wrong counts: wanted %s but didn't get it", elem.K)
				}

				// And that the values actually match.
				got, want := gotCounts.Get(elem.K), elem.V
				if diff := cmp.Diff(want, got); diff != "" {
					t.Errorf("wrong counts for %s: %s", want.Addr, diff)
				}

			}

			for _, elem := range gotCounts.Elems() {
				// Then, make sure we didn't get anything we didn't want.
				if !wantCounts.HasKey(elem.K) {
					t.Errorf("wrong counts: got %s but didn't want it", elem.K)
				}
			}
		})
	}
}

func TestApplyWithChangedValues(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, filepath.Join("with-single-input", "valid"))

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "1991-08-25T20:57:08Z")
	if err != nil {
		t.Fatal(err)
	}

	changesCh := make(chan stackplan.PlannedChange)
	diagsCh := make(chan tfdiags.Diagnostic)
	lock := depsfile.NewLocks()
	lock.SetProvider(
		addrs.NewDefaultProvider("testing"),
		providerreqs.MustParseVersion("0.0.0"),
		providerreqs.MustParseVersionConstraints("=0.0.0"),
		providerreqs.PreferredHashes([]providerreqs.Hash{}),
	)
	req := PlanRequest{
		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,

		ForcePlanTimestamp: &fakePlanTimestamp,

		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "input"}: {
				Value: cty.StringVal("hello"),
			},
		},
	}
	resp := PlanResponse{
		PlannedChanges: changesCh,
		Diagnostics:    diagsCh,
	}

	go Plan(ctx, &req, &resp)
	planChanges, diags := collectPlanOutput(changesCh, diagsCh)
	if len(diags) > 0 {
		t.Fatalf("expected no diagnostics, got %s", diags.ErrWithWarnings())
	}

	planLoader := stackplan.NewLoader()
	for _, change := range planChanges {
		proto, err := change.PlannedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			err = planLoader.AddRaw(rawMsg)
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	plan, err := planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	// Now deliberately edit the plan so that it'll produce different outputs.
	plan.RootInputValues[stackaddrs.InputVariable{Name: "input"}] = cty.StringVal("world")

	applyReq := ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	diagsCh = make(chan tfdiags.Diagnostic)

	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    diagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	_, applyDiags := collectApplyOutput(applyChangesCh, diagsCh)
	if len(applyDiags) != 1 {
		t.Fatalf("expected exactly one diagnostic, got %s", applyDiags.ErrWithWarnings())
	}

	// We don't care about the changes here, just that we got the diagnostic
	// we expected when we messed the plan up earlier.

	gotSeverity, wantSeverity := applyDiags[0].Severity(), tfdiags.Error
	gotSummary, wantSummary := applyDiags[0].Description().Summary, "Planned input variable value changed"
	gotDetail, wantDetail := applyDiags[0].Description().Detail, "The planned value for input variable \"input\" has changed between the planning and apply phases for component.self. This is a bug in Terraform - please report it."

	if gotSeverity != wantSeverity {
		t.Errorf("expected severity %q, got %q", wantSeverity, gotSeverity)
	}
	if gotSummary != wantSummary {
		t.Errorf("expected summary %q, got %q", wantSummary, gotSummary)
	}
	if gotDetail != wantDetail {
		t.Errorf("expected detail %q, got %q", wantDetail, gotDetail)
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

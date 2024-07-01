// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	terraformProvider "github.com/hashicorp/terraform/internal/builtin/providers/terraform"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
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

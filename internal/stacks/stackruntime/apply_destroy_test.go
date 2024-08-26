// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"context"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackplan"
	stacks_testing_provider "github.com/hashicorp/terraform/internal/stacks/stackruntime/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestApplyDestroyMissingResource(t *testing.T) {

	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, filepath.Join("with-single-input", "valid"))

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
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

	planReq := PlanRequest{
		PlanMode: plans.DestroyMode,

		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProvider(), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "input"}: {
				Value: cty.StringVal("hello"),
			},
		},

		// We have in the previous state a resource that is not in our
		// underlying data store. This simulates the case where someone went
		// in and manually deleted a resource that Terraform is managing.
		//
		// Some providers will return an error in this case, but some will
		// not. We need to ensure that we handle the second case gracefully.
		PrevState: stackstate.NewStateBuilder().
			AddResourceInstance(stackstate.NewResourceInstanceBuilder().
				SetAddr(mustAbsResourceInstanceObject("component.self.testing_resource.data")).
				SetProviderAddr(mustDefaultRootProvider("testing")).
				SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
					SchemaVersion: 0,
					AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
						"id":    "e84b59f2",
						"value": "hello",
					}),
					Status: states.ObjectReady,
				})).
			Build(),
	}

	planChangesCh := make(chan stackplan.PlannedChange)
	planDiagsCh := make(chan tfdiags.Diagnostic)
	planResp := PlanResponse{
		PlannedChanges: planChangesCh,
		Diagnostics:    planDiagsCh,
	}

	go Plan(ctx, &planReq, &planResp)
	planChanges, planDiags := collectPlanOutput(planChangesCh, planDiagsCh)
	if len(planDiags) > 0 {
		t.Fatalf("unexpected diagnostics during planning: %s", planDiags)
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
	applyDiagsCh := make(chan tfdiags.Diagnostic)
	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    applyDiagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, applyDiagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("unexpected diagnostics during apply: %s", applyDiags)
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
		// The resource that was in state but not in the data store should still
		// be included to be destroyed.
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			NewStateSrc:                nil, // We should be removing this from the state file.
			Schema:                     nil,
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, changesCmpOpts); diff != "" {
		t.Errorf("wrong changes\n%s", diff)
	}
}

func TestApplyDestroyWithDataSourceInState(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-data-source")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
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

	store := stacks_testing_provider.NewResourceStoreBuilder().
		AddResource("foo", cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("foo"),
			"value": cty.StringVal("hello"),
		})).Build()

	planReq := PlanRequest{
		PlanMode: plans.DestroyMode,

		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "id"}: {
				Value: cty.StringVal("foo"),
			},
			stackaddrs.InputVariable{Name: "resource"}: {
				Value: cty.StringVal("bar"),
			},
		},

		// We have a forgotten data source in the state file, this basically
		// means we've removed it from the config file since the last apply.
		// We should get a notice telling us that it is being removed.
		PrevState: stackstate.NewStateBuilder().
			AddResourceInstance(stackstate.NewResourceInstanceBuilder().
				SetAddr(mustAbsResourceInstanceObject("component.self.data.testing_data_source.missing")).
				SetProviderAddr(mustDefaultRootProvider("testing")).
				SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
					SchemaVersion: 0,
					AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
						"id":    "e84b59f2",
						"value": "hello",
					}),
					Status: states.ObjectReady,
				})).
			Build(),
	}

	planChangesCh := make(chan stackplan.PlannedChange)
	planDiagsCh := make(chan tfdiags.Diagnostic)
	planResp := PlanResponse{
		PlannedChanges: planChangesCh,
		Diagnostics:    planDiagsCh,
	}

	go Plan(ctx, &planReq, &planResp)
	planChanges, planDiags := collectPlanOutput(planChangesCh, planDiagsCh)
	if len(planDiags) > 0 {
		t.Fatalf("unexpected diagnostics during planning: %s", planDiags)
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
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	applyDiagsCh := make(chan tfdiags.Diagnostic)
	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    applyDiagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, applyDiagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("unexpected diagnostics during apply: %s", applyDiags)
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},

		// This is a bit of a quirk of the system, this wasn't in the state
		// file before so we don't need to emit this. But since Terraform
		// pushes data sources into the refresh state, it's very difficult to
		// tell the difference between this kind of change that doesn't need to
		// be emitted, and the next change that does need to be emitted. It's
		// better to emit both than to miss one, and emitting this doesn't
		// actually harm anything.
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.data.testing_data_source.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			Schema:                     nil,
			NewStateSrc:                nil, // deleted
		},

		// This was in the state file, so we're emitting the destroy notice.
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.data.testing_data_source.missing"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			Schema:                     nil,
			NewStateSrc:                nil,
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, changesCmpOpts); diff != "" {
		t.Fatalf("wrong changes\n%s", diff)
	}
}

func TestApplyDestroyWithDataSource(t *testing.T) {
	ctx := context.Background()
	cfg := loadMainBundleConfigForTest(t, "with-data-source")

	fakePlanTimestamp, err := time.Parse(time.RFC3339, "2021-01-01T00:00:00Z")
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

	store := stacks_testing_provider.NewResourceStoreBuilder().
		AddResource("foo", cty.ObjectVal(map[string]cty.Value{
			"id":    cty.StringVal("foo"),
			"value": cty.StringVal("hello"),
		})).Build()

	planReq := PlanRequest{
		PlanMode: plans.NormalMode,

		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "id"}: {
				Value: cty.StringVal("foo"),
			},
			stackaddrs.InputVariable{Name: "resource"}: {
				Value: cty.StringVal("bar"),
			},
		},

		// We have a forgotten data source in the state file, this basically
		// means we've removed it from the config file since the last apply.
		// We should get a notice telling us that it is being removed.
		PrevState: stackstate.NewStateBuilder().
			AddResourceInstance(stackstate.NewResourceInstanceBuilder().
				SetAddr(mustAbsResourceInstanceObject("component.self.data.testing_data_source.missing")).
				SetProviderAddr(mustDefaultRootProvider("testing")).
				SetResourceInstanceObjectSrc(states.ResourceInstanceObjectSrc{
					SchemaVersion: 0,
					AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
						"id":    "e84b59f2",
						"value": "hello",
					}),
					Status: states.ObjectReady,
				})).
			Build(),
	}

	planChangesCh := make(chan stackplan.PlannedChange)
	planDiagsCh := make(chan tfdiags.Diagnostic)
	planResp := PlanResponse{
		PlannedChanges: planChangesCh,
		Diagnostics:    planDiagsCh,
	}

	go Plan(ctx, &planReq, &planResp)
	planChanges, planDiags := collectPlanOutput(planChangesCh, planDiagsCh)
	if len(planDiags) > 0 {
		t.Fatalf("unexpected diagnostics during planning: %s", planDiags)
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
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh := make(chan stackstate.AppliedChange)
	applyDiagsCh := make(chan tfdiags.Diagnostic)
	applyResp := ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    applyDiagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags := collectApplyOutput(applyChangesCh, applyDiagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("unexpected diagnostics during apply: %s", applyDiags)
	}

	wantChanges := []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.data.testing_data_source.data"),
			NewStateSrc: &states.ResourceInstanceObjectSrc{
				AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
					"id":    "foo",
					"value": "hello",
				}),
				AttrSensitivePaths: make([]cty.Path, 0),
				Status:             states.ObjectReady,
			},
			ProviderConfigAddr: mustDefaultRootProvider("testing"),
			Schema:             stacks_testing_provider.TestingDataSourceSchema,
		},
		// This data source should be removed from the state file as it is no
		// longer in the configuration.
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.data.testing_data_source.missing"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			Schema:                     nil,
			NewStateSrc:                nil, // deleted
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
			NewStateSrc: &states.ResourceInstanceObjectSrc{
				AttrsJSON: mustMarshalJSONAttrs(map[string]interface{}{
					"id":    "bar",
					"value": "hello",
				}),
				Status: states.ObjectReady,
				Dependencies: []addrs.ConfigResource{
					mustAbsResourceInstance("data.testing_data_source.data").ConfigResource(),
				},
			},
			ProviderConfigAddr: mustDefaultRootProvider("testing"),
			Schema:             stacks_testing_provider.TestingResourceSchema,
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, changesCmpOpts); diff != "" {
		t.Fatalf("wrong changes\n%s", diff)
	}

	// Now, let's destroy everything.

	stateLoader := stackstate.NewLoader()
	for _, change := range applyChanges {
		proto, err := change.AppliedChangeProto()
		if err != nil {
			t.Fatal(err)
		}

		for _, rawMsg := range proto.Raw {
			if rawMsg.Value == nil {
				// This is a removal notice, so we don't need to add it to the
				// state.
				continue
			}
			err = stateLoader.AddRaw(rawMsg.Key, rawMsg.Value)
			if err != nil {
				t.Fatal(err)
			}
		}
	}

	planReq = PlanRequest{
		PlanMode: plans.DestroyMode,

		Config: cfg,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks:    *lock,
		ForcePlanTimestamp: &fakePlanTimestamp,
		InputValues: map[stackaddrs.InputVariable]ExternalInputValue{
			stackaddrs.InputVariable{Name: "id"}: {
				Value: cty.StringVal("foo"),
			},
			stackaddrs.InputVariable{Name: "resource"}: {
				Value: cty.StringVal("bar"),
			},
		},
		PrevState: stateLoader.State(),
	}

	planChangesCh = make(chan stackplan.PlannedChange)
	planDiagsCh = make(chan tfdiags.Diagnostic)
	planResp = PlanResponse{
		PlannedChanges: planChangesCh,
		Diagnostics:    planDiagsCh,
	}

	go Plan(ctx, &planReq, &planResp)
	planChanges, planDiags = collectPlanOutput(planChangesCh, planDiagsCh)
	if len(planDiags) > 0 {
		t.Fatalf("unexpected diagnostics during planning: %s", planDiags)
	}

	planLoader = stackplan.NewLoader()
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

	plan, err = planLoader.Plan()
	if err != nil {
		t.Fatal(err)
	}

	applyReq = ApplyRequest{
		Config: cfg,
		Plan:   plan,
		ProviderFactories: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("testing"): func() (providers.Interface, error) {
				return stacks_testing_provider.NewProviderWithData(store), nil
			},
		},
		DependencyLocks: *lock,
	}

	applyChangesCh = make(chan stackstate.AppliedChange)
	applyDiagsCh = make(chan tfdiags.Diagnostic)
	applyResp = ApplyResponse{
		AppliedChanges: applyChangesCh,
		Diagnostics:    applyDiagsCh,
	}

	go Apply(ctx, &applyReq, &applyResp)
	applyChanges, applyDiags = collectApplyOutput(applyChangesCh, applyDiagsCh)
	if len(applyDiags) > 0 {
		t.Fatalf("unexpected diagnostics during apply: %s", applyDiags)
	}

	wantChanges = []stackstate.AppliedChange{
		&stackstate.AppliedChangeComponentInstance{
			ComponentAddr:         mustAbsComponent("component.self"),
			ComponentInstanceAddr: mustAbsComponentInstance("component.self"),
			OutputValues:          make(map[addrs.OutputValue]cty.Value),
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.data.testing_data_source.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			Schema:                     nil,
			NewStateSrc:                nil, // deleted
		},
		&stackstate.AppliedChangeResourceInstanceObject{
			ResourceInstanceObjectAddr: mustAbsResourceInstanceObject("component.self.testing_resource.data"),
			ProviderConfigAddr:         mustDefaultRootProvider("testing"),
			Schema:                     nil,
			NewStateSrc:                nil, // deleted
		},
	}

	sort.SliceStable(applyChanges, func(i, j int) bool {
		return appliedChangeSortKey(applyChanges[i]) < appliedChangeSortKey(applyChanges[j])
	})

	if diff := cmp.Diff(wantChanges, applyChanges, changesCmpOpts); diff != "" {
		t.Fatalf("wrong changes\n%s", diff)
	}
}

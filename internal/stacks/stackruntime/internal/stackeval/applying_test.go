// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"log"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/collections"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackstate"
)

func TestApply_componentOrdering(t *testing.T) {
	// This verifies that component instances have their plans applied in a
	// suitable order during the apply phase, both for normal plans and for
	// destroy plans.
	//
	// This test also creates a plan using the normal planning logic, so
	// it partially acts as an integration test for planning and applying
	// with component inter-dependencies (since the plan phase is the one
	// responsible for actually calculating the dependencies.)
	//
	// Since this is testing some concurrent code, the test might produce
	// false-positives if things just happen to occur in the right order
	// despite the sequencing code being incorrect. Consider running this
	// test under the Go data race detector to find memory-safety-related
	// problems, but also keep in mind that not all sequencing problems are
	// caused by data races.
	//
	// If this test seems to be flaking and the race detector doesn't dig up
	// any clues, you might consider the following:
	//  - Is the code in function ApplyPlan waiting for all of the prerequisites
	//    captured in the plan? Is it honoring the reversed order expected
	//    for destroy plans?
	//  - Is the ChangeExec function, and its subsequent execution, correctly
	//    scheduling all of the apply tasks that were registered?
	//
	// If other tests in this package (or that call into this package) are
	// also consistently failing, it'd likely be more productive to debug and
	// fix those first, which might then give a clue as to what's making this
	// test misbehave.

	cfg := testStackConfig(t, "applying", "component_dependencies")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	testProviderAddr := addrs.NewBuiltInProvider("test")
	testProviderSchema := providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_report": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"marker": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
			},
		},
	}

	cmpAAddr := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.Component{
			Name: "a",
		},
	}
	cmpBAddr := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.Component{
			Name: "b",
		},
	}
	cmpBInst1Addr := stackaddrs.AbsComponentInstance{
		Stack: cmpBAddr.Stack,
		Item: stackaddrs.ComponentInstance{
			Component: cmpBAddr.Item,
			Key:       addrs.StringKey("i"),
		},
	}
	cmpCAddr := stackaddrs.AbsComponent{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.Component{
			Name: "c",
		},
	}
	cmpCInstAddr := stackaddrs.AbsComponentInstance{
		Stack: cmpCAddr.Stack,
		Item: stackaddrs.ComponentInstance{
			Component: cmpCAddr.Item,
			Key:       addrs.NoKey,
		},
	}

	// First we need to create a plan for this configuration, which will
	// include the calculated component dependencies.
	planOutput, err := promising.MainTask(ctx, func(ctx context.Context) (*planOutputTester, error) {
		main := NewForPlanning(cfg, stackstate.NewState(), PlanOpts{
			PlanningMode: plans.NormalMode,
			ProviderFactories: ProviderFactories{
				testProviderAddr: func() (providers.Interface, error) {
					return &testing_provider.MockProvider{
						GetProviderSchemaResponse: &testProviderSchema,
						PlanResourceChangeFn: func(prcr providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
							return providers.PlanResourceChangeResponse{
								PlannedState: prcr.ProposedNewState,
							}
						},
					}, nil
				},
			},
			PlanTimestamp: time.Now().UTC(),
		})

		outp, outpTester := testPlanOutput(t)
		main.PlanAll(ctx, outp)

		return outpTester, nil
	})
	if err != nil {
		t.Fatalf("planning failed: %s", err)
	}

	rawPlan := planOutput.RawChanges(t)
	plan, diags := planOutput.Close(t)
	assertNoDiagnostics(t, diags)

	// Before we proceed further we'll check that the plan contains the
	// expected dependency relationships, because missing dependency edges
	// will make the following tests invalid, and testing this is not
	// subject to concurrency-related false-positives.
	//
	// This is not comprehensive, because the dependency calculation logic
	// should already be tested more completely elsewhere. If this part fails
	// then hopefully at least one of the planning-specific tests is also
	// failing, and will give some more clues as to what's gone wrong here.
	if !plan.Applyable {
		m := prototext.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		for _, raw := range rawPlan {
			t.Log(m.Format(raw))
		}
		t.Fatalf("plan is not applyable")
	}
	{
		cmpPlan := plan.Components.Get(cmpCInstAddr)
		gotDeps := cmpPlan.Dependencies
		wantDeps := collections.NewSet[stackaddrs.AbsComponent]()
		wantDeps.Add(cmpBAddr)
		if diff := cmp.Diff(wantDeps, gotDeps, collections.CmpOptions); diff != "" {
			t.Fatalf("wrong dependencies for component.c\n%s", diff)
		}
	}
	{
		cmpPlan := plan.Components.Get(cmpBInst1Addr)
		gotDeps := cmpPlan.Dependencies
		wantDeps := collections.NewSet[stackaddrs.AbsComponent]()
		wantDeps.Add(cmpAAddr)
		if diff := cmp.Diff(wantDeps, gotDeps, collections.CmpOptions); diff != "" {
			t.Fatalf("wrong dependencies for component.b[\"i\"]\n%s", diff)
		}
	}

	type applyResultData struct {
		NewRawState    map[string]*anypb.Any
		NewState       *stackstate.State
		VisitedMarkers []string
	}

	// Now we're finally ready for the first apply, during which we expect
	// the component ordering decided during the plan phase to be respected.
	applyResult, err := promising.MainTask(ctx, func(ctx context.Context) (applyResultData, error) {
		var visitedMarkers []string
		var visitedMarkersMu sync.Mutex

		outp, outpTester := testApplyOutput(t, nil)

		main, err := ApplyPlan(ctx, cfg, plan, ApplyOpts{
			ProviderFactories: ProviderFactories{
				testProviderAddr: func() (providers.Interface, error) {
					return &testing_provider.MockProvider{
						GetProviderSchemaResponse: &testProviderSchema,
						ApplyResourceChangeFn: func(arcr providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
							markerStr := arcr.PlannedState.GetAttr("marker").AsString()
							log.Printf("[TRACE] TestApply_componentOrdering: visiting %q", markerStr)
							visitedMarkersMu.Lock()
							visitedMarkers = append(visitedMarkers, markerStr)
							visitedMarkersMu.Unlock()

							return providers.ApplyResourceChangeResponse{
								NewState: arcr.PlannedState,
							}
						},
					}, nil
				},
			},
		}, outp)
		if main != nil {
			defer main.DoCleanup(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}

		assertNoDiagnostics(t, outpTester.Diags())

		rawState := outpTester.RawUpdatedState(t)
		state, diags := outpTester.Close(t)
		assertNoDiagnostics(t, diags)

		return applyResultData{
			NewRawState:    rawState,
			NewState:       state,
			VisitedMarkers: visitedMarkers,
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	{
		if len(applyResult.VisitedMarkers) != 5 {
			t.Fatalf("apply didn't visit all of the resources\n%s", spew.Sdump(applyResult.VisitedMarkers))
		}
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"a", "b.i",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"a", "b.ii",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"a", "b.iii",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.i", "c",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.ii", "c",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.iii", "c",
		)
	}

	// If the initial plan and apply was successful and made its changes in
	// the correct order, then we'll also test creating and applying a
	// destroy-mode plan.
	t.Log("destroy plan")
	planOutput, err = promising.MainTask(ctx, func(ctx context.Context) (*planOutputTester, error) {
		main := NewForPlanning(cfg, applyResult.NewState, PlanOpts{
			PlanningMode: plans.DestroyMode,
			ProviderFactories: ProviderFactories{
				testProviderAddr: func() (providers.Interface, error) {
					return &testing_provider.MockProvider{
						GetProviderSchemaResponse: &testProviderSchema,
						PlanResourceChangeFn: func(prcr providers.PlanResourceChangeRequest) providers.PlanResourceChangeResponse {
							return providers.PlanResourceChangeResponse{
								PlannedState: prcr.ProposedNewState,
							}
						},
					}, nil
				},
			},
			PlanTimestamp: time.Now().UTC(),
		})

		outp, outpTester := testPlanOutput(t)
		main.PlanAll(ctx, outp)

		return outpTester, nil
	})
	if err != nil {
		t.Fatalf("planning failed: %s", err)
	}

	rawPlan = planOutput.RawChanges(t)
	plan, diags = planOutput.Close(t)
	assertNoDiagnostics(t, diags)
	if !plan.Applyable {
		m := prototext.MarshalOptions{
			Multiline: true,
			Indent:    "  ",
		}
		for _, raw := range rawPlan {
			t.Log(m.Format(raw))
		}
		t.Fatalf("plan is not applyable")
	}

	// When we apply the destroy plan, the components should be visited in
	// reverse dependency order to ensure that dependencies outlive their
	// dependents.
	t.Log("destroy apply")
	applyResult, err = promising.MainTask(ctx, func(ctx context.Context) (applyResultData, error) {
		var visitedMarkers []string
		var visitedMarkersMu sync.Mutex

		outp, outpTester := testApplyOutput(t, nil)

		main, err := ApplyPlan(ctx, cfg, plan, ApplyOpts{
			ProviderFactories: ProviderFactories{
				testProviderAddr: func() (providers.Interface, error) {
					return &testing_provider.MockProvider{
						GetProviderSchemaResponse: &testProviderSchema,
						ApplyResourceChangeFn: func(arcr providers.ApplyResourceChangeRequest) providers.ApplyResourceChangeResponse {
							markerStr := arcr.PriorState.GetAttr("marker").AsString()
							log.Printf("[TRACE] TestApply_componentOrdering: visiting %q", markerStr)
							visitedMarkersMu.Lock()
							visitedMarkers = append(visitedMarkers, markerStr)
							visitedMarkersMu.Unlock()

							return providers.ApplyResourceChangeResponse{
								NewState: arcr.PlannedState,
							}
						},
					}, nil
				},
			},
		}, outp)
		if main != nil {
			defer main.DoCleanup(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}

		assertNoDiagnostics(t, outpTester.Diags())

		rawState := outpTester.RawUpdatedState(t)
		state, diags := outpTester.Close(t)
		assertNoDiagnostics(t, diags)

		return applyResultData{
			NewRawState:    rawState,
			NewState:       state,
			VisitedMarkers: visitedMarkers,
		}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	{
		if len(applyResult.VisitedMarkers) != 5 {
			t.Fatalf("apply didn't visit all of the resources\n%s", spew.Sdump(applyResult.VisitedMarkers))
		}
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.i", "a",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.ii", "a",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"b.iii", "a",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"c", "b.i",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"c", "b.ii",
		)
		assertSliceElementsInRelativeOrder(
			t, applyResult.VisitedMarkers,
			"c", "b.iii",
		)
	}
}

func sliceElementsInRelativeOrder[S ~[]E, E comparable](s S, v1, v2 E) bool {
	idx1 := slices.Index(s, v1)
	idx2 := slices.Index(s, v2)
	if idx1 < 0 || idx2 < 0 {
		// both values must actually be present for this test to be meaningful
		return false
	}
	return idx1 < idx2
}

func assertSliceElementsInRelativeOrder[S ~[]E, E comparable](t *testing.T, s S, v1, v2 E) {
	t.Helper()

	if !sliceElementsInRelativeOrder(s, v1, v2) {
		t.Fatalf("incorrect element order\ngot: %s\nwant: %#v before %#v", strings.TrimSpace(spew.Sdump(s)), v1, v2)
	}
}

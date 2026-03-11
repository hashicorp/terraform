// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty-debug/ctydebug"
	"github.com/zclconf/go-cty/cty"
)

func TestChangeExec(t *testing.T) {
	ctx := context.Background()

	type FakeMain struct {
		results *ChangeExecResults
	}
	instAAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "A",
			},
		},
	}
	instBAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "B",
			},
		},
	}
	// We don't actually register a task for instCAddr; this one's here
	// to test how we handle requesting the result from an unregistered task.
	instCAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{
				Name: "C",
			},
		},
	}
	valueAddr := addrs.OutputValue{Name: "v"}.Absolute(addrs.RootModuleInstance)

	_, err := promising.MainTask(ctx, func(ctx context.Context) (FakeMain, error) {
		changeResults, begin := ChangeExec(ctx, func(ctx context.Context, reg *ChangeExecRegistry[FakeMain]) {
			t.Logf("begin setup phase")
			reg.RegisterComponentInstanceChange(ctx, instAAddr, func(ctx context.Context, main FakeMain) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
				t.Logf("producing result for A")
				return &ComponentInstanceApplyResult{
					FinalState: states.BuildState(func(ss *states.SyncState) {
						ss.SetOutputValue(valueAddr, cty.StringVal("a"), false)
					}),
				}, nil
			})
			reg.RegisterComponentInstanceChange(ctx, instBAddr, func(ctx context.Context, main FakeMain) (*ComponentInstanceApplyResult, tfdiags.Diagnostics) {
				t.Logf("B is waiting for A")
				aState, _, err := main.results.ComponentInstanceResult(ctx, instAAddr)
				if err != nil {
					return nil, nil
				}
				t.Logf("producing result for B")
				aOutputVal := aState.FinalState.OutputValue(valueAddr)
				if aOutputVal == nil {
					return nil, nil
				}
				return &ComponentInstanceApplyResult{
					FinalState: states.BuildState(func(ss *states.SyncState) {
						ss.SetOutputValue(
							valueAddr, cty.TupleVal([]cty.Value{aOutputVal.Value, cty.StringVal("b")}), false,
						)
					}),
				}, nil
			})
			t.Logf("end setup phase")
		})

		main := FakeMain{
			results: changeResults,
		}

		// We must call "begin" before this task returns, since internally
		// there's now a promise that our task is responsible for resolving.
		t.Logf("about to start execution phase")
		begin(ctx, main)

		// Now we'll pretend that we're doing normal stackeval stuff that
		// involves some interdependencies between the results. Specifically,
		// the "B" task depends on the result from the "A" task.
		var wg sync.WaitGroup
		wg.Add(3)
		var gotAResult, gotBResult, gotCResult *ComponentInstanceApplyResult
		var errA, errB, errC error
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result C")
			gotCResult, _, errC = main.results.ComponentInstanceResult(ctx, instCAddr)
			t.Logf("got result C")
			wg.Done()
		})
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result B")
			gotBResult, _, errB = main.results.ComponentInstanceResult(ctx, instBAddr)
			t.Logf("got result B")
			wg.Done()
		})
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result A")
			gotAResult, _, errA = main.results.ComponentInstanceResult(ctx, instAAddr)
			t.Logf("got result A")
			wg.Done()
		})
		wg.Wait()

		if errA != nil {
			t.Errorf("A failed: %s", errA)
		}
		if errB != nil {
			t.Errorf("B failed: %s", errB)
		}
		if diff := cmp.Diff(ErrChangeExecUnregistered{instCAddr}, errC); diff != "" {
			t.Errorf("wrong error for C\n%s", diff)
		}
		if errA != nil || errB != nil {
			t.FailNow()
		}
		if gotAResult == nil {
			t.Fatal("A state is nil")
		}
		if gotBResult == nil {
			t.Fatal("B state is nil")
		}
		if gotCResult != nil {
			t.Fatal("C state isn't nil, but should have been")
		}

		gotAOutputVal := gotAResult.FinalState.OutputValue(valueAddr)
		if gotAOutputVal == nil {
			t.Fatal("A state has no value")
		}
		gotBOutputVal := gotBResult.FinalState.OutputValue(valueAddr)
		if gotBOutputVal == nil {
			t.Fatal("B state has no value")
		}

		gotAVal := gotAOutputVal.Value
		wantAVal := cty.StringVal("a")
		gotBVal := gotBOutputVal.Value
		wantBVal := cty.TupleVal([]cty.Value{wantAVal, cty.StringVal("b")})
		if diff := cmp.Diff(wantAVal, gotAVal, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result for A\n%s", diff)
		}
		if diff := cmp.Diff(wantBVal, gotBVal, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result for B\n%s", diff)
		}

		return main, nil
	})
	if err != nil {
		t.Fatal(err)
	}

}

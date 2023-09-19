package stackeval

import (
	"context"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/promising"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
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
	_, err := promising.MainTask(ctx, func(ctx context.Context) (FakeMain, error) {
		changeResults, begin := ChangeExec(ctx, func(ctx context.Context, reg *ChangeExecRegistry[FakeMain]) {
			t.Logf("begin setup phase")
			reg.RegisterComponentInstanceChange(ctx, instAAddr, func(ctx context.Context, main FakeMain) cty.Value {
				t.Logf("producing result for A")
				return cty.StringVal("a")
			})
			reg.RegisterComponentInstanceChange(ctx, instBAddr, func(ctx context.Context, main FakeMain) cty.Value {
				t.Logf("B is waiting for A")
				aVal, err := main.results.ComponentInstanceResult(ctx, instAAddr)
				if err != nil {
					return cty.DynamicVal
				}
				t.Logf("producing result for B")
				return cty.TupleVal([]cty.Value{aVal, cty.StringVal("b")})
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
		var gotAVal, gotBVal, gotCVal cty.Value
		var errA, errB, errC error
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result C")
			gotCVal, errC = main.results.ComponentInstanceResult(ctx, instCAddr)
			t.Logf("C is %#v", gotCVal)
			wg.Done()
		})
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result B")
			gotBVal, errB = main.results.ComponentInstanceResult(ctx, instBAddr)
			t.Logf("B is %#v", gotBVal)
			wg.Done()
		})
		promising.AsyncTask(ctx, promising.NoPromises, func(ctx context.Context, _ promising.PromiseContainer) {
			t.Logf("requesting result A")
			gotAVal, errA = main.results.ComponentInstanceResult(ctx, instAAddr)
			t.Logf("A is %#v", gotAVal)
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

		wantAVal := cty.StringVal("a")
		wantBVal := cty.TupleVal([]cty.Value{wantAVal, cty.StringVal("b")})
		wantCVal := cty.DynamicVal
		if diff := cmp.Diff(wantAVal, gotAVal, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result for A\n%s", diff)
		}
		if diff := cmp.Diff(wantBVal, gotBVal, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result for B\n%s", diff)
		}
		if diff := cmp.Diff(wantCVal, gotCVal, ctydebug.CmpOptions); diff != "" {
			t.Errorf("wrong result for C\n%s", diff)
		}

		return main, nil
	})
	if err != nil {
		t.Fatal(err)
	}

}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestMetaRunOperation_UsesBackgroundContextWithoutCallerContext(t *testing.T) {
	backend := &testRunOperationBackend{
		runningOp: testCompletedRunningOperation(),
	}

	meta := &Meta{}
	_, err := meta.RunOperation(backend, &backendrun.Operation{
		View: &testRunOperationView{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if backend.operationCtx == nil {
		t.Fatal("backend Operation called with nil context")
	}
	if got, want := backend.operationCtx, context.Background(); got != want {
		t.Fatalf("wrong context used for backend operation\n got: %T %[1]v\nwant: %T %[2]v", got, want)
	}
}

func TestMetaRunOperation_UsesCallerContextWhenSet(t *testing.T) {
	t.Skip("enabled in task 2 when RunOperation propagates Meta.CommandContext")

	type ctxKey string

	callerCtx := context.WithValue(context.Background(), ctxKey("k"), "v")
	backend := &testRunOperationBackend{
		runningOp: testCompletedRunningOperation(),
	}

	meta := &Meta{
		CallerContext: callerCtx,
	}
	_, err := meta.RunOperation(backend, &backendrun.Operation{
		View: &testRunOperationView{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if got, want := backend.operationCtx, callerCtx; got != want {
		t.Fatalf("backend operation did not receive caller context\n got: %p\nwant: %p", got, want)
	}
	if got, want := backend.operationCtx.Value(ctxKey("k")), "v"; got != want {
		t.Fatalf("caller context value was not preserved\n got: %v\nwant: %v", got, want)
	}
}

func TestMetaRunOperation_SingleInterruptStopsOperation(t *testing.T) {
	order := &testRunOperationOrder{}
	view := &testRunOperationView{record: order.add}
	shutdownCh := make(chan struct{}, 1)

	doneCtx, doneCancel := context.WithCancel(context.Background())
	backend := &testRunOperationBackend{
		onOperation: func() {
			shutdownCh <- struct{}{}
		},
		runningOp: &backendrun.RunningOperation{
			Context: doneCtx,
			Stop: func() {
				order.add("stop")
				doneCancel()
			},
			Cancel: func() {
				order.add("cancel")
				doneCancel()
			},
			Result: backendrun.OperationFailure,
		},
	}

	meta := &Meta{
		ShutdownCh: shutdownCh,
	}
	_, err := meta.RunOperation(backend, &backendrun.Operation{
		View: view,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	got := order.snapshot()
	assertRunOperationOrderContains(t, got, "stop", "interrupted")
	assertRunOperationOrderNotContains(t, got, "cancel", "fatal-interrupt")
	if !beforeInSlice(got, "stop", "interrupted") {
		t.Fatalf("expected stop before interrupted callback, got order: %#v", got)
	}
}

func TestMetaRunOperation_DoubleInterruptCancelsOperation(t *testing.T) {
	order := &testRunOperationOrder{}
	view := &testRunOperationView{record: order.add}
	shutdownCh := make(chan struct{}, 2)

	doneCtx, doneCancel := context.WithCancel(context.Background())
	backend := &testRunOperationBackend{
		onOperation: func() {
			shutdownCh <- struct{}{}
			shutdownCh <- struct{}{}
		},
		runningOp: &backendrun.RunningOperation{
			Context: doneCtx,
			Stop: func() {
				order.add("stop")
				// Do not finish here so the second interrupt path is exercised.
			},
			Cancel: func() {
				order.add("cancel")
				doneCancel()
			},
			Result: backendrun.OperationFailure,
		},
	}

	meta := &Meta{
		ShutdownCh: shutdownCh,
	}
	_, err := meta.RunOperation(backend, &backendrun.Operation{
		View: view,
	})
	if err == nil {
		t.Fatal("expected error after forced cancel")
	}
	if got, want := err.Error(), "operation canceled"; got != want {
		t.Fatalf("wrong error\n got: %q\nwant: %q", got, want)
	}

	got := order.snapshot()
	assertRunOperationOrderContains(t, got, "stop", "interrupted", "fatal-interrupt", "cancel")
	if !beforeInSlice(got, "stop", "cancel") {
		t.Fatalf("expected stop before cancel, got order: %#v", got)
	}
	if !beforeInSlice(got, "interrupted", "fatal-interrupt") {
		t.Fatalf("expected Interrupted before FatalInterrupt, got order: %#v", got)
	}
}

type testRunOperationBackend struct {
	operationCtx context.Context
	operationReq *backendrun.Operation
	runningOp    *backendrun.RunningOperation
	onOperation  func()
}

func (b *testRunOperationBackend) ConfigSchema() *configschema.Block {
	return &configschema.Block{}
}

func (b *testRunOperationBackend) PrepareConfig(v cty.Value) (cty.Value, tfdiags.Diagnostics) {
	return v, nil
}

func (b *testRunOperationBackend) Configure(cty.Value) tfdiags.Diagnostics {
	return nil
}

func (b *testRunOperationBackend) StateMgr(string) (statemgr.Full, tfdiags.Diagnostics) {
	return nil, nil
}

func (b *testRunOperationBackend) DeleteWorkspace(string, bool) tfdiags.Diagnostics {
	return nil
}

func (b *testRunOperationBackend) Workspaces() ([]string, tfdiags.Diagnostics) {
	return []string{"default"}, nil
}

func (b *testRunOperationBackend) Operation(ctx context.Context, op *backendrun.Operation) (*backendrun.RunningOperation, error) {
	b.operationCtx = ctx
	b.operationReq = op
	if b.onOperation != nil {
		b.onOperation()
	}
	if b.runningOp == nil {
		b.runningOp = testCompletedRunningOperation()
	}
	return b.runningOp, nil
}

func (b *testRunOperationBackend) ServiceDiscoveryAliases() ([]backendrun.HostAlias, error) {
	return nil, nil
}

type testRunOperationView struct {
	mu     sync.Mutex
	events []string
	record func(string)
}

func (v *testRunOperationView) Interrupted() {
	v.add("interrupted")
}

func (v *testRunOperationView) FatalInterrupt() {
	v.add("fatal-interrupt")
}

func (v *testRunOperationView) Stopping()                             {}
func (v *testRunOperationView) Cancelled(plans.Mode)                  {}
func (v *testRunOperationView) EmergencyDumpState(*statefile.File) error { return nil }
func (v *testRunOperationView) PlannedChange(*plans.ResourceInstanceChangeSrc) {}
func (v *testRunOperationView) Plan(*plans.Plan, *terraform.Schemas)  {}
func (v *testRunOperationView) PlanNextStep(string, string)           {}
func (v *testRunOperationView) Diagnostics(tfdiags.Diagnostics)       {}

var _ views.Operation = (*testRunOperationView)(nil)

func (v *testRunOperationView) add(event string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.events = append(v.events, event)
	if v.record != nil {
		v.record(event)
	}
}

func (v *testRunOperationView) snapshot() []string {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]string, 0, len(v.events))
	out = append(out, v.events...)
	return out
}

type testRunOperationOrder struct {
	mu     sync.Mutex
	events []string
}

func (o *testRunOperationOrder) add(event string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, event)
}

func (o *testRunOperationOrder) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	out := make([]string, 0, len(o.events))
	out = append(out, o.events...)
	return out
}

func testCompletedRunningOperation() *backendrun.RunningOperation {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return &backendrun.RunningOperation{
		Context: ctx,
		Stop:    func() {},
		Cancel:  func() {},
		Result:  backendrun.OperationSuccess,
	}
}

func assertRunOperationOrderContains(t *testing.T, got []string, wantEvents ...string) {
	t.Helper()
	for _, want := range wantEvents {
		if !containsString(got, want) {
			t.Fatalf("missing event %q in order %#v", want, got)
		}
	}
}

func assertRunOperationOrderNotContains(t *testing.T, got []string, forbidden ...string) {
	t.Helper()
	for _, want := range forbidden {
		if containsString(got, want) {
			t.Fatalf("unexpected event %q in order %#v", want, got)
		}
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func beforeInSlice(items []string, earlier, later string) bool {
	earlierIdx, laterIdx := -1, -1
	for i, item := range items {
		switch item {
		case earlier:
			if earlierIdx == -1 {
				earlierIdx = i
			}
		case later:
			if laterIdx == -1 {
				laterIdx = i
			}
		}
	}
	return earlierIdx >= 0 && laterIdx >= 0 && earlierIdx < laterIdx
}

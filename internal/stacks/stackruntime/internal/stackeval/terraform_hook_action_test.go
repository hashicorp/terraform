package stackeval

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestActionHookForwarding(t *testing.T) {
	var plannedCount, statusCount int
	var lastPlannedAI, lastStatusAI *hooks.ActionInvocation

	hks := &Hooks{}
	hks.ReportActionInvocationPlanned = func(ctx context.Context, span any, ai *hooks.ActionInvocation) any {
		plannedCount++
		lastPlannedAI = ai
		return nil
	}
	hks.ReportActionInvocationStatus = func(ctx context.Context, span any, ai *hooks.ActionInvocation) any {
		statusCount++
		lastStatusAI = ai
		return nil
	}

	// Create a simple concrete component instance address for the hook
	compAddr := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{Name: "testcomp"},
			Key:       addrs.NoKey,
		},
	}

	// Create the componentInstanceTerraformHook with our Hooks
	c := &componentInstanceTerraformHook{
		ctx:   context.Background(),
		seq:   &hookSeq{},
		hooks: hks,
		addr:  compAddr,
	}

	// Prepare a HookActionIdentity with an invoke trigger
	id := terraform.HookActionIdentity{
		Addr:          addrs.AbsActionInstance{},
		ActionTrigger: &plans.InvokeActionTrigger{},
	}

	// StartAction should trigger the planned hook once
	_, _ = c.StartAction(id)
	if plannedCount != 1 {
		t.Fatalf("expected StartAction to trigger planned hook once, got %d", plannedCount)
	}
	if lastPlannedAI == nil {
		t.Fatalf("expected non-nil ActionInvocation in planned hook")
	}
	if !reflect.DeepEqual(lastPlannedAI.Trigger, id.ActionTrigger) {
		t.Fatalf("planned hook received unexpected trigger: %#v", lastPlannedAI.Trigger)
	}

	// ProgressAction should trigger a status hook
	_, _ = c.ProgressAction(id, "in-progress")
	if statusCount != 1 {
		t.Fatalf("expected ProgressAction to trigger status hook once, got %d", statusCount)
	}
	if lastStatusAI == nil {
		t.Fatalf("expected non-nil ActionInvocation in status hook")
	}
	if !reflect.DeepEqual(lastStatusAI.Trigger, id.ActionTrigger) {
		t.Fatalf("status hook received unexpected trigger: %#v", lastStatusAI.Trigger)
	}

	// CompleteAction should trigger another status hook
	_, _ = c.CompleteAction(id, nil)
	if statusCount != 2 {
		t.Fatalf("expected CompleteAction to trigger status hook again, total 2, got %d", statusCount)
	}
}

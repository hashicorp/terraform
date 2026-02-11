package stackeval

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/terraform"
)

func TestActionHookForwarding(t *testing.T) {
	var statusCount int
	var statuses []hooks.ActionInvocationStatus

	hks := &Hooks{}
	hks.ReportActionInvocationStatus = func(ctx context.Context, span any, data *hooks.ActionInvocationStatusHookData) any {
		statusCount++
		statuses = append(statuses, data.Status)
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
		ProviderAddr:  addrs.AbsProviderConfig{},
	}

	// StartAction should trigger a status hook with "Running" status
	_, _ = c.StartAction(id)
	if statusCount != 1 {
		t.Fatalf("expected StartAction to trigger status hook once, got %d", statusCount)
	}
	if statuses[0] != hooks.ActionInvocationRunning {
		t.Fatalf("expected ActionInvocationRunning status from StartAction, got %s", statuses[0].String())
	}

	// ProgressAction with "in-progress" should keep running status
	_, _ = c.ProgressAction(id, "in-progress")
	if statusCount != 2 {
		t.Fatalf("expected ProgressAction to trigger status hook, got %d total", statusCount)
	}
	if statuses[1] != hooks.ActionInvocationRunning {
		t.Fatalf("expected ActionInvocationRunning status from ProgressAction, got %s", statuses[1].String())
	}

	// ProgressAction with "pending" should switch to pending status
	_, _ = c.ProgressAction(id, "pending")
	if statusCount != 3 {
		t.Fatalf("expected ProgressAction to trigger status hook, got %d total", statusCount)
	}
	if statuses[2] != hooks.ActionInvocationPending {
		t.Fatalf("expected ActionInvocationPending status from ProgressAction('pending'), got %s", statuses[2].String())
	}

	// CompleteAction with no error should complete successfully
	_, _ = c.CompleteAction(id, nil)
	if statusCount != 4 {
		t.Fatalf("expected CompleteAction to trigger status hook, got %d total", statusCount)
	}
	if statuses[3] != hooks.ActionInvocationCompleted {
		t.Fatalf("expected ActionInvocationCompleted status, got %s", statuses[3].String())
	}

	// Test error case
	statusCount = 0
	statuses = statuses[:0]

	// CompleteAction with error should mark as errored
	_, _ = c.CompleteAction(id, context.DeadlineExceeded)
	if statusCount != 1 {
		t.Fatalf("expected CompleteAction to trigger status hook, got %d total", statusCount)
	}
	if statuses[0] != hooks.ActionInvocationErrored {
		t.Fatalf("expected ActionInvocationErrored status, got %s", statuses[0].String())
	}
}

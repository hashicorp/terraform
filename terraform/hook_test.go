package terraform

import (
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/states"
)

func TestNilHook_impl(t *testing.T) {
	var _ Hook = new(NilHook)
}

// testHook is a Hook implementation that logs the calls it receives.
// It is intended for testing that core code is emitting the correct hooks
// for a given situation.
type testHook struct {
	Calls []*testHookCall
}

var _ Hook = (*testHook)(nil)

// testHookCall represents a single call in testHook.
// This hook just logs string names to make it easy to write "want" expressions
// in tests that can DeepEqual against the real calls.
type testHookCall struct {
	Action     string
	InstanceID string
}

func (h *testHook) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreApply", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, err error) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostApply", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreDiff(addr addrs.AbsResourceInstance, gen states.Generation, priorState, proposedNewState cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreDiff", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostDiff(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostDiff", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreProvisionInstance", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvisionInstance(addr addrs.AbsResourceInstance, state cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostProvisionInstance", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreProvisionInstanceStep", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string, err error) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostProvisionInstanceStep", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, line string) {
	h.Calls = append(h.Calls, &testHookCall{"ProvisionOutput", addr.String()})
}

func (h *testHook) PreRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreRefresh", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostRefresh(addr addrs.AbsResourceInstance, gen states.Generation, priorState cty.Value, newState cty.Value) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostRefresh", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreImportState(addr addrs.AbsResourceInstance, importID string) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreImportState", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostImportState(addr addrs.AbsResourceInstance, imported []providers.ImportedResource) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostImportState", addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostStateUpdate(new *states.State) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostStateUpdate", ""})
	return HookActionContinue, nil
}

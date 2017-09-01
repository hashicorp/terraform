package terraform

import (
	"testing"
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

// testHookCall represents a single call in testHook.
// This hook just logs string names to make it easy to write "want" expressions
// in tests that can DeepEqual against the real calls.
type testHookCall struct {
	Action     string
	InstanceID string
}

func (h *testHook) PreApply(i *InstanceInfo, s *InstanceState, d *InstanceDiff) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreApply", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostApply(i *InstanceInfo, s *InstanceState, err error) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostApply", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PreDiff(i *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreDiff", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostDiff(i *InstanceInfo, d *InstanceDiff) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostDiff", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvisionResource(i *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreProvisionResource", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvisionResource(i *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostProvisionResource", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvision(i *InstanceInfo, n string) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreProvision", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvision(i *InstanceInfo, n string, err error) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostProvision", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) ProvisionOutput(i *InstanceInfo, n string, m string) {
	h.Calls = append(h.Calls, &testHookCall{"ProvisionOutput", i.ResourceAddress().String()})
}

func (h *testHook) PreRefresh(i *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreRefresh", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostRefresh(i *InstanceInfo, s *InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostRefresh", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PreImportState(i *InstanceInfo, n string) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PreImportState", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostImportState(i *InstanceInfo, ss []*InstanceState) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostImportState", i.ResourceAddress().String()})
	return HookActionContinue, nil
}

func (h *testHook) PostStateUpdate(s *State) (HookAction, error) {
	h.Calls = append(h.Calls, &testHookCall{"PostStateUpdate", ""})
	return HookActionContinue, nil
}

var _ Hook = new(testHook)

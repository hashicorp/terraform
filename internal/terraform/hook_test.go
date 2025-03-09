// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sync"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestNilHook_impl(t *testing.T) {
	var _ Hook = new(NilHook)
}

// testHook is a Hook implementation that logs the calls it receives.
// It is intended for testing that core code is emitting the correct hooks
// for a given situation.
type testHook struct {
	mu    sync.Mutex
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

func (h *testHook) PreApply(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreApply", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostApply(id HookResourceIdentity, dk addrs.DeposedKey, newState cty.Value, err error) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostApply", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreDiff(id HookResourceIdentity, dk addrs.DeposedKey, priorState, proposedNewState cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreDiff", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostDiff(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostDiff", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreProvisionInstance", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostProvisionInstance", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreProvisionInstanceStep(id HookResourceIdentity, typeName string) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreProvisionInstanceStep", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostProvisionInstanceStep(id HookResourceIdentity, typeName string, err error) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostProvisionInstanceStep", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) ProvisionOutput(id HookResourceIdentity, typeName string, line string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"ProvisionOutput", id.Addr.String()})
}

func (h *testHook) PreRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreRefresh", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value, newState cty.Value) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostRefresh", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreImportState(id HookResourceIdentity, importID string) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreImportState", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostImportState(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostImportState", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PrePlanImport(id HookResourceIdentity, importID string) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PrePlanImport", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostPlanImport(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostPlanImport", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreApplyImport", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostApplyImport", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PreEphemeralOp(id HookResourceIdentity, action plans.Action) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PreEphemeralOp", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) PostEphemeralOp(id HookResourceIdentity, action plans.Action, err error) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostEphemeralOp", id.Addr.String()})
	return HookActionContinue, nil
}

func (h *testHook) Stopping() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"Stopping", ""})
}

func (h *testHook) PostStateUpdate(new *states.State) (HookAction, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.Calls = append(h.Calls, &testHookCall{"PostStateUpdate", ""})
	return HookActionContinue, nil
}

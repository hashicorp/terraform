// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"errors"
	"sync/atomic"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

// stopHook is a private Hook implementation that Terraform uses to
// signal when to stop or cancel actions.
type stopHook struct {
	stop uint32
}

var _ Hook = (*stopHook)(nil)

func (h *stopHook) PreApply(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostApply(id HookResourceIdentity, dk addrs.DeposedKey, newState cty.Value, err error) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreDiff(id HookResourceIdentity, dk addrs.DeposedKey, priorState, proposedNewState cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostDiff(id HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostProvisionInstance(id HookResourceIdentity, state cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreProvisionInstanceStep(id HookResourceIdentity, typeName string) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostProvisionInstanceStep(id HookResourceIdentity, typeName string, err error) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) ProvisionOutput(id HookResourceIdentity, typeName string, line string) {
}

func (h *stopHook) PreRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostRefresh(id HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value, newState cty.Value) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreImportState(id HookResourceIdentity, importID string) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostImportState(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PrePlanImport(id HookResourceIdentity, importID string) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostPlanImport(id HookResourceIdentity, imported []providers.ImportedResource) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PreApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) PostApplyImport(id HookResourceIdentity, importing plans.ImportingSrc) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) Stopping() {}

func (h *stopHook) PostStateUpdate(new *states.State) (HookAction, error) {
	return h.hook()
}

func (h *stopHook) hook() (HookAction, error) {
	if h.Stopped() {
		return HookActionHalt, errors.New("execution halted")
	}

	return HookActionContinue, nil
}

// reset should be called within the lock context
func (h *stopHook) Reset() {
	atomic.StoreUint32(&h.stop, 0)
}

func (h *stopHook) Stop() {
	atomic.StoreUint32(&h.stop, 1)
}

func (h *stopHook) Stopped() bool {
	return atomic.LoadUint32(&h.stop) == 1
}

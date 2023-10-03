// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackeval

import (
	"context"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

// componentInstanceTerraformHook implements terraform.Hook for plan and apply
// operations on a specified component instance. It connects the standard
// terraform.Hook callbacks to the given stackruntime.Hooks callbacks.
//
// We unfortunately must embed a context.Context in this type, as the existing
// Terraform core hook interface does not support threading a context through.
// The lifetime of this hook instance is strictly smaller than its surrounding
// context, but we should migrate away from this for clarity when possible.
type componentInstanceTerraformHook struct {
	terraform.NilHook

	ctx   context.Context
	seq   *hookSeq
	hooks *Hooks
	addr  stackaddrs.AbsComponentInstance
}

func (h *componentInstanceTerraformHook) resourceInstanceAddr(addr addrs.AbsResourceInstance) stackaddrs.AbsResourceInstance {
	return stackaddrs.AbsResourceInstance{
		Component: h.addr,
		Item:      addr,
	}
}

func (h *componentInstanceTerraformHook) PreDiff(addr addrs.AbsResourceInstance, dk addrs.DeposedKey, priorState, proposedNewState cty.Value) (terraform.HookAction, error) {
	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceStatus, &hooks.ResourceInstanceStatusHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Status: hooks.ResourceInstancePlanning,
	})
	return terraform.HookActionContinue, nil
}

func (h *componentInstanceTerraformHook) PostDiff(addr addrs.AbsResourceInstance, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceStatus, &hooks.ResourceInstanceStatusHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Status: hooks.ResourceInstancePlanned,
	})
	return terraform.HookActionContinue, nil
}

func (h *componentInstanceTerraformHook) PreApply(addr addrs.AbsResourceInstance, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	if action != plans.NoOp {
		hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceStatus, &hooks.ResourceInstanceStatusHookData{
			Addr:   h.resourceInstanceAddr(addr),
			Status: hooks.ResourceInstanceApplying,
		})
	}
	return terraform.HookActionContinue, nil
}

func (h *componentInstanceTerraformHook) PostApply(addr addrs.AbsResourceInstance, dk addrs.DeposedKey, newState cty.Value, err error) (terraform.HookAction, error) {
	// FIXME: need to emit nothing if this was a no-op, which means tracking
	// the `action` argument to `PreApply`. See `jsonHook` for more on this.
	status := hooks.ResourceInstanceApplied
	if err != nil {
		status = hooks.ResourceInstanceErrored
	}

	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceStatus, &hooks.ResourceInstanceStatusHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Status: status,
	})
	return terraform.HookActionContinue, nil
}

func (h *componentInstanceTerraformHook) PreProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string) (terraform.HookAction, error) {
	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceProvisionerStatus, &hooks.ResourceInstanceProvisionerHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Name:   typeName,
		Status: hooks.ProvisionerProvisioning,
	})
	return terraform.HookActionContinue, nil
}

func (h *componentInstanceTerraformHook) ProvisionOutput(addr addrs.AbsResourceInstance, typeName string, msg string) {
	// TODO: determine whether we should continue line splitting as we do with jsonHook
	output := msg
	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceProvisionerStatus, &hooks.ResourceInstanceProvisionerHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Name:   typeName,
		Status: hooks.ProvisionerProvisioning,
		Output: &output,
	})
}

func (h *componentInstanceTerraformHook) PostProvisionInstanceStep(addr addrs.AbsResourceInstance, typeName string, err error) (terraform.HookAction, error) {
	status := hooks.ProvisionerProvisioned
	if err != nil {
		status = hooks.ProvisionerErrored
	}
	hookMore(h.ctx, h.seq, h.hooks.ReportResourceInstanceProvisionerStatus, &hooks.ResourceInstanceProvisionerHookData{
		Addr:   h.resourceInstanceAddr(addr),
		Name:   typeName,
		Status: status,
	})
	return terraform.HookActionContinue, nil
}

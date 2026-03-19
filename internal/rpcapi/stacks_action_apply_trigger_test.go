// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package rpcapi

import (
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

func TestApplyActionInvocationStatusIncludesResourceActionTrigger(t *testing.T) {
	component := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{Name: "web"},
		},
	}
	actionInvocationAddr := stackaddrs.AbsActionInvocationInstance{
		Component: component,
		Item:      addrs.RootModuleInstance.ActionInstance("notify", "deploy", addrs.NoKey),
	}

	backendTrigger := &plans.ResourceActionTrigger{
		TriggeringResourceAddr:  addrs.RootModuleInstance.ResourceInstance(addrs.ManagedResourceMode, "aws_instance", "backend", addrs.NoKey),
		ActionTriggerEvent:      configs.AfterCreate,
		ActionTriggerBlockIndex: 0,
		ActionsListIndex:        0,
	}
	frontendTrigger := &plans.ResourceActionTrigger{
		TriggeringResourceAddr:  addrs.RootModuleInstance.ResourceInstance(addrs.ManagedResourceMode, "aws_instance", "frontend", addrs.NoKey),
		ActionTriggerEvent:      configs.AfterCreate,
		ActionTriggerBlockIndex: 0,
		ActionsListIndex:        0,
	}

	backendMsg := &stacks.StackChangeProgress_ActionInvocationStatus{
		Addr: stacks.NewActionInvocationInStackAddr(actionInvocationAddr),
	}
	frontendMsg := &stacks.StackChangeProgress_ActionInvocationStatus{
		Addr: stacks.NewActionInvocationInStackAddr(actionInvocationAddr),
	}

	setActionInvocationStatusTrigger(backendMsg, component, backendTrigger)
	setActionInvocationStatusTrigger(frontendMsg, component, frontendTrigger)

	if got, want := backendMsg.GetAddr().GetActionInvocationInstanceAddr(), "action.notify.deploy"; got != want {
		t.Fatalf("wrong action invocation address for backend status\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := frontendMsg.GetAddr().GetActionInvocationInstanceAddr(), "action.notify.deploy"; got != want {
		t.Fatalf("wrong action invocation address for frontend status\ngot:  %q\nwant: %q", got, want)
	}

	backendRAT := backendMsg.GetResourceActionTrigger()
	if backendRAT == nil {
		t.Fatal("backend status is missing resource_action_trigger")
	}
	frontendRAT := frontendMsg.GetResourceActionTrigger()
	if frontendRAT == nil {
		t.Fatal("frontend status is missing resource_action_trigger")
	}

	if got, want := backendRAT.GetTriggeringResourceAddress().GetResourceInstanceAddr(), "aws_instance.backend"; got != want {
		t.Fatalf("wrong backend triggering resource in status\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := frontendRAT.GetTriggeringResourceAddress().GetResourceInstanceAddr(), "aws_instance.frontend"; got != want {
		t.Fatalf("wrong frontend triggering resource in status\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := backendRAT.GetTriggerEvent(), stacks.StackChangeProgress_AFTER_CREATE; got != want {
		t.Fatalf("wrong backend trigger event in status\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := frontendRAT.GetTriggerEvent(), stacks.StackChangeProgress_AFTER_CREATE; got != want {
		t.Fatalf("wrong frontend trigger event in status\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := backendRAT.GetActionTriggerBlockIndex(), int64(0); got != want {
		t.Fatalf("wrong backend action trigger block index in status\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := frontendRAT.GetActionTriggerBlockIndex(), int64(0); got != want {
		t.Fatalf("wrong frontend action trigger block index in status\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := backendRAT.GetActionsListIndex(), int64(0); got != want {
		t.Fatalf("wrong backend actions list index in status\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := frontendRAT.GetActionsListIndex(), int64(0); got != want {
		t.Fatalf("wrong frontend actions list index in status\ngot:  %d\nwant: %d", got, want)
	}
}

func TestApplyActionInvocationProgressIncludesResourceActionTrigger(t *testing.T) {
	component := stackaddrs.AbsComponentInstance{
		Stack: stackaddrs.RootStackInstance,
		Item: stackaddrs.ComponentInstance{
			Component: stackaddrs.Component{Name: "web"},
		},
	}
	actionInvocationAddr := stackaddrs.AbsActionInvocationInstance{
		Component: component,
		Item:      addrs.RootModuleInstance.ActionInstance("notify", "deploy", addrs.NoKey),
	}

	backendTrigger := &plans.ResourceActionTrigger{
		TriggeringResourceAddr:  addrs.RootModuleInstance.ResourceInstance(addrs.ManagedResourceMode, "aws_instance", "backend", addrs.NoKey),
		ActionTriggerEvent:      configs.AfterCreate,
		ActionTriggerBlockIndex: 1,
		ActionsListIndex:        2,
	}
	frontendTrigger := &plans.ResourceActionTrigger{
		TriggeringResourceAddr:  addrs.RootModuleInstance.ResourceInstance(addrs.ManagedResourceMode, "aws_instance", "frontend", addrs.NoKey),
		ActionTriggerEvent:      configs.AfterCreate,
		ActionTriggerBlockIndex: 1,
		ActionsListIndex:        2,
	}

	backendMsg := &stacks.StackChangeProgress_ActionInvocationProgress{
		Addr: stacks.NewActionInvocationInStackAddr(actionInvocationAddr),
	}
	frontendMsg := &stacks.StackChangeProgress_ActionInvocationProgress{
		Addr: stacks.NewActionInvocationInStackAddr(actionInvocationAddr),
	}

	setActionInvocationProgressTrigger(backendMsg, component, backendTrigger)
	setActionInvocationProgressTrigger(frontendMsg, component, frontendTrigger)

	if got, want := backendMsg.GetAddr().GetActionInvocationInstanceAddr(), "action.notify.deploy"; got != want {
		t.Fatalf("wrong action invocation address for backend progress\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := frontendMsg.GetAddr().GetActionInvocationInstanceAddr(), "action.notify.deploy"; got != want {
		t.Fatalf("wrong action invocation address for frontend progress\ngot:  %q\nwant: %q", got, want)
	}

	backendRAT := backendMsg.GetResourceActionTrigger()
	if backendRAT == nil {
		t.Fatal("backend progress is missing resource_action_trigger")
	}
	frontendRAT := frontendMsg.GetResourceActionTrigger()
	if frontendRAT == nil {
		t.Fatal("frontend progress is missing resource_action_trigger")
	}

	if got, want := backendRAT.GetTriggeringResourceAddress().GetResourceInstanceAddr(), "aws_instance.backend"; got != want {
		t.Fatalf("wrong backend triggering resource in progress\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := frontendRAT.GetTriggeringResourceAddress().GetResourceInstanceAddr(), "aws_instance.frontend"; got != want {
		t.Fatalf("wrong frontend triggering resource in progress\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := backendRAT.GetTriggerEvent(), stacks.StackChangeProgress_AFTER_CREATE; got != want {
		t.Fatalf("wrong backend trigger event in progress\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := frontendRAT.GetTriggerEvent(), stacks.StackChangeProgress_AFTER_CREATE; got != want {
		t.Fatalf("wrong frontend trigger event in progress\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := backendRAT.GetActionTriggerBlockIndex(), int64(1); got != want {
		t.Fatalf("wrong backend action trigger block index in progress\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := frontendRAT.GetActionTriggerBlockIndex(), int64(1); got != want {
		t.Fatalf("wrong frontend action trigger block index in progress\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := backendRAT.GetActionsListIndex(), int64(2); got != want {
		t.Fatalf("wrong backend actions list index in progress\ngot:  %d\nwant: %d", got, want)
	}
	if got, want := frontendRAT.GetActionsListIndex(), int64(2); got != want {
		t.Fatalf("wrong frontend actions list index in progress\ngot:  %d\nwant: %d", got, want)
	}
}

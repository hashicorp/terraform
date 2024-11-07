// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hooks

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// ComponentInstanceStatus is a UI-focused description of the overall status
// for a given component instance undergoing a Terraform plan or apply
// operation. The "pending" and "errored" status are used for both operation
// types, and the others will be used only for one of plan or apply.
type ComponentInstanceStatus rune

//go:generate go run golang.org/x/tools/cmd/stringer -type=ComponentInstanceStatus component_instance.go

const (
	ComponentInstanceStatusInvalid ComponentInstanceStatus = 0
	ComponentInstancePending       ComponentInstanceStatus = '.'
	ComponentInstancePlanning      ComponentInstanceStatus = 'p'
	ComponentInstancePlanned       ComponentInstanceStatus = 'P'
	ComponentInstanceApplying      ComponentInstanceStatus = 'a'
	ComponentInstanceApplied       ComponentInstanceStatus = 'A'
	ComponentInstanceErrored       ComponentInstanceStatus = 'E'
	ComponentInstanceDeferred      ComponentInstanceStatus = 'D'
)

// TODO: move this into the rpcapi package somewhere
func (s ComponentInstanceStatus) ForProtobuf() stacks.StackChangeProgress_ComponentInstanceStatus_Status {
	switch s {
	case ComponentInstancePending:
		return stacks.StackChangeProgress_ComponentInstanceStatus_PENDING
	case ComponentInstancePlanning:
		return stacks.StackChangeProgress_ComponentInstanceStatus_PLANNING
	case ComponentInstancePlanned:
		return stacks.StackChangeProgress_ComponentInstanceStatus_PLANNED
	case ComponentInstanceApplying:
		return stacks.StackChangeProgress_ComponentInstanceStatus_APPLYING
	case ComponentInstanceApplied:
		return stacks.StackChangeProgress_ComponentInstanceStatus_APPLIED
	case ComponentInstanceErrored:
		return stacks.StackChangeProgress_ComponentInstanceStatus_ERRORED
	case ComponentInstanceDeferred:
		return stacks.StackChangeProgress_ComponentInstanceStatus_DEFERRED
	default:
		return stacks.StackChangeProgress_ComponentInstanceStatus_INVALID
	}
}

// ComponentInstanceChange is the argument type for hook callbacks which
// signal a set of planned or applied changes for a component instance.
type ComponentInstanceChange struct {
	Addr   stackaddrs.AbsComponentInstance
	Add    int
	Change int
	Import int
	Remove int
	Defer  int
	Move   int
	Forget int
}

// Total sums all of the change counts as a forwards-compatibility measure. If
// we later add a new change type, older clients will still be able to detect
// that the component instance has some unknown changes, rather than falsely
// stating that there are no changes at all.
func (cic ComponentInstanceChange) Total() int {
	return cic.Add + cic.Change + cic.Import + cic.Remove + cic.Defer + cic.Move + cic.Forget
}

// CountNewAction increments zero or more of the count fields based on the
// given action.
func (cic *ComponentInstanceChange) CountNewAction(action plans.Action) {
	switch action {
	case plans.Create:
		cic.Add++
	case plans.Delete:
		cic.Remove++
	case plans.Update:
		cic.Change++
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		cic.Add++
		cic.Remove++
	case plans.Forget:
		cic.Forget++
	case plans.CreateThenForget:
		cic.Add++
		cic.Forget++
	}
}

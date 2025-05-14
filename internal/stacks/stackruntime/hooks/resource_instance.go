// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package hooks

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/rpcapi/terraform1/stacks"
	"github.com/hashicorp/terraform/internal/stacks/stackaddrs"
)

// ResourceInstanceStatus is a UI-focused description of the overall status
// for a given resource instance undergoing a Terraform plan or apply
// operation. The "pending" and "errored" status are used for both operation
// types, and the others will be used only for one of plan or apply.
type ResourceInstanceStatus rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type=ResourceInstanceStatus resource_instance.go

const (
	ResourceInstanceStatusInvalid ResourceInstanceStatus = 0
	ResourceInstancePending       ResourceInstanceStatus = '.'
	ResourceInstanceRefreshing    ResourceInstanceStatus = 'r'
	ResourceInstanceRefreshed     ResourceInstanceStatus = 'R'
	ResourceInstancePlanning      ResourceInstanceStatus = 'p'
	ResourceInstancePlanned       ResourceInstanceStatus = 'P'
	ResourceInstanceApplying      ResourceInstanceStatus = 'a'
	ResourceInstanceApplied       ResourceInstanceStatus = 'A'
	ResourceInstanceErrored       ResourceInstanceStatus = 'E'
)

// TODO: move this into the rpcapi package somewhere
func (s ResourceInstanceStatus) ForProtobuf() stacks.StackChangeProgress_ResourceInstanceStatus_Status {
	switch s {
	case ResourceInstancePending:
		return stacks.StackChangeProgress_ResourceInstanceStatus_PENDING
	case ResourceInstanceRefreshing:
		return stacks.StackChangeProgress_ResourceInstanceStatus_REFRESHING
	case ResourceInstanceRefreshed:
		return stacks.StackChangeProgress_ResourceInstanceStatus_REFRESHED
	case ResourceInstancePlanning:
		return stacks.StackChangeProgress_ResourceInstanceStatus_PLANNING
	case ResourceInstancePlanned:
		return stacks.StackChangeProgress_ResourceInstanceStatus_PLANNED
	case ResourceInstanceApplying:
		return stacks.StackChangeProgress_ResourceInstanceStatus_APPLYING
	case ResourceInstanceApplied:
		return stacks.StackChangeProgress_ResourceInstanceStatus_APPLIED
	case ResourceInstanceErrored:
		return stacks.StackChangeProgress_ResourceInstanceStatus_ERRORED
	default:
		return stacks.StackChangeProgress_ResourceInstanceStatus_INVALID
	}
}

// ProvisionerStatus is a UI-focused description of the progress of a given
// resource instance's provisioner during a Terraform apply operation. Each
// specified provisioner will start in "provisioning" state, and progress to
// either "provisioned" or "errored".
type ProvisionerStatus rune

//go:generate go tool golang.org/x/tools/cmd/stringer -type=ProvisionerStatus resource_instance.go

const (
	ProvisionerStatusInvalid ProvisionerStatus = 0
	ProvisionerProvisioning  ProvisionerStatus = 'p'
	ProvisionerProvisioned   ProvisionerStatus = 'P'
	ProvisionerErrored       ProvisionerStatus = 'E'
)

// TODO: move this into the rpcapi package somewhere
func (s ProvisionerStatus) ForProtobuf() stacks.StackChangeProgress_ProvisionerStatus_Status {
	switch s {
	case ProvisionerProvisioning:
		return stacks.StackChangeProgress_ProvisionerStatus_PROVISIONING
	case ProvisionerProvisioned:
		return stacks.StackChangeProgress_ProvisionerStatus_PROVISIONING
	case ProvisionerErrored:
		return stacks.StackChangeProgress_ProvisionerStatus_ERRORED
	default:
		return stacks.StackChangeProgress_ProvisionerStatus_INVALID
	}
}

// ResourceInstanceStatusHookData is the argument type for hook callbacks which
// signal a resource instance's status updates.
type ResourceInstanceStatusHookData struct {
	Addr         stackaddrs.AbsResourceInstanceObject
	ProviderAddr addrs.Provider
	Status       ResourceInstanceStatus
}

// ResourceInstanceProvisionerHookData is the argument type for hook callbacks
// which signal a resource instance's provisioner progress, including both
// status updates and optional provisioner output data.
type ResourceInstanceProvisionerHookData struct {
	Addr   stackaddrs.AbsResourceInstanceObject
	Name   string
	Status ProvisionerStatus
	Output *string
}

// ResourceInstanceChange is the argument type for hook callbacks which signal
// a detected or planned change for a resource instance resulting from a plan
// operation.
type ResourceInstanceChange struct {
	Addr   stackaddrs.AbsResourceInstanceObject
	Change *plans.ResourceInstanceChangeSrc
}

// DeferredResourceInstanceChange is the argument type for hook callbacks which
// signal a deferred change for a resource instance resulting from a plan
// operation.
type DeferredResourceInstanceChange struct {
	Reason providers.DeferredReason
	Change *ResourceInstanceChange
}

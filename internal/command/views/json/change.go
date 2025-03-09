// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func NewResourceInstanceChange(change *plans.ResourceInstanceChangeSrc) *ResourceInstanceChange {
	c := &ResourceInstanceChange{
		Resource:        newResourceAddr(change.Addr),
		Action:          changeAction(change.Action),
		Reason:          changeReason(change.ActionReason),
		GeneratedConfig: change.GeneratedConfig,
	}

	// The order here matters, we want the moved action to take precedence over
	// the import action. We're basically taking "the most recent action" as the
	// primary action in the streamed logs. That is to say, that if a resource
	// is imported and then moved in a single operation then the change for that
	// resource will be reported as ActionMove while the Importing flag will
	// still be set to true.
	//
	// Since both the moved and imported actions only overwrite a NoOp this
	// behaviour is consistent across the other actions as well. Something that
	// is imported and then updated, or moved and then updated, will have the
	// ActionUpdate as the recognised action for the change.

	if !change.Addr.Equal(change.PrevRunAddr) {
		if c.Action == ActionNoOp {
			c.Action = ActionMove
		}
		pr := newResourceAddr(change.PrevRunAddr)
		c.PreviousResource = &pr
	}
	if change.Importing != nil {
		if c.Action == ActionNoOp {
			c.Action = ActionImport
		}
		c.Importing = &Importing{ID: change.Importing.ID}
	}

	return c
}

type ResourceInstanceChange struct {
	Resource         ResourceAddr  `json:"resource"`
	PreviousResource *ResourceAddr `json:"previous_resource,omitempty"`
	Action           ChangeAction  `json:"action"`
	Reason           ChangeReason  `json:"reason,omitempty"`
	Importing        *Importing    `json:"importing,omitempty"`
	GeneratedConfig  string        `json:"generated_config,omitempty"`
}

func (c *ResourceInstanceChange) String() string {
	return fmt.Sprintf("%s: Plan to %s", c.Resource.Addr, c.Action)
}

type ChangeAction string

const (
	ActionNoOp    ChangeAction = "noop"
	ActionMove    ChangeAction = "move"
	ActionForget  ChangeAction = "remove"
	ActionCreate  ChangeAction = "create"
	ActionRead    ChangeAction = "read"
	ActionUpdate  ChangeAction = "update"
	ActionReplace ChangeAction = "replace"
	ActionDelete  ChangeAction = "delete"
	ActionImport  ChangeAction = "import"

	// While ephemeral resources do not represent a change
	// or participate in the plan in the same way as the above
	// we declare them here for convenience in helper functions.
	ActionOpen  ChangeAction = "open"
	ActionRenew ChangeAction = "renew"
	ActionClose ChangeAction = "close"
)

func changeAction(action plans.Action) ChangeAction {
	switch action {
	case plans.NoOp:
		return ActionNoOp
	case plans.Create:
		return ActionCreate
	case plans.Read:
		return ActionRead
	case plans.Update:
		return ActionUpdate
	case plans.DeleteThenCreate, plans.CreateThenDelete, plans.CreateThenForget:
		return ActionReplace
	case plans.Delete:
		return ActionDelete
	case plans.Forget:
		return ActionForget
	case plans.Open:
		return ActionOpen
	case plans.Renew:
		return ActionRenew
	case plans.Close:
		return ActionClose
	default:
		return ActionNoOp
	}
}

type ChangeReason string

const (
	ReasonNone               ChangeReason = ""
	ReasonTainted            ChangeReason = "tainted"
	ReasonRequested          ChangeReason = "requested"
	ReasonReplaceTriggeredBy ChangeReason = "replace_triggered_by"
	ReasonCannotUpdate       ChangeReason = "cannot_update"
	ReasonUnknown            ChangeReason = "unknown"

	ReasonDeleteBecauseNoResourceConfig ChangeReason = "delete_because_no_resource_config"
	ReasonDeleteBecauseWrongRepetition  ChangeReason = "delete_because_wrong_repetition"
	ReasonDeleteBecauseCountIndex       ChangeReason = "delete_because_count_index"
	ReasonDeleteBecauseEachKey          ChangeReason = "delete_because_each_key"
	ReasonDeleteBecauseNoModule         ChangeReason = "delete_because_no_module"
	ReasonDeleteBecauseNoMoveTarget     ChangeReason = "delete_because_no_move_target"
	ReasonReadBecauseConfigUnknown      ChangeReason = "read_because_config_unknown"
	ReasonReadBecauseDependencyPending  ChangeReason = "read_because_dependency_pending"
	ReasonReadBecauseCheckNested        ChangeReason = "read_because_check_nested"
)

func changeReason(reason plans.ResourceInstanceChangeActionReason) ChangeReason {
	switch reason {
	case plans.ResourceInstanceChangeNoReason:
		return ReasonNone
	case plans.ResourceInstanceReplaceBecauseTainted:
		return ReasonTainted
	case plans.ResourceInstanceReplaceByRequest:
		return ReasonRequested
	case plans.ResourceInstanceReplaceBecauseCannotUpdate:
		return ReasonCannotUpdate
	case plans.ResourceInstanceReplaceByTriggers:
		return ReasonReplaceTriggeredBy
	case plans.ResourceInstanceDeleteBecauseNoResourceConfig:
		return ReasonDeleteBecauseNoResourceConfig
	case plans.ResourceInstanceDeleteBecauseWrongRepetition:
		return ReasonDeleteBecauseWrongRepetition
	case plans.ResourceInstanceDeleteBecauseCountIndex:
		return ReasonDeleteBecauseCountIndex
	case plans.ResourceInstanceDeleteBecauseEachKey:
		return ReasonDeleteBecauseEachKey
	case plans.ResourceInstanceDeleteBecauseNoModule:
		return ReasonDeleteBecauseNoModule
	case plans.ResourceInstanceReadBecauseConfigUnknown:
		return ReasonReadBecauseConfigUnknown
	case plans.ResourceInstanceDeleteBecauseNoMoveTarget:
		return ReasonDeleteBecauseNoMoveTarget
	case plans.ResourceInstanceReadBecauseDependencyPending:
		return ReasonReadBecauseDependencyPending
	case plans.ResourceInstanceReadBecauseCheckNested:
		return ReasonReadBecauseCheckNested
	default:
		// This should never happen, but there's no good way to guarantee
		// exhaustive handling of the enum, so a generic fall back is better
		// than a misleading result or a panic
		return ReasonUnknown
	}
}

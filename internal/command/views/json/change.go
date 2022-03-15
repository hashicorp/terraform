package json

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func NewResourceInstanceChange(change *plans.ResourceInstanceChangeSrc) *ResourceInstanceChange {
	c := &ResourceInstanceChange{
		Resource: newResourceAddr(change.Addr),
		Action:   changeAction(change.Action),
		Reason:   changeReason(change.ActionReason),
	}
	if !change.Addr.Equal(change.PrevRunAddr) {
		if c.Action == ActionNoOp {
			c.Action = ActionMove
		}
		pr := newResourceAddr(change.PrevRunAddr)
		c.PreviousResource = &pr
	}

	return c
}

type ResourceInstanceChange struct {
	Resource         ResourceAddr  `json:"resource"`
	PreviousResource *ResourceAddr `json:"previous_resource,omitempty"`
	Action           ChangeAction  `json:"action"`
	Reason           ChangeReason  `json:"reason,omitempty"`
}

func (c *ResourceInstanceChange) String() string {
	return fmt.Sprintf("%s: Plan to %s", c.Resource.Addr, c.Action)
}

type ChangeAction string

const (
	ActionNoOp    ChangeAction = "noop"
	ActionMove    ChangeAction = "move"
	ActionCreate  ChangeAction = "create"
	ActionRead    ChangeAction = "read"
	ActionUpdate  ChangeAction = "update"
	ActionReplace ChangeAction = "replace"
	ActionDelete  ChangeAction = "delete"
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
	case plans.DeleteThenCreate, plans.CreateThenDelete:
		return ActionReplace
	case plans.Delete:
		return ActionDelete
	default:
		return ActionNoOp
	}
}

type ChangeReason string

const (
	ReasonNone         ChangeReason = ""
	ReasonTainted      ChangeReason = "tainted"
	ReasonRequested    ChangeReason = "requested"
	ReasonCannotUpdate ChangeReason = "cannot_update"
	ReasonUnknown      ChangeReason = "unknown"

	ReasonDeleteBecauseNoResourceConfig ChangeReason = "delete_because_no_resource_config"
	ReasonDeleteBecauseWrongRepetition  ChangeReason = "delete_because_wrong_repetition"
	ReasonDeleteBecauseCountIndex       ChangeReason = "delete_because_count_index"
	ReasonDeleteBecauseEachKey          ChangeReason = "delete_because_each_key"
	ReasonDeleteBecauseNoModule         ChangeReason = "delete_because_no_module"
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
	default:
		// This should never happen, but there's no good way to guarantee
		// exhaustive handling of the enum, so a generic fall back is better
		// than a misleading result or a panic
		return ReasonUnknown
	}
}

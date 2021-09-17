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
	default:
		// This should never happen, but there's no good way to guarantee
		// exhaustive handling of the enum, so a generic fall back is better
		// than a misleading result or a panic
		return ReasonUnknown
	}
}

package json

import (
	"fmt"

	"github.com/hashicorp/terraform/plans"
)

func NewResourceInstanceChange(change *plans.ResourceInstanceChangeSrc) *ResourceInstanceChange {
	c := &ResourceInstanceChange{
		Resource: newResourceAddr(change.Addr),
		Action:   changeAction(change.Action),
	}

	return c
}

type ResourceInstanceChange struct {
	Resource ResourceAddr `json:"resource"`
	Action   ChangeAction `json:"action"`
}

func (c *ResourceInstanceChange) String() string {
	return fmt.Sprintf("%s: Plan to %s", c.Resource.Addr, c.Action)
}

type ChangeAction string

const (
	ActionNoOp    ChangeAction = "noop"
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

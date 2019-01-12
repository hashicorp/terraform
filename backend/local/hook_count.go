package local

import (
	"sync"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

// CountHook is a hook that counts the number of resources
// added, removed, changed during the course of an apply.
type CountHook struct {
	Added   int
	Changed int
	Removed int

	ToAdd          int
	ToChange       int
	ToRemove       int
	ToRemoveAndAdd int

	pending map[string]plans.Action

	sync.Mutex
	terraform.NilHook
}

var _ terraform.Hook = (*CountHook)(nil)

func (h *CountHook) Reset() {
	h.Lock()
	defer h.Unlock()

	h.pending = nil
	h.Added = 0
	h.Changed = 0
	h.Removed = 0
}

func (h *CountHook) PreApply(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.pending == nil {
		h.pending = make(map[string]plans.Action)
	}

	h.pending[addr.String()] = action

	return terraform.HookActionContinue, nil
}

func (h *CountHook) PostApply(addr addrs.AbsResourceInstance, gen states.Generation, newState cty.Value, err error) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	if h.pending != nil {
		pendingKey := addr.String()
		if action, ok := h.pending[pendingKey]; ok {
			delete(h.pending, pendingKey)

			if err == nil {
				switch action {
				case plans.CreateThenDelete, plans.DeleteThenCreate:
					h.Added++
					h.Removed++
				case plans.Create:
					h.Added++
				case plans.Delete:
					h.Removed++
				case plans.Update:
					h.Changed++
				}
			}
		}
	}

	return terraform.HookActionContinue, nil
}

func (h *CountHook) PostDiff(addr addrs.AbsResourceInstance, gen states.Generation, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	h.Lock()
	defer h.Unlock()

	// We don't count anything for data resources
	if addr.Resource.Resource.Mode == addrs.DataResourceMode {
		return terraform.HookActionContinue, nil
	}

	switch action {
	case plans.CreateThenDelete, plans.DeleteThenCreate:
		h.ToRemoveAndAdd += 1
	case plans.Create:
		h.ToAdd += 1
	case plans.Delete:
		h.ToRemove += 1
	case plans.Update:
		h.ToChange += 1
	}

	return terraform.HookActionContinue, nil
}

package terraform

import (
	"log"
)

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Provider EvalNode
	State    **InstanceState
	Info     *InstanceInfo
	Output   **InstanceState
}

func (n *EvalRefresh) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider}, []EvalType{EvalTypeResourceProvider}
}

// TODO: test
func (n *EvalRefresh) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	provider := args[0].(ResourceProvider)
	state := *n.State

	// If we have no state, we don't do any refreshing
	if state == nil {
		log.Printf("[DEBUG] refresh: %s: no state, not refreshing", n.Info.Id)
		return nil, nil
	}

	// Call pre-refresh hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	// Refresh!
	state, err = provider.Refresh(n.Info, state)
	if err != nil {
		return nil, err
	}

	// Call post-refresh hook
	err = ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(n.Info, state)
	})
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = state
	}
	return state, nil
}

func (n *EvalRefresh) Type() EvalType {
	return EvalTypeInstanceState
}

package terraform

// EvalRefresh is an EvalNode implementation that does a refresh for
// a resource.
type EvalRefresh struct {
	Provider EvalNode
	State    EvalNode
	Info     *InstanceInfo
}

func (n *EvalRefresh) Args() ([]EvalNode, []EvalType) {
	return []EvalNode{n.Provider, n.State},
		[]EvalType{EvalTypeResourceProvider, EvalTypeInstanceState}
}

// TODO: test
func (n *EvalRefresh) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	var state *InstanceState
	provider := args[0].(ResourceProvider)
	if args[1] != nil {
		state = args[1].(*InstanceState)
	}

	// If we have no state, we don't do any refreshing
	if state == nil {
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

	return state, nil
}

func (n *EvalRefresh) Type() EvalType {
	return EvalTypeInstanceState
}

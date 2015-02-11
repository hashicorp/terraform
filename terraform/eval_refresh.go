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

	n.Info.ModulePath = ctx.Path()
	return provider.Refresh(n.Info, state)
}

func (n *EvalRefresh) Type() EvalType {
	return EvalTypeInstanceState
}

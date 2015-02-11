package terraform

// EvalInstanceInfo is an EvalNode implementation that fills in the
// InstanceInfo as much as it can.
type EvalInstanceInfo struct {
	Info *InstanceInfo
}

func (n *EvalInstanceInfo) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

// TODO: test
func (n *EvalInstanceInfo) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	n.Info.ModulePath = ctx.Path()
	return nil, nil
}

func (n *EvalInstanceInfo) Type() EvalType {
	return EvalTypeNull
}

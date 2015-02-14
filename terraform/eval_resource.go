package terraform

// EvalInstanceInfo is an EvalNode implementation that fills in the
// InstanceInfo as much as it can.
type EvalInstanceInfo struct {
	Info *InstanceInfo
}

// TODO: test
func (n *EvalInstanceInfo) Eval(ctx EvalContext) (interface{}, error) {
	n.Info.ModulePath = ctx.Path()
	return nil, nil
}

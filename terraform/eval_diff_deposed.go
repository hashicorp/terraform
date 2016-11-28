package terraform

// EvalDiffDeposed is an EvalNode implementation that marks DestroyDeposed
// in a diff if a resource has deposed instances that need destruction.
type EvalDiffDeposed struct {
	Name string
	Diff **InstanceDiff
}

// TODO: test
func (n *EvalDiffDeposed) Eval(ctx EvalContext) (interface{}, error) {
	// Check if there are any deposed items in the state
	deposed := false
	_, err := readInstanceFromState(ctx, n.Name, nil, func(rs *ResourceState) (*InstanceState, error) {
		if len(rs.Deposed) > 0 {
			deposed = true
		}

		return nil, nil
	})
	if err != nil {
		return nil, err
	}

	// If no deposed items, just return
	if !deposed {
		return nil, nil
	}

	// Set the flag to true
	if *n.Diff == nil {
		*n.Diff = new(InstanceDiff)
	}
	(*n.Diff).DestroyDeposed = true

	return nil, nil
}

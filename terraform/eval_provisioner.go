package terraform

import (
	"fmt"
)

// EvalInitProvisioner is an EvalNode implementation that initializes a provisioner
// and returns nothing. The provisioner can be retrieved again with the
// EvalGetProvisioner node.
type EvalInitProvisioner struct {
	Name string
}

func (n *EvalInitProvisioner) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalInitProvisioner) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	return ctx.InitProvisioner(n.Name)
}

func (n *EvalInitProvisioner) Type() EvalType {
	return EvalTypeNull
}

// EvalGetProvisioner is an EvalNode implementation that retrieves an already
// initialized provisioner instance for the given name.
type EvalGetProvisioner struct {
	Name string
}

func (n *EvalGetProvisioner) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalGetProvisioner) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	result := ctx.Provisioner(n.Name)
	if result == nil {
		return nil, fmt.Errorf("provisioner %s not initialized", n.Name)
	}

	return result, nil
}

func (n *EvalGetProvisioner) Type() EvalType {
	return EvalTypeResourceProvisioner
}

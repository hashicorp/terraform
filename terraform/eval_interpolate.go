package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalInterpolate is an EvalNode implementation that takes a raw
// configuration and interpolates it.
type EvalInterpolate struct {
	Config *config.RawConfig
}

func (n *EvalInterpolate) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalInterpolate) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	return nil, nil
}

func (n *EvalInterpolate) Type() EvalType {
	return EvalTypeConfig
}

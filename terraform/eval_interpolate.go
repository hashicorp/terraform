package terraform

import (
	"github.com/hashicorp/terraform/config"
)

// EvalInterpolate is an EvalNode implementation that takes a raw
// configuration and interpolates it.
type EvalInterpolate struct {
	Config   *config.RawConfig
	Resource *Resource
}

func (n *EvalInterpolate) Args() ([]EvalNode, []EvalType) {
	return nil, nil
}

func (n *EvalInterpolate) Eval(
	ctx EvalContext, args []interface{}) (interface{}, error) {
	return ctx.Interpolate(n.Config, n.Resource)
}

func (n *EvalInterpolate) Type() EvalType {
	return EvalTypeConfig
}

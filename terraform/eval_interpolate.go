package terraform

import (
	"log"

	"github.com/hashicorp/terraform/config"
)

// EvalInterpolate is an EvalNode implementation that takes a raw
// configuration and interpolates it.
type EvalInterpolate struct {
	Config   *config.RawConfig
	Resource *Resource
	Output   **ResourceConfig
}

func (n *EvalInterpolate) Eval(ctx EvalContext) (interface{}, error) {
	rc, err := ctx.Interpolate(n.Config, n.Resource)
	if err != nil {
		return nil, err
	}

	if n.Output != nil {
		*n.Output = rc
	}

	return nil, nil
}

// EvalTryInterpolate is an EvalNode implementation that takes a raw
// configuration and interpolates it, but only logs a warning on an
// interpolation error, and stops further Eval steps.
// This is used during Input where a value may not be known before Refresh, but
// we don't want to block Input.
type EvalTryInterpolate struct {
	Config   *config.RawConfig
	Resource *Resource
	Output   **ResourceConfig
}

func (n *EvalTryInterpolate) Eval(ctx EvalContext) (interface{}, error) {
	rc, err := ctx.Interpolate(n.Config, n.Resource)
	if err != nil {
		log.Printf("[WARN] Interpolation %q failed: %s", n.Config.Key, err)
		return nil, EvalEarlyExitError{}
	}

	if n.Output != nil {
		*n.Output = rc
	}

	return nil, nil
}

package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Refresh is a vestigial operation that is equivalent to call to Plan and
// then taking the prior state of the resulting plan.
//
// We retain this only as a measure of semi-backward-compatibility for
// automation relying on the "terraform refresh" subcommand. The modern way
// to get this effect is to create and then apply a plan in the refresh-only
// mode.
func (c *Context) Refresh(config *configs.Config, prevRunState *states.State, opts *PlanOpts) (*states.State, tfdiags.Diagnostics) {
	if opts == nil {
		// This fallback is only here for tests, not for real code.
		opts = &PlanOpts{
			Mode: plans.NormalMode,
		}
	}
	if opts.Mode != plans.NormalMode {
		panic("can only Refresh in the normal planning mode")
	}

	log.Printf("[DEBUG] Refresh is really just plan now, so creating a %s plan", opts.Mode)
	p, diags := c.Plan(config, prevRunState, opts)
	if diags.HasErrors() {
		return nil, diags
	}

	return p.PriorState, diags
}

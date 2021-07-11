package planner

import (
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

type Options struct {
	Mode             plans.Mode
	RootVariableVals map[string]cty.Value
	TargetAddrs      []addrs.Targetable
	Refresh          bool
}

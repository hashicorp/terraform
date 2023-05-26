package stackeval

import (
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
)

type PlanOpts struct {
	PlanningMode plans.Mode

	InputVariableValues map[string]cty.Value
}

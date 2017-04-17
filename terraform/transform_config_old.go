package terraform

import (
	"fmt"

	"github.com/hashicorp/terraform/config"
)

// varNameForVar returns the VarName value for an interpolated variable.
// This value is compared to the VarName() value for the nodes within the
// graph to build the graph edges.
func varNameForVar(raw config.InterpolatedVariable) string {
	switch v := raw.(type) {
	case *config.ModuleVariable:
		return fmt.Sprintf("module.%s.output.%s", v.Name, v.Field)
	case *config.ResourceVariable:
		return v.ResourceId()
	case *config.UserVariable:
		return fmt.Sprintf("var.%s", v.Name)
	default:
		return ""
	}
}

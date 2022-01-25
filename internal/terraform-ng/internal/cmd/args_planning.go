package cmd

import (
	"github.com/spf13/cobra"
)

var planningOptions = struct {
	refreshOnly    bool
	componentAddrs []string
}{}

func addPlanningOptionsToCommand(cmd *cobra.Command) {
	cmd.Flags().BoolVar(&planningOptions.refreshOnly, "refresh-only", false, "only detect changes from outside of Terraform; don't propose new actions")
	cmd.Flags().StringSliceVar(&planningOptions.componentAddrs, "component", nil, "update only a given component and those which depend on it")
}

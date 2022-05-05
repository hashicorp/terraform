package command

import (
	"strings"

	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/lang"
)

// MetadataFunctionsCommand is a Command implementation that describes the
// available functions in this version of Terraform.
type MetadataFunctionsCommand struct {
	Meta
}

func (c *MetadataFunctionsCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Set up view
	view := views.NewMetadata(c.View)

	scope := &lang.Scope{}
	functions := scope.Functions()

	diags := view.Functions(functions)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	return 0
}

func (c *MetadataFunctionsCommand) Help() string {
	helpText := `
Usage: terraform metadata functions

  Outputs language metadata describing the functions available in this version
  of Terraform. This command is intended for integration with terraform-ls.
`
	return strings.TrimSpace(helpText)
}

func (c *MetadataFunctionsCommand) Synopsis() string {
	return "Output language metadata for available functions"
}

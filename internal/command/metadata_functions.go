// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty/function"
)

var ignoredFunctions = []string{"map", "list", "core::map", "core::list"}

// MetadataFunctionsCommand is a Command implementation that prints out information
// about the available functions in Terraform.
type MetadataFunctionsCommand struct {
	Meta
}

func (c *MetadataFunctionsCommand) Help() string {
	return metadataFunctionsCommandHelp
}

func (c *MetadataFunctionsCommand) Synopsis() string {
	return "Show signatures and descriptions for the available functions"
}

func (c *MetadataFunctionsCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Process command-specific arguments.
	args, argDiags := arguments.ParseMetadataFunctions(rawArgs)
	diags = diags.Append(argDiags)

	// Prepare the view
	view := views.NewMetadataFunctions(args.ViewType, c.View)

	// Now the view is ready, process any error diagnostics from parsing arguments.
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	scope := &lang.Scope{}
	funcs := scope.Functions()
	filteredFuncs := make(map[string]function.Function)
	for k, v := range funcs {
		if isIgnoredFunction(k) {
			continue
		}
		filteredFuncs[k] = v
	}

	view.Functions(filteredFuncs)

	return 0
}

const metadataFunctionsCommandHelp = `
Usage: terraform [global options] metadata functions -json

  Prints out a json representation of the available function signatures.
`

func isIgnoredFunction(name string) bool {
	for _, i := range ignoredFunctions {
		if i == name {
			return true
		}
	}
	return false
}

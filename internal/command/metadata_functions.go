// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonfunction"
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

	args := c.Meta.process(rawArgs)
	parsedArgs, parsedArgDiags := arguments.ParseMetadataFunctions(args)
	diags = diags.Append(parsedArgDiags)
	if parsedArgDiags.HasErrors() {
		c.showDiagnostics(diags)
		if !parsedArgs.JSON {
			// Show help message if the -json flag is not provided
			c.Ui.Error(c.Help())
		}
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

	jsonFunctions, marshalDiags := jsonfunction.Marshal(filteredFuncs)
	if marshalDiags.HasErrors() {
		c.showDiagnostics(marshalDiags)
		return 1
	}
	c.Ui.Output(string(jsonFunctions))

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

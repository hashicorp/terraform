package command

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/terraform"
)

// GraphCommand is a Command implementation that takes a Terraform
// configuration and outputs the dependency tree in graphical form.
type GraphCommand struct {
	Meta
}

func (c *GraphCommand) Run(args []string) int {
	var drawCycles bool
	var graphTypeStr string
	var moduleDepth int
	var verbose bool

	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("graph")
	cmdFlags.BoolVar(&drawCycles, "draw-cycles", false, "draw-cycles")
	cmdFlags.StringVar(&graphTypeStr, "type", "", "type")
	cmdFlags.IntVar(&moduleDepth, "module-depth", -1, "module-depth")
	cmdFlags.BoolVar(&verbose, "verbose", false, "verbose")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		c.Ui.Error(fmt.Sprintf("Error parsing command-line flags: %s\n", err.Error()))
		return 1
	}

	configPath, err := ModulePath(cmdFlags.Args())
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Check if the path is a plan
	var plan *plans.Plan
	planFile, err := c.PlanFile(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if planFile != nil {
		// Reset for backend loading
		configPath = ""
	}

	var diags tfdiags.Diagnostics

	backendConfig, backendDiags := c.loadBackendConfig(configPath)
	diags = diags.Append(backendDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: backendConfig,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
	opReq.PlanFile = planFile
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	defer func() {
		err := opReq.StateLocker.Unlock(nil)
		if err != nil {
			c.Ui.Error(err.Error())
		}
	}()

	// Determine the graph type
	graphType := terraform.GraphTypePlan
	if plan != nil {
		graphType = terraform.GraphTypeApply
	}

	if graphTypeStr != "" {
		v, ok := terraform.GraphTypeMap[graphTypeStr]
		if !ok {
			c.Ui.Error(fmt.Sprintf("Invalid graph type requested: %s", graphTypeStr))
			return 1
		}

		graphType = v
	}

	// Skip validation during graph generation - we want to see the graph even if
	// it is invalid for some reason.
	g, graphDiags := ctx.Graph(graphType, &terraform.ContextGraphOpts{
		Verbose:  verbose,
		Validate: false,
	})
	diags = diags.Append(graphDiags)
	if graphDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	graphStr, err := terraform.GraphDot(g, &dag.DotOpts{
		DrawCycles: drawCycles,
		MaxDepth:   moduleDepth,
		Verbose:    verbose,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error converting graph: %s", err))
		return 1
	}

	if diags.HasErrors() {
		// For this command we only show diagnostics if there are errors,
		// because printing out naked warnings could upset a naive program
		// consuming our dot output.
		c.showDiagnostics(diags)
		return 1
	}

	c.Ui.Output(graphStr)

	return 0
}

func (c *GraphCommand) Help() string {
	helpText := `
Usage: terraform graph [options] [DIR]

  Outputs the visual execution graph of Terraform resources according to
  configuration files in DIR (or the current directory if omitted).

  The graph is outputted in DOT format. The typical program that can
  read this format is GraphViz, but many web services are also available
  to read this format.

  The -type flag can be used to control the type of graph shown. Terraform
  creates different graphs for different operations. See the options below
  for the list of types supported. The default type is "plan" if a
  configuration is given, and "apply" if a plan file is passed as an
  argument.

Options:

  -draw-cycles     Highlight any cycles in the graph with colored edges.
                   This helps when diagnosing cycle errors.

  -module-depth=n  Specifies the depth of modules to show in the output.	
                   By default this is -1, which will expand all.

  -type=plan       Type of graph to output. Can be: plan, plan-destroy, apply,
                   validate, input, refresh.

`
	return strings.TrimSpace(helpText)
}

func (c *GraphCommand) Synopsis() string {
	return "Create a visual graph of Terraform resources"
}

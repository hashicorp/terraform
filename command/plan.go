package command

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/terraform/terraform"
)

// PlanCommand is a Command implementation that compares a Terraform
// configuration to an actual infrastructure and shows the differences.
type PlanCommand struct {
	Meta
}

func (c *PlanCommand) Run(args []string) int {
	var destroy, refresh, detailed bool
	var outPath string
	var moduleDepth int

	args = c.Meta.process(args, true)

	cmdFlags := c.Meta.flagSet("plan")
	cmdFlags.BoolVar(&destroy, "destroy", false, "destroy")
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	c.addModuleDepthFlag(cmdFlags, &moduleDepth)
	cmdFlags.StringVar(&outPath, "out", "", "path")
	cmdFlags.IntVar(
		&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", DefaultStateFilename, "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&detailed, "detailed-exitcode", false, "detailed-exitcode")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	var path string
	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The plan command expects at most one argument with the path\n" +
				"to a Terraform configuration.\n")
		cmdFlags.Usage()
		return 1
	} else if len(args) == 1 {
		path = args[0]
	} else {
		var err error
		path, err = os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		}
	}

	countHook := new(CountHook)
	c.Meta.extraHooks = []terraform.Hook{countHook}

	ctx, _, err := c.Context(contextOpts{
		Destroy:     destroy,
		Path:        path,
		StatePath:   c.Meta.statePath,
		Parallelism: c.Meta.parallelism,
	})
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if err := ctx.Input(c.InputMode()); err != nil {
		c.Ui.Error(fmt.Sprintf("Error configuring: %s", err))
		return 1
	}

	if !validateContext(ctx, c.Ui) {
		return 1
	}

	if refresh {
		c.Ui.Output("Refreshing Terraform state prior to plan...\n")
		state, err := ctx.Refresh()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error refreshing state: %s", err))
			return 1
		}
		c.Ui.Output("")

		if state != nil {
			log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
			if err := c.Meta.PersistState(state); err != nil {
				c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
				return 1
			}
		}
	}

	plan, err := ctx.Plan()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error running plan: %s", err))
		return 1
	}

	if outPath != "" {
		log.Printf("[INFO] Writing plan output to: %s", outPath)
		f, err := os.Create(outPath)
		if err == nil {
			defer f.Close()
			err = terraform.WritePlan(plan, f)
		}
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error writing plan file: %s", err))
			return 1
		}
	}

	if plan.Diff.Empty() {
		c.Ui.Output(
			"No changes. Infrastructure is up-to-date. This means that Terraform\n" +
				"could not detect any differences between your configuration and\n" +
				"the real physical resources that exist. As a result, Terraform\n" +
				"doesn't need to do anything.")
		return 0
	}

	if outPath == "" {
		c.Ui.Output(strings.TrimSpace(planHeaderNoOutput) + "\n")
	} else {
		c.Ui.Output(fmt.Sprintf(
			strings.TrimSpace(planHeaderYesOutput)+"\n",
			outPath))
	}

	c.Ui.Output(FormatPlan(&FormatPlanOpts{
		Plan:        plan,
		Color:       c.Colorize(),
		ModuleDepth: moduleDepth,
	}))

	c.Ui.Output(c.Colorize().Color(fmt.Sprintf(
		"[reset][bold]Plan:[reset] "+
			"%d to add, %d to change, %d to destroy.",
		countHook.ToAdd+countHook.ToRemoveAndAdd,
		countHook.ToChange,
		countHook.ToRemove+countHook.ToRemoveAndAdd)))

	if detailed {
		return 2
	}
	return 0
}

func (c *PlanCommand) Help() string {
	helpText := `
Usage: terraform plan [options] [dir]

  Generates an execution plan for Terraform.

  This execution plan can be reviewed prior to running apply to get a
  sense for what Terraform will do. Optionally, the plan can be saved to
  a Terraform plan file, and apply can take this plan file to execute
  this plan exactly.

Options:

  -backup=path        Path to backup the existing state file before
                      modifying. Defaults to the "-state-out" path with
                      ".backup" extension. Set to "-" to disable backup.

  -destroy            If set, a plan will be generated to destroy all resources
                      managed by the given configuration and state.

  -detailed-exitcode  Return detailed exit codes when the command exits. This
                      will change the meaning of exit codes to:
                      0 - Succeeded, diff is empty (no changes)
                      1 - Errored
                      2 - Succeeded, there is a diff

  -input=true         Ask for input for variables if not directly set.

  -module-depth=n     Specifies the depth of modules to show in the output.
                      This does not affect the plan itself, only the output
                      shown. By default, this is -1, which will expand all.

  -no-color           If specified, output won't contain any color.

  -out=path           Write a plan file to the given path. This can be used as
                      input to the "apply" command.

  -parallelism=n      Limit the number of concurrent operations. Defaults to 10.

  -refresh=true       Update state prior to checking for differences.

  -state=statefile    Path to a Terraform state file to use to look
                      up Terraform-managed resources. By default it will
                      use the state "terraform.tfstate" if it exists.

  -target=resource    Resource to target. Operation will be limited to this
                      resource and its dependencies. This flag can be used
                      multiple times.

  -var 'foo=bar'      Set a variable in the Terraform configuration. This
                      flag can be set multiple times.

  -var-file=foo       Set variables in the Terraform configuration from
                      a file. If "terraform.tfvars" is present, it will be
                      automatically loaded if this flag is not specified.
`
	return strings.TrimSpace(helpText)
}

func (c *PlanCommand) Synopsis() string {
	return "Generate and show an execution plan"
}

const planHeaderNoOutput = `
The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed.

Note: You didn't specify an "-out" parameter to save this plan, so when
"apply" is called, Terraform can't guarantee this is what will execute.
`

const planHeaderYesOutput = `
The Terraform execution plan has been generated and is shown below.
Resources are shown in alphabetical order for quick scanning. Green resources
will be created (or destroyed and then created if an existing resource
exists), yellow resources are being changed in-place, and red resources
will be destroyed.

Your plan was also saved to the path below. Call the "apply" subcommand
with this plan file and Terraform will exactly execute this execution
plan.

Path: %s
`

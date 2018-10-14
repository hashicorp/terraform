package command

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/go-getter"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/repl"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// ApplyCommand is a Command implementation that applies a Terraform
// configuration and actually builds or changes infrastructure.
type ApplyCommand struct {
	Meta

	// If true, then this apply command will become the "destroy"
	// command. It is just like apply but only processes a destroy.
	Destroy bool
}

func (c *ApplyCommand) Run(args []string) int {
	var destroyForce, refresh, autoApprove bool
	args, err := c.Meta.process(args, true)
	if err != nil {
		return 1
	}

	cmdName := "apply"
	if c.Destroy {
		cmdName = "destroy"
	}

	cmdFlags := c.Meta.flagSet(cmdName)
	cmdFlags.BoolVar(&autoApprove, "auto-approve", false, "skip interactive approval of plan before applying")
	if c.Destroy {
		cmdFlags.BoolVar(&destroyForce, "force", false, "deprecated: same as auto-approve")
	}
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	// Get the args. The "maybeInit" flag tracks whether we may need to
	// initialize the configuration from a remote path. This is true as long
	// as we have an argument.
	args = cmdFlags.Args()
	maybeInit := len(args) == 1
	configPath, err := ModulePath(args)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	if !c.Destroy && maybeInit {
		// We need the pwd for the getter operation below
		pwd, err := os.Getwd()
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
			return 1
		}

		// Do a detect to determine if we need to do an init + apply.
		if detected, err := getter.Detect(configPath, pwd, getter.Detectors); err != nil {
			c.Ui.Error(fmt.Sprintf("Invalid path: %s", err))
			return 1
		} else if !strings.HasPrefix(detected, "file") {
			// If this isn't a file URL then we're doing an init +
			// apply.
			var init InitCommand
			init.Meta = c.Meta
			if code := init.Run([]string{detected}); code != 0 {
				return code
			}

			// Change the config path to be the cwd
			configPath = pwd
		}
	}

	// Check if the path is a plan
	planFile, err := c.PlanFile(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if c.Destroy && planFile != nil {
		c.Ui.Error(fmt.Sprintf("Destroy can't be called with a plan file."))
		return 1
	}
	if planFile != nil {
		// Reset the config path for backend loading
		configPath = ""
	}

	var diags tfdiags.Diagnostics

	// Load the backend
	var be backend.Enhanced
	var beDiags tfdiags.Diagnostics
	if planFile == nil {
		backendConfig, configDiags := c.loadBackendConfig(configPath)
		diags = diags.Append(configDiags)
		if configDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}

		be, beDiags = c.Backend(&BackendOpts{
			Config: backendConfig,
		})
	} else {
		plan, err := planFile.ReadPlan()
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				fmt.Sprintf("Cannot read the plan from the given plan file: %s.", err),
			))
			c.showDiagnostics(diags)
			return 1
		}
		if plan.Backend.Config == nil {
			// Should never happen; always indicates a bug in the creation of the plan file
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read plan from plan file",
				fmt.Sprintf("The given plan file does not have a valid backend configuration. This is a bug in the Terraform command that generated this plan file."),
			))
			c.showDiagnostics(diags)
			return 1
		}
		be, beDiags = c.BackendForPlan(plan.Backend)
	}
	diags = diags.Append(beDiags)
	if beDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Before we delegate to the backend, we'll print any warning diagnostics
	// we've accumulated here, since the backend will start fresh with its own
	// diagnostics.
	c.showDiagnostics(diags)
	diags = nil

	// Build the operation
	opReq := c.Operation(be)
	opReq.AutoApprove = autoApprove
	opReq.Destroy = c.Destroy
	opReq.ConfigDir = configPath
	opReq.PlanFile = planFile
	opReq.PlanRefresh = refresh
	opReq.Type = backend.OperationTypeApply
	opReq.AutoApprove = autoApprove
	opReq.DestroyForce = destroyForce
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}
	{
		var moreDiags tfdiags.Diagnostics
		opReq.Variables, moreDiags = c.collectVariableValues()
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}

	op, err := c.RunOperation(be, opReq)
	if err != nil {
		c.showDiagnostics(err)
		return 1
	}
	if op.Result != backend.OperationSuccess {
		return op.Result.ExitStatus()
	}

	if !c.Destroy {
		if outputs := outputsAsString(op.State, addrs.RootModuleInstance, true); outputs != "" {
			c.Ui.Output(c.Colorize().Color(outputs))
		}
	}

	return op.Result.ExitStatus()
}

func (c *ApplyCommand) Help() string {
	if c.Destroy {
		return c.helpDestroy()
	}

	return c.helpApply()
}

func (c *ApplyCommand) Synopsis() string {
	if c.Destroy {
		return "Destroy Terraform-managed infrastructure"
	}

	return "Builds or changes infrastructure"
}

func (c *ApplyCommand) helpApply() string {
	helpText := `
Usage: terraform apply [options] [DIR-OR-PLAN]

  Builds or changes infrastructure according to Terraform configuration
  files in DIR.

  By default, apply scans the current directory for the configuration
  and applies the changes appropriately. However, a path to another
  configuration or an execution plan can be provided. Execution plans can be
  used to only execute a pre-determined set of actions.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -auto-approve          Skip interactive approval of plan before applying.

  -lock=true             Lock the state file when locking is supported.

  -lock-timeout=0s       Duration to retry a state lock.

  -input=true            Ask for input for variables if not directly set.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of parallel resource operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" or any ".auto.tfvars"
                         files are present, they will be automatically loaded.


`
	return strings.TrimSpace(helpText)
}

func (c *ApplyCommand) helpDestroy() string {
	helpText := `
Usage: terraform destroy [options] [DIR]

  Destroy Terraform-managed infrastructure.

Options:

  -backup=path           Path to backup the existing state file before
                         modifying. Defaults to the "-state-out" path with
                         ".backup" extension. Set to "-" to disable backup.

  -auto-approve          Skip interactive approval before destroying.

  -force                 Deprecated: same as auto-approve.

  -lock=true             Lock the state file when locking is supported.

  -lock-timeout=0s       Duration to retry a state lock.

  -no-color              If specified, output won't contain any color.

  -parallelism=n         Limit the number of concurrent operations.
                         Defaults to 10.

  -refresh=true          Update state prior to checking for differences. This
                         has no effect if a plan file is given to apply.

  -state=path            Path to read and save state (unless state-out
                         is specified). Defaults to "terraform.tfstate".

  -state-out=path        Path to write state to that is different than
                         "-state". This can be used to preserve the old
                         state.

  -target=resource       Resource to target. Operation will be limited to this
                         resource and its dependencies. This flag can be used
                         multiple times.

  -var 'foo=bar'         Set a variable in the Terraform configuration. This
                         flag can be set multiple times.

  -var-file=foo          Set variables in the Terraform configuration from
                         a file. If "terraform.tfvars" or any ".auto.tfvars"
                         files are present, they will be automatically loaded.


`
	return strings.TrimSpace(helpText)
}

func outputsAsString(state *states.State, modPath addrs.ModuleInstance, includeHeader bool) string {
	if state == nil {
		return ""
	}

	ms := state.Module(modPath)
	if ms == nil {
		return ""
	}

	outputs := ms.OutputValues
	outputBuf := new(bytes.Buffer)
	if len(outputs) > 0 {
		if includeHeader {
			outputBuf.WriteString("[reset][bold][green]\nOutputs:\n\n")
		}

		// Output the outputs in alphabetical order
		keyLen := 0
		ks := make([]string, 0, len(outputs))
		for key, _ := range outputs {
			ks = append(ks, key)
			if len(key) > keyLen {
				keyLen = len(key)
			}
		}
		sort.Strings(ks)

		for _, k := range ks {
			v := outputs[k]
			if v.Sensitive {
				outputBuf.WriteString(fmt.Sprintf("%s = <sensitive>\n", k))
				continue
			}

			// Our formatter still wants an old-style raw interface{} value, so
			// for now we'll just shim it.
			// FIXME: Port the formatter to work with cty.Value directly.
			legacyVal := hcl2shim.ConfigValueFromHCL2(v.Value)
			result, err := repl.FormatResult(legacyVal)
			if err != nil {
				// We can't really return errors from here, so we'll just have
				// to stub this out. This shouldn't happen in practice anyway.
				result = "<error during formatting>"
			}

			outputBuf.WriteString(fmt.Sprintf("%s = %s\n", k, result))
		}
	}

	return strings.TrimSpace(outputBuf.String())
}

const outputInterrupt = `Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...`

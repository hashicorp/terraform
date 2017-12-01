package command

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/terraform"
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
	if c.Destroy {
		cmdFlags.BoolVar(&destroyForce, "force", false, "force")
	}
	cmdFlags.BoolVar(&refresh, "refresh", true, "refresh")
	if !c.Destroy {
		cmdFlags.BoolVar(&autoApprove, "auto-approve", false, "skip interactive approval of plan before applying")
	}
	cmdFlags.IntVar(
		&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
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
			c.Ui.Error(fmt.Sprintf(
				"Invalid path: %s", err))
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
	plan, err := c.Plan(configPath)
	if err != nil {
		c.Ui.Error(err.Error())
		return 1
	}
	if c.Destroy && plan != nil {
		c.Ui.Error(fmt.Sprintf(
			"Destroy can't be called with a plan file."))
		return 1
	}
	if plan != nil {
		// Reset the config path for backend loading
		configPath = ""
	}

	// Load the module if we don't have one yet (not running from plan)
	var mod *module.Tree
	if plan == nil {
		mod, err = c.Module(configPath)
		if err != nil {
			err = errwrap.Wrapf("Failed to load root config module: {{err}}", err)
			c.showDiagnostics(err)
			return 1
		}
	}

	/*
		terraform.SetDebugInfo(DefaultDataDir)

		// Check for the legacy graph
		if experiment.Enabled(experiment.X_legacyGraph) {
			c.Ui.Output(c.Colorize().Color(
				"[reset][bold][yellow]" +
					"Legacy graph enabled! This will use the graph from Terraform 0.7.x\n" +
					"to execute this operation. This will be removed in the future so\n" +
					"please report any issues causing you to use this to the Terraform\n" +
					"project.\n\n"))
		}
	*/

	var conf *config.Config
	if mod != nil {
		conf = mod.Config()
	}

	// Load the backend
	b, err := c.Backend(&BackendOpts{
		Config: conf,
		Plan:   plan,
	})
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load backend: %s", err))
		return 1
	}

	// Build the operation
	opReq := c.Operation()
	opReq.Destroy = c.Destroy
	opReq.Module = mod
	opReq.Plan = plan
	opReq.PlanRefresh = refresh
	opReq.Type = backend.OperationTypeApply
	opReq.AutoApprove = autoApprove
	opReq.DestroyForce = destroyForce

	// Perform the operation
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	op, err := b.Operation(ctx, opReq)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error starting operation: %s", err))
		return 1
	}

	// Wait for the operation to complete or an interrupt to occur
	select {
	case <-c.ShutdownCh:
		// Cancel our context so we can start gracefully exiting
		ctxCancel()

		// notify tests that the command context was canceled
		if testShutdownHook != nil {
			testShutdownHook()
		}

		// Notify the user
		c.Ui.Output(outputInterrupt)

		// Still get the result, since there is still one
		select {
		case <-c.ShutdownCh:
			c.Ui.Error(
				"Two interrupts received. Exiting immediately. Note that data\n" +
					"loss may have occurred.")
			return 1
		case <-op.Done():
		}
	case <-op.Done():
		if err := op.Err; err != nil {
			c.showDiagnostics(err)
			return 1
		}
	}

	if !c.Destroy {
		// Get the right module that we used. If we ran a plan, then use
		// that module.
		if plan != nil {
			mod = plan.Module
		}

		if outputs := outputsAsString(op.State, terraform.RootModulePath, mod.Config().Outputs, true); outputs != "" {
			c.Ui.Output(c.Colorize().Color(outputs))
		}
	}

	return 0
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

  -lock=true             Lock the state file when locking is supported.

  -lock-timeout=0s       Duration to retry a state lock.

  -auto-approve          Skip interactive approval of plan before applying.

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

  -force                 Don't ask for input for destroy confirmation.

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

func outputsAsString(state *terraform.State, modPath []string, schema []*config.Output, includeHeader bool) string {
	if state == nil {
		return ""
	}

	ms := state.ModuleByPath(modPath)
	if ms == nil {
		return ""
	}

	outputs := ms.Outputs
	outputBuf := new(bytes.Buffer)
	if len(outputs) > 0 {
		schemaMap := make(map[string]*config.Output)
		if schema != nil {
			for _, s := range schema {
				schemaMap[s.Name] = s
			}
		}

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
			schema, ok := schemaMap[k]
			if ok && schema.Sensitive {
				outputBuf.WriteString(fmt.Sprintf("%s = <sensitive>\n", k))
				continue
			}

			v := outputs[k]
			switch typedV := v.Value.(type) {
			case string:
				outputBuf.WriteString(fmt.Sprintf("%s = %s\n", k, typedV))
			case []interface{}:
				outputBuf.WriteString(formatListOutput("", k, typedV))
				outputBuf.WriteString("\n")
			case map[string]interface{}:
				outputBuf.WriteString(formatMapOutput("", k, typedV))
				outputBuf.WriteString("\n")
			}
		}
	}

	return strings.TrimSpace(outputBuf.String())
}

const outputInterrupt = `Interrupt received.
Please wait for Terraform to exit or data loss may occur.
Gracefully shutting down...`

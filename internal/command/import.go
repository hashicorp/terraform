package command

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ImportCommand is a cli.Command implementation that imports resources
// into the Terraform state.
type ImportCommand struct {
	Meta
}

func (c *ImportCommand) Run(args []string) int {
	// Get the pwd since its our default -config flag value
	pwd, err := os.Getwd()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Error getting pwd: %s", err))
		return 1
	}

	var configPath string
	var bulkPath string
	args = c.Meta.process(args)

	cmdFlags := c.Meta.extendedFlagSet("import")
	cmdFlags.BoolVar(&c.ignoreRemoteVersion, "ignore-remote-version", false, "continue even if remote and local Terraform versions are incompatible")
	cmdFlags.IntVar(&c.Meta.parallelism, "parallelism", DefaultParallelism, "parallelism")
	cmdFlags.StringVar(&c.Meta.statePath, "state", "", "path")
	cmdFlags.StringVar(&c.Meta.stateOutPath, "state-out", "", "path")
	cmdFlags.StringVar(&c.Meta.backupPath, "backup", "", "path")
	cmdFlags.StringVar(&configPath, "config", pwd, "path")
	cmdFlags.BoolVar(&c.Meta.stateLock, "lock", true, "lock state")
	cmdFlags.DurationVar(&c.Meta.stateLockTimeout, "lock-timeout", 0, "lock timeout")
	cmdFlags.StringVar(&bulkPath, "bulk", "", "import resources in bulk from file")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if bulkPath != "" {
		if len(args) != 0 {
			c.Ui.Error("The import command doesn't accept arguments when -bulk option is given")
			cmdFlags.Usage()
			return 1
		}
		return c.importBulk(configPath, bulkPath)
	}
	if len(args) != 2 {
		c.Ui.Error("The import command expects two arguments.")
		cmdFlags.Usage()
		return 1
	}
	return c.importSingle(configPath, args)
}

func (c *ImportCommand) importBulk(configPath string, importFile string) int {
	var err error
	var input io.Reader
	if importFile == "-" {
		input = os.Stdin
	} else {
		input, err = os.Open(importFile)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Unable to open file %s: %s", importFile, err))
			return 1
		}
	}
	var targets []*terraform.ImportTarget
	scanner := bufio.NewScanner(input)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ")
		// This assumes that there aren't any spaces in the ID. But as far
		// as I know, that is a good assumption.
		splitIdx := strings.LastIndex(line, " ")
		if splitIdx < 0 {
			c.Ui.Error("Each line of input must be a resource and id seperated by a space.")
			return 1
		}
		target, ok := c.getTarget(line[:splitIdx], line[splitIdx+1:])
		if !ok {
			return 1
		}
		targets = append(targets, target)
	}
	if err = scanner.Err(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error reading input: %s.", err))
		return 1
	}

	return c.importTargets(configPath, targets)
}

func (c *ImportCommand) importSingle(configPath string, args []string) int {

	// Parse the provided resource address.
	target, ok := c.getTarget(args[0], args[1])
	if !ok {
		return 1
	}
	if !ok {
		return 1
	}

	return c.importTargets(configPath, []*terraform.ImportTarget{
		target,
	})
}

func (c *ImportCommand) importTargets(configPath string, targets []*terraform.ImportTarget) int {
	var diags tfdiags.Diagnostics
	var err error

	if !c.dirIsConfigPath(configPath) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No Terraform configuration files",
			Detail: fmt.Sprintf(
				"The directory %s does not contain any Terraform configuration files (.tf or .tf.json). To specify a different configuration directory, use the -config=\"...\" command line option.",
				configPath,
			),
		})
		c.showDiagnostics(diags)
		return 1
	}

	// Load the full config, so we can verify that the target resource is
	// already configured.
	config, configDiags := c.loadConfig(configPath)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	var missingResources = false
	// Verify that the given addresses point to something that exists in config.
	// This is to reduce the risk that a typo in the resource address will
	// import something that Terraform will want to immediately destroy on
	// the next plan, and generally acts as a reassurance of user intent.
	for _, target := range targets {
		addr := target.Addr
		targetConfig := config.DescendentForInstance(addr.Module)
		if targetConfig == nil {
			modulePath := addr.Module.String()
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Import to non-existent module",
				Detail: fmt.Sprintf(
					"%s is not defined in the configuration. Please add configuration for this module before importing into it.",
					modulePath,
				),
			})
			c.showDiagnostics(diags)
			return 1
		}
		targetMod := targetConfig.Module
		rcs := targetMod.ManagedResources
		var rc *configs.Resource
		resourceRelAddr := addr.Resource.Resource
		for _, thisRc := range rcs {
			if resourceRelAddr.Type == thisRc.Type && resourceRelAddr.Name == thisRc.Name {
				rc = thisRc
				break
			}
		}
		if rc == nil {
			modulePath := addr.Module.String()
			if modulePath == "" {
				modulePath = "the root module"
			}

			c.showDiagnostics(diags)

			// This is not a diagnostic because currently our diagnostics printer
			// doesn't support having a code example in the detail, and there's
			// a code example in this message.
			// TODO: Improve the diagnostics printer so we can use it for this
			// message.
			c.Ui.Error(fmt.Sprintf(
				importCommandMissingResourceFmt,
				addr, modulePath, resourceRelAddr.Type, resourceRelAddr.Name,
			))
			return 1
		}
	}

	// Check for user-supplied plugin path
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Load the backend
	b, backendDiags := c.Backend(&BackendOpts{
		Config: config.Module.Backend,
	})
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a backend.Local to build a context.
	// This isn't necessarily a "local.Local" backend, which provides local
	// operations, however that is the only current implementation. A
	// "local.Local" backend also doesn't necessarily provide local state, as
	// that may be delegated to a "remotestate.Backend".
	local, ok := b.(backend.Local)
	if !ok {
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}
	opReq.Hooks = []terraform.Hook{c.uiHook()}
	{
		var moreDiags tfdiags.Diagnostics
		opReq.Variables, moreDiags = c.collectVariableValues()
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			c.showDiagnostics(diags)
			return 1
		}
	}
	opReq.View = views.NewOperation(arguments.ViewHuman, c.RunningInAutomation, c.View)

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, opReq.Workspace)
	diags = diags.Append(remoteVersionDiags)
	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	// Get the context
	lr, state, ctxDiags := local.LocalRun(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Successfully creating the context can result in a lock, so ensure we release it
	defer func() {
		diags := opReq.StateLocker.Unlock()
		if diags.HasErrors() {
			c.showDiagnostics(diags)
		}
	}()

	// Perform the import. Note that as you can see it is possible for this
	// API to import more than one resource at once. For now, we only allow
	// one while we stabilize this feature.
	newState, importDiags := lr.Core.Import(lr.Config, lr.InputState, &terraform.ImportOpts{
		Targets: targets,

		// The LocalRun idea is designed around our primary operations, so
		// the input variables end up represented as plan options even though
		// this particular operation isn't really a plan.
		SetVariables: lr.PlanOpts.SetVariables,
	})
	diags = diags.Append(importDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Get schemas, if possible, before writing state
	var schemas *terraform.Schemas
	if isCloudMode(b) {
		var schemaDiags tfdiags.Diagnostics
		schemas, schemaDiags = c.MaybeGetSchemas(newState, nil)
		diags = diags.Append(schemaDiags)
	}

	// Persist the final state
	log.Printf("[INFO] Writing state output to: %s", c.Meta.StateOutPath())
	if err := state.WriteState(newState); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}
	if err := state.PersistState(schemas); err != nil {
		c.Ui.Error(fmt.Sprintf("Error writing state file: %s", err))
		return 1
	}

	c.Ui.Output(c.Colorize().Color("[reset][green]\n" + importCommandSuccessMsg))

	c.showDiagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	return 0
}

func (c *ImportCommand) getTarget(resource string, id string) (*terraform.ImportTarget, bool) {
	traversalSrc := []byte(resource)
	traversal, travDiags := hclsyntax.ParseTraversalAbs(traversalSrc, "<import-address>", hcl.Pos{Line: 1, Column: 1})
	if travDiags.HasErrors() {
		c.registerSynthConfigSource("<import-address>", traversalSrc) // so we can include a source snippet
		c.showDiagnostics(travDiags)
		c.Ui.Info(importCommandInvalidAddressReference)
		return nil, false
	}
	addr, addrDiags := addrs.ParseAbsResourceInstance(traversal)
	if addrDiags.HasErrors() {
		c.registerSynthConfigSource("<import-address>", traversalSrc) // so we can include a source snippet
		c.showDiagnostics(addrDiags)
		c.Ui.Info(importCommandInvalidAddressReference)
		return nil, false
	}

	if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
		diags := errors.New("A managed resource address is required. Importing into a data resource is not allowed.")
		c.showDiagnostics(diags)
		return nil, false
	}

	return &terraform.ImportTarget{
		Addr: addr,
		ID:   id,
	}, true
}

func (c *ImportCommand) Help() string {
	helpText := `
Usage: terraform [global options] import [options] [ADDR ID]

  Import existing infrastructure into your Terraform state.

  This will find and import the specified resource into your Terraform
  state, allowing existing infrastructure to come under Terraform
  management without having to be initially created by Terraform.

  The ADDR specified is the address to import the resource to. Please
  see the documentation online for resource addresses. The ID is a
  resource-specific ID to identify that resource being imported. Please
  reference the documentation for the resource type you're importing to
  determine the ID syntax to use. It typically matches directly to the ID
  that the provider uses.

  The current implementation of Terraform import can only import resources
  into the state. It does not generate configuration. A future version of
  Terraform will also generate configuration.

  Because of this, prior to running terraform import it is necessary to write
  a resource configuration block for the resource manually, to which the
  imported object will be attached.

  This command will not modify your infrastructure, but it will make
  network requests to inspect parts of your infrastructure relevant to
  the resource being imported.

Options:

  -config=path            Path to a directory of Terraform configuration files
                          to use to configure the provider. Defaults to pwd.
                          If no config files are present, they must be provided
                          via the input prompts or env vars.

  -bulk=path              Import resources in bulk from a file. If this option is
                          supplied, then ADDR and ID should not be given on the
                          command line. Instead, the file at the path should
                          contain lines with addresses and ids seperated by a
                          space on each line. A path of "-" will cause it to
                          read from stdin.

  -input=false            Disable interactive input prompts.

  -lock=false             Don't hold a state lock during the operation. This is
                          dangerous if others might concurrently run commands
                          against the same workspace.

  -lock-timeout=0s        Duration to retry a state lock.

  -no-color               If specified, output won't contain any color.

  -var 'foo=bar'          Set a variable in the Terraform configuration. This
                          flag can be set multiple times. This is only useful
                          with the "-config" flag.

  -var-file=foo           Set variables in the Terraform configuration from
                          a file. If "terraform.tfvars" or any ".auto.tfvars"
                          files are present, they will be automatically loaded.

  -ignore-remote-version  A rare option used for the remote backend only. See
                          the remote backend documentation for more information.

  -state, state-out, and -backup are legacy options supported for the local
  backend only. For more information, see the local backend's documentation.

`
	return strings.TrimSpace(helpText)
}

func (c *ImportCommand) Synopsis() string {
	return "Associate existing infrastructure with a Terraform resource"
}

const importCommandInvalidAddressReference = `For information on valid syntax, see:
https://www.terraform.io/docs/cli/state/resource-addressing.html`

const importCommandMissingResourceFmt = `[reset][bold][red]Error:[reset][bold] resource address %q does not exist in the configuration.[reset]

Before importing this resource, please create its configuration in %s. For example:

resource %q %q {
  # (resource arguments)
}
`

const importCommandSuccessMsg = `Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
`

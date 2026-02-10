// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
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

func (c *ImportCommand) Run(rawArgs []string) int {
	var diags tfdiags.Diagnostics

	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	// Propagate -no-color for legacy use of Ui. The remote backend and
	// cloud package use this; it should be removed when/if they are
	// migrated to views.
	c.Meta.color = !common.NoColor
	c.Meta.Color = c.Meta.color

	// Parse and validate flags
	args, diags := arguments.ParseImport(rawArgs)

	// Instantiate the view, even if there are flag errors, so that we render
	// diagnostics according to the desired view
	view := views.NewImport(c.View)

	if diags.HasErrors() {
		view.Diagnostics(diags)
		view.HelpPrompt()
		return 1
	}

	// FIXME: the -input flag value is needed to initialize the backend and the
	// operation, but there is no clear path to pass this value down, so we
	// continue to mutate the Meta object state for now.
	c.Meta.input = args.InputEnabled

	// FIXME: the -parallelism flag is used to control the concurrency of
	// Terraform operations. At the moment, this value is used both to
	// initialize the backend via the ContextOpts field inside CLIOpts, and to
	// set a largely unused field on the Operation request. Again, there is no
	// clear path to pass this value down, so we continue to mutate the Meta
	// object state for now.
	c.Meta.parallelism = args.Parallelism

	// FIXME: we need to apply the state arguments to the meta object here
	// because they are later used when initializing the backend. Carving a
	// path to pass these arguments to the functions that need them is
	// difficult but would make their use easier to understand.
	c.Meta.applyStateArguments(args.State)

	c.ignoreRemoteVersion = args.IgnoreRemoteVersion

	// Determine config path, defaulting to pwd
	configPath := args.ConfigPath
	if configPath == "" {
		pwd, err := os.Getwd()
		if err != nil {
			diags = diags.Append(fmt.Errorf("Error getting pwd: %s", err))
			view.Diagnostics(diags)
			return 1
		}
		configPath = pwd
	}

	// Parse the provided resource address.
	traversalSrc := []byte(args.Addr)
	traversal, travDiags := hclsyntax.ParseTraversalAbs(traversalSrc, "<import-address>", hcl.Pos{Line: 1, Column: 1})
	diags = diags.Append(travDiags)
	if travDiags.HasErrors() {
		c.registerSynthConfigSource("<import-address>", traversalSrc) // so we can include a source snippet
		view.Diagnostics(diags)
		view.InvalidAddressReference()
		return 1
	}
	addr, addrDiags := addrs.ParseAbsResourceInstance(traversal)
	diags = diags.Append(addrDiags)
	if addrDiags.HasErrors() {
		c.registerSynthConfigSource("<import-address>", traversalSrc) // so we can include a source snippet
		view.Diagnostics(diags)
		view.InvalidAddressReference()
		return 1
	}

	if addr.Resource.Resource.Mode != addrs.ManagedResourceMode {
		diags = diags.Append(errors.New("A managed resource address is required. Importing into a data resource is not allowed."))
		view.Diagnostics(diags)
		return 1
	}

	if !c.dirIsConfigPath(configPath) {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "No Terraform configuration files",
			Detail: fmt.Sprintf(
				"The directory %s does not contain any Terraform configuration files (.tf or .tf.json). To specify a different configuration directory, use the -config=\"...\" command line option.",
				configPath,
			),
		})
		view.Diagnostics(diags)
		return 1
	}

	// Load the full config, so we can verify that the target resource is
	// already configured.
	config, configDiags := c.loadConfig(configPath)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Verify that the given address points to something that exists in config.
	// This is to reduce the risk that a typo in the resource address will
	// import something that Terraform will want to immediately destroy on
	// the next plan, and generally acts as a reassurance of user intent.
	targetConfig := config.DescendantForInstance(addr.Module)
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
		view.Diagnostics(diags)
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

		view.Diagnostics(diags)

		// This is not a diagnostic because currently our diagnostics printer
		// doesn't support having a code example in the detail, and there's
		// a code example in this message.
		// TODO: Improve the diagnostics printer so we can use it for this
		// message.
		view.MissingResourceConfig(addr.String(), modulePath, resourceRelAddr.Type, resourceRelAddr.Name)
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(fmt.Errorf("Error loading plugin path: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	// Load the backend
	b, backendDiags := c.backend(".", arguments.ViewHuman)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// We require a backendrun.Local to build a context.
	// This isn't necessarily a "local.Local" backend, which provides local
	// operations, however that is the only current implementation. A
	// "local.Local" backend also doesn't necessarily provide local state, as
	// that may be delegated to a "remotestate.Backend".
	local, ok := b.(backendrun.Local)
	if !ok {
		diags = diags.Append(errors.New(ErrUnsupportedLocalOp))
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b, arguments.ViewHuman)
	opReq.ConfigDir = configPath
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		view.Diagnostics(diags)
		return 1
	}
	opReq.Hooks = []terraform.Hook{c.uiHook()}
	{
		moreDiags := c.GatherVariables(opReq, args.Vars)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			view.Diagnostics(diags)
			return 1
		}
	}
	opReq.View = views.NewOperation(arguments.ViewHuman, c.RunningInAutomation, c.View)

	// Check remote Terraform version is compatible
	remoteVersionDiags := c.remoteVersionCheck(b, opReq.Workspace)
	diags = diags.Append(remoteVersionDiags)
	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	// Get the context
	lr, state, ctxDiags := local.LocalRun(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Successfully creating the context can result in a lock, so ensure we release it
	defer func() {
		diags := opReq.StateLocker.Unlock()
		if diags.HasErrors() {
			view.Diagnostics(diags)
		}
	}()

	// Perform the import. Note that as you can see it is possible for this
	// API to import more than one resource at once. For now, we only allow
	// one while we stabilize this feature.
	newState, importDiags := lr.Core.Import(lr.Config, lr.InputState, &terraform.ImportOpts{
		Targets: []*terraform.ImportTarget{
			{
				LegacyAddr: addr,
				LegacyID:   args.ID,
			},
		},

		// The LocalRun idea is designed around our primary operations, so
		// the input variables end up represented as plan options even though
		// this particular operation isn't really a plan.
		SetVariables: lr.PlanOpts.SetVariables,
	})
	diags = diags.Append(importDiags)
	if diags.HasErrors() {
		view.Diagnostics(diags)
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
		diags = diags.Append(fmt.Errorf("Error writing state file: %s", err))
		view.Diagnostics(diags)
		return 1
	}
	if err := state.PersistState(schemas); err != nil {
		diags = diags.Append(fmt.Errorf("Error writing state file: %s", err))
		view.Diagnostics(diags)
		return 1
	}

	view.Success()

	view.Diagnostics(diags)
	if diags.HasErrors() {
		return 1
	}

	return 0
}

// GatherVariables collects variable values from the arguments and populates
// the operation request.
func (c *ImportCommand) GatherVariables(opReq *backendrun.Operation, args *arguments.Vars) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// FIXME the arguments package currently trivially gathers variable related
	// arguments in a heterogenous slice, in order to minimize the number of
	// code paths gathering variables during the transition to this structure.
	// Once all commands that gather variables have been converted to this
	// structure, we could move the variable gathering code to the arguments
	// package directly, removing this shim layer.

	varArgs := args.All()
	items := make([]arguments.FlagNameValue, len(varArgs))
	for i := range varArgs {
		items[i].Name = varArgs[i].Name
		items[i].Value = varArgs[i].Value
	}
	c.Meta.variableArgs = arguments.FlagNameValueSlice{Items: &items}
	opReq.Variables, diags = c.collectVariableValues()

	return diags
}

func (c *ImportCommand) Help() string {
	helpText := `
Usage: terraform [global options] import [options] ADDR ID

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

  This command will not modify your infrastructure, but it will make
  network requests to inspect parts of your infrastructure relevant to
  the resource being imported.

Options:

  -config=path            Path to a directory of Terraform configuration files
                          to use to configure the provider. Defaults to pwd.
                          If no config files are present, they must be provided
                          via the input prompts or env vars.

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

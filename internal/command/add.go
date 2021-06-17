package command

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// AddCommand is a Command implementation that generates resource configuration templates.
type AddCommand struct {
	Meta
}

func (c *AddCommand) Run(rawArgs []string) int {
	// Parse and apply global view arguments
	common, rawArgs := arguments.ParseView(rawArgs)
	c.View.Configure(common)

	args, diags := arguments.ParseAdd(rawArgs)
	view := views.NewAdd(args.ViewType, c.View, args)
	if diags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error loading plugin path",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Apply the state arguments to the meta object here because they are later
	// used when initializing the backend.
	c.Meta.applyStateArguments(args.State)

	// Load the backend
	b, backendDiags := c.Backend(nil)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backend.Local)
	if !ok {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported backend",
			ErrUnsupportedLocalOp,
		))
		view.Diagnostics(diags)
		return 1
	}

	// This is a read-only command (until -import is implemented)
	c.ignoreRemoteBackendVersionConflict(b)

	cwd, err := os.Getwd()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error determining current working directory",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Build the operation
	opReq := c.Operation(b)
	opReq.AllowUnsetVariables = true
	opReq.ConfigDir = cwd
	opReq.ConfigLoader, err = c.initConfigLoader()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error initializing config loader",
			err.Error(),
		))
		view.Diagnostics(diags)
		return 1
	}

	// Get the context
	ctx, _, ctxDiags := local.Context(opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		view.Diagnostics(diags)
		return 1
	}

	// load the configuration to verify that the resource address doesn't
	// already exist in the config.
	var module *configs.Module
	if args.Addr.Module.IsRoot() {
		module = ctx.Config().Module
	} else {
		// This is weird, but users can potentially specify non-existant module names
		cfg := ctx.Config().Root.Descendent(args.Addr.Module.Module())
		if cfg != nil {
			module = cfg.Module
		}
	}

	if module == nil {
		// It's fine if the module doesn't actually exist; we don't need to check if the resource exists.
	} else {
		if rs, ok := module.ManagedResources[args.Addr.ContainingResource().Config().String()]; ok {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Resource already in configuration",
				Detail:   fmt.Sprintf("The resource %s is already in this configuration at %s. Resource names must be unique per type in each module.", args.Addr, rs.DeclRange),
				Subject:  &rs.DeclRange,
			})
			c.View.Diagnostics(diags)
			return 1
		}
	}

	// Get the schemas from the context
	schemas := ctx.Schemas()

	// Determine the correct provider config address. The provider-related
	// variables may get updated below
	absProviderConfig := args.Provider
	var providerLocalName string
	rs := args.Addr.Resource.Resource

	// If we are getting the values from state, get the AbsProviderConfig
	// directly from state as well.
	var resource *states.Resource
	var moreDiags tfdiags.Diagnostics
	if args.FromState {
		resource, moreDiags = c.getResource(b, args.Addr.ContainingResource())
		if moreDiags.HasErrors() {
			diags = diags.Append(moreDiags)
			c.View.Diagnostics(diags)
			return 1
		}
		absProviderConfig = &resource.ProviderConfig
	}

	if absProviderConfig == nil {
		ip := rs.ImpliedProvider()
		if module != nil {
			provider := module.ImpliedProviderForUnqualifiedType(ip)
			providerLocalName = module.LocalNameForProvider(provider)
			absProviderConfig = &addrs.AbsProviderConfig{
				Provider: provider,
				Module:   args.Addr.Module.Module(),
			}
		} else {
			// lacking any configuration to query, we'll go with a default provider.
			absProviderConfig = &addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider(ip),
			}
			providerLocalName = ip
		}
	} else {
		if module != nil {
			providerLocalName = module.LocalNameForProvider(absProviderConfig.Provider)
		} else {
			providerLocalName = absProviderConfig.Provider.Type
		}
	}

	localProviderConfig := addrs.LocalProviderConfig{
		LocalName: providerLocalName,
		Alias:     absProviderConfig.Alias,
	}

	// Get the schemas from the context
	if _, exists := schemas.Providers[absProviderConfig.Provider]; !exists {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Missing schema for provider",
			fmt.Sprintf("No schema found for provider %s. Please verify that this provider exists in the configuration.", absProviderConfig.Provider.String()),
		))
		c.View.Diagnostics(diags)
		return 1
	}

	// Get the schema for the resource
	schema, schemaVersion := schemas.ResourceTypeConfig(absProviderConfig.Provider, rs.Mode, rs.Type)
	if schema == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Missing resource schema from provider",
			fmt.Sprintf("No resource schema found for %s.", rs.Type),
		))
		c.View.Diagnostics(diags)
		return 1
	}

	stateVal := cty.NilVal
	// Now that we have the schema, we can decode the previously-acquired resource state
	if args.FromState {
		ri := resource.Instance(args.Addr.Resource.Key)
		if ri.Current == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No state for resource",
				fmt.Sprintf("There is no state found for the resource %s, so add cannot populate values.", rs.String()),
			))
			c.View.Diagnostics(diags)
			return 1
		}

		rio, err := ri.Current.Decode(schema.ImpliedType())
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Error decoding state",
				fmt.Sprintf("Error decoding state for resource %s: %s", rs.String(), err.Error()),
			))
			c.View.Diagnostics(diags)
			return 1
		}

		if ri.Current.SchemaVersion != schemaVersion {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Schema version mismatch",
				fmt.Sprintf("schema version %d for %s in state does not match version %d from the provider", ri.Current.SchemaVersion, rs.String(), schemaVersion),
			))
			c.View.Diagnostics(diags)
			return 1
		}

		stateVal = rio.Value
	}

	diags = diags.Append(view.Resource(args.Addr, schema, localProviderConfig, stateVal))
	if diags.HasErrors() {
		c.View.Diagnostics(diags)
		return 1
	}

	return 0
}

func (c *AddCommand) Help() string {
	helpText := `
Usage: terraform [global options] add [options] ADDRESS

  Generates a blank resource template. With no additional options,
  the template will be displayed in the terminal. 

Options:

-from-state=true		Fill the template with values from an existing resource.
                        Defaults to false.

-out=string 			Write the template to a file. If the file already
                        exists, the template will be appended to the file.

-optional=true          Include optional attributes. Defaults to false.

-provider=provider		Override the configured provider for the resource. Conflicts
                        with -from-state
`
	return strings.TrimSpace(helpText)
}

func (c *AddCommand) Synopsis() string {
	return "Generate a resource configuration template"
}

func (c *AddCommand) getResource(b backend.Enhanced, addr addrs.AbsResource) (*states.Resource, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	// Get the state
	env, err := c.Workspace()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error selecting workspace",
			err.Error(),
		))
		return nil, diags
	}

	stateMgr, err := b.StateMgr(env)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error loading state",
			fmt.Sprintf(errStateLoadingState, err),
		))
		return nil, diags
	}

	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Error refreshing state",
			err.Error(),
		))
		return nil, diags
	}

	state := stateMgr.State()
	if state == nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"No state",
			"There is no state found for the current workspace, so add cannot populate values.",
		))
		return nil, diags
	}

	return state.Resource(addr), nil
}

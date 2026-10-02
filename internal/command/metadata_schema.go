// the metadata_schema command is similar to providers schemas, in that it
// returns schemas for providers found in state and configuration (defaults to
// both, can use either),  but it also includes provisioner schemas, can get
// schemas for providers or provisioners not in the local config (using any
// configured installation methods), and can filter results.
package command

import (
	"context"
	"fmt"
	"maps"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type MetadataSchemaCommand struct {
	Meta
}

func (c *MetadataSchemaCommand) Help() string {
	return metadataSchemaCommandHelp
}

func (c *MetadataSchemaCommand) Synopsis() string {
	return "Show schemas for providers and provisioners"
}

func (c *MetadataSchemaCommand) Run(args []string) int {
	parsedArgs, diags := arguments.ParseMetadataSchemas(c.Meta.process(args))
	if diags.HasErrors() {
		// set up the view if possible?
		c.showDiagnostics(diags) // TODO: this should return json if the view was set up
		return 1
	}
	viewType := arguments.ViewJSON // JSON output is required; omitting -json results in a flag parsing error

	// Check for user-supplied plugin path
	var err error
	if c.pluginPath, err = c.loadPluginPath(); err != nil {
		c.Ui.Error(fmt.Sprintf("Error loading plugin path: %s", err))
		return 1
	}

	// Load the backend
	b, backendDiags := c.backend(".", viewType)
	diags = diags.Append(backendDiags)
	if backendDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// We require a local backend
	local, ok := b.(backendrun.Local)
	if !ok {
		c.showDiagnostics(diags) // in case of any warnings in here
		c.Ui.Error(ErrUnsupportedLocalOp)
		return 1
	}

	// This is a read-only command
	c.ignoreRemoteVersionConflict(b)

	// Get the config directory
	cwd := c.WorkingDir.RootModuleDir()

	// Build the operation
	opReq := c.Operation(b, arguments.ViewJSON)
	opReq.ConfigDir = cwd
	opReq.ConfigLoader, err = c.initConfigLoader()
	opReq.AllowUnsetVariables = true
	if err != nil {
		diags = diags.Append(err)
		c.showDiagnostics(diags)
		return 1
	}

	var varDiags tfdiags.Diagnostics
	opReq.Variables, varDiags = parsedArgs.Vars.CollectValues(func(filename string, src []byte) {
		opReq.ConfigLoader.Parser().ForceFileSource(filename, src)
	})
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Get the context
	lr, _, ctxDiags := local.LocalRun(context.Background(), opReq)
	diags = diags.Append(ctxDiags)
	if ctxDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	if parsedArgs.Filters.ConfigOnly && parsedArgs.Filters.StateOnly {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error,
			"Failed to parse command-line flags",
			"-config-only and -state-only are mutually exclusive. Both cannot be true",
		))
		c.showDiagnostics(diags)
		return 1
	}

	if parsedArgs.Filters.ConfigOnly {
		if lr.Config == nil {
			// return error or maybe successful nothing
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error,
				"Invalid inputs",
				"-config-only was true but no config was found",
			))
			c.showDiagnostics(diags)
			return 1
		} else {
			lr.InputState = nil
		}
	}

	if parsedArgs.Filters.StateOnly {
		if lr.InputState == nil {
			// return error or maybe successful nothing
			diags = diags.Append(tfdiags.Sourceless(tfdiags.Error,
				"Invalid inputs",
				"-state-only was true but no state was found",
			))
			c.showDiagnostics(diags)
			return 1
		} else {
			lr.Config = nil
		}
	}

	allSchemas, schemaDiags := lr.Core.Schemas(lr.Config, lr.InputState)
	diags = diags.Append(schemaDiags)
	if schemaDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	filteredSchemas := &terraform.Schemas{
		Providers:    map[addrs.Provider]providers.ProviderSchema{},
		Provisioners: map[string]*configschema.Block{},
	}

	// the next two blocks could actually run before we pull all schemas if we
	// "just" consult the installer (not implemented yet because I talked myself
	// out of this command entirely)
	if parsedArgs.Filters.ProviderFilter != nil {
		// get provider locally, or download - no config/state needed
		// but for now I'm going to do this the providers schema
		p := parsedArgs.Filters.ProviderFilter
		filteredSchemas := &terraform.Schemas{
			Providers: map[addrs.Provider]providers.ProviderSchema{
				addrs.Provider(*p): filterProvider(allSchemas.Providers[*p], parsedArgs.Filters),
			},
		}
		outDiags := c.marshalAndOutputSchemas(filteredSchemas)
		diags.Append(outDiags)
		if diags.HasErrors() {
			return 1
		}
		return 0
	} else if parsedArgs.Filters.ProvisionerFilter != "" {
		// get provisioner locally, or download - no config/state needed
		// but for now I'm only to implement the used config/state path
		p := parsedArgs.Filters.ProvisionerFilter
		filteredSchemas := &terraform.Schemas{
			Provisioners: map[string]*configschema.Block{
				parsedArgs.Filters.ProvisionerFilter: allSchemas.Provisioners[p],
			},
		}
		outDiags := c.marshalAndOutputSchemas(filteredSchemas)
		diags.Append(outDiags)
		if diags.HasErrors() {
			return 1
		}
		return 0
	} else {
		// if we're not returning a single provider or provisioner, filter a copy of allSchemas
		maps.Copy(filteredSchemas.Providers, allSchemas.Providers)
		maps.Copy(filteredSchemas.Provisioners, allSchemas.Provisioners)
	}

	// apply (additional) filters here, so we only marshal what the user requested
	var filterDiags tfdiags.Diagnostics
	filteredSchemas, filterDiags = filterSchemas(filteredSchemas, parsedArgs.Filters)
	diags = diags.Append(filterDiags)
	if filterDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	var pruneDiags tfdiags.Diagnostics
	if parsedArgs.Filters.Prune {
		if lr.Config != nil {
			filteredSchemas, pruneDiags = pruneSchemas(filteredSchemas, lr.Config)
			diags = diags.Append(pruneDiags)
			if schemaDiags.HasErrors() {
				c.showDiagnostics(diags)
				return 1
			}
		} else {
			// if you run providers schemas with no config, you get terraform schemas
			// but when -prune is used in a directory with no config, that should be an error
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"No terraform configuration found",
				"-prune can only be used with config",
			))
		}
	}

	// provisioner marshalling TBD
	if filteredSchemas != nil {
		if len(filteredSchemas.Providers) > 0 {
			jsonSchemas, err := jsonprovider.Marshal(filteredSchemas)
			if err != nil {
				diags = diags.Append(err)
				c.showDiagnostics(diags)
				return 1
			}
			c.Ui.Output(string(jsonSchemas))
		}
	}

	return 0 // good job
}

func (c *MetadataSchemaCommand) marshalAndOutputSchemas(schemas *terraform.Schemas) (diags tfdiags.Diagnostics) {
	if schemas != nil {
		if len(schemas.Providers) > 0 {
			jsonSchemas, err := jsonprovider.Marshal(schemas)
			if err != nil {
				diags = diags.Append(err)
			}
			c.Ui.Output(string(jsonSchemas))
		}
	}
	return nil
}

func filterSchemas(schemas *terraform.Schemas, filters arguments.SchemaFilters) (*terraform.Schemas, tfdiags.Diagnostics) {
	// easy case: only returning provisioners (typename filtering doesn't apply)
	if filters.RType == arguments.Provisioner {
		return &terraform.Schemas{Provisioners: schemas.Provisioners}, nil
	}

	filteredSchema := &terraform.Schemas{
		Providers:    map[addrs.Provider]providers.ProviderSchema{},
		Provisioners: map[string]*configschema.Block{},
	}

	// we've already apply single-provider filtering; now apply type and type name filters
	for p, schema := range schemas.Providers {
		providerSchema := filterProvider(schema, filters)
		filteredSchema.Providers[p] = providerSchema
	}

	// we've already returned schemas if a specific provisioner was requested,
	// now we're only including provisioner schemas if they weren't omitted.
	// Note that this is ignoring type name in favor of the provisioner filter
	// but both could be supported.
	if filters.RType == arguments.NoType {
		maps.Copy(filteredSchema.Provisioners, schemas.Provisioners)
	}

	return filteredSchema, nil
}

// filterProvider walks a single provider schema and removes types and types names as determined by the SchemaFilters.
func filterProvider(schema providers.ProviderSchema, filters arguments.SchemaFilters) providers.ProviderSchema {
	// nothing to do!
	if filters.TypeName == "" && filters.RType == arguments.NoType {
		return schema
	}

	filteredSchemas := providers.ProviderSchema{}
	// single type return
	if filters.RType != arguments.NoType {
		switch filters.RType {
		case arguments.Resource:
			filteredSchemas.ResourceTypes = mapFilter(schema.ResourceTypes, filters.TypeName)
		case arguments.DataSource:
			filteredSchemas.DataSources = mapFilter(schema.DataSources, filters.TypeName)
		case arguments.EphemeralResource:
			filteredSchemas.EphemeralResourceTypes = mapFilter(schema.EphemeralResourceTypes, filters.TypeName)
		case arguments.ListResource:
			filteredSchemas.ListResourceTypes = mapFilter(schema.ListResourceTypes, filters.TypeName)
		case arguments.Action:
			filteredSchemas.Actions = mapFilter(schema.Actions, filters.TypeName)
		case arguments.Function:
			filteredSchemas.Functions = mapFilter(schema.Functions, filters.TypeName)
		case arguments.Provider:
			filteredSchemas.Provider = schema.Provider
		case arguments.StateStore:
			filteredSchemas.StateStores = mapFilter(schema.StateStores, filters.TypeName)
		case arguments.Provisioner:
			// not relevant here
		default:
			// should be impossible!
			panic("unrecognized resource type filter!")
		}
		return filteredSchemas
	}

	// if we're not returning just one type, we're returning them all
	filteredSchemas.ResourceTypes = mapFilter(schema.ResourceTypes, filters.TypeName)
	filteredSchemas.DataSources = mapFilter(schema.DataSources, filters.TypeName)
	filteredSchemas.EphemeralResourceTypes = mapFilter(schema.EphemeralResourceTypes, filters.TypeName)
	filteredSchemas.ListResourceTypes = mapFilter(schema.ListResourceTypes, filters.TypeName)
	filteredSchemas.Actions = mapFilter(schema.Actions, filters.TypeName)
	filteredSchemas.Functions = mapFilter(schema.Functions, filters.TypeName)
	filteredSchemas.Provider = schema.Provider

	return filteredSchemas
}

func pruneSchemas(schemas *terraform.Schemas, config *configs.Config) (*terraform.Schemas, tfdiags.Diagnostics) {
	// collect used types
	tys := struct {
		resourceTys    map[string]any
		dataSourceTys  map[string]any
		ephemeralTys   map[string]any
		listTys        map[string]any
		actionTys      map[string]any
		functionTys    map[string]any
		provisionerTys map[string]any
		providerTys    map[string]any // not in love with this implementation - providerConfigTys?
		stateStoreTys  map[string]any
	}{
		resourceTys:    map[string]any{},
		dataSourceTys:  map[string]any{},
		ephemeralTys:   map[string]any{},
		listTys:        map[string]any{},
		actionTys:      map[string]any{},
		functionTys:    map[string]any{},
		provisionerTys: map[string]any{},
		providerTys:    map[string]any{},
		stateStoreTys:  map[string]any{},
	}

	config.DeepEach(func(c *configs.Config) {
		for _, resource := range c.Module.ManagedResources {
			tys.resourceTys[resource.Type] = true
			for _, p := range resource.Managed.Provisioners {
				tys.provisionerTys[p.Type] = true
			}
		}
		for _, datasource := range c.Module.DataResources {
			tys.dataSourceTys[datasource.Type] = true
		}
		for _, ephemeral := range c.Module.EphemeralResources {
			tys.ephemeralTys[ephemeral.Type] = true
		}
		for _, list := range c.Module.ListResources {
			tys.listTys[list.Type] = true
		}
		for _, action := range c.Module.Actions {
			tys.actionTys[action.Type] = true
		}
		if c.Module.StateStore != nil {
			tys.stateStoreTys[c.Module.StateStore.Type] = true
		}
		for pc := range c.Module.ProviderConfigs {
			tys.providerTys[pc] = true
		}
	})

	for p, pSchema := range schemas.Providers {
		for r := range pSchema.ResourceTypes {
			if _, ok := tys.resourceTys[r]; !ok {
				delete(schemas.Providers[p].ResourceTypes, r)
			}
		}
		for d := range pSchema.DataSources {
			if _, ok := tys.dataSourceTys[d]; !ok {
				delete(pSchema.DataSources, d)
			}
		}
		for e := range pSchema.EphemeralResourceTypes {
			if _, ok := tys.ephemeralTys[e]; !ok {
				delete(pSchema.EphemeralResourceTypes, e)
			}
		}
		for l := range pSchema.ListResourceTypes {
			if _, ok := tys.listTys[l]; !ok {
				delete(pSchema.ListResourceTypes, l)
			}
		}
		for a := range pSchema.Actions {
			if _, ok := tys.actionTys[a]; !ok {
				delete(pSchema.Actions, a)
			}
		}
		for s := range pSchema.StateStores {
			if _, ok := tys.stateStoreTys[s]; !ok {
				delete(pSchema.StateStores, s)
			}
		}
	}

	for p := range schemas.Provisioners {
		if _, ok := tys.provisionerTys[p]; !ok {
			delete(schemas.Provisioners, p)
		}
	}

	return schemas, nil
}

// exact matches only
func mapFilter[V any](schema map[string]V, name string) map[string]V {
	if name == "" {
		return schema
	}

	ret := make(map[string]V)
	if gotSchema, ok := schema[name]; ok {
		ret[name] = gotSchema
	}
	return ret
}

const metadataSchemaCommandHelp = `
Usage: terraform [global options] metadata schema -json

  Prints out a json representation of provider and provisioner schemas.
`

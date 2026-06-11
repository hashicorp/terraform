// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// ProvidersCommand is a Command implementation that prints out information
// about the providers used in the current configuration/state.
type ProvidersSchemaCommand struct {
	Meta
}

func (c *ProvidersSchemaCommand) Help() string {
	return providersSchemaCommandHelp
}

func (c *ProvidersSchemaCommand) Synopsis() string {
	return "Show schemas for the providers used in the configuration"
}

func (c *ProvidersSchemaCommand) Run(args []string) int {
	parsedArgs, diags := arguments.ParseProvidersSchema(c.Meta.process(args))
	if diags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	viewType := arguments.ViewJSON // See above; enforced use of JSON output

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

	schemas, moreDiags := lr.Core.Schemas(lr.Config, lr.InputState)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	// Apply any selector filtering after schemas are loaded. With no selectors
	// this is a pass-through that preserves the unfiltered output exactly.
	filtered, emit, filters, filterDiags := filterProviderSchemas(schemas, providersSchemaSelectors(parsedArgs))
	diags = diags.Append(filterDiags)
	if filterDiags.HasErrors() {
		c.showDiagnostics(diags)
		return 1
	}

	jsonSchemas, err := jsonprovider.MarshalWithFilters(filtered, emit, filters)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to marshal provider schemas to json: %s", err))
		return 1
	}
	c.Ui.Output(string(jsonSchemas))

	return 0
}

// selectors holds the normalized, post-parse filter selections for the
// providers schema command. A zero selectors value means "no filtering", which
// must reproduce the unfiltered output byte-for-byte.
type selectors struct {
	// provider is the normalized fully-qualified provider address to keep. It
	// is meaningful only when providerSet is true.
	provider    addrs.Provider
	providerSet bool

	// kind is the canonical schema category to keep. It is meaningful only when
	// kindSet is true.
	kind    arguments.Kind
	kindSet bool

	// typ is the exact, case-sensitive object type key to keep. It is
	// meaningful only when typeSet is true.
	typ     string
	typeSet bool
}

// any reports whether at least one selector dimension was supplied.
func (s selectors) any() bool {
	return s.providerSet || s.kindSet || s.typeSet
}

// emit translates the kind selector into the jsonprovider resource-emission
// directive. The directive only resolves the resource vs. resource-identity
// collision; every other category is governed by which maps survive pruning.
func (s selectors) emit() jsonprovider.ResourceEmit {
	if !s.kindSet {
		return jsonprovider.EmitAll
	}
	switch s.kind {
	case arguments.KindResource:
		return jsonprovider.ResourceBlockOnly
	case arguments.KindResourceIdentity:
		return jsonprovider.ResourceIdentityOnly
	default:
		return jsonprovider.EmitAll
	}
}

// filtersEcho builds the top-level filters metadata for the response, or nil
// when no selector was supplied. The provider is echoed as its normalized FQN,
// the kind as its canonical label, and the type verbatim.
func (s selectors) filtersEcho() *jsonprovider.Filters {
	if !s.any() {
		return nil
	}
	f := &jsonprovider.Filters{}
	if s.providerSet {
		f.Provider = s.provider.String()
	}
	if s.kindSet {
		f.Kind = string(s.kind)
	}
	if s.typeSet {
		f.Type = s.typ
	}
	return f
}

// providersSchemaSelectors translates parsed CLI arguments into the
// command-internal selectors value consumed by filterProviderSchemas. The
// individual selector dimensions are populated by their respective selector
// implementations.
func providersSchemaSelectors(args *arguments.ProvidersSchema) selectors {
	return selectors{}
}

// filterProviderSchemas prunes the loaded provider schemas according to the
// selectors and returns the filtered schemas, the resource-emission directive
// for the JSON marshaler, the filters echo (nil when no selector was
// supplied), and any diagnostics.
//
// It never mutates the input. Pruned levels are rebuilt as new maps while leaf
// schema values are shared by reference: the loaded ProviderSchema maps are the
// same instances held in the process-global providers.SchemaCache, so an
// in-place delete would corrupt the cache for every other consumer in the
// process (see proposals/provider-subcommand-filtering/design_decisions.md #9).
func filterProviderSchemas(schemas *terraform.Schemas, sel selectors) (*terraform.Schemas, jsonprovider.ResourceEmit, *jsonprovider.Filters, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	emit := sel.emit()
	filters := sel.filtersEcho()

	// No selectors supplied: return the loaded schemas unchanged so unfiltered
	// output stays byte-for-byte identical.
	if !sel.any() {
		return schemas, emit, nil, diags
	}

	// Selector-specific pruning (-provider, -kind, -type) is layered in by the
	// individual selector implementations. Until then the schemas pass through
	// unchanged while the directive and filters echo are threaded to the
	// marshaler.
	return schemas, emit, filters, diags
}

const providersSchemaCommandHelp = `
Usage: terraform [global options] providers schema -json

  Prints out a json representation of the schemas for all providers used
  in the current configuration.

Options:

  -var 'foo=bar'      Set a value for one of the input variables in the root
                      module of the configuration. Use this option more than
                      once to set more than one variable.

  -var-file=filename  Load variable values from the given file, in addition
                      to the default files terraform.tfvars and *.auto.tfvars.
                      Use this option more than once to include more than one
                      variables file.
`

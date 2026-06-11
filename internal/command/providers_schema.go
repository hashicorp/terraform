// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/providers"
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
// command-internal selectors value consumed by filterProviderSchemas.
func providersSchemaSelectors(args *arguments.ProvidersSchema) selectors {
	return selectors{
		provider:    args.Provider,
		providerSet: args.ProviderSet,
		kind:        args.Kind,
		kindSet:     args.KindSet,
		typ:         args.Type,
		typeSet:     args.TypeSet,
	}
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

	// Narrow to the selected provider (if any), checking existence among the
	// loaded providers. A -provider that parsed but isn't loaded is an error
	// that lists the loaded providers; -kind/-type no-match is empty success
	// (see proposals/provider-subcommand-filtering/design_decisions.md #4, #12).
	candidates := schemas.Providers
	if sel.providerSet {
		ps, ok := schemas.Providers[sel.provider]
		if !ok {
			diags = diags.Append(providerNotLoadedDiag(sel.provider, schemas))
			return schemas, emit, filters, diags
		}
		candidates = map[addrs.Provider]providers.ProviderSchema{sel.provider: ps}
	}

	// Prune each candidate provider by -kind/-type, dropping providers that
	// have no selected output.
	out := make(map[addrs.Provider]providers.ProviderSchema, len(candidates))
	for addr, ps := range candidates {
		if pruned, keep := pruneProviderSchema(ps, sel); keep {
			out[addr] = pruned
		}
	}

	return &terraform.Schemas{
		Providers:    out,
		Provisioners: schemas.Provisioners,
	}, emit, filters, diags
}

// pruneProviderSchema returns a filtered copy of a single provider's schema
// according to the -kind/-type selectors, and reports whether the result has
// any selected output (providers with none are dropped by the caller).
//
// New maps are allocated for every pruned category; leaf schema values are
// shared by reference and never mutated.
func pruneProviderSchema(ps providers.ProviderSchema, sel selectors) (providers.ProviderSchema, bool) {
	// With neither -kind nor -type, keep the entire provider schema. This is
	// the -provider-only path.
	if !sel.kindSet && !sel.typeSet {
		return ps, true
	}

	// wants reports whether the given kind should contribute output. With -kind
	// omitted (wildcard) every map-backed category participates; the
	// non-map-backed provider config is never searched by -type, so it only
	// contributes when -kind=provider is explicitly set.
	wants := func(k arguments.Kind) bool {
		if sel.kindSet {
			return sel.kind == k
		}
		return k.IsMapBacked()
	}

	var out providers.ProviderSchema

	if wants(arguments.KindProvider) {
		out.Provider = ps.Provider
	}

	// resource and resource-identity are both derived from ResourceTypes. With
	// a wildcard kind we keep the type-matched entries and let EmitAll fan out
	// to both resource_schemas and resource_identity_schemas. With an explicit
	// resource-identity selection we keep only entries that have an identity
	// schema (and ResourceIdentityOnly renders identity-only).
	switch {
	case sel.kindSet && sel.kind == arguments.KindResourceIdentity:
		out.ResourceTypes = selectByType(identityResourceTypes(ps.ResourceTypes), sel.typ, sel.typeSet)
	case sel.kindSet && sel.kind == arguments.KindResource:
		out.ResourceTypes = selectByType(ps.ResourceTypes, sel.typ, sel.typeSet)
	case !sel.kindSet:
		out.ResourceTypes = selectByType(ps.ResourceTypes, sel.typ, sel.typeSet)
	}

	if wants(arguments.KindDataSource) {
		out.DataSources = selectByType(ps.DataSources, sel.typ, sel.typeSet)
	}
	if wants(arguments.KindEphemeralResource) {
		out.EphemeralResourceTypes = selectByType(ps.EphemeralResourceTypes, sel.typ, sel.typeSet)
	}
	if wants(arguments.KindListResource) {
		out.ListResourceTypes = selectByType(ps.ListResourceTypes, sel.typ, sel.typeSet)
	}
	if wants(arguments.KindFunction) {
		out.Functions = selectByType(ps.Functions, sel.typ, sel.typeSet)
	}
	if wants(arguments.KindAction) {
		out.Actions = selectByType(ps.Actions, sel.typ, sel.typeSet)
	}
	if wants(arguments.KindStateStore) {
		out.StateStores = selectByType(ps.StateStores, sel.typ, sel.typeSet)
	}

	return out, providerSchemaHasContent(out)
}

// selectByType returns a new map containing the entries of src selected by the
// -type filter. With typeSet false, all entries are copied; with typeSet true,
// only the exact, case-sensitive key match (if any) is kept. It returns nil
// when nothing is selected so the category drops out via omitempty after
// marshaling. The source map is never mutated; leaf values are shared.
func selectByType[V any](src map[string]V, typ string, typeSet bool) map[string]V {
	if len(src) == 0 {
		return nil
	}
	if typeSet {
		if v, ok := src[typ]; ok {
			return map[string]V{typ: v}
		}
		return nil
	}
	out := make(map[string]V, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// identityResourceTypes returns a new map containing only the ResourceTypes
// entries that have an identity schema. It returns nil when none qualify.
func identityResourceTypes(src map[string]providers.Schema) map[string]providers.Schema {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]providers.Schema, len(src))
	for k, v := range src {
		if v.Identity != nil {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// providerSchemaHasContent reports whether a pruned provider schema has any
// content that would be marshaled, so empty providers can be dropped.
func providerSchemaHasContent(ps providers.ProviderSchema) bool {
	return ps.Provider.Body != nil ||
		len(ps.ResourceTypes) > 0 ||
		len(ps.DataSources) > 0 ||
		len(ps.EphemeralResourceTypes) > 0 ||
		len(ps.ListResourceTypes) > 0 ||
		len(ps.Functions) > 0 ||
		len(ps.Actions) > 0 ||
		len(ps.StateStores) > 0
}

// providerNotLoadedDiag builds the actionable error returned when a -provider
// selector parses but is not among the loaded provider schemas. The detail
// lists the loaded providers, sorted, so a consumer can self-correct (see
// proposals/provider-subcommand-filtering/design_decisions.md #12).
func providerNotLoadedDiag(p addrs.Provider, schemas *terraform.Schemas) tfdiags.Diagnostic {
	loaded := make([]string, 0, len(schemas.Providers))
	for addr := range schemas.Providers {
		loaded = append(loaded, addr.String())
	}
	sort.Strings(loaded)

	var detail string
	if len(loaded) == 0 {
		detail = "The current configuration did not load any providers."
	} else {
		detail = fmt.Sprintf(
			"The current configuration loaded these providers: %s.",
			strings.Join(loaded, ", "),
		)
	}

	return tfdiags.Sourceless(
		tfdiags.Error,
		fmt.Sprintf("Provider %s was not found in the loaded provider schemas", p.String()),
		detail,
	)
}

const providersSchemaCommandHelp = `
Usage: terraform [global options] providers schema -json

  Prints out a json representation of the schemas for all providers used
  in the current configuration.

Options:

  -provider=ADDR      Filter the output to a single provider, given as a
                      provider source address such as "aws", "hashicorp/aws",
                      or "registry.terraform.io/hashicorp/aws". The address is
                      normalized to its fully-qualified form in the "filters"
                      echo. It is an error if the named provider is not among
                      the providers loaded for the current configuration.

  -kind=CATEGORY      Filter the output to a single schema category. Valid
                      values are: action, data-source, ephemeral-resource,
                      function, list-resource, provider, resource,
                      resource-identity, state-store. A valid category that
                      selects nothing yields an empty success.

  -type=TYPE          Filter the output to a single object type, matched
                      exactly and case-sensitively (for example
                      "aws_instance"). When -kind is omitted, the type is
                      matched against every object-keyed category, so one type
                      may appear as a resource, a resource identity, and a data
                      source at once. The provider configuration category is
                      never searched, so -type cannot be combined with
                      -kind=provider. A valid type that selects nothing yields
                      an empty success.

  -var 'foo=bar'      Set a value for one of the input variables in the root
                      module of the configuration. Use this option more than
                      once to set more than one variable.

  -var-file=filename  Load variable values from the given file, in addition
                      to the default files terraform.tfvars and *.auto.tfvars.
                      Use this option more than once to include more than one
                      variables file.
`

package config

import (
	"fmt"
	"sort"
	"strings"

	gohcl2 "github.com/hashicorp/hcl2/gohcl"
	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2parse "github.com/hashicorp/hcl2/hclparse"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/zclconf/go-cty/cty"
)

// hcl2Configurable is an implementation of configurable that knows
// how to turn a HCL Body into a *Config object.
type hcl2Configurable struct {
	SourceFilename string
	Body           hcl2.Body
}

// hcl2Loader is a wrapper around a HCL parser that provides a fileLoaderFunc.
type hcl2Loader struct {
	Parser *hcl2parse.Parser
}

// For the moment we'll just have a global loader since we don't have anywhere
// better to stash this.
// TODO: refactor the loader API so that it uses some sort of object we can
// stash the parser inside.
var globalHCL2Loader = newHCL2Loader()

// newHCL2Loader creates a new hcl2Loader containing a new HCL Parser.
//
// HCL parsers retain information about files that are loaded to aid in
// producing diagnostic messages, so all files within a single configuration
// should be loaded with the same parser to ensure the availability of
// full diagnostic information.
func newHCL2Loader() hcl2Loader {
	return hcl2Loader{
		Parser: hcl2parse.NewParser(),
	}
}

// loadFile is a fileLoaderFunc that knows how to read a HCL2 file and turn it
// into a hcl2Configurable.
func (l hcl2Loader) loadFile(filename string) (configurable, []string, error) {
	var f *hcl2.File
	var diags hcl2.Diagnostics
	if strings.HasSuffix(filename, ".json") {
		f, diags = l.Parser.ParseJSONFile(filename)
	} else {
		f, diags = l.Parser.ParseHCLFile(filename)
	}
	if diags.HasErrors() {
		// Return diagnostics as an error; callers may type-assert this to
		// recover the original diagnostics, if it doesn't end up wrapped
		// in another error.
		return nil, nil, diags
	}

	return &hcl2Configurable{
		SourceFilename: filename,
		Body:           f.Body,
	}, nil, nil
}

func (t *hcl2Configurable) Config() (*Config, error) {
	config := &Config{}

	// these structs are used only for the initial shallow decoding; we'll
	// expand this into the main, public-facing config structs afterwards.
	type atlas struct {
		Name    string    `hcl:"name"`
		Include *[]string `hcl:"include"`
		Exclude *[]string `hcl:"exclude"`
	}
	type provider struct {
		Name    string    `hcl:"name,label"`
		Alias   *string   `hcl:"alias,attr"`
		Version *string   `hcl:"version,attr"`
		Config  hcl2.Body `hcl:",remain"`
	}
	type module struct {
		Name    string  `hcl:"name,label"`
		Source  string  `hcl:"source,attr"`
		Version *string `hcl:"version,attr"`
		// FIXME, maps not working
		// Providers *map[string]string `hcl:"providers,attr"`
		Config hcl2.Body `hcl:",remain"`
	}
	type resourceLifecycle struct {
		CreateBeforeDestroy *bool     `hcl:"create_before_destroy,attr"`
		PreventDestroy      *bool     `hcl:"prevent_destroy,attr"`
		IgnoreChanges       *[]string `hcl:"ignore_changes,attr"`
	}
	type connection struct {
		Config hcl2.Body `hcl:",remain"`
	}
	type provisioner struct {
		Type string `hcl:"type,label"`

		When      *string `hcl:"when,attr"`
		OnFailure *string `hcl:"on_failure,attr"`

		Connection *connection `hcl:"connection,block"`
		Config     hcl2.Body   `hcl:",remain"`
	}
	type managedResource struct {
		Type string `hcl:"type,label"`
		Name string `hcl:"name,label"`

		CountExpr hcl2.Expression `hcl:"count,attr"`
		Provider  *string         `hcl:"provider,attr"`
		DependsOn *[]string       `hcl:"depends_on,attr"`

		Lifecycle    *resourceLifecycle `hcl:"lifecycle,block"`
		Provisioners []provisioner      `hcl:"provisioner,block"`
		Connection   *connection        `hcl:"connection,block"`

		Config hcl2.Body `hcl:",remain"`
	}
	type dataResource struct {
		Type string `hcl:"type,label"`
		Name string `hcl:"name,label"`

		CountExpr hcl2.Expression `hcl:"count,attr"`
		Provider  *string         `hcl:"provider,attr"`
		DependsOn *[]string       `hcl:"depends_on,attr"`

		Config hcl2.Body `hcl:",remain"`
	}
	type variable struct {
		Name string `hcl:"name,label"`

		DeclaredType *string    `hcl:"type,attr"`
		Default      *cty.Value `hcl:"default,attr"`
		Description  *string    `hcl:"description,attr"`
		Sensitive    *bool      `hcl:"sensitive,attr"`
	}
	type output struct {
		Name string `hcl:"name,label"`

		ValueExpr   hcl2.Expression `hcl:"value,attr"`
		DependsOn   *[]string       `hcl:"depends_on,attr"`
		Description *string         `hcl:"description,attr"`
		Sensitive   *bool           `hcl:"sensitive,attr"`
	}
	type locals struct {
		Definitions hcl2.Attributes `hcl:",remain"`
	}
	type backend struct {
		Type   string    `hcl:"type,label"`
		Config hcl2.Body `hcl:",remain"`
	}
	type terraform struct {
		RequiredVersion *string  `hcl:"required_version,attr"`
		Backend         *backend `hcl:"backend,block"`
	}
	type topLevel struct {
		Atlas     *atlas            `hcl:"atlas,block"`
		Datas     []dataResource    `hcl:"data,block"`
		Modules   []module          `hcl:"module,block"`
		Outputs   []output          `hcl:"output,block"`
		Providers []provider        `hcl:"provider,block"`
		Resources []managedResource `hcl:"resource,block"`
		Terraform *terraform        `hcl:"terraform,block"`
		Variables []variable        `hcl:"variable,block"`
		Locals    []*locals         `hcl:"locals,block"`
	}

	var raw topLevel
	diags := gohcl2.DecodeBody(t.Body, nil, &raw)
	if diags.HasErrors() {
		// Do some minimal decoding to see if we can at least get the
		// required Terraform version, which might help explain why we
		// couldn't parse the rest.
		if raw.Terraform != nil && raw.Terraform.RequiredVersion != nil {
			config.Terraform = &Terraform{
				RequiredVersion: *raw.Terraform.RequiredVersion,
			}
		}

		// We return the diags as an implementation of error, which the
		// caller than then type-assert if desired to recover the individual
		// diagnostics.
		// FIXME: The current API gives us no way to return warnings in the
		// absense of any errors.
		return config, diags
	}

	if raw.Terraform != nil {
		var reqdVersion string
		var backend *Backend

		if raw.Terraform.RequiredVersion != nil {
			reqdVersion = *raw.Terraform.RequiredVersion
		}
		if raw.Terraform.Backend != nil {
			backend = new(Backend)
			backend.Type = raw.Terraform.Backend.Type

			// We don't permit interpolations or nested blocks inside the
			// backend config, so we can decode the config early here and
			// get direct access to the values, which is important for the
			// config hashing to work as expected.
			var config map[string]string
			configDiags := gohcl2.DecodeBody(raw.Terraform.Backend.Config, nil, &config)
			diags = append(diags, configDiags...)

			raw := make(map[string]interface{}, len(config))
			for k, v := range config {
				raw[k] = v
			}

			var err error
			backend.RawConfig, err = NewRawConfig(raw)
			if err != nil {
				diags = append(diags, &hcl2.Diagnostic{
					Severity: hcl2.DiagError,
					Summary:  "Invalid backend configuration",
					Detail:   fmt.Sprintf("Error in backend configuration: %s", err),
				})
			}
		}

		config.Terraform = &Terraform{
			RequiredVersion: reqdVersion,
			Backend:         backend,
		}
	}

	if raw.Atlas != nil {
		var include, exclude []string
		if raw.Atlas.Include != nil {
			include = *raw.Atlas.Include
		}
		if raw.Atlas.Exclude != nil {
			exclude = *raw.Atlas.Exclude
		}
		config.Atlas = &AtlasConfig{
			Name:    raw.Atlas.Name,
			Include: include,
			Exclude: exclude,
		}
	}

	for _, rawM := range raw.Modules {
		m := &Module{
			Name:      rawM.Name,
			Source:    rawM.Source,
			RawConfig: NewRawConfigHCL2(rawM.Config),
		}

		if rawM.Version != nil {
			m.Version = *rawM.Version
		}

		//if rawM.Providers != nil {
		//    m.Providers = *rawM.Providers
		//}

		config.Modules = append(config.Modules, m)
	}

	for _, rawV := range raw.Variables {
		v := &Variable{
			Name: rawV.Name,
		}
		if rawV.DeclaredType != nil {
			v.DeclaredType = *rawV.DeclaredType
		}
		if rawV.Default != nil {
			v.Default = hcl2shim.ConfigValueFromHCL2(*rawV.Default)
		}
		if rawV.Description != nil {
			v.Description = *rawV.Description
		}

		config.Variables = append(config.Variables, v)
	}

	for _, rawO := range raw.Outputs {
		o := &Output{
			Name: rawO.Name,
		}

		if rawO.Description != nil {
			o.Description = *rawO.Description
		}
		if rawO.DependsOn != nil {
			o.DependsOn = *rawO.DependsOn
		}
		if rawO.Sensitive != nil {
			o.Sensitive = *rawO.Sensitive
		}

		// The result is expected to be a map like map[string]interface{}{"value": something},
		// so we'll fake that with our hcl2shim.SingleAttrBody shim.
		o.RawConfig = NewRawConfigHCL2(hcl2shim.SingleAttrBody{
			Name: "value",
			Expr: rawO.ValueExpr,
		})

		config.Outputs = append(config.Outputs, o)
	}

	for _, rawR := range raw.Resources {
		r := &Resource{
			Mode: ManagedResourceMode,
			Type: rawR.Type,
			Name: rawR.Name,
		}
		if rawR.Lifecycle != nil {
			var l ResourceLifecycle
			if rawR.Lifecycle.CreateBeforeDestroy != nil {
				l.CreateBeforeDestroy = *rawR.Lifecycle.CreateBeforeDestroy
			}
			if rawR.Lifecycle.PreventDestroy != nil {
				l.PreventDestroy = *rawR.Lifecycle.PreventDestroy
			}
			if rawR.Lifecycle.IgnoreChanges != nil {
				l.IgnoreChanges = *rawR.Lifecycle.IgnoreChanges
			}
			r.Lifecycle = l
		}
		if rawR.Provider != nil {
			r.Provider = *rawR.Provider
		}
		if rawR.DependsOn != nil {
			r.DependsOn = *rawR.DependsOn
		}

		var defaultConnInfo *RawConfig
		if rawR.Connection != nil {
			defaultConnInfo = NewRawConfigHCL2(rawR.Connection.Config)
		}

		for _, rawP := range rawR.Provisioners {
			p := &Provisioner{
				Type: rawP.Type,
			}

			switch {
			case rawP.When == nil:
				p.When = ProvisionerWhenCreate
			case *rawP.When == "create":
				p.When = ProvisionerWhenCreate
			case *rawP.When == "destroy":
				p.When = ProvisionerWhenDestroy
			default:
				p.When = ProvisionerWhenInvalid
			}

			switch {
			case rawP.OnFailure == nil:
				p.OnFailure = ProvisionerOnFailureFail
			case *rawP.When == "fail":
				p.OnFailure = ProvisionerOnFailureFail
			case *rawP.When == "continue":
				p.OnFailure = ProvisionerOnFailureContinue
			default:
				p.OnFailure = ProvisionerOnFailureInvalid
			}

			if rawP.Connection != nil {
				p.ConnInfo = NewRawConfigHCL2(rawP.Connection.Config)
			} else {
				p.ConnInfo = defaultConnInfo
			}

			p.RawConfig = NewRawConfigHCL2(rawP.Config)

			r.Provisioners = append(r.Provisioners, p)
		}

		// The old loader records the count expression as a weird RawConfig with
		// a single-element map inside. Since the rest of the world is assuming
		// that, we'll mimic it here.
		{
			countBody := hcl2shim.SingleAttrBody{
				Name: "count",
				Expr: rawR.CountExpr,
			}

			r.RawCount = NewRawConfigHCL2(countBody)
			r.RawCount.Key = "count"
		}

		r.RawConfig = NewRawConfigHCL2(rawR.Config)

		config.Resources = append(config.Resources, r)

	}

	for _, rawR := range raw.Datas {
		r := &Resource{
			Mode: DataResourceMode,
			Type: rawR.Type,
			Name: rawR.Name,
		}

		if rawR.Provider != nil {
			r.Provider = *rawR.Provider
		}
		if rawR.DependsOn != nil {
			r.DependsOn = *rawR.DependsOn
		}

		// The old loader records the count expression as a weird RawConfig with
		// a single-element map inside. Since the rest of the world is assuming
		// that, we'll mimic it here.
		{
			countBody := hcl2shim.SingleAttrBody{
				Name: "count",
				Expr: rawR.CountExpr,
			}

			r.RawCount = NewRawConfigHCL2(countBody)
			r.RawCount.Key = "count"
		}

		r.RawConfig = NewRawConfigHCL2(rawR.Config)

		config.Resources = append(config.Resources, r)
	}

	for _, rawP := range raw.Providers {
		p := &ProviderConfig{
			Name: rawP.Name,
		}

		if rawP.Alias != nil {
			p.Alias = *rawP.Alias
		}
		if rawP.Version != nil {
			p.Version = *rawP.Version
		}

		// The result is expected to be a map like map[string]interface{}{"value": something},
		// so we'll fake that with our hcl2shim.SingleAttrBody shim.
		p.RawConfig = NewRawConfigHCL2(rawP.Config)

		config.ProviderConfigs = append(config.ProviderConfigs, p)
	}

	for _, rawL := range raw.Locals {
		names := make([]string, 0, len(rawL.Definitions))
		for n := range rawL.Definitions {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			attr := rawL.Definitions[n]
			l := &Local{
				Name: n,
				RawConfig: NewRawConfigHCL2(hcl2shim.SingleAttrBody{
					Name: "value",
					Expr: attr.Expr,
				}),
			}
			config.Locals = append(config.Locals, l)
		}
	}

	// FIXME: The current API gives us no way to return warnings in the
	// absense of any errors.
	var err error
	if diags.HasErrors() {
		err = diags
	}

	return config, err
}

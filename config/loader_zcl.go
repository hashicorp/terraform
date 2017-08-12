package config

import (
	"strings"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-zcl/gozcl"
	"github.com/zclconf/go-zcl/zcl"
	"github.com/zclconf/go-zcl/zclparse"
)

// zclConfigurable is an implementation of configurable that knows
// how to turn a zcl Body into a *Config object.
type zclConfigurable struct {
	SourceFilename string
	Body           zcl.Body
}

// zclLoader is a wrapper around a zcl parser that provides a fileLoaderFunc.
type zclLoader struct {
	Parser *zclparse.Parser
}

// For the moment we'll just have a global loader since we don't have anywhere
// better to stash this.
// TODO: refactor the loader API so that it uses some sort of object we can
// stash the parser inside.
var globalZclLoader = newZclLoader()

// newZclLoader creates a new zclLoader containing a new zcl Parser.
//
// zcl parsers retain information about files that are loaded to aid in
// producing diagnostic messages, so all files within a single configuration
// should be loaded with the same parser to ensure the availability of
// full diagnostic information.
func newZclLoader() zclLoader {
	return zclLoader{
		Parser: zclparse.NewParser(),
	}
}

// loadFile is a fileLoaderFunc that knows how to read a zcl
// files and turn it into a zclConfigurable.
func (l zclLoader) loadFile(filename string) (configurable, []string, error) {
	var f *zcl.File
	var diags zcl.Diagnostics
	if strings.HasSuffix(filename, ".json") {
		f, diags = l.Parser.ParseJSONFile(filename)
	} else {
		f, diags = l.Parser.ParseZCLFile(filename)
	}
	if diags.HasErrors() {
		// Return diagnostics as an error; callers may type-assert this to
		// recover the original diagnostics, if it doesn't end up wrapped
		// in another error.
		return nil, nil, diags
	}

	return &zclConfigurable{
		SourceFilename: filename,
		Body:           f.Body,
	}, nil, nil
}

func (t *zclConfigurable) Config() (*Config, error) {
	config := &Config{}

	// these structs are used only for the initial shallow decoding; we'll
	// expand this into the main, public-facing config structs afterwards.
	type atlas struct {
		Name    string    `zcl:"name"`
		Include *[]string `zcl:"include"`
		Exclude *[]string `zcl:"exclude"`
	}
	type module struct {
		Name   string   `zcl:"name,label"`
		Source string   `zcl:"source,attr"`
		Config zcl.Body `zcl:",remain"`
	}
	type provider struct {
		Name    string   `zcl:"name,label"`
		Alias   *string  `zcl:"alias,attr"`
		Version *string  `zcl:"version,attr"`
		Config  zcl.Body `zcl:",remain"`
	}
	type resourceLifecycle struct {
		CreateBeforeDestroy *bool     `zcl:"create_before_destroy,attr"`
		PreventDestroy      *bool     `zcl:"prevent_destroy,attr"`
		IgnoreChanges       *[]string `zcl:"ignore_changes,attr"`
	}
	type connection struct {
		Config zcl.Body `zcl:",remain"`
	}
	type provisioner struct {
		Type string `zcl:"type,label"`

		When      *string `zcl:"when,attr"`
		OnFailure *string `zcl:"on_failure,attr"`

		Connection *connection `zcl:"connection,block"`
		Config     zcl.Body    `zcl:",remain"`
	}
	type resource struct {
		Type string `zcl:"type,label"`
		Name string `zcl:"name,label"`

		CountExpr zcl.Expression `zcl:"count,attr"`
		Provider  *string        `zcl:"provider,attr"`
		DependsOn *[]string      `zcl:"depends_on,attr"`

		Lifecycle    *resourceLifecycle `zcl:"lifecycle,block"`
		Provisioners []provisioner      `zcl:"provisioner,block"`

		Config zcl.Body `zcl:",remain"`
	}
	type variable struct {
		Name string `zcl:"name,label"`

		DeclaredType *string    `zcl:"type,attr"`
		Default      *cty.Value `zcl:"default,attr"`
		Description  *string    `zcl:"description,attr"`
		Sensitive    *bool      `zcl:"sensitive,attr"`
	}
	type output struct {
		Name string `zcl:"name,label"`

		Value       zcl.Expression `zcl:"value,attr"`
		DependsOn   *[]string      `zcl:"depends_on,attr"`
		Description *string        `zcl:"description,attr"`
		Sensitive   *bool          `zcl:"sensitive,attr"`
	}
	type locals struct {
		Definitions zcl.Attributes `zcl:",remain"`
	}
	type backend struct {
		Type   string   `zcl:"type,label"`
		Config zcl.Body `zcl:",remain"`
	}
	type terraform struct {
		RequiredVersion *string  `zcl:"required_version,attr"`
		Backend         *backend `zcl:"backend,block"`
	}
	type topLevel struct {
		Atlas     *atlas     `zcl:"atlas,block"`
		Datas     []resource `zcl:"data,block"`
		Modules   []module   `zcl:"module,block"`
		Outputs   []output   `zcl:"output,block"`
		Providers []provider `zcl:"provider,block"`
		Resources []resource `zcl:"resource,block"`
		Terraform *terraform `zcl:"terraform,block"`
		Variables []variable `zcl:"variable,block"`
	}

	var raw topLevel
	diags := gozcl.DecodeBody(t.Body, nil, &raw)
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

	for _, rawV := range raw.Variables {
		v := &Variable{
			Name: rawV.Name,
		}
		if rawV.DeclaredType != nil {
			v.DeclaredType = *rawV.DeclaredType
		}
		if rawV.Default != nil {
			// TODO: decode this to a raw interface like the rest of Terraform
			// is expecting, using some shared "turn cty value into what
			// Terraform expects" function.
		}
		if rawV.Description != nil {
			v.Description = *rawV.Description
		}

		config.Variables = append(config.Variables, v)
	}

	for _, rawR := range raw.Resources {
		r := &Resource{
			Mode: ManagedResourceMode,
			Type: rawR.Type,
			Name: rawR.Name,
		}
		if rawR.Lifecycle != nil {
			l := &ResourceLifecycle{}
			if rawR.Lifecycle.CreateBeforeDestroy != nil {
				l.CreateBeforeDestroy = *rawR.Lifecycle.CreateBeforeDestroy
			}
			if rawR.Lifecycle.PreventDestroy != nil {
				l.PreventDestroy = *rawR.Lifecycle.PreventDestroy
			}
			if rawR.Lifecycle.IgnoreChanges != nil {
				l.IgnoreChanges = *rawR.Lifecycle.IgnoreChanges
			}
		}

		// TODO: provider, provisioners, depends_on, count, and the config itself

		config.Resources = append(config.Resources, r)

	}

	return config, nil
}

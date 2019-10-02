package projectconfigs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	hcljson "github.com/hashicorp/hcl/v2/json"

	"github.com/hashicorp/terraform/tfdiags"
)

// Config represents a workspace configuration as loaded from a workspace
// configuration file.
//
// A Config object produced by function Load and not subsequently modified has
// had static validation applied to it, but may produce further errors on
// later evaluation when checked against actual root module configurations,
// remote state backends, etc.
type Config struct {
	// ProjectRoot and ConfigFile are the project root directory and the
	// configuration file within that directory respectively.
	ProjectRoot string
	ConfigFile  string

	// Source is the raw source code of the configuration file, for use in
	// rendering diagnostic message snippets.
	Source []byte

	Context    map[string]*ContextValue
	Locals     map[string]*LocalValue
	Workspaces map[string]*Workspace
	Upstreams  map[string]*Upstream
}

func loadConfigDir(rootDir string) (*Config, tfdiags.Diagnostics) {
	// We'll try to normalize the root directory as a relative path, so
	// that it'll be more compact if we show it in the UI and more portable
	// if we save it anywhere. If we don't succeed here then we'll just
	// run with the path as given, and then probably fail in some other
	// way below anyway.
	cwd, err := os.Getwd()
	if err == nil {
		normalRootDir, err := filepath.Rel(cwd, rootDir)
		if err == nil {
			rootDir = normalRootDir
		}
	}

	var diags tfdiags.Diagnostics

	nativeSyntaxFile := filepath.Join(rootDir, ProjectConfigFilenameNative)
	jsonSyntaxFile := filepath.Join(rootDir, ProjectConfigFilenameJSON)
	nativeSrc, nativeErr := ioutil.ReadFile(nativeSyntaxFile)
	jsonSrc, jsonErr := ioutil.ReadFile(jsonSyntaxFile)

	switch {
	case nativeErr == nil && jsonErr == nil:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Multiple project configuration files",
			fmt.Sprintf(
				"Project root directory %q has both %s and %s. Only one project configuration file is allowed per project root.",
				rootDir, ProjectConfigFilenameNative, ProjectConfigFilenameJSON,
			),
		))
		return nil, diags
	case nativeErr != nil && jsonErr != nil:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to read project configuration file",
			fmt.Sprintf(
				"Could not read the configuration file for project directory %s: %s.",
				rootDir, nativeErr, // we'll just use the native one, arbitrarily
			),
		))
		return nil, diags
	case nativeErr == nil:
		f, hclDiags := hclsyntax.ParseConfig(nativeSrc, nativeSyntaxFile, hcl.Pos{Line: 1, Column: 1})
		diags = diags.Append(hclDiags)
		if diags.HasErrors() {
			return &Config{
				ProjectRoot: rootDir,
				ConfigFile:  nativeSyntaxFile,
				Source:      nativeSrc,
			}, diags
		}
		return loadConfig(rootDir, nativeSyntaxFile, nativeSrc, f.Body)
	case jsonErr == nil:
		f, hclDiags := hcljson.Parse(nativeSrc, nativeSyntaxFile)
		diags = diags.Append(hclDiags)
		if diags.HasErrors() {
			return &Config{
				ProjectRoot: rootDir,
				ConfigFile:  jsonSyntaxFile,
				Source:      jsonSrc,
			}, diags
		}
		return loadConfig(rootDir, jsonSyntaxFile, jsonSrc, f.Body)
	default:
		// The above cases are exhaustive.
		panic("unhandled case in error handling of project config loading")
	}
}

func loadConfig(rootDir, filename string, src []byte, body hcl.Body) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	config := &Config{
		ProjectRoot: rootDir,
		ConfigFile:  filename,
		Source:      src,

		Context:    make(map[string]*ContextValue),
		Locals:     make(map[string]*LocalValue),
		Workspaces: make(map[string]*Workspace),
	}

	content, hclDiags := body.Content(rootSchema)
	diags = diags.Append(hclDiags)

	// There are no attributes defined in rootSchema, so content.Attributes
	// is always empty.

	for _, block := range content.Blocks {
		switch block.Type {
		case "context":
			cv, moreDiags := decodeContextBlock(block)

			diags = diags.Append(moreDiags)
			if cv != nil {
				if existing, exists := config.Context[cv.Name]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate context block",
						Detail:   fmt.Sprintf("A context %q block was already declared at %s.", cv.Name, existing.DeclRange.StartString()),
						Subject:  cv.NameRange.ToHCL().Ptr(),
						Context:  cv.DeclRange.ToHCL().Ptr(),
					})
					continue
				}

				config.Context[cv.Name] = cv
			}

		case "locals":
			attrs, moreDiags := block.Body.JustAttributes()
			diags = diags.Append(moreDiags)
			for name, attr := range attrs {
				lv, moreDiags := decodeLocalValueAttr(attr)
				diags = diags.Append(moreDiags)
				if lv != nil {
					if existing, exists := config.Locals[lv.Name]; exists {
						diags = diags.Append(&hcl.Diagnostic{
							Severity: hcl.DiagError,
							Summary:  "Duplicate workspace block",
							Detail:   fmt.Sprintf("A local value named %q was already declared at %s.", lv.Name, existing.SrcRange.StartString()),
							Subject:  attr.NameRange.Ptr(),
							Context:  attr.Range.Ptr(),
						})
						continue
					}
					config.Locals[name] = &LocalValue{
						Name:      name,
						Value:     attr.Expr,
						SrcRange:  tfdiags.SourceRangeFromHCL(attr.Range),
						NameRange: tfdiags.SourceRangeFromHCL(attr.NameRange),
					}
				}
			}

		case "workspace":
			ws, moreDiags := decodeWorkspaceBlock(block)
			diags = diags.Append(moreDiags)
			if ws != nil {
				if existing, exists := config.Workspaces[ws.Name]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate workspace block",
						Detail:   fmt.Sprintf("A workspace %q block was already declared at %s.", ws.Name, existing.DeclRange.StartString()),
						Subject:  ws.NameRange.ToHCL().Ptr(),
						Context:  ws.DeclRange.ToHCL().Ptr(),
					})
					continue
				}

				config.Workspaces[ws.Name] = ws
			}

		case "upstream":
			u, moreDiags := decodeUpstreamBlock(block)
			diags = diags.Append(moreDiags)
			if u != nil {
				if existing, exists := config.Upstreams[u.Name]; exists {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Duplicate \"upstream\" block",
						Detail:   fmt.Sprintf("An upstream %q block was already declared at %s.", u.Name, existing.DeclRange.StartString()),
						Subject:  u.NameRange.ToHCL().Ptr(),
						Context:  u.DeclRange.ToHCL().Ptr(),
					})
					continue
				}

				config.Upstreams[u.Name] = u
			}

		default:
			// No other block types in our schema, so anything else is a bug.
			panic(fmt.Sprintf("unexpected block type %q", block.Type))
		}
	}

	return config, diags
}

var rootSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "context", LabelNames: []string{"name"}},
		{Type: "locals"},
		{Type: "workspace", LabelNames: []string{"name"}},
		{Type: "upstream", LabelNames: []string{"name"}},
	},
}

package tfcomponents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// Config represents the contents of a single ".tfcomponents.hcl" file, having
// been subjected to decoding and simple static validation but not yet
// evaluated to produce a full tree with component groups and individual
// component instances.
type Config struct {
	Components map[string]*Component
	Groups     map[string]*ComponentGroup

	InputVariables map[string]*InputVariable
	LocalValues    map[string]*LocalValue
	OutputValues   map[string]*OutputValue

	Filename string
}

func LoadConfigFile(filename string) (*Config, tfdiags.Diagnostics) {
	src, err := os.ReadFile(filename)
	if err != nil {
		var diags tfdiags.Diagnostics
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't open configuration file",
			fmt.Sprintf("Error while loading %s: %s.", filename, err),
		))
		return nil, diags
	}
	return LoadConfig(filename, src)
}

func LoadConfig(filename string, src []byte) (*Config, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if !(strings.HasSuffix(filename, ".tfcomponents.hcl") || strings.HasSuffix(filename, "/tfcomponents.hcl")) {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid components configuration",
			fmt.Sprintf("Can't use %q as a component group file: filename must have the .tfcomponents.hcl suffix.", filename),
		))
		return nil, diags
	}

	f, hclDiags := hclsyntax.ParseConfig(src, filename, hcl.InitialPos)
	diags = diags.Append(hclDiags)

	ret := &Config{
		Filename: filepath.ToSlash(filepath.Clean(filename)),

		Components:     make(map[string]*Component),
		Groups:         make(map[string]*ComponentGroup),
		InputVariables: make(map[string]*InputVariable),
		LocalValues:    make(map[string]*LocalValue),
		OutputValues:   make(map[string]*OutputValue),
	}

	content, hclDiags := f.Body.Content(rootSchema)
	for _, block := range content.Blocks {
		switch block.Type {
		case "component":
			component, moreDiags := decodeComponentBlock(block)
			diags = diags.Append(moreDiags)
			// TODO: Detect duplicate declarations of the same name
			ret.Components[component.Name] = component
		case "group":
			group, moreDiags := decodeComponentGroupBlock(block)
			diags = diags.Append(moreDiags)
			// TODO: Detect duplicate declarations of the same name
			ret.Groups[group.Name] = group
		case "variable":
			variable, moreDiags := decodeInputVariableBlock(block)
			diags = diags.Append(moreDiags)
			// TODO: Detect duplicate declarations of the same name
			ret.InputVariables[variable.Name] = variable
		case "locals":
			newLocals, moreDiags := decodeLocalValuesBlock(block)
			diags = diags.Append(moreDiags)
			for _, lv := range newLocals {
				// TODO: Detect duplicate declarations of the same name
				ret.LocalValues[lv.Name] = lv
			}
		case "output":
			outputValue, moreDiags := decodeOutputValueBlock(block)
			diags = diags.Append(moreDiags)
			// TODO: Detect duplicate declarations of the same name
			ret.OutputValues[outputValue.Name] = outputValue
		default:
			// If we get here then it's a bug either in our schema or in HCL.
			panic(fmt.Sprintf("unexpected block type %q", block.Type))
		}
	}

	return ret, diags
}

var rootSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{Type: "component", LabelNames: []string{"name"}},
		{Type: "group", LabelNames: []string{"name"}},
		{Type: "variable", LabelNames: []string{"name"}},
		{Type: "locals"},
		{Type: "output", LabelNames: []string{"name"}},
	},
}

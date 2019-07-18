package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/hcl2/hcl/hclsyntax"
	hcljson "github.com/hashicorp/hcl2/hcl/json"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// VarEnvPrefix is the prefix for environment variables that represent values
// for root module input variables.
const VarEnvPrefix = "TF_VAR_"

// collectVariableValues inspects the various places that root module input variable
// values can come from and constructs a map ready to be passed to the
// backend as part of a backend.Operation.
//
// This method returns diagnostics relating to the collection of the values,
// but the values themselves may produce additional diagnostics when finally
// parsed.
func (m *Meta) collectVariableValues() (map[string]backend.UnparsedVariableValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := map[string]backend.UnparsedVariableValue{}

	// First we'll deal with environment variables, since they have the lowest
	// precedence.
	{
		env := os.Environ()
		for _, raw := range env {
			if !strings.HasPrefix(raw, VarEnvPrefix) {
				continue
			}
			raw = raw[len(VarEnvPrefix):] // trim the prefix

			eq := strings.Index(raw, "=")
			if eq == -1 {
				// Seems invalid, so we'll ignore it.
				continue
			}

			name := raw[:eq]
			rawVal := raw[eq+1:]

			ret[name] = unparsedVariableValueString{
				str:        rawVal,
				name:       name,
				sourceType: terraform.ValueFromEnvVar,
			}
		}
	}

	// Next up we have some implicit files that are loaded automatically
	// if they are present. There's the original terraform.tfvars
	// (DefaultVarsFilename) along with the later-added search for all files
	// ending in .auto.tfvars.
	if _, err := os.Stat(DefaultVarsFilename); err == nil {
		moreDiags := m.addVarsFromFile(DefaultVarsFilename, terraform.ValueFromAutoFile, ret)
		diags = diags.Append(moreDiags)
	}
	const defaultVarsFilenameJSON = DefaultVarsFilename + ".json"
	if _, err := os.Stat(defaultVarsFilenameJSON); err == nil {
		moreDiags := m.addVarsFromFile(defaultVarsFilenameJSON, terraform.ValueFromAutoFile, ret)
		diags = diags.Append(moreDiags)
	}
	if infos, err := ioutil.ReadDir("."); err == nil {
		// "infos" is already sorted by name, so we just need to filter it here.
		for _, info := range infos {
			name := info.Name()
			if !isAutoVarFile(name) {
				continue
			}
			moreDiags := m.addVarsFromFile(name, terraform.ValueFromAutoFile, ret)
			diags = diags.Append(moreDiags)
		}
	}

	// Finally we process values given explicitly on the command line, either
	// as individual literal settings or as additional files to read.
	for _, rawFlag := range m.variableArgs.AllItems() {
		switch rawFlag.Name {
		case "-var":
			// Value should be in the form "name=value", where value is a
			// raw string whose interpretation will depend on the variable's
			// parsing mode.
			raw := rawFlag.Value
			eq := strings.Index(raw, "=")
			if eq == -1 {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid -var option",
					fmt.Sprintf("The given -var option %q is not correctly specified. Must be a variable name and value separated by an equals sign, like -var=\"key=value\".", raw),
				))
				continue
			}
			name := raw[:eq]
			rawVal := raw[eq+1:]
			ret[name] = unparsedVariableValueString{
				str:        rawVal,
				name:       name,
				sourceType: terraform.ValueFromCLIArg,
			}

		case "-var-file":
			moreDiags := m.addVarsFromFile(rawFlag.Value, terraform.ValueFromNamedFile, ret)
			diags = diags.Append(moreDiags)

		default:
			// Should never happen; always a bug in the code that built up
			// the contents of m.variableArgs.
			diags = diags.Append(fmt.Errorf("unsupported variable option name %q (this is a bug in Terraform)", rawFlag.Name))
		}
	}

	return ret, diags
}

func (m *Meta) addVarsFromFile(filename string, sourceType terraform.ValueSourceType, to map[string]backend.UnparsedVariableValue) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	src, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read variables file",
				fmt.Sprintf("Given variables file %s does not exist.", filename),
			))
		} else {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Failed to read variables file",
				fmt.Sprintf("Error while reading %s: %s.", filename, err),
			))
		}
		return diags
	}

	loader, err := m.initConfigLoader()
	if err != nil {
		diags = diags.Append(err)
		return diags
	}

	// Record the file source code for snippets in diagnostic messages.
	loader.Parser().ForceFileSource(filename, src)

	var f *hcl.File
	if strings.HasSuffix(filename, ".json") {
		var hclDiags hcl.Diagnostics
		f, hclDiags = hcljson.Parse(src, filename)
		diags = diags.Append(hclDiags)
		if f == nil || f.Body == nil {
			return diags
		}
	} else {
		var hclDiags hcl.Diagnostics
		f, hclDiags = hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
		diags = diags.Append(hclDiags)
		if f == nil || f.Body == nil {
			return diags
		}
	}

	// Before we do our real decode, we'll probe to see if there are any blocks
	// of type "variable" in this body, since it's a common mistake for new
	// users to put variable declarations in tfvars rather than variable value
	// definitions, and otherwise our error message for that case is not so
	// helpful.
	{
		content, _, _ := f.Body.PartialContent(&hcl.BodySchema{
			Blocks: []hcl.BlockHeaderSchema{
				{
					Type:       "variable",
					LabelNames: []string{"name"},
				},
			},
		})
		for _, block := range content.Blocks {
			name := block.Labels[0]
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Variable declaration in .tfvars file",
				Detail:   fmt.Sprintf("A .tfvars file is used to assign values to variables that have already been declared in .tf files, not to declare new variables. To declare variable %q, place this block in one of your .tf files, such as variables.tf.\n\nTo set a value for this variable in %s, use the definition syntax instead:\n    %s = <value>", name, block.TypeRange.Filename, name),
				Subject:  &block.TypeRange,
			})
		}
		if diags.HasErrors() {
			// If we already found problems then JustAttributes below will find
			// the same problems with less-helpful messages, so we'll bail for
			// now to let the user focus on the immediate problem.
			return diags
		}
	}

	attrs, hclDiags := f.Body.JustAttributes()
	diags = diags.Append(hclDiags)

	for name, attr := range attrs {
		to[name] = unparsedVariableValueExpression{
			expr:       attr.Expr,
			sourceType: sourceType,
		}
	}
	return diags
}

// unparsedVariableValueLiteral is a backend.UnparsedVariableValue
// implementation that was actually already parsed (!). This is
// intended to deal with expressions inside "tfvars" files.
type unparsedVariableValueExpression struct {
	expr       hcl.Expression
	sourceType terraform.ValueSourceType
}

func (v unparsedVariableValueExpression) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val, hclDiags := v.expr.Value(nil) // nil because no function calls or variable references are allowed here
	diags = diags.Append(hclDiags)

	rng := tfdiags.SourceRangeFromHCL(v.expr.Range())

	return &terraform.InputValue{
		Value:       val,
		SourceType:  v.sourceType,
		SourceRange: rng,
	}, diags
}

// unparsedVariableValueString is a backend.UnparsedVariableValue
// implementation that parses its value from a string. This can be used
// to deal with values given directly on the command line and via environment
// variables.
type unparsedVariableValueString struct {
	str        string
	name       string
	sourceType terraform.ValueSourceType
}

func (v unparsedVariableValueString) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	val, hclDiags := mode.Parse(v.name, v.str)
	diags = diags.Append(hclDiags)

	return &terraform.InputValue{
		Value:      val,
		SourceType: v.sourceType,
	}, diags
}

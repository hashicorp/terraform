package command

import (
	"strings"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// collectVariables inspects the various places that root module input variable
// values can come from and constructs a map ready to be passed to the
// backend as part of a backend.Operation.
//
// This method does not return an error or diagnostics directly, but one or
// more of the returned unparsed value objects may produce diagnostics when
// finally parsed within the backend code.
func (m *Meta) collectVariables() map[string]backend.UnparsedVariableValue {
	ret := map[string]backend.UnparsedVariableValue{}

	// First we'll deal with environment variables, since they have the lowest
	// precedence.
	{
		env := os.Environ()
		for _, raw := range env {
			if !strings.HasPrefix(raw, terraform.VarEnvPrefix) {
				continue
			}
			raw = raw[len(terraform.VarEnvPrefix):] // trim the prefix

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
}

// unparsedVariableValueExpression is a backend.UnparsedVariableValue
// implementation that evaluates an already-parsed HCL expression. This is
// intended to deal with expressions inside "tfvars" files.
type unparsedVariableValueExpression struct {
	expr       hcl.Expression
	sourceType terraform.ValueSourceType
}

func (v unparsedVariableValueExpression) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val, hclDiags := v.expr.Value(nil) // nil because no function calls or variable references are allowed here
	diags = diags.Append(hclDiags)

	rng := tfdiags.SourceRangeFromHCL(v.expr.SourceRange())

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
		Value:       val,
		SourceType:  v.sourceType,
	}, diags
}

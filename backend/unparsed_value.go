package backend

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// UnparsedVariableValue represents a variable value provided by the caller
// whose parsing must be deferred until configuration is available.
//
// This exists to allow processing of variable-setting arguments (e.g. in the
// command package) to be separated from parsing (in the backend package).
type UnparsedVariableValue interface {
	// ParseVariableValue information in the provided variable configuration
	// to parse (if necessary) and return the variable value encapsulated in
	// the receiver.
	//
	// If error diagnostics are returned, the resulting value may be invalid
	// or incomplete.
	ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics)
}

func ParseVariableValues(vv map[string]UnparsedVariableValue, decls map[string]*configs.Variable) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(terraform.InputValues, len(vv))

	for name, rv := range vv {
		var mode configs.VariableParsingMode
		config, declared := decls[name]
		if declared {
			mode = config.ParsingMode
		} else {
			mode = configs.VariableParseLiteral
		}

		val, valDiags := rv.ParseVariableValue(mode)
		diags = diags.Append(valDiags)
		if valDiags.HasErrors() {
			continue
		}

		if !declared {
			switch val.SourceType {
			case terraform.ValueFromConfig, terraform.ValueFromFile:
				// These source types have source ranges, so we can produce
				// a nice error message with good context.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Value for undeclared variable",
					Detail:   fmt.Sprintf("The root module does not declare a variable named %q. To use this value, add a \"variable\" block to the configuration.", name),
					Subject:  val.SourceRange.ToHCL().Ptr(),
				})
			case terraform.ValueFromEnvVar:
				// We allow and ignore undeclared names for environment
				// variables, because users will often set these globally
				// when they are used across many (but not necessarily all)
				// configurations.
			case terraform.ValueFromCLIArg:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Value for undeclared variable",
					fmt.Sprintf("A variable named %q was assigned on the command line, but the root module does not declare a variable of that name. To use this value, add a \"variable\" block to the configuration.", name),
				))
			default:
				// For all other source types we are more vague, but other situations
				// don't generally crop up at this layer in practice.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Value for undeclared variable",
					fmt.Sprintf("A variable named %q was assigned a value, but the root module does not declare a variable of that name. To use this value, add a \"variable\" block to the configuration.", name),
				))
			}
			continue
		}

		ret[name] = val
	}

	return ret, diags
}

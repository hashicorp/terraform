package backend

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
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

// ParseVariableValues processes a map of unparsed variable values by
// correlating each one with the given variable declarations which should
// be from a root module.
//
// The map of unparsed variable values should include variables from all
// possible root module declarations sources such that it is as complete as
// it can possibly be for the current operation. If any declared variables
// are not included in the map, ParseVariableValues will either substitute
// a configured default value or produce an error.
//
// If this function returns without any errors in the diagnostics, the
// resulting input values map is guaranteed to be valid and ready to pass
// to terraform.NewContext. If the diagnostics contains errors, the returned
// InputValues may be incomplete but will include the subset of variables
// that were successfully processed, allowing for careful analysis of the
// partial result.
func ParseVariableValues(vv map[string]UnparsedVariableValue, decls map[string]*configs.Variable) (terraform.InputValues, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := make(terraform.InputValues, len(vv))

	// Currently we're generating only warnings for undeclared variables
	// defined in files (see below) but we only want to generate a few warnings
	// at a time because existing deployments may have lots of these and
	// the result can therefore be overwhelming.
	seenUndeclaredInFile := 0

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
			case terraform.ValueFromConfig, terraform.ValueFromAutoFile, terraform.ValueFromNamedFile:
				// We allow undeclared names for variable values from files and warn in case
				// users have forgotten a variable {} declaration or have a typo in their var name.
				// Some users will actively ignore this warning because they use a .tfvars file
				// across multiple configurations.
				if seenUndeclaredInFile < 2 {
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Value for undeclared variable",
						fmt.Sprintf("The root module does not declare a variable named %q but a value was found in file %q. If you meant to use this value, add a \"variable\" block to the configuration.\n\nTo silence these warnings, use TF_VAR_... environment variables to provide certain \"global\" settings to all configurations in your organization. To reduce the verbosity of these warnings, use the -compact-warnings option.", name, val.SourceRange.Filename),
					))
				}
				seenUndeclaredInFile++

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

	if seenUndeclaredInFile > 2 {
		extras := seenUndeclaredInFile - 2
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagWarning,
			Summary:  "Values for undeclared variables",
			Detail:   fmt.Sprintf("In addition to the other similar warnings shown, %d other variable(s) defined without being declared.", extras),
		})
	}

	// By this point we should've gathered all of the required root module
	// variables from one of the many possible sources. We'll now populate
	// any we haven't gathered as their defaults and fail if any of the
	// missing ones are required.
	for name, vc := range decls {
		if _, defined := ret[name]; defined {
			continue
		}

		if vc.Required() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "No value for required variable",
				Detail:   fmt.Sprintf("The root module input variable %q is not set, and has no default value. Use a -var or -var-file command line argument to provide a value for this variable.", name),
				Subject:  vc.DeclRange.Ptr(),
			})

			// We'll include a placeholder value anyway, just so that our
			// result is complete for any calling code that wants to cautiously
			// analyze it for diagnostic purposes. Since our diagnostics now
			// includes an error, normal processing will ignore this result.
			ret[name] = &terraform.InputValue{
				Value:       cty.DynamicVal,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(vc.DeclRange),
			}
		} else {
			ret[name] = &terraform.InputValue{
				Value:       vc.Default,
				SourceType:  terraform.ValueFromConfig,
				SourceRange: tfdiags.SourceRangeFromHCL(vc.DeclRange),
			}
		}
	}

	return ret, diags
}

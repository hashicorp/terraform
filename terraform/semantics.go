package terraform

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/dag"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty/convert"
)

// GraphSemanticChecker is the interface that semantic checks across
// the entire Terraform graph implement.
//
// The graph should NOT be modified by the semantic checker.
type GraphSemanticChecker interface {
	Check(*dag.Graph) error
}

// UnorderedSemanticCheckRunner is an implementation of GraphSemanticChecker
// that runs a list of SemanticCheckers against the vertices of the graph
// in no specified order.
type UnorderedSemanticCheckRunner struct {
	Checks []SemanticChecker
}

func (sc *UnorderedSemanticCheckRunner) Check(g *dag.Graph) error {
	var err error
	for _, v := range g.Vertices() {
		for _, check := range sc.Checks {
			if e := check.Check(g, v); e != nil {
				err = multierror.Append(err, e)
			}
		}
	}

	return err
}

// SemanticChecker is the interface that semantic checks across the
// Terraform graph implement. Errors are accumulated. Even after an error
// is returned, child vertices in the graph will still be visited.
//
// The graph should NOT be modified by the semantic checker.
//
// The order in which vertices are visited is left unspecified, so the
// semantic checks should not rely on that.
type SemanticChecker interface {
	Check(*dag.Graph, dag.Vertex) error
}

// checkInputVariables ensures that variable values supplied at the UI conform
// to their corresponding declarations in configuration.
//
// The set of values is considered valid only if the returned diagnostics
// does not contain errors. A valid set of values may still produce warnings,
// which should be returned to the user.
func checkInputVariables(vcs map[string]*configs.Variable, vs InputValues) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for name, vc := range vcs {
		val, isSet := vs[name]
		if !isSet {
			// Always an error, since the caller should already have included
			// default values from the configuration in the values map.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unassigned variable",
				fmt.Sprintf("The input variable %q has not been assigned a value. This is a bug in Terraform; please report it in a GitHub issue.", name),
			))
			continue
		}

		wantType := vc.Type

		// A given value is valid if it can convert to the desired type.
		_, err := convert.Convert(val.Value, wantType)
		if err != nil {
			switch val.SourceType {
			case ValueFromConfig, ValueFromFile:
				// We have source location information for these.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid value for input variable",
					Detail:   fmt.Sprintf("The given value is not valid for variable %q: %s.", name, err),
					Subject:  val.SourceRange.ToHCL().Ptr(),
				})
			case ValueFromEnvVar:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The environment variable TF_VAR_%s does not contain a valid value for variable %q: %s.", name, name, err),
				))
			case ValueFromCLIArg:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The argument -var=\"%s=...\" does not contain a valid value for variable %q: %s.", name, name, err),
				))
			case ValueFromInput:
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The value entered for variable %q is not valid: %s.", name, err),
				))
			default:
				// The above gets us good coverage for the situations users
				// are likely to encounter with their own inputs. The other
				// cases are generally implementation bugs, so we'll just
				// use a generic error for these.
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Invalid value for input variable",
					fmt.Sprintf("The value provided for variable %q is not valid: %s.", name, err),
				))
			}
		}
	}

	// Check for any variables that are assigned without being configured.
	// This is always an implementation error in the caller, because we
	// expect undefined variables to be caught during context construction
	// where there is better context to report it well.
	for name := range vs {
		if _, defined := vcs[name]; !defined {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Value assigned to undeclared variable",
				fmt.Sprintf("A value was assigned to an undeclared input variable %q.", name),
			))
		}
	}

	return diags
}

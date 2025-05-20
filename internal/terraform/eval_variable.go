// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func prepareFinalInputVariableValue(addr addrs.AbsInputVariableInstance, raw *InputValue, cfg *configs.Variable) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	convertTy := cfg.ConstraintType
	log.Printf("[TRACE] prepareFinalInputVariableValue: preparing %s", addr)

	var defaultVal cty.Value
	if cfg.Default != cty.NilVal {
		log.Printf("[TRACE] prepareFinalInputVariableValue: %s has a default value", addr)
		var err error
		defaultVal, err = convert.Convert(cfg.Default, convertTy)
		if err != nil {
			// Validation of the declaration should typically catch this,
			// but we'll check it here too to be robust.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid default value for module argument",
				Detail: fmt.Sprintf(
					"The default value for variable %q is incompatible with its type constraint: %s.",
					cfg.Name, err,
				),
				Subject: &cfg.DeclRange,
			})
			// We'll return a placeholder unknown value to avoid producing
			// redundant downstream errors.
			return cty.UnknownVal(cfg.Type), diags
		}
	}

	var sourceRange tfdiags.SourceRange
	var nonFileSource string
	if raw.HasSourceRange() {
		sourceRange = raw.SourceRange
	} else {
		// If the value came from a place that isn't a file and thus doesn't
		// have its own source range, we'll use the declaration range as
		// our source range and generate some slightly different error
		// messages.
		sourceRange = tfdiags.SourceRangeFromHCL(cfg.DeclRange)
		switch raw.SourceType {
		case ValueFromCLIArg:
			nonFileSource = fmt.Sprintf("set using -var=\"%s=...\"", addr.Variable.Name)
		case ValueFromEnvVar:
			nonFileSource = fmt.Sprintf("set using the TF_VAR_%s environment variable", addr.Variable.Name)
		case ValueFromInput:
			nonFileSource = "set using an interactive prompt"
		default:
			nonFileSource = "set from outside of the configuration"
		}
	}

	given := raw.Value
	if given == cty.NilVal { // The variable wasn't set at all (even to null)
		log.Printf("[TRACE] prepareFinalInputVariableValue: %s has no defined value", addr)
		if cfg.Required() {
			// NOTE: The CLI layer typically checks for itself whether all of
			// the required _root_ module variables are set, which would
			// mask this error with a more specific one that refers to the
			// CLI features for setting such variables. We can get here for
			// child module variables, though.
			log.Printf("[ERROR] prepareFinalInputVariableValue: %s is required but is not set", addr)
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  `Required variable not set`,
				Detail:   fmt.Sprintf(`The variable %q is required, but is not set.`, addr.Variable.Name),
				Subject:  cfg.DeclRange.Ptr(),
			})
			// We'll return a placeholder unknown value to avoid producing
			// redundant downstream errors.
			return cty.UnknownVal(cfg.Type), diags
		}

		given = defaultVal // must be set, because we checked above that the variable isn't required
	}

	// Apply defaults from the variable's type constraint to the converted value,
	// unless the converted value is null. We do not apply defaults to top-level
	// null values, as doing so could prevent assigning null to a nullable
	// variable.
	if cfg.TypeDefaults != nil && !given.IsNull() {
		given = cfg.TypeDefaults.Apply(given)
	}

	val, err := convert.Convert(given, convertTy)
	if err != nil {
		log.Printf("[ERROR] prepareFinalInputVariableValue: %s has unsuitable type\n  got:  %s\n  want: %s", addr, given.Type(), convertTy)
		var detail string
		var subject *hcl.Range
		if nonFileSource != "" {
			detail = fmt.Sprintf(
				"Unsuitable value for %s %s: %s.",
				addr, nonFileSource, err,
			)
			subject = cfg.DeclRange.Ptr()
		} else {
			detail = fmt.Sprintf(
				"The given value is not suitable for %s declared at %s: %s.",
				addr, cfg.DeclRange.String(), err,
			)
			subject = sourceRange.ToHCL().Ptr()

			// In some workflows, the operator running terraform does not have access to the variables
			// themselves. They are for example stored in encrypted files that will be used by the CI toolset
			// and not by the operator directly. In such a case, the failing secret value should not be
			// displayed to the operator
			if cfg.Sensitive {
				detail = fmt.Sprintf(
					"The given value is not suitable for %s, which is sensitive: %s. Invalid value defined at %s.",
					addr, err, sourceRange.ToHCL(),
				)
				subject = cfg.DeclRange.Ptr()
			}
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid value for input variable",
			Detail:   detail,
			Subject:  subject,
		})
		// We'll return a placeholder unknown value to avoid producing
		// redundant downstream errors.
		return cty.UnknownVal(cfg.Type), diags
	}

	// By the time we get here, we know:
	// - val matches the variable's type constraint
	// - val is definitely not cty.NilVal, but might be a null value if the given was already null.
	//
	// That means we just need to handle the case where the value is null,
	// which might mean we need to use the default value, or produce an error.
	//
	// For historical reasons we do this only for a "non-nullable" variable.
	// Nullable variables just appear as null if they were set to null,
	// regardless of any default value.
	if val.IsNull() && !cfg.Nullable {
		log.Printf("[TRACE] prepareFinalInputVariableValue: %s is defined as null", addr)
		if defaultVal != cty.NilVal {
			val = defaultVal
		} else {
			log.Printf("[ERROR] prepareFinalInputVariableValue: %s is non-nullable but set to null, and is required", addr)
			if nonFileSource != "" {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Required variable not set`,
					Detail: fmt.Sprintf(
						"Unsuitable value for %s %s: required variable may not be set to null.",
						addr, nonFileSource,
					),
					Subject: cfg.DeclRange.Ptr(),
				})
			} else {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Required variable not set`,
					Detail: fmt.Sprintf(
						"The given value is not suitable for %s defined at %s: required variable may not be set to null.",
						addr, cfg.DeclRange.String(),
					),
					Subject: sourceRange.ToHCL().Ptr(),
				})
			}
			// Stub out our return value so that the semantic checker doesn't
			// produce redundant downstream errors.
			val = cty.UnknownVal(cfg.Type)
		}
	}

	if cfg.Ephemeral {
		// An ephemeral input variable always has an ephemeral value inside the
		// module, even if the value assigned to it from outside is not. This
		// is a useful simplification so that module authors can be explicit
		// about what guarantees they are intending to make (regardless of
		// current implementation details). Changing the ephemerality of an
		// input variable is a breaking change to a module's API.
		val = val.Mark(marks.Ephemeral)
	} else {
		if marks.Contains(val, marks.Ephemeral) {
			var subject hcl.Range
			if raw.HasSourceRange() {
				subject = raw.SourceRange.ToHCL()
			} else {
				// We shouldn't typically get here for ephemeral values, because
				// all of the source types that can represent expressions that
				// could potentially produce ephemeral values are those which
				// have source locations. This is just here for robustness.
				subject = cfg.DeclRange
			}

			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Ephemeral value not allowed",
				Detail:   "This input variable is not declared as accepting a ephemeral values, so it cannot be set to a result derived from an ephemeral value.",
				Subject:  subject.Ptr(),
			})
		}
	}

	return val, diags
}

// evalVariableValidations ensures that all of the configured custom validations
// for a variable are passing.
//
// This must be used only after any side-effects that make the value of the
// variable available for use in expression evaluation, such as
// EvalModuleCallArgument for variables in descendant modules.
func evalVariableValidations(addr addrs.AbsInputVariableInstance, ctx EvalContext, rules []*configs.CheckRule, valueRng hcl.Range, validateWalk bool) (diags tfdiags.Diagnostics) {
	if len(rules) == 0 {
		log.Printf("[TRACE] evalVariableValidations: no validation rules declared for %s, so skipping", addr)
		return nil
	}
	log.Printf("[TRACE] evalVariableValidations: validating %s", addr)

	checkState := ctx.Checks()
	if !checkState.ConfigHasChecks(addr.ConfigCheckable()) {
		// We have nothing to do if this object doesn't have any checks,
		// but the "rules" slice should agree that we don't.
		if ct := len(rules); ct != 0 {
			panic(fmt.Sprintf("check state says that %s should have no rules, but it has %d", addr, ct))
		}
		return diags
	}

	// We'll build just one evaluation context covering the data needed by
	// all of the rules together, since that'll minimize lock contention
	// on the state, plan, etc.
	scope := ctx.EvaluationScope(nil, nil, EvalDataForNoInstanceKey)
	var refs []*addrs.Reference
	for _, rule := range rules {
		condRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, rule.Condition)
		diags = diags.Append(moreDiags)
		msgRefs, moreDiags := langrefs.ReferencesInExpr(addrs.ParseRef, rule.ErrorMessage)
		diags = diags.Append(moreDiags)
		refs = append(refs, condRefs...)
		refs = append(refs, msgRefs...)
	}
	if diags.HasErrors() {
		// If any of the references were invalid then evaluating the expressions
		// will duplicate those errors, so we'll bail out early.
		return diags
	}
	hclCtx, moreDiags := scope.EvalContext(refs)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return diags
	}

	// HACK: Historically we manually built a very constrained hcl.EvalContext
	// here, which only included the value of the one specific input variable
	// we're validating, since we didn't yet support referring to anything
	// else. That accidentally bypassed our rule that input variables are
	// always unknown during the validate walk, and thus accidentally created
	// a useful behavior of actually checking constant-only values against
	// their validation rules just during "terraform validate", rather than
	// having to run "terraform plan".
	//
	// Although that behavior was accidental, it makes simple validation rules
	// more useful and is protected by compatibility promises, and so we'll
	// fake it here by overwriting the unknown value that scope.EvalContext
	// will have inserted with a possibly-more-known value using the same
	// strategy our special code used to use.
	ourVal := ctx.NamedValues().GetInputVariableValue(addr)
	if ourVal != cty.NilVal {
		// (it would be weird for ourVal to be nil here, but we'll tolerate it
		// because it was scope.EvalContext's responsibility to check for the
		// absent final value, and even if it didn't we'll just get an
		// evaluation error when evaluating the expressions below anyway.)

		// Our goal here is to make sure that a reference to the variable
		// we're checking will evaluate to ourVal, regardless of what else
		// scope.EvalContext might have put in the variables table.
		if hclCtx.Variables == nil {
			hclCtx.Variables = make(map[string]cty.Value)
		}
		if varsVal, ok := hclCtx.Variables["var"]; ok {
			// Unfortunately we need to unpack and repack the object here,
			// because cty values are immutable.
			attrs := varsVal.AsValueMap()
			attrs[addr.Variable.Name] = ourVal
			hclCtx.Variables["var"] = cty.ObjectVal(attrs)
		} else {
			hclCtx.Variables["var"] = cty.ObjectVal(map[string]cty.Value{
				addr.Variable.Name: ourVal,
			})
		}
	}

	for ix, validation := range rules {
		result, ruleDiags := evalVariableValidation(validation, hclCtx, valueRng, addr, ix, validateWalk)
		diags = diags.Append(ruleDiags)

		log.Printf("[TRACE] evalVariableValidations: %s status is now %s", addr, result.Status)
		if result.Status == checks.StatusFail {
			checkState.ReportCheckFailure(addr, addrs.InputValidation, ix, result.FailureMessage)
		} else {
			checkState.ReportCheckResult(addr, addrs.InputValidation, ix, result.Status)
		}
	}

	return diags
}

func evalVariableValidation(validation *configs.CheckRule, hclCtx *hcl.EvalContext, valueRng hcl.Range, addr addrs.AbsInputVariableInstance, ix int, validateWalk bool) (checkResult, tfdiags.Diagnostics) {
	const errInvalidCondition = "Invalid variable validation result"
	const errInvalidValue = "Invalid value for variable"
	var diags tfdiags.Diagnostics

	result, moreDiags := validation.Condition.Value(hclCtx)
	diags = diags.Append(moreDiags)
	errorValue, errorDiags := validation.ErrorMessage.Value(hclCtx)

	// The following error handling is a workaround to preserve backwards
	// compatibility. Due to an implementation quirk, all prior versions of
	// Terraform would treat error messages specified using JSON
	// configuration syntax (.tf.json) as string literals, even if they
	// contained the "${" template expression operator. This behaviour did
	// not match that of HCL configuration syntax, where a template
	// expression would result in a validation error.
	//
	// As a result, users writing or generating JSON configuration syntax
	// may have specified error messages which are invalid template
	// expressions. As we add support for error message expressions, we are
	// unable to perfectly distinguish between these two cases.
	//
	// To ensure that we don't break backwards compatibility, we have the
	// below fallback logic if the error message fails to evaluate. This
	// should only have any effect for JSON configurations. The gohcl
	// DecodeExpression function behaves differently when the source of the
	// expression is a JSON configuration file and a nil context is passed.
	if errorDiags.HasErrors() {
		// Attempt to decode the expression as a string literal. Passing
		// nil as the context forces a JSON syntax string value to be
		// interpreted as a string literal.
		var errorString string
		moreErrorDiags := gohcl.DecodeExpression(validation.ErrorMessage, nil, &errorString)
		if !moreErrorDiags.HasErrors() {
			// Decoding succeeded, meaning that this is a JSON syntax
			// string value. We rewrap that as a cty value to allow later
			// decoding to succeed.
			errorValue = cty.StringVal(errorString)

			// This warning diagnostic explains this odd behaviour, while
			// giving us an escape hatch to change this to a hard failure
			// in some future Terraform 1.x version.
			errorDiags = hcl.Diagnostics{
				&hcl.Diagnostic{
					Severity:    hcl.DiagWarning,
					Summary:     "Validation error message expression is invalid",
					Detail:      fmt.Sprintf("The error message provided could not be evaluated as an expression, so Terraform is interpreting it as a string literal.\n\nIn future versions of Terraform, this will be considered an error. Please file a GitHub issue if this would break your workflow.\n\n%s", errorDiags.Error()),
					Subject:     validation.ErrorMessage.Range().Ptr(),
					Context:     validation.DeclRange.Ptr(),
					Expression:  validation.ErrorMessage,
					EvalContext: hclCtx,
				},
			}
		}

		// We want to either report the original diagnostics if the
		// fallback failed, or the warning generated above if it succeeded.
		diags = diags.Append(errorDiags)
	}

	if diags.HasErrors() {
		log.Printf("[TRACE] evalVariableValidations: %s rule %s check rule evaluation failed: %s", addr, validation.DeclRange, diags.Err().Error())
	}
	if !result.IsKnown() {
		log.Printf("[TRACE] evalVariableValidations: %s rule %s condition value is unknown, so skipping validation for now", addr, validation.DeclRange)

		return checkResult{Status: checks.StatusUnknown}, diags // We'll wait until we've learned more, then.
	}
	if result.IsNull() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      "Validation condition expression must return either true or false, not null.",
			Subject:     validation.Condition.Range().Ptr(),
			Expression:  validation.Condition,
			EvalContext: hclCtx,
		})
		return checkResult{Status: checks.StatusError}, diags
	}
	var err error
	result, err = convert.Convert(result, cty.Bool)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     errInvalidCondition,
			Detail:      fmt.Sprintf("Invalid validation condition result value: %s.", tfdiags.FormatError(err)),
			Subject:     validation.Condition.Range().Ptr(),
			Expression:  validation.Condition,
			EvalContext: hclCtx,
		})
		return checkResult{Status: checks.StatusError}, diags
	}

	// Validation condition may be marked if the input variable is bound to
	// a sensitive value. This is irrelevant to the validation process, so
	// we discard the marks now.
	result, _ = result.Unmark()
	status := checks.StatusForCtyValue(result)

	if status != checks.StatusFail {
		return checkResult{Status: status}, diags
	}

	if !errorValue.IsKnown() {
		if validateWalk {
			log.Printf("[DEBUG] evalVariableValidations: %s rule %s error_message value is unknown, so skipping validation for now", addr, validation.DeclRange)
			return checkResult{Status: checks.StatusUnknown}, diags
		}

		diags = diags.Append(&hcl.Diagnostic{
			Severity:    hcl.DiagError,
			Summary:     "Invalid error message",
			Detail:      "Unsuitable value for error message: expression refers to values that won't be known until the apply phase.",
			Subject:     validation.ErrorMessage.Range().Ptr(),
			Expression:  validation.ErrorMessage,
			EvalContext: hclCtx,
			Extra:       diagnosticCausedByUnknown(true),
		})
		return checkResult{
			Status: checks.StatusError,
		}, diags
	}

	var errorMessage string
	if !errorDiags.HasErrors() && !errorValue.IsNull() {
		var err error
		errorValue, err = convert.Convert(errorValue, cty.String)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity:    hcl.DiagError,
				Summary:     "Invalid error message",
				Detail:      fmt.Sprintf("Unsuitable value for error message: %s.", tfdiags.FormatError(err)),
				Subject:     validation.ErrorMessage.Range().Ptr(),
				Expression:  validation.ErrorMessage,
				EvalContext: hclCtx,
			})
		} else {
			if marks.Has(errorValue, marks.Sensitive) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,

					Summary: "Error message refers to sensitive values",
					Detail: `The error expression used to explain this condition refers to sensitive values. Terraform will not display the resulting message.

You can correct this by removing references to sensitive values, or by carefully using the nonsensitive() function if the expression will not reveal the sensitive data.`,

					Subject:     validation.ErrorMessage.Range().Ptr(),
					Expression:  validation.ErrorMessage,
					EvalContext: hclCtx,
				})
				errorMessage = "The error message included a sensitive value, so it will not be displayed."
			} else if marks.Has(errorValue, marks.Ephemeral) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,

					Summary: "Error message refers to ephemeral values",
					Detail: `The error expression used to explain this condition refers to ephemeral values. Terraform will not display the resulting message.

You can correct this by removing references to ephemeral values, or by carefully using the ephemeralasnull() function if the expression will not reveal the ephemeral data.`,

					Subject:     validation.ErrorMessage.Range().Ptr(),
					Expression:  validation.ErrorMessage,
					EvalContext: hclCtx,
				})
				errorMessage = "The error message included a sensitive value, so it will not be displayed."
			} else {
				errorMessage = strings.TrimSpace(errorValue.AsString())
			}
		}
	}
	if errorMessage == "" {
		errorMessage = "Failed to evaluate condition error message."
	}

	diags = diags.Append(&hcl.Diagnostic{
		Severity:    hcl.DiagError,
		Summary:     errInvalidValue,
		Detail:      fmt.Sprintf("%s\n\nThis was checked by the validation rule at %s.", errorMessage, validation.DeclRange.String()),
		Subject:     valueRng.Ptr(),
		Expression:  validation.Condition,
		EvalContext: hclCtx,
		Extra: &addrs.CheckRuleDiagnosticExtra{
			CheckRule: addr.CheckRule(addrs.InputValidation, ix),
		},
	})

	return checkResult{
		Status:         status,
		FailureMessage: errorMessage,
	}, diags
}

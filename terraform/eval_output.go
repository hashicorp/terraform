package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/objchange"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// EvalPlanOutput is an EvalNode implementation that creates a planned change
// for a specific output value and records it in the planned changeset.
type EvalPlanOutput struct {
	Addr         addrs.OutputValue
	Config       *configs.Output
	ForceDestroy bool
}

// Eval implements EvalNode
func (n *EvalPlanOutput) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	addr := n.Addr.Absolute(ctx.Path())

	changes := ctx.Changes()
	state := ctx.State()
	var os *states.OutputValue
	if state != nil {
		os = state.OutputValue(addr)
	}

	before := cty.NullVal(cty.DynamicPseudoType)
	if os != nil {
		before = os.Value
	}
	sensitive := false
	if os != nil {
		before = os.Value
		if os.Sensitive {
			sensitive = true
		}
	}
	if n.Config != nil {
		if n.Config.Sensitive {
			sensitive = true
		}
	}

	var change *plans.OutputChange
	switch {
	case n.Config == nil || n.ForceDestroy:
		change = &plans.OutputChange{
			Addr: addr,
			Change: plans.Change{
				Action: plans.Delete,
				Before: before,
				After:  cty.NullVal(cty.DynamicPseudoType),
			},
			Sensitive: sensitive,
		}
	default:
		after, moreDiags := ctx.EvaluateExpr(n.Config.Expr, cty.DynamicPseudoType, nil)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags.Err()
		}

		eqV := after.Equals(before)
		eq := eqV.IsKnown() && eqV.True()
		var action plans.Action
		switch {
		case eq:
			action = plans.NoOp
		case os == nil:
			action = plans.Create
		default:
			action = plans.Update
		}

		change = &plans.OutputChange{
			Addr: addr,
			Change: plans.Change{
				Action: action,
				Before: before,
				After:  after,
			},
			Sensitive: sensitive,
		}
	}

	changeSrc, err := change.Encode()
	if err != nil {
		return nil, fmt.Errorf("failed to encode plan for %s: %s", addr, err)
	}
	log.Printf("[TRACE] EvalPlanOutput: Recording %s change for %s", changeSrc.Action, addr)
	changes.AppendOutputChange(changeSrc)

	// We'll also record the planned value in the state for consistency,
	// but expression evaluation during the plan walk should always prefer
	// to use the value from the changeset because the state can't represent
	// unknown values.
	state.SetOutputValue(addr, cty.UnknownAsNull(change.After), change.Sensitive)

	return nil, diags.ErrWithWarnings()
}

// EvalApplyOutput is an EvalNode implementation that handles a
// previously-planned change to an output value.
type EvalApplyOutput struct {
	Addr addrs.OutputValue
	Expr hcl.Expression
}

// Eval implements EvalNode
func (n *EvalApplyOutput) Eval(ctx EvalContext) (interface{}, error) {
	var diags tfdiags.Diagnostics
	addr := n.Addr.Absolute(ctx.Path())

	state := ctx.State()
	changes := ctx.Changes()
	if changes == nil {
		// This is unexpected, but we'll tolerate it so that we can run
		// context tests with incomplete mocks.
		log.Printf("[WARN] EvalApplyOutput for %s with no active changeset is no-op", addr)
		return nil, nil
	}

	changeSrc := changes.GetOutputChange(addr)
	if changeSrc == nil || changeSrc.Action == plans.NoOp {
		log.Printf("[WARN] EvalApplyOutput: %s has no change planned", addr)
		return nil, nil
	}

	change, err := changeSrc.Decode()
	if err != nil {
		// This shouldn't happen unless someone tampered with the plan file
		// or there is a bug in the plan file reader/writer.
		return nil, fmt.Errorf("failed to decode plan for %s: %s", addr, err)
	}

	log.Printf("[TRACE] EvalApplyOutput: applying %s change for %s", change.Action, addr)

	switch change.Action {
	case plans.Delete:
		log.Printf("[TRACE] EvalApplyOutput: Removing %s from state (it was deleted)", addr)
		state.RemoveOutputValue(addr)
		changes.RemoveOutputChange(addr) // change is no longer pending
		return nil, diags.ErrWithWarnings()
	default:
		// The "after" value in our planned change might be incomplete if
		// it was constructed from unknown values during planning, so we
		// need to re-evaluate it here to incorporate new values we've
		// learned so far during the apply walk.
		val, moreDiags := ctx.EvaluateExpr(n.Expr, cty.DynamicPseudoType, nil)
		diags = diags.Append(moreDiags)
		if moreDiags.HasErrors() {
			return nil, diags.Err()
		}

		if !val.IsWhollyKnown() {
			// If am output's expression includes a reference to a resource
			// that wasn't created yet and the user prevented that resource
			// from being created by using the -target CLI option then we
			// will end up here, because the not-yet-created resource references
			// will return unknown values.
			//
			// Not updating an output value is likely to have downstream
			// consequences, especially if it's a root module output that is
			// consumed by terraform_remote_state, etc. Therefore we'll produce
			// a warning to make sure the user is aware and also use this
			// opportunity to remind about -target being for exceptional
			// circumstances only.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagWarning,
				Summary:  "Output value incomplete",
				Detail: fmt.Sprintf(
					"The expression for %s has references to resources that are not yet created and were excluded from this operation using the -target option. Terraform has therefore left its value unset.\n\nThe -target operation is provided for exceptional circumstances only and should not be used as part of a routine Terraform workflow.",
					addr,
				),
				Subject: n.Expr.Range().Ptr(),
			})

			log.Printf("[TRACE] EvalApplyOutput: Removing %s from state (it depends on resources not yet created)", addr)
			state.RemoveOutputValue(addr)

			return nil, diags.ErrWithWarnings()
		}

		if errs := objchange.AssertValueCompatible(change.After, val); len(errs) > 0 {
			// This should not happen, but one way it could happen is if
			// a resource in the configuration is written with the legacy
			// SDK and is thus exempted from the usual provider result safety
			// checks that would otherwise have caught this upstream.
			if change.Sensitive {
				// A more general message to avoid disclosing any details about
				// the sensitive value.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Output has inconsistent result during apply",
					Detail: fmt.Sprintf(
						"When updating %s to include new values learned so far during apply, the value changed unexpectedly.\n\nThis usually indicates a bug in a provider whose results are used in this output's value expression.",
						addr,
					),
					Subject: n.Expr.Range().Ptr(),
				})
			} else {
				for _, err := range errs {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Output has inconsistent result during apply",
						Detail: fmt.Sprintf(
							"When updating %s to include new values learned so far during apply, the value changed unexpectedly: %s.\n\nThis usually indicates a bug in a provider whose results are used in this output's value expression.",
							addr, tfdiags.FormatError(err),
						),
						Subject: n.Expr.Range().Ptr(),
					})
				}
			}

			// NOTE: We do still proceed to save the updated value below,
			// in case a subsequent codepath inspects it. This is consistent
			// with how we handle inconsistent results from apply for
			// resources.
		}

		// If we had an unknown value during planning then we would've planned
		// an update, but that unknown can turn out to be null, so we'll
		// handle that as a special case here.
		if val.IsNull() {
			log.Printf("[TRACE] EvalApplyOutput: Removing %s from state (it is now null)", addr)
			state.RemoveOutputValue(addr)
		} else {
			log.Printf("[TRACE] EvalApplyOutput: Saving new value for %s in state", addr)
			state.SetOutputValue(addr, val, change.Sensitive)
		}
		changes.RemoveOutputChange(addr) // change is no longer pending
	}

	return nil, diags.ErrWithWarnings()
}

// EvalRefreshOutput is an EvalNode implementation that re-evaluates a given
// output value and updates its cached value in the state.
//
// This EvalNode is only for walks where no direct (user-initiated) changes to
// output values are expected, such as the refresh walk. The plan and apply
// walks must instead use EvalPlanOutput and EvalApplyOutput respectively.
type EvalRefreshOutput struct {
	Addr      addrs.OutputValue
	Sensitive bool
	Expr      hcl.Expression
}

// Eval implements EvalNode
func (n *EvalRefreshOutput) Eval(ctx EvalContext) (interface{}, error) {
	addr := n.Addr.Absolute(ctx.Path())

	// This has to run before we have a state lock, since evaluation also
	// reads the state
	val, diags := ctx.EvaluateExpr(n.Expr, cty.DynamicPseudoType, nil)
	// We'll handle errors below, after we have loaded the module.

	changes := ctx.Changes()
	state := ctx.State()
	if state == nil {
		return nil, nil
	}

	// handling the interpolation error
	if diags.HasErrors() {
		return nil, diags.Err()
	}

	if val.IsNull() {
		log.Printf("[TRACE] EvalRefreshOutput: Removing %s from state (it is now null)", addr)
		state.RemoveOutputValue(addr)
		if changes != nil {
			changes.RemoveOutputChange(addr)
		}
	} else {
		log.Printf("[TRACE] EvalRefreshOutput: Saving value for %s in state", addr)
		state.SetOutputValue(addr, val, n.Sensitive)

		// If there is a changeset active then we'll also write into that.
		// We use the changeset to represent when the value of an output
		// isn't known yet, because the state can represent only known values.
		// In this case we create only a stub change (forced to always be
		// a create) because it's only used as a placeholder in preparation
		// walks like refreshing; EvalPlanOutput will eventually create a
		// proper plan with an appropriate action and old value.
		if changes != nil {
			changes.RemoveOutputChange(addr)
			if !val.IsWhollyKnown() {
				change := &plans.OutputChange{
					Addr:      addr,
					Sensitive: n.Sensitive,
					Change: plans.Change{
						Action: plans.Create,
						Before: cty.NullVal(cty.DynamicPseudoType),
						After:  val,
					},
				}
				changeSrc, err := change.Encode()
				if err != nil { // Should never happen, since we control the full input
					return nil, fmt.Errorf("failed to encode output plan while refreshing: %s", err)
				}
				changes.AppendOutputChange(changeSrc)
			}
		}
	}

	return nil, diags.ErrWithWarnings()
}

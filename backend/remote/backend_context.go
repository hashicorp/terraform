package remote

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// Context implements backend.Enhanced.
func (b *Remote) Context(op *backend.Operation) (*terraform.Context, statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if op.LockState {
		op.StateLocker = clistate.NewLocker(context.Background(), op.StateLockTimeout, b.CLI, b.cliColorize())
	} else {
		op.StateLocker = clistate.NewNoopLocker()
	}

	// Get the remote workspace name.
	workspace := op.Workspace
	switch {
	case op.Workspace == backend.DefaultStateName:
		workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix):
		workspace = b.prefix + op.Workspace
	}

	// Get the latest state.
	log.Printf("[TRACE] backend/remote: requesting state manager for workspace %q", workspace)
	stateMgr, err := b.StateMgr(op.Workspace)
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading state: {{err}}", err))
		return nil, nil, diags
	}

	log.Printf("[TRACE] backend/remote: requesting state lock for workspace %q", workspace)
	if err := op.StateLocker.Lock(stateMgr, op.Type.String()); err != nil {
		diags = diags.Append(errwrap.Wrapf("Error locking state: {{err}}", err))
		return nil, nil, diags
	}

	defer func() {
		// If we're returning with errors, and thus not producing a valid
		// context, we'll want to avoid leaving the remote workspace locked.
		if diags.HasErrors() {
			err := op.StateLocker.Unlock(nil)
			if err != nil {
				diags = diags.Append(errwrap.Wrapf("Error unlocking state: {{err}}", err))
			}
		}
	}()

	log.Printf("[TRACE] backend/remote: reading remote state for workspace %q", workspace)
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading state: {{err}}", err))
		return nil, nil, diags
	}

	// Initialize our context options
	var opts terraform.ContextOpts
	if v := b.ContextOpts; v != nil {
		opts = *v
	}

	// Copy set options from the operation
	opts.Destroy = op.Destroy
	opts.Targets = op.Targets
	opts.UIInput = op.UIIn

	// Load the latest state. If we enter contextFromPlanFile below then the
	// state snapshot in the plan file must match this, or else it'll return
	// error diagnostics.
	log.Printf("[TRACE] backend/remote: retrieving remote state snapshot for workspace %q", workspace)
	opts.State = stateMgr.State()

	log.Printf("[TRACE] backend/remote: loading configuration for the current working directory")
	config, configDiags := op.ConfigLoader.LoadConfig(op.ConfigDir)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, nil, diags
	}
	opts.Config = config

	log.Printf("[TRACE] backend/remote: retrieving variables from workspace %q", workspace)
	tfeVariables, err := b.client.Variables.List(context.Background(), tfe.VariableListOptions{
		Organization: tfe.String(b.organization),
		Workspace:    tfe.String(workspace),
	})
	if err != nil && err != tfe.ErrResourceNotFound {
		diags = diags.Append(errwrap.Wrapf("Error loading variables: {{err}}", err))
		return nil, nil, diags
	}

	if op.AllowUnsetVariables {
		// If we're not going to use the variables in an operation we'll be
		// more lax about them, stubbing out any unset ones as unknown.
		// This gives us enough information to produce a consistent context,
		// but not enough information to run a real operation (plan, apply, etc)
		opts.Variables = stubAllVariables(op.Variables, config.Module.Variables)
	} else {
		if tfeVariables != nil {
			if op.Variables == nil {
				op.Variables = make(map[string]backend.UnparsedVariableValue)
			}
			for _, v := range tfeVariables.Items {
				op.Variables[v.Key] = &remoteStoredVariableValue{
					definition: v,
				}
			}
		}

		if op.Variables != nil {
			variables, varDiags := backend.ParseVariableValues(op.Variables, config.Module.Variables)
			diags = diags.Append(varDiags)
			if diags.HasErrors() {
				return nil, nil, diags
			}
			opts.Variables = variables
		}
	}

	tfCtx, ctxDiags := terraform.NewContext(&opts)
	diags = diags.Append(ctxDiags)

	log.Printf("[TRACE] backend/remote: finished building terraform.Context")

	return tfCtx, stateMgr, diags
}

func stubAllVariables(vv map[string]backend.UnparsedVariableValue, decls map[string]*configs.Variable) terraform.InputValues {
	ret := make(terraform.InputValues, len(decls))

	for name, cfg := range decls {
		raw, exists := vv[name]
		if !exists {
			ret[name] = &terraform.InputValue{
				Value:      cty.UnknownVal(cfg.Type),
				SourceType: terraform.ValueFromConfig,
			}
			continue
		}

		val, diags := raw.ParseVariableValue(cfg.ParsingMode)
		if diags.HasErrors() {
			ret[name] = &terraform.InputValue{
				Value:      cty.UnknownVal(cfg.Type),
				SourceType: terraform.ValueFromConfig,
			}
			continue
		}
		ret[name] = val
	}

	return ret
}

// remoteStoredVariableValue is a backend.UnparsedVariableValue implementation
// that translates from the go-tfe representation of stored variables into
// the Terraform Core backend representation of variables.
type remoteStoredVariableValue struct {
	definition *tfe.Variable
}

var _ backend.UnparsedVariableValue = (*remoteStoredVariableValue)(nil)

func (v *remoteStoredVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	var val cty.Value

	switch {
	case v.definition.Sensitive:
		// If it's marked as sensitive then it's not available for use in
		// local operations. We'll use an unknown value as a placeholder for
		// it so that operations that don't need it might still work, but
		// we'll also produce a warning about it to add context for any
		// errors that might result here.
		val = cty.DynamicVal
		if !v.definition.HCL {
			// If it's not marked as HCL then we at least know that the
			// value must be a string, so we'll set that in case it allows
			// us to do some more precise type checking.
			val = cty.UnknownVal(cty.String)
		}

		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			fmt.Sprintf("Value for var.%s unavailable", v.definition.Key),
			fmt.Sprintf("The value of variable %q is marked as sensitive in the remote workspace. This operation always runs locally, so the value for that variable is not available.", v.definition.Key),
		))

	case v.definition.HCL:
		// If the variable value is marked as being in HCL syntax, we need to
		// parse it the same way as it would be interpreted in a .tfvars
		// file because that is how it would get passed to Terraform CLI for
		// a remote operation and we want to mimic that result as closely as
		// possible.
		var exprDiags hcl.Diagnostics
		expr, exprDiags := hclsyntax.ParseExpression([]byte(v.definition.Value), "<remote workspace>", hcl.Pos{Line: 1, Column: 1})
		if expr != nil {
			var moreDiags hcl.Diagnostics
			val, moreDiags = expr.Value(nil)
			exprDiags = append(exprDiags, moreDiags...)
		} else {
			// We'll have already put some errors in exprDiags above, so we'll
			// just stub out the value here.
			val = cty.DynamicVal
		}

		// We don't have sufficient context to return decent error messages
		// for syntax errors in the remote values, so we'll just return a
		// generic message instead for now.
		// (More complete error messages will still result from true remote
		// operations, because they'll run on the remote system where we've
		// materialized the values into a tfvars file we can report from.)
		if exprDiags.HasErrors() {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				fmt.Sprintf("Invalid expression for var.%s", v.definition.Key),
				fmt.Sprintf("The value of variable %q is marked in the remote workspace as being specified in HCL syntax, but the given value is not valid HCL. Stored variable values must be valid literal expressions and may not contain references to other variables or calls to functions.", v.definition.Key),
			))
		}

	default:
		// A variable value _not_ marked as HCL is always be a string, given
		// literally.
		val = cty.StringVal(v.definition.Value)
	}

	return &terraform.InputValue{
		Value: val,

		// We mark these as "from input" with the rationale that entering
		// variable values into the Terraform Cloud or Enterprise UI is,
		// roughly speaking, a similar idea to entering variable values at
		// the interactive CLI prompts. It's not a perfect correspondance,
		// but it's closer than the other options.
		SourceType: terraform.ValueFromInput,
	}, diags
}

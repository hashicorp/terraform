package local

import (
	"context"
	"errors"

	"github.com/hashicorp/errwrap"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// backend.Local implementation.
func (b *Local) Context(op *backend.Operation) (*terraform.Context, state.State, tfdiags.Diagnostics) {
	// Make sure the type is invalid. We use this as a way to know not
	// to ask for input/validate.
	op.Type = backend.OperationTypeInvalid

	if op.LockState {
		op.StateLocker = clistate.NewLocker(context.Background(), op.StateLockTimeout, b.CLI, b.Colorize())
	} else {
		op.StateLocker = clistate.NewNoopLocker()
	}

	return b.context(op)
}

func (b *Local) context(op *backend.Operation) (*terraform.Context, state.State, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the state.
	s, err := b.State(op.Workspace)
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading state: {{err}}", err))
		return nil, nil, diags
	}

	if err := op.StateLocker.Lock(s, op.Type.String()); err != nil {
		diags = diags.Append(errwrap.Wrapf("Error locking state: {{err}}", err))
		return nil, nil, diags
	}

	if err := s.RefreshState(); err != nil {
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
	if op.Variables != nil {
		opts.Variables = op.Variables
	}

	// FIXME: Configuration is temporarily stubbed out here to artificially
	// create a stopping point in our work to switch to the new config loader.
	// This means no backend-provided Terraform operations will actually work.
	// This will be addressed in a subsequent commit.
	opts.Module = nil

	// Load our state
	// By the time we get here, the backend creation code in "command" took
	// care of making s.State() return a state compatible with our plan,
	// if any, so we can safely pass this value in both the plan context
	// and new context cases below.
	opts.State = s.State()

	// Build the context
	var tfCtx *terraform.Context
	if op.Plan != nil {
		tfCtx, err = op.Plan.Context(&opts)
	} else {
		tfCtx, err = terraform.NewContext(&opts)
	}

	// any errors resolving plugins returns this
	if rpe, ok := err.(*terraform.ResourceProviderError); ok {
		b.pluginInitRequired(rpe)
		// we wrote the full UI error here, so return a generic error for flow
		// control in the command.
		diags = diags.Append(errors.New("Can't satisfy plugin requirements"))
		return nil, nil, diags
	}

	if err != nil {
		diags = diags.Append(err)
		return nil, nil, diags
	}

	// If we have an operation, then we automatically do the input/validate
	// here since every option requires this.
	if op.Type != backend.OperationTypeInvalid {
		// If input asking is enabled, then do that
		if op.Plan == nil && b.OpInput {
			mode := terraform.InputModeProvider
			mode |= terraform.InputModeVar
			mode |= terraform.InputModeVarUnset

			if err := tfCtx.Input(mode); err != nil {
				diags = diags.Append(errwrap.Wrapf("Error asking for user input: {{err}}", err))
				return nil, nil, diags
			}
		}

		// If validation is enabled, validate
		if b.OpValidation {
			validateDiags := tfCtx.Validate()
			diags = diags.Append(validateDiags)
		}
	}

	return tfCtx, s, diags
}

const validateWarnHeader = `
There are warnings related to your configuration. If no errors occurred,
Terraform will continue despite these warnings. It is a good idea to resolve
these warnings in the near future.

Warnings:
`

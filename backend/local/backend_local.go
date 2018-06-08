package local

import (
	"context"
	"errors"
	"log"

	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/command/format"

	"github.com/hashicorp/terraform/tfdiags"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// backend.Local implementation.
func (b *Local) Context(op *backend.Operation) (*terraform.Context, state.State, error) {
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

func (b *Local) context(op *backend.Operation) (*terraform.Context, state.State, error) {
	// Get the state.
	s, err := b.State(op.Workspace)
	if err != nil {
		return nil, nil, errwrap.Wrapf("Error loading state: {{err}}", err)
	}

	if err := op.StateLocker.Lock(s, op.Type.String()); err != nil {
		return nil, nil, errwrap.Wrapf("Error locking state: {{err}}", err)
	}

	if err := s.RefreshState(); err != nil {
		return nil, nil, errwrap.Wrapf("Error loading state: {{err}}", err)
	}

	// Initialize our context options
	var opts terraform.ContextOpts
	if v := b.ContextOpts; v != nil {
		opts = *v
	}

	// Copy set options from the operation
	opts.Destroy = op.Destroy
	opts.Module = op.Module
	opts.Targets = op.Targets
	opts.UIInput = op.UIIn
	if op.Variables != nil {
		opts.Variables = op.Variables
	}

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
		return nil, nil, errors.New("error satisfying plugin requirements")
	}

	if err != nil {
		return nil, nil, err
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
				return nil, nil, errwrap.Wrapf("Error asking for user input: {{err}}", err)
			}
		}

		// If validation is enabled, validate
		if b.OpValidation {
			diags := tfCtx.Validate()
			if len(diags) > 0 {
				if diags.HasErrors() {
					// If there are warnings _and_ errors then we'll take this
					// path and return them all together in this error.
					return nil, nil, diags.Err()
				}

				// For now we can't propagate warnings any further without
				// printing them directly to the UI, so we'll need to
				// format them here ourselves.
				for _, diag := range diags {
					if diag.Severity() != tfdiags.Warning {
						continue
					}
					if b.CLI != nil {
						b.CLI.Warn(format.Diagnostic(diag, b.Colorize(), 72))
					} else {
						desc := diag.Description()
						log.Printf("[WARN] backend/local: %s", desc.Summary)
					}
				}

				// Make a newline before continuing
				b.CLI.Output("")
			}
		}
	}

	return tfCtx, s, nil
}

const validateWarnHeader = `
There are warnings related to your configuration. If no errors occurred,
Terraform will continue despite these warnings. It is a good idea to resolve
these warnings in the near future.

Warnings:
`

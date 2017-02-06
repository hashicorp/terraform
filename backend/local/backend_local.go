package local

import (
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// backend.Local implementation.
func (b *Local) Context(op *backend.Operation) (*terraform.Context, state.State, error) {
	// Make sure the type is invalid. We use this as a way to know not
	// to ask for input/validate.
	op.Type = backend.OperationTypeInvalid

	return b.context(op)
}

func (b *Local) context(op *backend.Operation) (*terraform.Context, state.State, error) {
	// Get the state.
	s, err := b.State()
	if err != nil {
		return nil, nil, errwrap.Wrapf("Error loading state: {{err}}", err)
	}

	if s, ok := s.(state.Locker); op.LockState && ok {
		if err := s.Lock(op.Type.String()); err != nil {
			return nil, nil, errwrap.Wrapf("Error locking state: {{err}}", err)
		}
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
	opts.State = s.State()

	// Build the context
	var tfCtx *terraform.Context
	if op.Plan != nil {
		tfCtx, err = op.Plan.Context(&opts)
	} else {
		tfCtx, err = terraform.NewContext(&opts)
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
			// We ignore warnings here on purpose. We expect users to be listening
			// to the terraform.Hook called after a validation.
			_, es := tfCtx.Validate()
			if len(es) > 0 {
				return nil, nil, multierror.Append(nil, es...)
			}
		}
	}

	return tfCtx, s, nil
}

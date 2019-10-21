package remote

import (
	"context"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	tfe "github.com/hashicorp/go-tfe"
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
				if v.Sensitive {
					v.Value = "<sensitive>"
				}
				op.Variables[v.Key] = &unparsedVariableValue{
					value:  v.Value,
					source: terraform.ValueFromEnvVar,
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

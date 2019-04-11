package remote

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

// Context implements backend.Enhanced.
func (b *Remote) Context(op *backend.Operation) (*terraform.Context, state.State, error) {
	if op.LockState {
		op.StateLocker = clistate.NewLocker(context.Background(), op.StateLockTimeout, b.CLI, b.cliColorize())
	} else {
		op.StateLocker = clistate.NewNoopLocker()
	}

	// Configure the remote workspace name.
	workspace := op.Workspace
	switch {
	case op.Workspace == backend.DefaultStateName:
		workspace = b.workspace
	case b.prefix != "" && !strings.HasPrefix(op.Workspace, b.prefix):
		workspace = b.prefix + op.Workspace
	}

	// Get the latest state.
	log.Printf("[TRACE] backend/remote: requesting state manager for workspace %q", workspace)
	s, err := b.State(op.Workspace)
	if err != nil {
		return nil, nil, errwrap.Wrapf("Error loading state: {{err}}", err)
	}

	log.Printf("[TRACE] backend/remote: requesting state lock for workspace %q", workspace)
	if err := op.StateLocker.Lock(s, op.Type.String()); err != nil {
		return nil, nil, errwrap.Wrapf("Error locking state: {{err}}", err)
	}

	log.Printf("[TRACE] backend/remote: reading remote state for workspace %q", workspace)
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

	// Load the latest state.
	log.Printf("[TRACE] backend/remote: retrieving remote state snapshot for workspace %q", workspace)
	opts.State = s.State()

	log.Printf("[TRACE] backend/remote: retrieving variables from workspace %q", workspace)
	tfeVariables, err := b.client.Variables.List(context.Background(), tfe.VariableListOptions{
		Organization: tfe.String(b.organization),
		Workspace:    tfe.String(workspace),
	})
	if err != nil && err != tfe.ErrResourceNotFound {
		return nil, nil, fmt.Errorf("error loading variables: %v", err)
	}

	if tfeVariables != nil {
		if opts.Variables == nil {
			opts.Variables = make(map[string]interface{})
		}
		for _, v := range tfeVariables.Items {
			if v.Sensitive {
				v.Value = "<sensitive>"
			}
			opts.Variables[v.Key] = v.Value
		}
	}

	tfCtx, err := terraform.NewContext(&opts)

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

	log.Printf("[TRACE] backend/remote: finished building terraform.Context")

	return tfCtx, s, nil
}

func (b *Remote) pluginInitRequired(providerErr *terraform.ResourceProviderError) {
	b.CLI.Output(b.Colorize().Color(fmt.Sprintf(
		strings.TrimSpace(errPluginInit)+"\n",
		providerErr)))
}

// this relies on multierror to format the plugin errors below the copy
const errPluginInit = `
[reset][bold][yellow]Plugin reinitialization required. Please run "terraform init".[reset]
[yellow]Reason: Could not satisfy plugin requirements.

Plugins are external binaries that Terraform uses to access and manipulate
resources. The configuration provided requires plugins which can't be located,
don't satisfy the version constraints, or are otherwise incompatible.

[reset][red]%s

[reset][yellow]Terraform automatically discovers provider requirements from your
configuration, including providers used in child modules. To see the
requirements and constraints from each module, run "terraform providers".
`

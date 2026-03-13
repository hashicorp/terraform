// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"fmt"
	"log"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LocalRun implements backendrun.Local
func (b *Cloud) LocalRun(op *backendrun.Operation) (*backendrun.LocalRun, statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &backendrun.LocalRun{
		PlanOpts: &terraform.PlanOpts{
			Mode:    op.PlanMode,
			Targets: op.Targets,
		},
	}

	op.StateLocker = op.StateLocker.WithContext(context.Background())

	// Get the remote workspace name.
	remoteWorkspaceName := b.getRemoteWorkspaceName(op.Workspace)

	// Get the latest state.
	log.Printf("[TRACE] cloud: requesting state manager for workspace %q", remoteWorkspaceName)
	stateMgr, sDiags := b.StateMgr(op.Workspace)
	if sDiags.HasErrors() {
		diags = diags.Append(fmt.Errorf("error loading state: %w", sDiags.Err()))
		return nil, nil, diags
	}

	log.Printf("[TRACE] cloud: requesting state lock for workspace %q", remoteWorkspaceName)
	if diags := op.StateLocker.Lock(stateMgr, op.Type.String()); diags.HasErrors() {
		return nil, nil, diags
	}

	defer func() {
		// If we're returning with errors, and thus not producing a valid
		// context, we'll want to avoid leaving the remote workspace locked.
		if diags.HasErrors() {
			diags = diags.Append(op.StateLocker.Unlock())
		}
	}()

	log.Printf("[TRACE] cloud: reading remote state for workspace %q", remoteWorkspaceName)
	if err := stateMgr.RefreshState(); err != nil {
		diags = diags.Append(fmt.Errorf("error loading state: %w", err))
		return nil, nil, diags
	}

	// Initialize our context options
	var opts terraform.ContextOpts
	if v := b.ContextOpts; v != nil {
		opts = *v
	}

	// Copy set options from the operation
	opts.UIInput = op.UIIn

	// Load the latest state. If we enter contextFromPlanFile below then the
	// state snapshot in the plan file must match this, or else it'll return
	// error diagnostics.
	log.Printf("[TRACE] cloud: retrieving remote state snapshot for workspace %q", remoteWorkspaceName)
	ret.InputState = stateMgr.State()

	log.Printf("[TRACE] cloud: loading root module for the current working directory")
	rootMod, configDiags := op.ConfigLoader.LoadRootModule(op.ConfigDir)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, nil, diags
	}

	// If we're not going to use the variables in an operation we'll be
	// more lax about them, stubbing out any unset ones as unknown.
	// This gives us enough information to produce a consistent context,
	// but not enough information to run a real operation (plan, apply, etc).
	//
	// However, const variables must always be resolved since they're
	// needed during early configuration loading (e.g. module sources).
	// We fetch backend variables so const vars can be satisfied.
	fetchedVars, fetchDiags := b.FetchVariables(context.Background(), op.Workspace)
	diags = diags.Append(fetchDiags)
	if fetchDiags.HasErrors() {
		return nil, nil, diags
	}
	if len(fetchedVars) > 0 {
		if op.Variables == nil {
			op.Variables = make(map[string]arguments.UnparsedVariableValue)
		}
		for k, v := range fetchedVars {
			if _, ok := op.Variables[k]; !ok {
				op.Variables[k] = v
			}
		}
	}
	var variables terraform.InputValues
	var varDiags tfdiags.Diagnostics
	if op.AllowUnsetVariables {
		variables, varDiags = backendrun.ParseConstVariableValues(op.Variables, rootMod.Variables)
	} else {
		variables, varDiags = backendrun.ParseVariableValues(op.Variables, rootMod.Variables)
	}
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		return nil, nil, diags
	}
	ret.PlanOpts.SetVariables = variables

	tfCtx, ctxDiags := terraform.NewContext(&opts)
	diags = diags.Append(ctxDiags)
	ret.Core = tfCtx
	if diags.HasErrors() {
		return nil, nil, diags
	}

	log.Printf("[TRACE] cloud: building configuration for the current working directory")

	config, buildDiags := terraform.BuildConfigWithGraph(
		rootMod,
		op.ConfigLoader.ModuleWalker(),
		ret.PlanOpts.SetVariables,
		configs.MockDataLoaderFunc(op.ConfigLoader.LoadExternalMockData),
	)
	diags = diags.Append(buildDiags)
	if diags.HasErrors() {
		return nil, nil, diags
	}

	ret.Config = config

	log.Printf("[TRACE] cloud: finished building terraform.Context")

	return ret, stateMgr, diags
}

func (b *Cloud) getRemoteWorkspaceName(localWorkspaceName string) string {
	switch {
	case localWorkspaceName == backend.DefaultStateName:
		// The default workspace name is a special case
		return b.WorkspaceMapping.Name
	default:
		return localWorkspaceName
	}
}

func (b *Cloud) getRemoteWorkspace(ctx context.Context, localWorkspaceName string) (*tfe.Workspace, error) {
	remoteWorkspaceName := b.getRemoteWorkspaceName(localWorkspaceName)

	log.Printf("[TRACE] cloud: looking up workspace for %s/%s", b.Organization, remoteWorkspaceName)
	remoteWorkspace, err := b.client.Workspaces.Read(ctx, b.Organization, remoteWorkspaceName)
	if err != nil {
		return nil, err
	}

	return remoteWorkspace, nil
}

func (b *Cloud) getRemoteWorkspaceID(ctx context.Context, localWorkspaceName string) (string, error) {
	remoteWorkspace, err := b.getRemoteWorkspace(ctx, localWorkspaceName)
	if err != nil {
		return "", err
	}

	return remoteWorkspace.ID, nil
}

// FetchVariables implements backendrun.ConstVariableSupplier by retrieving
// Terraform variables from the HCP Terraform or Terraform Enterprise workspace.
func (b *Cloud) FetchVariables(ctx context.Context, workspace string) (map[string]arguments.UnparsedVariableValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	remoteWorkspaceID, err := b.getRemoteWorkspaceID(ctx, workspace)
	if err != nil {
		diags = diags.Append(fmt.Errorf("error finding remote workspace: %w", err))
		return nil, diags
	}

	w, err := b.fetchWorkspace(ctx, b.Organization, workspace)
	if err != nil {
		diags = diags.Append(fmt.Errorf("error loading workspace: %w", err))
		return nil, diags
	}

	if isLocalExecutionMode(w.ExecutionMode) {
		log.Printf("[TRACE] cloud: skipping variable fetch for workspace %s/%s (%s), workspace is in Local Execution mode", b.getRemoteWorkspaceName(workspace), b.Organization, remoteWorkspaceID)
		return nil, nil
	}

	log.Printf("[TRACE] cloud: retrieving variables from workspace %s/%s (%s)", b.getRemoteWorkspaceName(workspace), b.Organization, remoteWorkspaceID)
	tfeVariables, err := b.client.Variables.ListAll(ctx, remoteWorkspaceID, nil)
	if err != nil && err != tfe.ErrResourceNotFound {
		diags = diags.Append(fmt.Errorf("error loading variables: %w", err))
		return nil, diags
	}

	result := make(map[string]arguments.UnparsedVariableValue)
	if tfeVariables != nil {
		for _, v := range tfeVariables.Items {
			if v.Category == tfe.CategoryTerraform {
				result[v.Key] = &remoteStoredVariableValue{
					definition: v,
				}
			}
		}
	}

	return result, nil
}

// remoteStoredVariableValue is a backendrun.UnparsedVariableValue implementation
// that translates from the go-tfe representation of stored variables into
// the Terraform Core backend representation of variables.
type remoteStoredVariableValue struct {
	definition *tfe.Variable
}

var _ arguments.UnparsedVariableValue = (*remoteStoredVariableValue)(nil)

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
		// variable values into the HCP Terraform or Enterprise UI is,
		// roughly speaking, a similar idea to entering variable values at
		// the interactive CLI prompts. It's not a perfect correspondance,
		// but it's closer than the other options.
		SourceType: terraform.ValueFromInput,
	}, diags
}

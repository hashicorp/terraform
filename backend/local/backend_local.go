package local

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/command/clistate"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// backend.Local implementation.
func (b *Local) Context(op *backend.Operation) (*terraform.Context, statemgr.Full, tfdiags.Diagnostics) {
	// Make sure the type is invalid. We use this as a way to know not
	// to ask for input/validate.
	op.Type = backend.OperationTypeInvalid

	if op.LockState {
		op.StateLocker = clistate.NewLocker(context.Background(), op.StateLockTimeout, b.CLI, b.Colorize())
	} else {
		op.StateLocker = clistate.NewNoopLocker()
	}

	ctx, _, stateMgr, diags := b.context(op)
	return ctx, stateMgr, diags
}

func (b *Local) context(op *backend.Operation) (*terraform.Context, *configload.Snapshot, statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Get the latest state.
	log.Printf("[TRACE] backend/local: requesting state manager for workspace %q", op.Workspace)
	s, err := b.StateMgr(op.Workspace)
	if err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading state: {{err}}", err))
		return nil, nil, nil, diags
	}
	log.Printf("[TRACE] backend/local: requesting state lock for workspace %q", op.Workspace)
	if err := op.StateLocker.Lock(s, op.Type.String()); err != nil {
		diags = diags.Append(errwrap.Wrapf("Error locking state: {{err}}", err))
		return nil, nil, nil, diags
	}
	log.Printf("[TRACE] backend/local: reading remote state for workspace %q", op.Workspace)
	if err := s.RefreshState(); err != nil {
		diags = diags.Append(errwrap.Wrapf("Error loading state: {{err}}", err))
		return nil, nil, nil, diags
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
	log.Printf("[TRACE] backend/local: retrieving local state snapshot for workspace %q", op.Workspace)
	opts.State = s.State()

	var tfCtx *terraform.Context
	var ctxDiags tfdiags.Diagnostics
	var configSnap *configload.Snapshot
	if op.PlanFile != nil {
		var stateMeta *statemgr.SnapshotMeta
		// If the statemgr implements our optional PersistentMeta interface then we'll
		// additionally verify that the state snapshot in the plan file has
		// consistent metadata, as an additional safety check.
		if sm, ok := s.(statemgr.PersistentMeta); ok {
			m := sm.StateSnapshotMeta()
			stateMeta = &m
		}
		log.Printf("[TRACE] backend/local: building context from plan file")
		tfCtx, configSnap, ctxDiags = b.contextFromPlanFile(op.PlanFile, opts, stateMeta)
		// Write sources into the cache of the main loader so that they are
		// available if we need to generate diagnostic message snippets.
		op.ConfigLoader.ImportSourcesFromSnapshot(configSnap)
	} else {
		log.Printf("[TRACE] backend/local: building context for current working directory")
		tfCtx, configSnap, ctxDiags = b.contextDirect(op, opts)
	}
	diags = diags.Append(ctxDiags)
	if diags.HasErrors() {
		return nil, nil, nil, diags
	}
	log.Printf("[TRACE] backend/local: finished building terraform.Context")

	// If we have an operation, then we automatically do the input/validate
	// here since every option requires this.
	if op.Type != backend.OperationTypeInvalid {
		// If input asking is enabled, then do that
		if op.PlanFile == nil && b.OpInput {
			mode := terraform.InputModeProvider

			log.Printf("[TRACE] backend/local: requesting interactive input, if necessary")
			inputDiags := tfCtx.Input(mode)
			diags = diags.Append(inputDiags)
			if inputDiags.HasErrors() {
				return nil, nil, nil, diags
			}
		}

		// If validation is enabled, validate
		if b.OpValidation {
			log.Printf("[TRACE] backend/local: running validation operation")
			validateDiags := tfCtx.Validate()
			diags = diags.Append(validateDiags)
		}
	}

	return tfCtx, configSnap, s, diags
}

func (b *Local) contextDirect(op *backend.Operation, opts terraform.ContextOpts) (*terraform.Context, *configload.Snapshot, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Load the configuration using the caller-provided configuration loader.
	config, configSnap, configDiags := op.ConfigLoader.LoadConfigWithSnapshot(op.ConfigDir)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, nil, diags
	}
	opts.Config = config

	// If interactive input is enabled, we might gather some more variable
	// values through interactive prompts.
	// TODO: Need to route the operation context through into here, so that
	// the interactive prompts can be sensitive to its timeouts/etc.
	rawVariables := b.interactiveCollectVariables(context.TODO(), op.Variables, config.Module.Variables, opts.UIInput)

	variables, varDiags := backend.ParseVariableValues(rawVariables, config.Module.Variables)
	diags = diags.Append(varDiags)
	if diags.HasErrors() {
		return nil, nil, diags
	}
	opts.Variables = variables

	tfCtx, ctxDiags := terraform.NewContext(&opts)
	diags = diags.Append(ctxDiags)
	return tfCtx, configSnap, diags
}

func (b *Local) contextFromPlanFile(pf *planfile.Reader, opts terraform.ContextOpts, currentStateMeta *statemgr.SnapshotMeta) (*terraform.Context, *configload.Snapshot, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	const errSummary = "Invalid plan file"

	// A plan file has a snapshot of configuration embedded inside it, which
	// is used instead of whatever configuration might be already present
	// in the filesystem.
	snap, err := pf.ReadConfigSnapshot()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errSummary,
			fmt.Sprintf("Failed to read configuration snapshot from plan file: %s.", err),
		))
		return nil, snap, diags
	}
	loader := configload.NewLoaderFromSnapshot(snap)
	config, configDiags := loader.LoadConfig(snap.Modules[""].Dir)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, snap, diags
	}
	opts.Config = config

	// A plan file also contains a snapshot of the prior state the changes
	// are intended to apply to.
	priorStateFile, err := pf.ReadStateFile()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errSummary,
			fmt.Sprintf("Failed to read prior state snapshot from plan file: %s.", err),
		))
		return nil, snap, diags
	}
	if currentStateMeta != nil {
		// If the caller sets this, we require that the stored prior state
		// has the same metadata, which is an extra safety check that nothing
		// has changed since the plan was created. (All of the "real-world"
		// state manager implementstions support this, but simpler test backends
		// may not.)
		if currentStateMeta.Lineage != "" && priorStateFile.Lineage != "" {
			if priorStateFile.Serial != currentStateMeta.Serial || priorStateFile.Lineage != currentStateMeta.Lineage {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Saved plan is stale",
					"The given plan file can no longer be applied because the state was changed by another operation after the plan was created.",
				))
			}
		}
	}
	// The caller already wrote the "current state" here, but we're overriding
	// it here with the prior state. These two should actually be identical in
	// normal use, particularly if we validated the state meta above, but
	// we do this here anyway to ensure consistent behavior.
	opts.State = priorStateFile.State

	plan, err := pf.ReadPlan()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			errSummary,
			fmt.Sprintf("Failed to read plan from plan file: %s.", err),
		))
		return nil, snap, diags
	}

	variables := terraform.InputValues{}
	for name, dyVal := range plan.VariableValues {
		val, err := dyVal.Decode(cty.DynamicPseudoType)
		if err != nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				errSummary,
				fmt.Sprintf("Invalid value for variable %q recorded in plan file: %s.", name, err),
			))
			continue
		}

		variables[name] = &terraform.InputValue{
			Value:      val,
			SourceType: terraform.ValueFromPlan,
		}
	}
	opts.Variables = variables
	opts.Changes = plan.Changes
	opts.Targets = plan.TargetAddrs
	opts.ProviderSHA256s = plan.ProviderSHA256s

	tfCtx, ctxDiags := terraform.NewContext(&opts)
	diags = diags.Append(ctxDiags)
	return tfCtx, snap, diags
}

// interactiveCollectVariables attempts to complete the given existing
// map of variables by interactively prompting for any variables that are
// declared as required but not yet present.
//
// If interactive input is disabled for this backend instance then this is
// a no-op. If input is enabled but fails for some reason, the resulting
// map will be incomplete. For these reasons, the caller must still validate
// that the result is complete and valid.
//
// This function does not modify the map given in "existing", but may return
// it unchanged if no modifications are required. If modifications are required,
// the result is a new map with all of the elements from "existing" plus
// additional elements as appropriate.
//
// Interactive prompting is a "best effort" thing for first-time user UX and
// not something we expect folks to be relying on for routine use. Terraform
// is primarily a non-interactive tool and so we prefer to report in error
// messages that variables are not set rather than reporting that input failed:
// the primary resolution to missing variables is to provide them by some other
// means.
func (b *Local) interactiveCollectVariables(ctx context.Context, existing map[string]backend.UnparsedVariableValue, vcs map[string]*configs.Variable, uiInput terraform.UIInput) map[string]backend.UnparsedVariableValue {
	var needed []string
	if b.OpInput && uiInput != nil {
		for name, vc := range vcs {
			if !vc.Required() {
				continue // We only prompt for required variables
			}
			if _, exists := existing[name]; !exists {
				needed = append(needed, name)
			}
		}
	} else {
		log.Print("[DEBUG] backend/local: Skipping interactive prompts for variables because input is disabled")
	}
	if len(needed) == 0 {
		return existing
	}

	log.Printf("[DEBUG] backend/local: will prompt for input of unset required variables %s", needed)

	// If we get here then we're planning to prompt for at least one additional
	// variable's value.
	sort.Strings(needed) // prompt in lexical order
	ret := make(map[string]backend.UnparsedVariableValue, len(vcs))
	for k, v := range existing {
		ret[k] = v
	}
	for _, name := range needed {
		vc := vcs[name]
		rawValue, err := uiInput.Input(ctx, &terraform.InputOpts{
			Id:          fmt.Sprintf("var.%s", name),
			Query:       fmt.Sprintf("var.%s", name),
			Description: vc.Description,
		})
		if err != nil {
			// Since interactive prompts are best-effort, we'll just continue
			// here and let subsequent validation report this as a variable
			// not specified.
			log.Printf("[WARN] backend/local: Failed to request user input for variable %q: %s", name, err)
			continue
		}
		ret[name] = unparsedInteractiveVariableValue{Name: name, RawValue: rawValue}
	}
	return ret
}

type unparsedInteractiveVariableValue struct {
	Name, RawValue string
}

var _ backend.UnparsedVariableValue = unparsedInteractiveVariableValue{}

func (v unparsedInteractiveVariableValue) ParseVariableValue(mode configs.VariableParsingMode) (*terraform.InputValue, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	val, valDiags := mode.Parse(v.Name, v.RawValue)
	diags = diags.Append(valDiags)
	if diags.HasErrors() {
		return nil, diags
	}
	return &terraform.InputValue{
		Value:      val,
		SourceType: terraform.ValueFromInput,
	}, diags
}

const validateWarnHeader = `
There are warnings related to your configuration. If no errors occurred,
Terraform will continue despite these warnings. It is a good idea to resolve
these warnings in the near future.

Warnings:
`

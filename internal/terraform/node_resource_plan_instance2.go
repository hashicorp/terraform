package terraform

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/instances"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/moduletest/mocking"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/deferring"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

type PlanResourceManager ResourceStateManager[*NodePlannableResourceInstance]

// ResourceData holds the shared data during execution
type ResourceData struct {
	// inputs
	Addr         addrs.AbsResourceInstance
	Importing    bool
	ImportTarget cty.Value
	SkipPlanning bool
	LightMode    bool

	// these are set during the execution
	InstanceRefreshState *states.ResourceInstanceObject
	Provider             providers.Interface
	ProviderSchema       providers.ProviderSchema
	ResourceSchema       providers.Schema
	Deferred             *providers.Deferred
	CheckRuleSeverity    tfdiags.Severity
	RefreshNeeded        bool
}

func (n *NodePlannableResourceInstance) Execute2(ctx EvalContext, op walkOperation) tfdiags.Diagnostics {
	stateManager := NewResourceStateManager(n)
	steps := []ResourceState[*NodePlannableResourceInstance]{}
	stateManager.AddHook(func(state ResourceState[*NodePlannableResourceInstance], manager *ResourceStateManager[*NodePlannableResourceInstance]) {
		steps = append(steps, state)
	})
	init := &InitializationStep{n.ResourceAddr().Resource.Mode}
	diags := stateManager.Execute(init, ctx)

	// Log the steps taken
	str := strings.Builder{}
	str.WriteString(fmt.Sprintf("Executing %s %s", n.Addr, op))
	str.WriteString(fmt.Sprintf(" in %d steps:", len(steps)))
	for _, step := range steps {
		str.WriteString(fmt.Sprintf(" -> %T", step))
	}
	log.Printf("[TRACE] %s\n", str.String())
	return diags
}

// InitializationStep is the first step in the state machine.
// It initializes the resource data and sets up the provider.
type InitializationStep struct {
	Mode addrs.ResourceMode
}

func (s *InitializationStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Initialize basic data
	data.Addr = node.ResourceInstanceAddr()
	data.Importing = node.importTarget != cty.NilVal && !node.preDestroyRefresh
	data.ImportTarget = node.importTarget
	data.SkipPlanning = node.skipPlanChanges
	data.LightMode = ctx.PlanCtx().LightMode
	data.RefreshNeeded = !node.skipRefresh // by default, refresh is needed, unless asked to skip it. Any step that doesn't need it will set this to false.

	// Determine check rule severity
	data.CheckRuleSeverity = tfdiags.Error
	if node.skipPlanChanges || node.preDestroyRefresh {
		data.CheckRuleSeverity = tfdiags.Warning
	}

	// Set up provider
	provider, providerSchema, err := getProvider(ctx, node.ResolvedProvider)
	diags = diags.Append(err)
	if diags.HasErrors() {
		return nil, diags
	}

	data.Provider = provider
	data.ProviderSchema = providerSchema
	data.ResourceSchema = data.ProviderSchema.SchemaForResourceAddr(node.Addr.Resource.Resource)
	if data.ResourceSchema.Body == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		diags = diags.Append(fmt.Errorf("provider does not support resource type for %q", node.Addr))
		return nil, diags
	}

	// Validate configuration if present
	if node.Config != nil {
		diags = diags.Append(validateSelfRef(data.Addr.Resource, node.Config.Config, providerSchema))
		if diags.HasErrors() {
			return nil, diags
		}
	}

	// Data source planning goes through a different path
	if s.Mode == addrs.DataResourceMode {
		return &PlanDataSourceStep{}, diags
	}

	// Start importing process.
	if data.Importing {
		return &ImportingStep{ImportTarget: node.importTarget}, diags
	}

	// Check if we need to upgrade the schema. If we do, we must
	// refresh the resource instance state to match the new schema.
	upgradeRequired, diags := node.schemaUpgradeRequired(ctx, providerSchema, data.Addr)
	if diags.HasErrors() {
		return nil, diags
	}
	if upgradeRequired {
		data.RefreshNeeded = upgradeRequired
	}

	// Read the resource instance from the state
	data.InstanceRefreshState, diags = node.readResourceInstanceState(ctx, node.ResourceInstanceAddr())
	if diags.HasErrors() {
		return nil, diags
	}
	return &SaveSnapshotStep{}, diags
}

// ImportingStep handles the importing of resources
type ImportingStep struct {
	ImportTarget cty.Value
}

func (s *ImportingStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	addr := node.ResourceInstanceAddr()

	// If the target was already in the state, import target would be nil and we
	// would not have gotten here, but let's double-check.
	if s.ImportTarget == cty.NilVal {
		return nil, diags
	}

	// Happy path: the import target id is known. Let's import it.
	if s.ImportTarget.IsWhollyKnown() {
		return &ProviderImportStep{ImportTarget: s.ImportTarget}, diags
	}

	// Unknown config. Mark as deferred without importing.
	// We can only get here because we allowed unknowns in the
	// import target, a behavior that is only supported when
	// we allow deferrals.
	data.Deferred = &providers.Deferred{
		Reason: providers.DeferredReasonResourceConfigUnknown,
	}

	// Handle config generation
	if node.Config == nil && len(node.generateConfigPath) > 0 {
		// Then we're supposed to be generating configuration for this
		// resource, but we can't because the configuration is unknown.
		//
		// Normally, the next step would just be about
		// planning the known configuration to make sure everything we
		// do know about it is correct, but we can't even do that here.
		// If we attempt to do that, (a) We're going to panic later when it complains
		// about having no configuration, and (b) the rest of the
		// function isn't doing anything as there is no configuration
		// to validate.
		//
		// What we'll do instead is write out the address as being deferred with
		// an entirely unknown value. Therefore we can skip the planning steps
		// and go straight to the post-plan deferral step.
		impliedType := data.ProviderSchema.ResourceTypes[addr.Resource.Resource.Type].Body.ImpliedType()
		return &PostPlanDeferralStep{
			Change: &plans.ResourceInstanceChange{
				Addr:         addr,
				PrevRunAddr:  addr,
				ProviderAddr: node.ResolvedProvider,
				Change: plans.Change{
					Action: plans.NoOp, // assume we'll get the config generation correct.
					Before: cty.NullVal(impliedType),
					After:  cty.UnknownVal(impliedType),
					Importing: &plans.Importing{
						Target: s.ImportTarget,
					},
				},
			},
		}, diags
	}

	// We can go straight to planning the import, since we know it has no
	// state, and thus nothing to refresh.
	return &PlanningStep{}, diags
}

// ProviderImportStep handles the import of resources with the provider.
type ProviderImportStep struct {
	ImportTarget cty.Value
}

func (s *ProviderImportStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	addr := node.ResourceInstanceAddr()
	deferralAllowed := ctx.Deferrals().DeferralAllowed()
	var diags tfdiags.Diagnostics
	absAddr := addr.Resource.Absolute(ctx.Path())
	hookResourceID := HookResourceIdentity{
		Addr:         absAddr,
		ProviderAddr: node.ResolvedProvider.Provider,
	}
	provider := data.Provider

	diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PrePlanImport(hookResourceID, s.ImportTarget)
	}))
	if diags.HasErrors() {
		return nil, diags
	}

	importType := "ID"
	var importValue string

	var resp providers.ImportResourceStateResponse
	if node.override != nil {
		// For overriding resources that are being imported, we cheat a little
		// bit and look ahead at the configuration the user has provided and
		// we'll use that as the basis for the resource we're going to make up
		// that is due to be overridden.

		// Note, we know we have configuration as it's impossible to enable
		// config generation during tests, and the validation that config exists
		// if configuration generation is off has already happened.
		if node.Config == nil {
			// But, just in case we change this at some point in the future,
			// let's add a specific error message here we can test for to
			// document the expectation somewhere. This shouldn't happen in
			// production, so we don't bother with a pretty error.
			diags = diags.Append(fmt.Errorf("override blocks do not support config generation"))
			return nil, diags
		}

		forEach, _, _ := evaluateForEachExpression(node.Config.ForEach, ctx, false)
		keyData := EvalDataForInstanceKey(node.ResourceInstanceAddr().Resource.Key, forEach)
		configVal, _, configDiags := ctx.EvaluateBlock(node.Config.Config, data.ResourceSchema.Body, nil, keyData)
		if configDiags.HasErrors() {
			// We have an overridden resource so we're definitely in a test and
			// the users config is not valid. So give up and just report the
			// problems in the users configuration. Normally, we'd import the
			// resource before giving up but for a test it doesn't matter, the
			// test fails in the same way and the state is just lost anyway.
			//
			// If there were only warnings from the config then we'll duplicate
			// them if we include them (as the config will be loaded again
			// later), so only add the configDiags into the main diags if we
			// found actual errors.
			diags = diags.Append(configDiags)
			return nil, diags
		}
		configVal, _ = configVal.UnmarkDeep()

		// Let's pretend we're reading the value as a data source so we
		// pre-compute values now as if the resource has already been created.
		override, overrideDiags := mocking.ComputedValuesForDataSource(configVal, &mocking.MockedData{
			Value:             node.override.Values,
			Range:             node.override.Range,
			ComputedAsUnknown: false,
		}, data.ResourceSchema.Body)
		resp = providers.ImportResourceStateResponse{
			ImportedResources: []providers.ImportedResource{
				{
					TypeName: addr.Resource.Resource.Type,
					State:    ephemeral.StripWriteOnlyAttributes(override, data.ResourceSchema.Body),
				},
			},
			Diagnostics: overrideDiags.InConfigBody(node.Config.Config, absAddr.String()),
		}
	} else {
		if s.ImportTarget.Type().IsObjectType() {
			// Identity-based import
			resp = provider.ImportResourceState(providers.ImportResourceStateRequest{
				TypeName:           addr.Resource.Resource.Type,
				Identity:           s.ImportTarget,
				ClientCapabilities: ctx.ClientCapabilities(),
			})
			importType = "Identity"
			importValue = tfdiags.ObjectToString(s.ImportTarget)
		} else {
			// ID-based/string import
			resp = provider.ImportResourceState(providers.ImportResourceStateRequest{
				TypeName:           addr.Resource.Resource.Type,
				ID:                 s.ImportTarget.AsString(),
				ClientCapabilities: ctx.ClientCapabilities(),
			})
			importValue = s.ImportTarget.AsString()
		}
	}

	data.Deferred = resp.Deferred
	// If we don't support deferrals, but the provider reports a deferral and does not
	// emit any error level diagnostics, we should emit an error.
	if resp.Deferred != nil && !deferralAllowed && !resp.Diagnostics.HasErrors() {
		diags = diags.Append(deferring.UnexpectedProviderDeferralDiagnostic(node.Addr))
	}
	diags = diags.Append(resp.Diagnostics)
	if diags.HasErrors() {
		return nil, diags
	}

	count := len(resp.ImportedResources)
	switch {
	case count > 1:
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Multiple import states not supported",
			fmt.Sprintf("While attempting to import with %s %s, the provider "+
				"returned multiple resource instance states. This "+
				"is not currently supported.",
				importType, importValue,
			),
		))
	case count == 0:
		// Sanity check against the providers. If the provider defers the response, it may not have been able to return a state, so we'll only error if no deferral was returned.
		if resp.Deferred == nil {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Import returned no resources",
				fmt.Sprintf("While attempting to import with %s %s, the provider"+
					"returned no instance states.",
					importType, importValue,
				),
			))
			return nil, diags
		}

		// If we were deferred, then let's make up a resource to represent the
		// state we're going to import.
		state := providers.ImportedResource{
			TypeName: addr.Resource.Resource.Type,
			State:    cty.NullVal(data.ResourceSchema.Body.ImpliedType()),
		}

		// We skip the read and further validation since we make up the state
		// of the imported resource anyways.
		data.InstanceRefreshState = states.NewResourceInstanceObjectFromIR(state)
		data.RefreshNeeded = false
		return &PlanningStep{}, nil
	}

	return &PostImportStep{
		ImportType:        importType,
		ImportValue:       importValue,
		ImportedResources: resp.ImportedResources,
		HookResourceID:    hookResourceID}, diags
}

type PostImportStep struct {
	ImportType        string
	ImportValue       string
	ImportedResources []providers.ImportedResource
	HookResourceID    HookResourceIdentity
}

func (s *PostImportStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	addr := node.ResourceInstanceAddr()
	deferred := data.Deferred
	var diags tfdiags.Diagnostics
	imported := s.ImportedResources

	absAddr := addr.Resource.Absolute(ctx.Path())
	for _, obj := range imported {
		log.Printf("[TRACE] PostImportStep: import %s %q produced instance object of type %s", absAddr.String(), s.ImportValue, obj.TypeName)
	}

	// We can only call the hooks and validate the imported state if we have
	// actually done the import.
	if deferred == nil {
		// call post-import hook
		diags = diags.Append(ctx.Hook(func(h Hook) (HookAction, error) {
			return h.PostPlanImport(s.HookResourceID, imported)
		}))
	}

	if imported[0].TypeName == "" {
		diags = diags.Append(fmt.Errorf("import of %s didn't set type", node.Addr.String()))
		return nil, diags
	}

	// Providers are supposed to return null values for all write-only attributes
	writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
		"Import returned a non-null value for a write-only attribute",
		func(path cty.Path) string {
			return fmt.Sprintf(
				"While attempting to import with %s %s, the provider %q returned a value for the write-only attribute \"%s%s\". Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
				s.ImportType, s.ImportValue, node.ResolvedProvider, node.Addr, tfdiags.FormatCtyPath(path),
			)
		},
		imported[0].State,
		data.ResourceSchema.Body,
	)
	diags = diags.Append(writeOnlyDiags)

	if writeOnlyDiags.HasErrors() {
		return nil, diags
	}

	importedState := states.NewResourceInstanceObjectFromIR(imported[0])
	if deferred == nil && importedState.Value.IsNull() {
		// It's actually okay for a deferred import to have returned a null.
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Import returned null resource",
			fmt.Sprintf("While attempting to import with %s %s, the provider"+
				"returned an instance with no state.",
				s.ImportType, s.ImportValue,
			),
		))

	}
	data.InstanceRefreshState = importedState
	return &ProviderRefreshStep{}, diags
}

// SaveSnapshotStep saves a snapshot of the resource instance state
// before refreshing the resource.
type SaveSnapshotStep struct{}

func (s *SaveSnapshotStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Only write the state if the change isn't being deferred.
	if data.Deferred == nil {
		// We'll save a snapshot of what we just read from the state into the
		// prevRunState before we do anything else, since this will capture the
		// result of any schema upgrading that readResourceInstanceState just did,
		// but not include any out-of-band changes we might detect in the
		// subsequent provider refresh step.
		diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, prevRunState))
		if diags.HasErrors() {
			return nil, diags
		}
		// Also the refreshState, because that should still reflect schema upgrades
		// even if it doesn't reflect upstream changes.
		diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, refreshState))
		if diags.HasErrors() {
			return nil, diags
		}
	}

	// we may need to detect a change in CreateBeforeDestroy to ensure it's
	// stored when we are not refreshing
	updated := updateCreateBeforeDestroy(node, data.InstanceRefreshState)

	// If we are in light mode, we may not need to refresh the state.
	// If we find out that we have to after planning, the planning step will send us there.
	if !data.RefreshNeeded || data.LightMode {
		if updated {
			// CreateBeforeDestroy must be set correctly in the state which is used
			// to create the apply graph, so if we did not refresh the state make
			// sure we still update any changes to CreateBeforeDestroy.
			diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, refreshState))
			if diags.HasErrors() {
				return nil, diags
			}
		}

		// If we only want to refresh the state, then we can skip the
		// planning phase.
		if data.SkipPlanning {
			return &RefreshOnlyStep{prevInstanceState: data.InstanceRefreshState}, diags
		}

		// Go straight to planning, since we don't need to refresh the state
		return &PlanningStep{}, diags
	}

	return &ProviderRefreshStep{}, diags
}

// ProviderRefreshStep handles refreshing the resource's state
// with the provider.
type ProviderRefreshStep struct{}

func (s *ProviderRefreshStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// This is the state of the resource before we refresh the value in the provider, we need to keep track
	// of this to report this as the before value if the refresh is deferred.
	preRefreshInstanceState := data.InstanceRefreshState

	var refreshWasDeferred bool
	// Perform the refresh
	refreshedState, refreshDeferred, refreshDiags := node.refresh(
		ctx, states.NotDeposed, data.InstanceRefreshState, ctx.Deferrals().DeferralAllowed(),
	)
	diags = diags.Append(refreshDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	data.InstanceRefreshState = refreshedState

	if data.InstanceRefreshState != nil {
		// When refreshing we start by merging the stored dependencies and
		// the configured dependencies. The configured dependencies will be
		// stored to state once the changes are applied. If the plan
		// results in no changes, we will re-write these dependencies
		// below.
		data.InstanceRefreshState.Dependencies = mergeDeps(
			node.Dependencies, data.InstanceRefreshState.Dependencies,
		)
	}

	if data.Deferred == nil && refreshDeferred != nil {
		data.Deferred = refreshDeferred
	}
	refreshWasDeferred = refreshDeferred != nil

	if data.Deferred == nil {
		diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, refreshState))
	}
	if diags.HasErrors() {
		return nil, diags
	}

	// Handle import validation and config generation if needed
	if data.Importing {
		importDiags := s.handleImportValidationAndConfigGen(ctx, node, data, refreshWasDeferred)
		diags = diags.Append(importDiags)
		if diags.HasErrors() {
			return nil, diags
		}
	}

	data.RefreshNeeded = false // we just refreshed, we shouldn't need to refresh again

	// If we only want to refresh the state, then we can skip the
	// planning phase.
	if data.SkipPlanning {
		return &RefreshOnlyStep{prevInstanceState: preRefreshInstanceState}, diags
	}

	return &PlanningStep{}, diags
}

// handleImportValidationAndConfigGen handles the import validation and config generation
// after a resource has been refreshed.
func (s *ProviderRefreshStep) handleImportValidationAndConfigGen(
	ctx EvalContext,
	node *NodePlannableResourceInstance,
	data *ResourceData,
	refreshWasDeferred bool,
) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	// We only need to handle import validation and config generation
	// when we're importing and the import target is wholly known
	if !data.ImportTarget.IsWhollyKnown() {
		return diags
	}

	if !refreshWasDeferred && data.InstanceRefreshState.Value.IsNull() {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Cannot import non-existent remote object",
			fmt.Sprintf(
				"While attempting to import an existing object to %q, "+
					"the provider detected that no object exists with the given id. "+
					"Only pre-existing objects can be imported; check that the id "+
					"is correct and that it is associated with the provider's "+
					"configured region or endpoint, or use \"terraform apply\" to "+
					"create a new remote object for this resource.",
				node.Addr,
			),
		))
		return diags
	}

	// If we're importing and generating config, generate it now. We only
	// generate config if the import isn't being deferred. We should generate
	// the configuration in the plan that the import is actually happening in.
	if data.Deferred == nil && len(node.generateConfigPath) > 0 {
		if node.Config != nil {
			return diags.Append(fmt.Errorf("tried to generate config for %s, but it already exists", node.Addr))
		}

		// Generate the HCL string first, then parse the HCL body from it.
		// First we generate the contents of the resource block for use within
		// the planning node. Then we wrap it in an enclosing resource block to
		// pass into the plan for rendering.
		generatedResource, generatedDiags := node.generateHCLResourceDef(ctx, node.Addr, data.InstanceRefreshState.Value)
		diags = diags.Append(generatedDiags)

		// This wraps the content of the resource block in an enclosing resource block
		// to pass into the plan for rendering.
		node.generatedConfigHCL = generatedResource.String()

		// parse the "file" body as HCL to get the hcl.Body
		synthHCLFile, hclDiags := hclsyntax.ParseConfig(generatedResource.Body, filepath.Base(node.generateConfigPath), hcl.Pos{Byte: 0, Line: 1, Column: 1})
		diags = diags.Append(hclDiags)
		if hclDiags.HasErrors() {
			return diags
		}

		// We have to do a kind of mini parsing of the content here to correctly
		// mark attributes like 'provider' as hiddenode. We only care about the
		// resulting content, so it's remain that gets passed into the resource
		// as the config.
		_, remain, resourceDiags := synthHCLFile.Body.PartialContent(configs.ResourceBlockSchema)
		diags = diags.Append(resourceDiags)
		if resourceDiags.HasErrors() {
			return diags
		}

		node.Config = &configs.Resource{
			Mode:     addrs.ManagedResourceMode,
			Type:     node.Addr.Resource.Resource.Type,
			Name:     node.Addr.Resource.Resource.Name,
			Config:   remain,
			Managed:  &configs.ManagedResource{},
			Provider: node.ResolvedProvider.Provider,
		}
	}

	return diags
}

// RefreshOnlyStep handles the refresh-only planning mode
type RefreshOnlyStep struct {
	// This is the state of the resource before we refresh the value
	prevInstanceState *states.ResourceInstanceObject
}

func (s *RefreshOnlyStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// In refresh-only mode we need to evaluate the for-each expression in
	// order to supply the value to the pre- and post-condition check
	// blocks. This has the unfortunate edge case of a refresh-only plan
	// executing with a for-each map which has the same keys but different
	// values, which could result in a post-condition check relying on that
	// value being inaccurate. Unless we decide to store the value of the
	// for-each expression in state, this is unavoidable.
	forEach, _, _ := evaluateForEachExpression(node.Config.ForEach, ctx, false)
	repeatData := EvalDataForInstanceKey(data.Addr.Resource.Key, forEach)

	// Evaluate preconditions
	checkDiags := evalCheckRules(
		addrs.ResourcePrecondition,
		node.Config.Preconditions,
		ctx, data.Addr, repeatData,
		data.CheckRuleSeverity,
	)
	diags = diags.Append(checkDiags)

	// Even if we don't plan changes, we do still need to at least update
	// the working state to reflect the refresh result. If not, then e.g.
	// any output values refering to this will not react to the drift.
	// (Even if we didn't actually refresh above, this will still save
	// the result of any schema upgrading we did in readResourceInstanceState.)
	diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, workingState))
	if diags.HasErrors() {
		return nil, diags
	}

	// Evaluate postconditions
	checkDiags = evalCheckRules(
		addrs.ResourcePostcondition,
		node.Config.Postconditions,
		ctx, data.Addr, repeatData,
		data.CheckRuleSeverity,
	)
	diags = diags.Append(checkDiags)

	// Report deferral if needed
	if data.Deferred != nil {
		// Make sure we have a valid state before using it
		var beforeValue cty.Value
		if s.prevInstanceState != nil {
			beforeValue = s.prevInstanceState.Value
		} else {
			beforeValue = cty.NullVal(data.InstanceRefreshState.Value.Type())
		}

		ctx.Deferrals().ReportResourceInstanceDeferred(
			data.Addr,
			data.Deferred.Reason,
			&plans.ResourceInstanceChange{
				Addr:         node.Addr,
				PrevRunAddr:  node.Addr,
				ProviderAddr: node.ResolvedProvider,
				Change: plans.Change{
					Action: plans.Read,
					Before: beforeValue,
					After:  data.InstanceRefreshState.Value,
				},
			},
		)
	}

	// no more steps.
	return nil, diags
}

// PlanningStep handles the planning phase
type PlanningStep struct{}

func (s *PlanningStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// Initialize repetition data for replace triggers
	repData := instances.RepetitionData{}
	switch k := data.Addr.Resource.Key.(type) {
	case addrs.IntKey:
		repData.CountIndex = k.Value()
	case addrs.StringKey:
		repData.EachKey = k.Value()
		repData.EachValue = cty.DynamicVal
	}

	// Check for triggered replacements
	diags = diags.Append(node.replaceTriggered(ctx, repData))
	if diags.HasErrors() {
		return nil, diags
	}

	// Plan the changes
	change, instancePlanState, planDeferred, repeatData, planDiags := node.plan(
		ctx, nil, data.InstanceRefreshState, node.ForceCreateBeforeDestroy, node.forceReplace,
	)
	diags = diags.Append(planDiags)
	if diags.HasErrors() {
		// Special case for import with config generation
		// If we are importing and generating a configuration, we need to
		// ensure the change is written out so the configuration can be
		// captured.
		if planDeferred == nil && len(node.generateConfigPath) > 0 {
			// Update our return plan
			change := &plans.ResourceInstanceChange{
				Addr:         node.Addr,
				PrevRunAddr:  node.prevRunAddr(ctx),
				ProviderAddr: node.ResolvedProvider,
				Change: plans.Change{
					// we only need a placeholder, so this will be a NoOp
					Action:          plans.NoOp,
					Before:          data.InstanceRefreshState.Value,
					After:           data.InstanceRefreshState.Value,
					GeneratedConfig: node.generatedConfigHCL,
				},
			}
			diags = diags.Append(node.writeChange(ctx, change, ""))
		}
		return nil, diags
	}

	if data.Deferred == nil && planDeferred != nil {
		data.Deferred = planDeferred
	}

	// Update import metadata if needed
	if data.Importing {
		// There is a subtle difference between the import by identity
		// and the import by ID. When importing by identity, we need to
		// make sure to use the complete identity return by the provider
		// instead of the (potential) incomplete one from the configuration.
		if node.importTarget.Type().IsObjectType() {
			change.Importing = &plans.Importing{Target: data.InstanceRefreshState.Identity}
		} else {
			change.Importing = &plans.Importing{Target: node.importTarget}
		}
	}

	// FIXME: here we update the change to reflect the reason for
	// replacement, but we still overload forceReplace to get the correct
	// change planned.
	if len(node.replaceTriggeredBy) > 0 {
		change.ActionReason = plans.ResourceInstanceReplaceByTriggers
	}

	// Determine if we need to refresh and re-plan
	// In light mode, if we didn't refresh before planning but the provider
	// has indicated that changes are needed, we need to refresh and re-plan to
	// ensure we have the most up-to-date state
	refreshChangedResource := data.RefreshNeeded && change.Action != plans.NoOp && data.LightMode
	if refreshChangedResource {
		// Go back to the refresh step and plan again
		return &ProviderRefreshStep{}, nil
	}

	// FIXME: here we update the change to reflect the reason for
	// replacement, but we still overload forceReplace to get the correct
	// change planned.
	if len(node.replaceTriggeredBy) > 0 {
		change.ActionReason = plans.ResourceInstanceReplaceByTriggers
	}

	return &PostPlanDeferralStep{
		RepeatData: repeatData,
		PlanState:  instancePlanState,
		Change:     change,
	}, diags
}

// PostPlanDeferralStep handles the deferral of changes after planning
type PostPlanDeferralStep struct {
	RepeatData instances.RepetitionData
	PlanState  *states.ResourceInstanceObject
	Change     *plans.ResourceInstanceChange
}

func (s *PostPlanDeferralStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	deferrals := ctx.Deferrals()
	if data.Deferred != nil {
		// Then this resource has been deferred either during the import,
		// refresh or planning stage. We'll report the deferral and
		// store what we could produce in the deferral tracker.
		deferrals.ReportResourceInstanceDeferred(data.Addr, data.Deferred.Reason, s.Change)
		return nil, diags
	}

	// We intentionally write the change before the subsequent checks, because
	// all of the checks below this point are for problems caused by the
	// context surrounding the change, rather than the change itself, and
	// so it's helpful to still include the valid-in-isolation change as
	// part of the plan as additional context in our error output.
	//
	// FIXME: it is currently important that we write resource changes to
	// the plan (n.writeChange) before we write the corresponding state
	// (n.writeResourceInstanceState).
	//
	// This is because the planned resource state will normally have the
	// status of states.ObjectPlanned, which causes later logic to refer to
	// the contents of the plan to retrieve the resource data. Because
	// there is no shared lock between these two data structures, reversing
	// the order of these writes will cause a brief window of inconsistency
	// which can lead to a failed safety check.
	//
	// Future work should adjust these APIs such that it is impossible to
	// update these two data structures incorrectly through any objects
	// reachable via the terraform.EvalContext API.
	if !deferrals.ShouldDeferResourceInstanceChanges(node.Addr, node.Dependencies) {
		// Write the change
		diags = diags.Append(node.writeChange(ctx, s.Change, ""))
		if diags.HasErrors() {
			return nil, diags
		}

		// Update the working state
		diags = diags.Append(node.writeResourceInstanceState(ctx, s.PlanState, workingState))
		if diags.HasErrors() {
			return nil, diags
		}

		// Check for prevent_destroy violations
		diags = diags.Append(node.checkPreventDestroy(s.Change))
		if diags.HasErrors() {
			return nil, diags
		}

		// If this plan resulted in a NoOp, then apply won't have a chance to make
		// any changes to the stored dependencies. Since this is a NoOp we know
		// that the stored dependencies will have no effect during apply, and we can
		// write them out now.
		if s.Change.Action == plans.NoOp && !depsEqual(data.InstanceRefreshState.Dependencies, node.Dependencies) {
			// the refresh state will be the final state for this resource, so
			// finalize the dependencies here if they need to be updated.
			data.InstanceRefreshState.Dependencies = node.Dependencies
			diags = diags.Append(node.writeResourceInstanceState(ctx, data.InstanceRefreshState, refreshState))
			if diags.HasErrors() {
				return nil, diags
			}
		}

		return &CheckingPostconditionsStep{s.RepeatData}, diags
	}

	// If we get here, it means that the deferrals tracker says that
	// that we must defer changes for
	// this resource instance, presumably due to a dependency on an
	// upstream object that was already deferred. Therefore we just
	// report our own deferral (capturing a placeholder value in the
	// deferral tracker) and don't add anything to the plan or
	// working state.
	// In this case, the expression evaluator should use the placeholder
	// value registered here as the value of this resource instance,
	// instead of using the plan.
	deferrals.ReportResourceInstanceDeferred(node.Addr, providers.DeferredReasonDeferredPrereq, s.Change)
	return nil, diags
}

// CheckingPostconditionsStep evaluates postconditions
type CheckingPostconditionsStep struct {
	RepeatData instances.RepetitionData
}

func (s *CheckingPostconditionsStep) Execute(ctx EvalContext, node *NodePlannableResourceInstance, data *ResourceData) (ResourceState[*NodePlannableResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	// Post-conditions might block completion. We intentionally do this
	// _after_ writing the state/diff because we want to check against
	// the result of the operation, and to fail on future operations
	// until the user makes the condition succeed.
	// (Note that some preconditions will end up being skipped during
	// planning, because their conditions depend on values not yet known.)

	// Check postconditions
	checkDiags := evalCheckRules(
		addrs.ResourcePostcondition,
		node.Config.Postconditions,
		ctx, node.ResourceInstanceAddr(), s.RepeatData,
		data.CheckRuleSeverity,
	)
	diags = diags.Append(checkDiags)

	// End of execution
	return nil, diags
}

func updateCreateBeforeDestroy(n *NodePlannableResourceInstance, currentState *states.ResourceInstanceObject) (updated bool) {
	if n.Config != nil && n.Config.Managed != nil && currentState != nil {
		newCBD := n.Config.Managed.CreateBeforeDestroy || n.ForceCreateBeforeDestroy
		updated = currentState.CreateBeforeDestroy != newCBD
		currentState.CreateBeforeDestroy = newCBD
		return updated
	}
	return false
}

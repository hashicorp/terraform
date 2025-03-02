// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package refactoring

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/ephemeral"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// crossTypeMover is a collection of data that is needed to calculate the
// cross-provider move state changes.
type crossTypeMover struct {
	State             *states.State
	ProviderFactories map[addrs.Provider]providers.Factory
	ProviderCache     map[addrs.Provider]providers.Interface
}

// close ensures the cached providers are closed.
func (m *crossTypeMover) close() tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, provider := range m.ProviderCache {
		diags = diags.Append(provider.Close())
	}
	return diags
}

func (m *crossTypeMover) getProvider(providers addrs.Provider) (providers.Interface, error) {
	if provider, ok := m.ProviderCache[providers]; ok {
		return provider, nil
	}

	if factory, ok := m.ProviderFactories[providers]; ok {
		provider, err := factory()
		if err != nil {
			return nil, err
		}

		m.ProviderCache[providers] = provider
		return provider, nil
	}

	// Then we don't have a provider in the cache - this represents a bug in
	// Terraform since we should have already loaded all the providers in the
	// configuration and the state.
	return nil, fmt.Errorf("provider %s implementation not found; this is a bug in Terraform - please report it", providers)
}

// prepareCrossTypeMove checks if the provided MoveStatement is a cross-type
// move and if so, prepares the data needed to perform the move.
func (m *crossTypeMover) prepareCrossTypeMove(stmt *MoveStatement, source, target addrs.AbsResource) (*crossTypeMove, tfdiags.Diagnostics) {
	if stmt.Provider == nil {
		// This means the resource was not in the configuration at all, so we
		// can't process this. It'll be picked up in the validation errors
		// later.
		return nil, nil
	}

	targetProviderAddr := stmt.Provider
	sourceProviderAddr := m.State.Resource(source).ProviderConfig

	if targetProviderAddr.Provider.Equals(sourceProviderAddr.Provider) {
		if source.Resource.Type == target.Resource.Type {
			// Then this is a move within the same provider and type, so we
			// don't need to do anything special.
			return nil, nil
		}
	}

	var diags tfdiags.Diagnostics
	var err error

	targetProvider, err := m.getProvider(targetProviderAddr.Provider)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(tfdiags.Error, "Failed to initialise provider", err.Error()))
		return nil, diags
	}

	targetSchema := targetProvider.GetProviderSchema()
	diags = diags.Append(targetSchema.Diagnostics)
	if targetSchema.Diagnostics.HasErrors() {
		return nil, diags
	}

	if !targetSchema.ServerCapabilities.MoveResourceState {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported `moved` across resource types",
			Detail:   fmt.Sprintf("The provider %q does not support moved operations across resource types and providers.", targetProviderAddr.Provider),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return nil, diags
	}
	targetResourceSchema, targetResourceSchemaVersion := targetSchema.SchemaForResourceAddr(target.Resource)
	return &crossTypeMove{
		targetProvider:              targetProvider,
		targetProviderAddr:          *targetProviderAddr,
		targetResourceSchema:        targetResourceSchema,
		targetResourceSchemaVersion: targetResourceSchemaVersion,
		sourceProviderAddr:          sourceProviderAddr,
	}, diags
}

type crossTypeMove struct {
	targetProvider              providers.Interface
	targetProviderAddr          addrs.AbsProviderConfig
	targetResourceSchema        *configschema.Block
	targetResourceSchemaVersion uint64

	sourceProviderAddr addrs.AbsProviderConfig
}

// applyCrossTypeMove will update the provider states.SyncState so that value
// at source is the result of the providers move operation. Note, that this
// doesn't actually move the resource in the state file, it just updates the
// value at source ready to be moved.
func (move *crossTypeMove) applyCrossTypeMove(stmt *MoveStatement, source, target addrs.AbsResourceInstance, state *states.SyncState) tfdiags.Diagnostics {
	if move == nil {
		// Then we don't need to do any data transformation.
		return nil
	}

	var diags tfdiags.Diagnostics

	// First, build the request.

	src := state.ResourceInstance(source).Current
	request := providers.MoveResourceStateRequest{
		SourceProviderAddress: move.sourceProviderAddr.Provider.String(),
		SourceTypeName:        source.Resource.Resource.Type,
		SourceSchemaVersion:   int64(src.SchemaVersion),
		SourceStateJSON:       src.AttrsJSON,
		SourcePrivate:         src.Private,
		TargetTypeName:        target.Resource.Resource.Type,
	}

	// Second, ask the provider to transform the value into the type expected by
	// the new resource type.

	resp := move.targetProvider.MoveResourceState(request)
	diags = diags.Append(resp.Diagnostics)
	if resp.Diagnostics.HasErrors() {
		return diags
	}

	// Providers are supposed to return null values for all write-only attributes
	writeOnlyDiags := ephemeral.ValidateWriteOnlyAttributes(
		"Provider returned invalid value",
		func(path cty.Path) string {
			return fmt.Sprintf(
				"The provider %q returned a value for the write-only attribute \"%s%s\" during an across type move operation to %s. Write-only attributes cannot be read back from the provider. This is a bug in the provider, which should be reported in the provider's own issue tracker.",
				move.targetProviderAddr, target, tfdiags.FormatCtyPath(path), target,
			)
		},
		resp.TargetState,
		move.targetResourceSchema,
	)
	diags = diags.Append(writeOnlyDiags)

	if writeOnlyDiags.HasErrors() {
		return diags
	}

	if resp.TargetState == cty.NilVal {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider returned invalid value",
			Detail:   fmt.Sprintf("The provider returned an invalid value during an across type move operation to %s. This is a bug in the relevant provider; Please report it.", target),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}
	if !resp.TargetState.IsWhollyKnown() {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Provider returned invalid value",
			Detail: fmt.Sprintf("The provider %s returned an invalid value during an across type move operation: The returned state contains unknown values. This is a bug in the relevant provider; Please report it.",
				move.targetProviderAddr),
			Subject: stmt.DeclRange.ToHCL().Ptr(),
		})
	}

	// Finally, we can update the source value with the new value.

	newValue := &states.ResourceInstanceObject{
		Value:               resp.TargetState,
		Private:             resp.TargetPrivate,
		Status:              src.Status,
		Dependencies:        src.Dependencies,
		CreateBeforeDestroy: src.CreateBeforeDestroy,
	}

	data, err := newValue.Encode(move.targetResourceSchema.ImpliedType(), move.targetResourceSchemaVersion)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to encode source value",
			Detail:   fmt.Sprintf("Terraform failed to encode the value in state for %s: %v. This is a bug in Terraform; Please report it.", source.String(), err),
			Subject:  stmt.DeclRange.ToHCL().Ptr(),
		})
		return diags
	}

	state.SetResourceInstanceCurrent(source, data, move.targetProviderAddr)
	return diags
}

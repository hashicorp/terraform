package terraform

import (
	"fmt"
	"log"

	hcl "github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/tfdiags"
)

// evalReadDataTransient is an EvalNode implementation that deals with the
// special "transient" storage mode for a data resource, where it must always
// be ready to resolve and cannot be deferred to a later run.
type evalReadDataTransient struct {
	evalReadData
}

func (n *evalReadDataTransient) Eval(ctx EvalContext) (interface{}, error) {
	absAddr := n.Addr.Absolute(ctx.Path())
	log.Printf("[TRACE] evalReadDataTransient: reading %s", absAddr)

	var diags tfdiags.Diagnostics
	var configVal cty.Value

	if n.ProviderSchema == nil || *n.ProviderSchema == nil {
		return nil, fmt.Errorf("provider schema not available for %s", n.Addr)
	}

	config := *n.Config
	providerSchema := *n.ProviderSchema
	schema, _ := providerSchema.SchemaForResourceAddr(n.Addr.ContainingResource())
	if schema == nil {
		// Should be caught during validation, so we don't bother with a pretty error here
		return nil, fmt.Errorf("provider %q does not support data source %q", n.ProviderAddr.Provider.String(), n.Addr.Resource.Type)
	}

	// Transient data resources are never persisted, so their prior values are
	// always null.
	objTy := schema.ImpliedType()
	priorVal := cty.NullVal(objTy)

	forEach, _ := evaluateForEachExpression(config.ForEach, ctx)
	keyData := EvalDataForInstanceKey(n.Addr.Key, forEach)

	var configDiags tfdiags.Diagnostics
	configVal, _, configDiags = ctx.EvaluateBlock(config.Config, schema, nil, keyData)
	diags = diags.Append(configDiags)
	if configDiags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	if !configVal.IsWhollyKnown() {
		// A transient data resource may not depend on any unknown values,
		// because we need to be able to read it during every run, even if
		// we've not created managed resources yet.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid transient data resource",
			Detail:   `The configuration for this data resource depends on resource attributes that cannot be determined until apply, and so it cannot be used with transient storage. Transient-storage data resources must have a known configuration for all phases because their results are discarded after use.`,
			Subject:  config.DeclRange.Ptr(),
		})
		return nil, diags.ErrWithWarnings()
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PreRefresh(absAddr, states.CurrentGen, priorVal)
	}); err != nil {
		diags = diags.Append(err)
		return nil, diags.ErrWithWarnings()
	}

	newVal, readDiags := n.readDataSource(ctx, configVal)
	diags = diags.Append(readDiags)
	if diags.HasErrors() {
		return nil, diags.ErrWithWarnings()
	}

	// Transient data resource results go only in the state, and are marked as
	// transient so that we know to drop them when creating persistent
	// snapshots.
	*n.State = &states.ResourceInstanceObject{
		Value:  newVal,
		Status: states.ObjectTransient, // will be excluded from persisted state snapshots
	}

	if err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostRefresh(absAddr, states.CurrentGen, priorVal, newVal)
	}); err != nil {
		return nil, err
	}

	return nil, diags.ErrWithWarnings()
}

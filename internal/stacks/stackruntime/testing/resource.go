// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package testing

import (
	"fmt"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// resource is an interface that represents a resource that can be managed by
// the mock provider defined in this package.
type resource interface {
	// Read reads the current state of the resource from the store.
	Read(request providers.ReadResourceRequest, store *ResourceStore) providers.ReadResourceResponse

	// Plan plans the resource for creation.
	Plan(request providers.PlanResourceChangeRequest, store *ResourceStore) providers.PlanResourceChangeResponse

	// Apply applies the planned changes to the resource.
	Apply(request providers.ApplyResourceChangeRequest, store *ResourceStore) providers.ApplyResourceChangeResponse
}

func getResource(name string) resource {
	switch name {
	case "testing_resource":
		return &testingResource{}
	case "testing_deferred_resource":
		return &deferredResource{}
	case "testing_failed_resource":
		return &failedResource{}
	case "testing_blocked_resource":
		return &blockedResource{}
	default:
		panic("unknown resource: " + name)
	}
}

var (
	_ resource = (*testingResource)(nil)
	_ resource = (*deferredResource)(nil)
	_ resource = (*failedResource)(nil)
	_ resource = (*blockedResource)(nil)
)

// testingResource is a simple resource that can be managed by the mock provider
// defined in this package.
type testingResource struct{}

func (t *testingResource) Read(request providers.ReadResourceRequest, store *ResourceStore) (response providers.ReadResourceResponse) {
	id := request.PriorState.GetAttr("id").AsString()
	var exists bool
	response.NewState, exists = store.Get(id)
	if !exists {
		response.NewState = cty.NullVal(TestingResourceSchema.ImpliedType())
	}
	return
}

func (t *testingResource) Plan(request providers.PlanResourceChangeRequest, store *ResourceStore) (response providers.PlanResourceChangeResponse) {
	if request.ProposedNewState.IsNull() {
		response.PlannedState = request.ProposedNewState
		return
	}

	response.PlannedState = planEnsureId(request.ProposedNewState)
	replace, err := validateId(response.PlannedState, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "testingResource error", err.Error()))
		return
	}
	if replace {
		response.RequiresReplace = []cty.Path{cty.GetAttrPath("id")}
	}
	return
}

func (t *testingResource) Apply(request providers.ApplyResourceChangeRequest, store *ResourceStore) (response providers.ApplyResourceChangeResponse) {
	if request.PlannedState.IsNull() {
		store.Delete(request.PriorState.GetAttr("id").AsString())
		response.NewState = request.PlannedState
		return
	}

	value := applyEnsureId(request.PlannedState)
	replace, err := validateId(value, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "testingResource error", err.Error()))
		return
	}
	response.NewState = value

	if replace {
		store.Delete(request.PriorState.GetAttr("id").AsString())
	}
	store.Set(response.NewState.GetAttr("id").AsString(), response.NewState)
	return
}

// deferredResource is a resource that can defer itself based on the provided
// configuration.
type deferredResource struct{}

func (d *deferredResource) Read(request providers.ReadResourceRequest, store *ResourceStore) (response providers.ReadResourceResponse) {
	id := request.PriorState.GetAttr("id").AsString()
	var exists bool
	response.NewState, exists = store.Get(id)
	if !exists {
		response.NewState = cty.NullVal(DeferredResourceSchema.ImpliedType())
	}
	return
}

func (d *deferredResource) Plan(request providers.PlanResourceChangeRequest, store *ResourceStore) (response providers.PlanResourceChangeResponse) {
	if request.ProposedNewState.IsNull() {
		if deferred := request.PriorState.GetAttr("deferred"); !deferred.IsNull() && deferred.IsKnown() && deferred.True() {
			response.Deferred = &providers.Deferred{
				Reason: providers.DeferredReasonResourceConfigUnknown,
			}
		}
		response.PlannedState = request.ProposedNewState
		return
	}

	response.PlannedState = planEnsureId(request.ProposedNewState)
	replace, err := validateId(response.PlannedState, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "deferredResource error", err.Error()))
		return
	}
	if deferred := response.PlannedState.GetAttr("deferred"); !deferred.IsNull() && deferred.IsKnown() && deferred.True() {
		response.Deferred = &providers.Deferred{
			Reason: providers.DeferredReasonResourceConfigUnknown,
		}
	}
	if replace {
		response.RequiresReplace = []cty.Path{cty.GetAttrPath("id")}
	}
	return
}

func (d *deferredResource) Apply(request providers.ApplyResourceChangeRequest, store *ResourceStore) (response providers.ApplyResourceChangeResponse) {
	if request.PlannedState.IsNull() {
		store.Delete(request.PriorState.GetAttr("id").AsString())
		response.NewState = request.PlannedState
		return
	}

	value := applyEnsureId(request.PlannedState)
	replace, err := validateId(value, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "deferredResource error", err.Error()))
		return
	}
	response.NewState = value

	if replace {
		store.Delete(request.PriorState.GetAttr("id").AsString())
	}
	store.Set(response.NewState.GetAttr("id").AsString(), response.NewState)
	return
}

// failedResource is a resource that can be set to fail during Plan or Apply.
type failedResource struct{}

func (f *failedResource) Read(request providers.ReadResourceRequest, store *ResourceStore) (response providers.ReadResourceResponse) {
	id := request.PriorState.GetAttr("id").AsString()
	var exists bool
	response.NewState, exists = store.Get(id)
	if !exists {
		response.NewState = cty.NullVal(FailedResourceSchema.ImpliedType())
	}
	return
}

func (f *failedResource) Plan(request providers.PlanResourceChangeRequest, store *ResourceStore) (response providers.PlanResourceChangeResponse) {
	if request.ProposedNewState.IsNull() {
		response.PlannedState = request.ProposedNewState
		if attr := request.PriorState.GetAttr("fail_plan"); !attr.IsNull() && attr.IsKnown() && attr.True() {
			response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "failedResource error", "failed during plan"))
			return
		}
		return
	}

	response.PlannedState = planEnsureId(request.ProposedNewState)
	replace, err := validateId(response.PlannedState, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "failedResource error", err.Error()))
		return
	}

	setUnknown(response.PlannedState, "fail_apply")
	setUnknown(response.PlannedState, "fail_plan")

	if attr := response.PlannedState.GetAttr("fail_plan"); !attr.IsNull() && attr.IsKnown() && attr.True() {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "failedResource error", "failed during plan"))
	}
	if replace {
		response.RequiresReplace = []cty.Path{cty.GetAttrPath("id")}
	}

	return
}

func (f *failedResource) Apply(request providers.ApplyResourceChangeRequest, store *ResourceStore) (response providers.ApplyResourceChangeResponse) {
	if request.PlannedState.IsNull() {
		if attr := request.PriorState.GetAttr("fail_apply"); !attr.IsNull() && attr.IsKnown() && attr.True() {
			response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "failedResource error", "failed during apply"))
			return
		}
		response.NewState = request.PlannedState
		store.Delete(request.PriorState.GetAttr("id").AsString())
		return
	}

	value := applyEnsureId(request.PlannedState)
	replace, err := validateId(value, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "testingResource error", err.Error()))
		return
	}

	setKnown(value, "fail_apply", cty.False)
	setKnown(value, "fail_plan", cty.False)

	if attr := value.GetAttr("fail_apply"); !attr.IsNull() && attr.IsKnown() && attr.True() {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "failedResource error", "failed during apply"))
		return
	}
	response.NewState = value

	if replace {
		store.Delete(request.PriorState.GetAttr("id").AsString())
	}
	store.Set(response.NewState.GetAttr("id").AsString(), response.NewState)
	return
}

// blockedResource is a resource that accepts a list of required resource ids
// and will fail to apply if those resources don't exist. They will also fail to
// destroy if the resources do not exist - this ensures they have to be created
// and destroyed in the correct order.
type blockedResource struct{}

func (b *blockedResource) Read(request providers.ReadResourceRequest, store *ResourceStore) (response providers.ReadResourceResponse) {
	id := request.PriorState.GetAttr("id").AsString()
	var exists bool
	response.NewState, exists = store.Get(id)
	if !exists {
		response.NewState = cty.NullVal(DeferredResourceSchema.ImpliedType())
	}
	return
}

func (b *blockedResource) Plan(request providers.PlanResourceChangeRequest, store *ResourceStore) (response providers.PlanResourceChangeResponse) {
	if request.ProposedNewState.IsNull() {
		response.PlannedState = request.ProposedNewState
		return
	}

	response.PlannedState = planEnsureId(request.ProposedNewState)
	replace, err := validateId(response.PlannedState, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "testingResource error", err.Error()))
		return
	}
	if replace {
		response.RequiresReplace = []cty.Path{cty.GetAttrPath("id")}
	}
	return
}

func (b *blockedResource) Apply(request providers.ApplyResourceChangeRequest, store *ResourceStore) (response providers.ApplyResourceChangeResponse) {
	if request.PlannedState.IsNull() {
		if required := request.PriorState.GetAttr("required_resources"); !required.IsNull() && required.IsKnown() {
			for _, id := range required.AsValueSlice() {
				if _, exists := store.Get(id.AsString()); !exists {
					response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "blockedResource error", fmt.Sprintf("required resource %q does not exists, so can't destroy self", id.AsString())))
					return
				}
			}
		}

		store.Delete(request.PriorState.GetAttr("id").AsString())
		response.NewState = request.PlannedState
		return
	}

	value := applyEnsureId(request.PlannedState)
	replace, err := validateId(value, request.PriorState, store)
	if err != nil {
		response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "testingResource error", err.Error()))
		return
	}

	if required := value.GetAttr("required_resources"); !required.IsNull() && required.IsKnown() {
		for _, id := range required.AsValueSlice() {
			if _, exists := store.Get(id.AsString()); !exists {
				response.Diagnostics = append(response.Diagnostics, tfdiags.Sourceless(tfdiags.Error, "blockedResource error", fmt.Sprintf("required resource %q does not exist, so can't apply self", id.AsString())))
				return
			}
		}
	}
	response.NewState = value

	if replace {
		store.Delete(request.PriorState.GetAttr("id").AsString())
	}
	store.Set(response.NewState.GetAttr("id").AsString(), response.NewState)
	return
}

func validateId(target cty.Value, prior cty.Value, store *ResourceStore) (bool, error) {
	if prior.IsNull() {
		// Then we're creating a resource, we want to make sure we're not
		// creating a resource with an existing ID.
		if id := target.GetAttr("id"); id.IsKnown() {
			if _, exists := store.Get(id.AsString()); exists {
				return false, fmt.Errorf("resource with id %q already exists", id.AsString())
			}
		}

		return false, nil
	}

	if attr := target.GetAttr("id"); !attr.IsKnown() {
		// Then the attribute has been set to unknown, which means we're
		// potentially changing the id.
		return true, nil
	}

	// Now, we know that the ID is known in both the prior and target states.
	if result := prior.GetAttr("id").Equals(target.GetAttr("id")); result.False() {
		// Then the ID value is changing, so we need to delete the old ID
		// and create the new one.
		return true, nil
	}

	return false, nil
}

func planEnsureId(value cty.Value) cty.Value {
	return setUnknown(value, "id")
}

func applyEnsureId(value cty.Value) cty.Value {
	return setKnown(value, "id", cty.StringVal(mustGenerateUUID()))
}

func setUnknown(value cty.Value, attr string) cty.Value {
	if v := value.GetAttr(attr); v.IsNull() {
		vals := value.AsValueMap()
		vals[attr] = cty.UnknownVal(cty.String)
		return cty.ObjectVal(vals)
	}
	return value
}

func setKnown(value cty.Value, attr string, attrValue cty.Value) cty.Value {
	if v := value.GetAttr(attr); !v.IsKnown() {
		vals := value.AsValueMap()
		vals[attr] = attrValue
		return cty.ObjectVal(vals)
	}
	return value
}

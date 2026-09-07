// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

func dataStoreResourceSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id": {Type: cty.String, Computed: true},

				// forces replacement of the entire resource when changed
				"triggers_replace": {Type: cty.DynamicPseudoType, Optional: true},

				// input is reflected in output after apply, and changes to
				// input always result in a re-computation of output.
				"input":  {Type: cty.DynamicPseudoType, Optional: true},
				"output": {Type: cty.DynamicPseudoType, Computed: true},
			},
			BlockTypes: map[string]*configschema.NestedBlock{
				"store": {
					Nesting: configschema.NestingSingle,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							// The input attribute will be exposed as a stable
							// value in store.output or store.sensitive_output.
							"input":            {Type: cty.DynamicPseudoType, Optional: true, WriteOnly: true},
							"output":           {Type: cty.DynamicPseudoType, Computed: true},
							"sensitive_output": {Type: cty.DynamicPseudoType, Computed: true, Sensitive: true},
							// If there is a version value, a change in that
							// value will trigger a change in the stored output
							// or sensitive_output. If there is no version
							// value, then input will be compared directly
							// against output.
							"version": {Type: cty.DynamicPseudoType, Optional: true},

							"sensitive": {Type: cty.Bool, Optional: true},

							// replace causes the resource to be replaced when
							// there is a change to a store output value.
							"replace": {Type: cty.Bool, Optional: true},
						},
					},
				},
			},
		},

		Identity: dataStoreResourceIdentitySchema().Body,
	}
}

func dataStoreResourceIdentitySchema() providers.IdentitySchema {
	return providers.IdentitySchema{
		Version: 0,
		Body: &configschema.Object{
			Attributes: map[string]*configschema.Attribute{
				"id": {
					Type:        cty.String,
					Description: "The unique identifier for the data store.",
					Required:    true,
				},
			},
			Nesting: configschema.NestingSingle,
		},
	}
}

func validateDataStoreResourceConfig(req providers.ValidateResourceConfigRequest) (resp providers.ValidateResourceConfigResponse) {
	if req.Config.IsNull() {
		return resp
	}

	// Core does not currently validate computed values are not set in the
	// configuration.
	for _, attr := range []string{"id", "output"} {
		if !req.Config.GetAttr(attr).IsNull() {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf(`%q attribute is read-only`, attr))
		}
	}
	return resp
}

func upgradeDataStoreResourceState(req providers.UpgradeResourceStateRequest) (resp providers.UpgradeResourceStateResponse) {
	// We've only added new nullable block attributes, so unmarshaling from json
	// will complete the data structure correctly.
	val, err := ctyjson.Unmarshal(req.RawStateJSON, dataStoreResourceSchema().Body.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	resp.UpgradedState = val
	return resp
}

func upgradeDataStoreResourceIdentity(providers.UpgradeResourceIdentityRequest) (resp providers.UpgradeResourceIdentityResponse) {
	resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("The builtin provider does not support provider upgrades since it has not changed the identity schema yet."))
	return resp
}

func readDataStoreResourceState(req providers.ReadResourceRequest) (resp providers.ReadResourceResponse) {
	resp.NewState = req.PriorState
	resp.Private = req.Private
	return resp
}

func planDataStoreResourceChange(req providers.PlanResourceChangeRequest) (resp providers.PlanResourceChangeResponse) {
	resp.PlannedState = req.ProposedNewState

	if req.ProposedNewState.IsNull() {
		// destroy op
		return resp
	}

	planned := req.ProposedNewState.AsValueMap()
	prior := req.PriorState

	// first determine if this is a create or replace
	if !prior.IsNull() && !prior.GetAttr("triggers_replace").RawEquals(req.ProposedNewState.GetAttr("triggers_replace")) {
		// trigger changed, so we need to replace the entire instance
		resp.RequiresReplace = append(resp.RequiresReplace, cty.GetAttrPath("triggers_replace"))

		// set the prior value to null so that that everything else is treated
		// as if it's a new instance
		prior = cty.NullVal(req.ProposedNewState.Type())
	}

	// creating a new instance, so we need a new ID
	if prior.IsNull() {
		// New instances, so set the id value to unknown.
		planned["id"] = cty.UnknownVal(cty.String).RefineNotNull()
	}

	// check the input/output for changes
	input := req.ProposedNewState.GetAttr("input")
	priorInput := cty.NullVal(cty.DynamicPseudoType)
	if !prior.IsNull() {
		priorInput = prior.GetAttr("input")
	}

	if !priorInput.RawEquals(input) {
		if input.IsNull() {
			// we reflect the type even if the value is null
			planned["output"] = cty.NullVal(input.Type())
		} else {
			// input changed, so we need to re-compute output
			planned["output"] = cty.UnknownVal(input.Type())
		}
	}

	// check the store object for changes
	if store := req.ProposedNewState.GetAttr("store"); !store.IsNull() {
		objMap := storeMap(store.AsValueMap())
		priorVersion := cty.NullVal(cty.DynamicPseudoType)
		priorSensitive := cty.NullVal(cty.Bool)

		for _, mustKnow := range []string{"sensitive", "replace"} {
			if !store.GetAttr(mustKnow).IsKnown() {
				resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"unexpected unknown value",
					fmt.Sprintf("the %q attribute must be known in order to plan changes to this resource", mustKnow),
					cty.GetAttrPath("store").GetAttr(mustKnow),
				))
				return resp
			}
		}

		if !prior.IsNull() && !prior.GetAttr("store").IsNull() {
			priorVersion = prior.GetAttr("store").GetAttr("version")
			priorSensitive = prior.GetAttr("store").GetAttr("sensitive")
		}

		// if sensitive changed, just move the data between outputs
		if !priorSensitive.RawEquals(objMap.sensitive()) {
			objMap.swapOutputs()
		}

		// Plan an update if the version changed, or if the input and output don't
		// match in the absence of a version value.
		switch {
		// if input and outputs are all null, just pass through a possible null type.
		case objMap.valuesNull():
			objMap.storeNull()

		// if there is a version, checked if it has changed
		case objMap.hasVersion():
			// The version value comparison is done within this case, because we
			// don't want to fall into the input comparison case when there is a
			// version, nor do we want to prevent evaluating that case if the
			// input and output changed.
			if !objMap.version().RawEquals(priorVersion) {
				objMap.storeChange()
			}

		// if there is no version, we automatically update if the input and output
		// don't match
		case objMap.hasChange():
			objMap.storeChange()
		}

		// see if we want store to replace the resource
		if objMap.replace() {
			planned["id"] = cty.UnknownVal(cty.String)
			resp.RequiresReplace = append(resp.RequiresReplace, cty.GetAttrPath("store").GetAttr("input"))
		}

		// and the input must always be returned as the unset null value because it
		// is write-only
		objMap.clearInput()

		planned["store"] = cty.ObjectVal(objMap)
	}

	resp.PlannedState = cty.ObjectVal(planned)
	return resp
}

// storeMap encapsulates some of the logic around handling the various
// combinations of the object attributes. There are a few accessors and simple
// set functions just to make accessing the data consistent, so nothing needs to
// index the map directly.
type storeMap map[string]cty.Value

func (d storeMap) valuesNull() bool {
	return d["input"].IsNull() && d["output"].IsNull() && d["sensitive_output"].IsNull()
}

func (d storeMap) isSensitive() bool {
	return !d["sensitive"].IsNull() && d["sensitive"].True()
}

func (d storeMap) sensitive() cty.Value {
	return d["sensitive"]
}

func (d storeMap) hasVersion() bool {
	return !d["version"].IsNull()
}

func (d storeMap) version() cty.Value {
	return d["version"]
}

func (d storeMap) storeNull() {
	d.write(cty.NullVal(d["input"].Type()))
}

func (d storeMap) storeChange() {
	d.write(cty.UnknownVal(d["input"].Type()))
}

func (d storeMap) swapOutputs() {
	output := d["output"]
	if tmp := d["sensitive_output"]; !tmp.IsNull() {
		output = tmp
	}

	d.write(output)
}

func (d storeMap) write(v cty.Value) {
	if d.isSensitive() {
		d["sensitive_output"] = v
		d["output"] = cty.NullVal(cty.DynamicPseudoType)
		return
	}

	d["output"] = v
	d["sensitive_output"] = cty.NullVal(cty.DynamicPseudoType)
}

func (d storeMap) clearInput() {
	d["input"] = cty.NullVal(cty.DynamicPseudoType)
}

func (d storeMap) hasChange() bool {
	old := d["output"]
	if !d["sensitive_output"].IsNull() {
		old = d["sensitive_output"]
	}

	return !old.RawEquals(d["input"])
}

func (d storeMap) replace() bool {
	return d["replace"].True() && !(d["output"].IsKnown() && d["sensitive_output"].IsKnown())
}

var testUUIDHook func() string

func applyDataStoreResourceChange(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	if req.PlannedState.IsNull() {
		resp.NewState = req.PlannedState
		return resp
	}

	// Applying a plan only consists of filling in any unknown values. We can
	// write this as a single transformation, and base the logic on the path of
	// the transform value.
	resp.NewState, _ = cty.Transform(req.PlannedState, func(path cty.Path, val cty.Value) (cty.Value, error) {
		if val.IsKnown() {
			return val, nil
		}

		// val is unknown, so find the correct value based on our path
		switch {
		case path.Equals(cty.GetAttrPath("id")):
			idString, err := uuid.GenerateUUID()
			// Terraform would probably never get this far without a good random
			// source, but catch the error anyway.
			if err != nil {
				diag := tfdiags.AttributeValue(
					tfdiags.Error,
					"Error generating id",
					err.Error(),
					cty.GetAttrPath("id"),
				)

				resp.Diagnostics = resp.Diagnostics.Append(diag)
			}

			if testUUIDHook != nil {
				idString = testUUIDHook()
			}
			return cty.StringVal(idString), nil

		case path.Equals(cty.GetAttrPath("output")):
			return req.PlannedState.GetAttr("input"), nil

		case path.Equals(cty.GetAttrPath("store").GetAttr("output")):
			// input is write-only, so won't be in the planned state. We ned to get
			// the latest ephemeral value directly from the config.
			return req.Config.GetAttr("store").GetAttr("input"), nil

		case path.Equals(cty.GetAttrPath("store").GetAttr("sensitive_output")):
			// input is write-only, so won't be in the planned state. We ned to get
			// the latest ephemeral value directly from the config.
			return req.Config.GetAttr("store").GetAttr("input"), nil
		}
		return val, nil
	})

	return resp
}

// TODO: This isn't very useful even for examples, because terraform_data has
// no way to refresh the full resource value from only the import ID. This
// minimal implementation allows the import to succeed, and can be extended
// once the configuration is available during import.
func importDataStore(req providers.ImportResourceStateRequest) (resp providers.ImportResourceStateResponse) {
	schema := dataStoreResourceSchema()
	v := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal(req.ID),
	})
	state, err := schema.Body.CoerceValue(v)
	resp.Diagnostics = resp.Diagnostics.Append(err)

	resp.ImportedResources = []providers.ImportedResource{
		{
			TypeName: req.TypeName,
			State:    state,
		},
	}
	return resp
}

// moveDataStoreResourceState enables moving from the official null_resource
// managed resource to the terraform_data managed resource.
func moveDataStoreResourceState(req providers.MoveResourceStateRequest) (resp providers.MoveResourceStateResponse) {
	// Verify that the source provider is an official hashicorp/null provider,
	// but ignore the hostname for mirrors.
	if !strings.HasSuffix(req.SourceProviderAddress, "hashicorp/null") {
		diag := tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported source provider for move operation",
			"Only moving from the official hashicorp/null provider to terraform_data is supported.",
		)
		resp.Diagnostics = resp.Diagnostics.Append(diag)

		return resp
	}

	// Verify that the source resource type name is null_resource.
	if req.SourceTypeName != "null_resource" {
		diag := tfdiags.Sourceless(
			tfdiags.Error,
			"Unsupported source resource type for move operation",
			"Only moving from the null_resource managed resource to terraform_data is supported.",
		)
		resp.Diagnostics = resp.Diagnostics.Append(diag)

		return resp
	}

	nullResourceSchemaType := nullResourceSchema().Body.ImpliedType()
	nullResourceValue, err := ctyjson.Unmarshal(req.SourceStateJSON, nullResourceSchemaType)

	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)

		return resp
	}

	triggersReplace := nullResourceValue.GetAttr("triggers")

	// PlanResourceChange uses RawEquals comparison, which will show a
	// difference between cty.NullVal(cty.Map(cty.String)) and
	// cty.NullVal(cty.DynamicPseudoType).
	if triggersReplace.IsNull() {
		triggersReplace = cty.NullVal(cty.DynamicPseudoType)
	} else {
		// PlanResourceChange uses RawEquals comparison, which will show a
		// difference between cty.MapVal(...) and cty.ObjectVal(...). Given that
		// triggers is typically configured using direct configuration syntax of
		// {...}, which is a cty.ObjectVal, over a map typed variable or
		// explicitly type converted map, this pragmatically chooses to convert
		// the triggers value to cty.ObjectVal to prevent an immediate plan
		// difference for the more typical case.
		triggersReplace = cty.ObjectVal(triggersReplace.AsValueMap())
	}

	schema := dataStoreResourceSchema()
	v := cty.ObjectVal(map[string]cty.Value{
		"id":               nullResourceValue.GetAttr("id"),
		"triggers_replace": triggersReplace,
	})

	state, err := schema.Body.CoerceValue(v)

	// null_resource did not use private state, so it is unnecessary to move.
	resp.Diagnostics = resp.Diagnostics.Append(err)
	resp.TargetState = state

	return resp
}

func nullResourceSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"id":       {Type: cty.String, Computed: true},
				"triggers": {Type: cty.Map(cty.String), Optional: true},
			},
		},
	}
}

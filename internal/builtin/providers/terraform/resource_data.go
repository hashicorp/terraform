// Copyright (c) HashiCorp, Inc.
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
		Version: 1,
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
				"sensitive": {
					Nesting: configschema.NestingGroup,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							// sensitive input and output work just like the top
							// level input and output, but otherwise are marked
							// as sensitive.
							"input":  {Type: cty.DynamicPseudoType, Optional: true, Sensitive: true},
							"output": {Type: cty.DynamicPseudoType, Computed: true, Sensitive: true},

							// replace causes the resource to be replaced when
							// there is a change to the output value.
							"replace": {Type: cty.Bool, Optional: true},
						},
					},
				},
				"write_only": {
					Nesting: configschema.NestingGroup,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							// The input attribute will be exposed as a stable
							// value in write_only.output.
							"input":  {Type: cty.DynamicPseudoType, Optional: true, WriteOnly: true},
							"output": {Type: cty.DynamicPseudoType, Computed: true},

							// If there is a version value, a change in that
							// value will trigger a change in the stored
							// write-only.output value. If there is no version
							// value, then input will be compared directly
							// against output.
							"version": {Type: cty.DynamicPseudoType, Optional: true},

							// replace causes the resource to be replaced when
							// there is a change to the output value.
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
	val, err := ctyjson.Unmarshal(req.RawStateJSON, dataStoreResourceSchema().Body.ImpliedType())
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	obj := val.AsValueMap()

	// write_only and sensitive blocks are NestingGroup, so they may must not be
	// null.
	if obj["write_only"].IsNull() {
		obj["write_only"] = cty.ObjectVal(map[string]cty.Value{
			"input":   cty.NullVal(cty.DynamicPseudoType),
			"output":  cty.NullVal(cty.DynamicPseudoType),
			"version": cty.NullVal(cty.DynamicPseudoType),
			"replace": cty.NullVal(cty.Bool),
		})
	}

	if obj["sensitive"].IsNull() {
		obj["sensitive"] = cty.ObjectVal(map[string]cty.Value{
			"input":   cty.NullVal(cty.DynamicPseudoType),
			"output":  cty.NullVal(cty.DynamicPseudoType),
			"replace": cty.NullVal(cty.Bool),
		})
	}

	resp.UpgradedState = cty.ObjectVal(obj)
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
	if req.ProposedNewState.IsNull() {
		// destroy op
		resp.PlannedState = req.ProposedNewState
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

	// check the sensitive object for changes
	proposedSensitiveInput := planned["sensitive"].GetAttr("input")
	priorSensitiveInput := cty.NullVal(cty.DynamicPseudoType)
	if !prior.IsNull() {
		priorSensitiveInput = prior.GetAttr("sensitive").GetAttr("input")
	}

	if !proposedSensitiveInput.RawEquals(priorSensitiveInput) {
		output := cty.NullVal(proposedSensitiveInput.Type())
		replace := planned["sensitive"].GetAttr("replace")
		if !proposedSensitiveInput.IsNull() {
			// input changed, so we need to re-compute output
			output = cty.UnknownVal(proposedSensitiveInput.Type())

			if replace.True() {
				// a change in sensitive output will cause this to be replaced
				resp.RequiresReplace = append(resp.RequiresReplace, cty.GetAttrPath("sensitive").GetAttr("output"))
			}
		}

		planned["sensitive"] = cty.ObjectVal(map[string]cty.Value{
			"input":   proposedSensitiveInput,
			"output":  output,
			"replace": replace,
		})
	}

	// check the write_only object for changes
	writeOnly := req.ProposedNewState.GetAttr("write_only").AsValueMap()
	priorWOTrigger := cty.NullVal(cty.DynamicPseudoType)
	if !prior.IsNull() {
		priorWOTrigger = prior.GetAttr("write_only").GetAttr("version")
	}

	// Plan an update if the version changed, or if the input and output don't
	// match in the absence of a version value.
	switch {
	// if the input is null, then like the other input+output pairs output is
	// also a known null during plan
	case writeOnly["input"].IsNull():
		writeOnly["output"] = cty.NullVal(writeOnly["input"].Type())

	// if there is a version, checked if it has changed
	case !writeOnly["version"].IsNull():
		// The version value comparison is done within this case, because we
		// don't want to fall into the input comparison case.
		if !writeOnly["version"].RawEquals(priorWOTrigger) {
			writeOnly["output"] = cty.UnknownVal(writeOnly["input"].Type())
		}

	// if there is no version, we automatically update if the input and output
	// don't match
	case !writeOnly["input"].RawEquals(writeOnly["output"]):
		writeOnly["output"] = cty.UnknownVal(writeOnly["input"].Type())
	}

	// see if we want write-only to replace the resource
	if planned["write_only"].GetAttr("replace").True() && !writeOnly["output"].IsKnown() {
		resp.RequiresReplace = append(resp.RequiresReplace, cty.GetAttrPath("write_only").GetAttr("output"))
	}

	// and the input must always be returned as the unset null value because it
	// is write-only
	writeOnly["input"] = cty.NullVal(cty.DynamicPseudoType)

	planned["write_only"] = cty.ObjectVal(writeOnly)
	resp.PlannedState = cty.ObjectVal(planned)
	return resp
}

var testUUIDHook func() string

func applyDataStoreResourceChange(req providers.ApplyResourceChangeRequest) (resp providers.ApplyResourceChangeResponse) {
	if req.PlannedState.IsNull() {
		resp.NewState = req.PlannedState
		return resp
	}

	// The new state will be created from the PlannedState, by filling in the
	// unknowns, and removing the write-only input attribute.
	newState := req.PlannedState.AsValueMap()

	if !req.PlannedState.GetAttr("id").IsKnown() {
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

		newState["id"] = cty.StringVal(idString)
	}

	if !req.PlannedState.GetAttr("output").IsKnown() {
		newState["output"] = req.PlannedState.GetAttr("input")
	}

	sensitive := newState["sensitive"].AsValueMap()
	if !sensitive["output"].IsKnown() {
		sensitive["output"] = sensitive["input"]
	}
	newState["sensitive"] = cty.ObjectVal(sensitive)

	writeOnly := newState["write_only"].AsValueMap()
	if !writeOnly["output"].IsKnown() {
		// input is write-only, so won't be in the planned state. We ned to get
		// the latest ephemeral value directly from the config.
		writeOnly["output"] = req.Config.GetAttr("write_only").GetAttr("input")
	}
	writeOnly["input"] = cty.NullVal(cty.DynamicPseudoType)
	newState["write_only"] = cty.ObjectVal(writeOnly)

	resp.NewState = cty.ObjectVal(newState)

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

package schema

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/hcl2shim"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/zclconf/go-cty/cty"

	ctyjson "github.com/zclconf/go-cty/cty/json"
)

// DiffFromValues takes the current state and desired state as cty.Values and
// derives a terraform.InstanceDiff to give to the legacy providers. This is
// used to take the states provided by the new ApplyResourceChange method and
// convert them to a state+diff required for the legacy Apply method.
func DiffFromValues(prior, planned cty.Value, res *Resource) (*terraform.InstanceDiff, error) {
	return diffFromValues(prior, planned, res, nil)
}

// diffFromValues takes an additional CustomizeDiffFunc, so we can generate our
// test fixtures from the legacy tests. In the new provider protocol the diff
// only needs to be created for the apply operation, and any customizations
// have already been done.
func diffFromValues(prior, planned cty.Value, res *Resource, cust CustomizeDiffFunc) (*terraform.InstanceDiff, error) {
	instanceState := InstanceStateFromStateValue(prior, res.SchemaVersion)

	configSchema := res.CoreConfigSchema()

	cfg := terraform.NewResourceConfigShimmed(planned, configSchema)

	return schemaMap(res.Schema).Diff(instanceState, cfg, cust, nil)
}

// ApplyDiff takes a cty.Value state and applies a terraform.InstanceDiff to
// get a new cty.Value state. This is used to convert the diff returned from
// the legacy provider Diff method to the state required for the new
// PlanResourceChange method.
func ApplyDiff(state cty.Value, d *terraform.InstanceDiff, schemaBlock *configschema.Block) (cty.Value, error) {
	// No diff means the state is unchanged.
	if d.Empty() {
		return state, nil
	}

	// Create an InstanceState attributes from our existing state.
	// We can use this to more easily apply the diff changes.
	attrs := hcl2shim.FlatmapValueFromHCL2(state)
	if attrs == nil {
		attrs = map[string]string{}
	}

	if d.Destroy || d.DestroyDeposed || d.DestroyTainted {
		// to mark a destroy, we remove all attributes
		attrs = map[string]string{}
	}

	for attr, diff := range d.Attributes {
		old, exists := attrs[attr]

		if old != diff.Old && exists {
			return state, fmt.Errorf("mismatched diff: %q != %q", old, diff.Old)
		}

		if diff.NewComputed {
			attrs[attr] = config.UnknownVariableValue
			continue
		}

		if diff.NewRemoved {
			delete(attrs, attr)
			continue
		}

		attrs[attr] = diff.New
	}

	val, err := hcl2shim.HCL2ValueFromFlatmap(attrs, schemaBlock.ImpliedType())
	if err != nil {
		return val, err
	}

	return schemaBlock.CoerceValue(val)
}

// StateValueToJSONMap converts a cty.Value to generic JSON map via the cty JSON
// encoding.
func StateValueToJSONMap(val cty.Value, ty cty.Type) (map[string]interface{}, error) {
	js, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return nil, err
	}

	var m map[string]interface{}
	if err := json.Unmarshal(js, &m); err != nil {
		return nil, err
	}

	return m, nil
}

// JSONMapToStateValue takes a generic json map[string]interface{} and converts it
// to the specific type, ensuring that the values conform to the schema.
func JSONMapToStateValue(m map[string]interface{}, block *configschema.Block) (cty.Value, error) {
	var val cty.Value

	js, err := json.Marshal(m)
	if err != nil {
		return val, err
	}

	val, err = ctyjson.Unmarshal(js, block.ImpliedType())
	if err != nil {
		return val, err
	}

	return block.CoerceValue(val)
}

// StateValueFromInstanceState converts a terraform.InstanceState to a
// cty.Value as described by the provided cty.Type, and maintains the resource
// ID as the "id" attribute.
func StateValueFromInstanceState(is *terraform.InstanceState, ty cty.Type) (cty.Value, error) {
	if is == nil {
		// if the state is nil, we need to construct a complete cty.Value with
		// null attributes, rather than a single cty.NullVal(ty)
		is = &terraform.InstanceState{}
	}

	// make sure ID is included in the attributes. The InstanceState.ID value
	// takes precedent.
	if is.Attributes == nil {
		is.Attributes = map[string]string{}
	}

	if is.ID != "" {
		is.Attributes["id"] = is.ID
	}

	return hcl2shim.HCL2ValueFromFlatmap(is.Attributes, ty)
}

// InstanceStateFromStateValue converts a cty.Value to a
// terraform.InstanceState. This function requires the schema version used by
// the provider, because the legacy providers used the private Meta data in the
// InstanceState to store the schema version.
func InstanceStateFromStateValue(state cty.Value, schemaVersion int) *terraform.InstanceState {
	attrs := hcl2shim.FlatmapValueFromHCL2(state)
	return &terraform.InstanceState{
		ID:         attrs["id"],
		Attributes: attrs,
		Meta: map[string]interface{}{
			"schema_version": schemaVersion,
		},
	}
}

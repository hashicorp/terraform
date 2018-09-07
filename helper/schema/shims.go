package schema

import (
	"encoding/json"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
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
	return d.ApplyToValue(state, schemaBlock)
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
	return is.AttrsAsObjectValue(ty)
}

// InstanceStateFromStateValue converts a cty.Value to a
// terraform.InstanceState. This function requires the schema version used by
// the provider, because the legacy providers used the private Meta data in the
// InstanceState to store the schema version.
func InstanceStateFromStateValue(state cty.Value, schemaVersion int) *terraform.InstanceState {
	return terraform.NewInstanceStateShimmedFromValue(state, schemaVersion)
}

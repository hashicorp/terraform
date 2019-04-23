package schema

import (
	"encoding/json"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/config"
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
	instanceState, err := res.ShimInstanceStateFromValue(prior)
	if err != nil {
		return nil, err
	}

	configSchema := res.CoreConfigSchema()

	cfg := terraform.NewResourceConfigShimmed(planned, configSchema)
	removeConfigUnknowns(cfg.Config)
	removeConfigUnknowns(cfg.Raw)

	diff, err := schemaMap(res.Schema).Diff(instanceState, cfg, cust, nil, false)
	if err != nil {
		return nil, err
	}

	return diff, err
}

// During apply the only unknown values are those which are to be computed by
// the resource itself. These may have been marked as unknown config values, and
// need to be removed to prevent the UnknownVariableValue from appearing the diff.
func removeConfigUnknowns(cfg map[string]interface{}) {
	for k, v := range cfg {
		switch v := v.(type) {
		case string:
			if v == config.UnknownVariableValue {
				delete(cfg, k)
			}
		case []interface{}:
			for _, i := range v {
				if m, ok := i.(map[string]interface{}); ok {
					removeConfigUnknowns(m)
				}
			}
		case map[string]interface{}:
			removeConfigUnknowns(v)
		}
	}
}

// ApplyDiff takes a cty.Value state and applies a terraform.InstanceDiff to
// get a new cty.Value state. This is used to convert the diff returned from
// the legacy provider Diff method to the state required for the new
// PlanResourceChange method.
func ApplyDiff(base cty.Value, d *terraform.InstanceDiff, schema *configschema.Block) (cty.Value, error) {
	return d.ApplyToValue(base, schema)
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

// LegacyResourceSchema takes a *Resource and returns a deep copy with 0.12 specific
// features removed. This is used by the shims to get a configschema that
// directly matches the structure of the schema.Resource.
func LegacyResourceSchema(r *Resource) *Resource {
	if r == nil {
		return nil
	}
	// start with a shallow copy
	newResource := new(Resource)
	*newResource = *r
	newResource.Schema = map[string]*Schema{}

	for k, s := range r.Schema {
		newResource.Schema[k] = LegacySchema(s)
	}

	return newResource
}

// LegacySchema takes a *Schema and returns a deep copy with some 0.12-specific
// features disabled. This is used by the shims to get a configschema that
// better reflects the given schema.Resource, without any adjustments we
// make for when sending a schema to Terraform Core.
func LegacySchema(s *Schema) *Schema {
	if s == nil {
		return nil
	}
	// start with a shallow copy
	newSchema := new(Schema)
	*newSchema = *s
	newSchema.SkipCoreTypeCheck = false

	switch e := newSchema.Elem.(type) {
	case *Schema:
		newSchema.Elem = LegacySchema(e)
	case *Resource:
		newSchema.Elem = LegacyResourceSchema(e)
	}

	return newSchema
}

package resource

import (
	"fmt"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/config/hcl2shim"

	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/terraform"
)

func mustShimNewState(newState *states.State, schemas *terraform.Schemas) *terraform.State {
	s, err := shimNewState(newState, schemas)
	if err != nil {
		panic(err)
	}
	return s
}

// shimState takes a new *states.State and reverts it to a legacy state for the provider ACC tests
func shimNewState(newState *states.State, schemas *terraform.Schemas) (*terraform.State, error) {
	state := terraform.NewState()

	// in the odd case of a nil state, let the helper packages handle it
	if newState == nil {
		return nil, nil
	}

	for _, newMod := range newState.Modules {
		mod := state.AddModule(newMod.Addr)

		for name, out := range newMod.OutputValues {
			outputType := ""
			val := hcl2shim.ConfigValueFromHCL2(out.Value)
			ty := out.Value.Type()
			switch {
			case ty == cty.String:
				outputType = "string"
			case ty.IsTupleType() || ty.IsListType():
				outputType = "list"
			case ty.IsMapType():
				outputType = "map"
			}

			mod.Outputs[name] = &terraform.OutputState{
				Type:      outputType,
				Value:     val,
				Sensitive: out.Sensitive,
			}
		}

		for _, res := range newMod.Resources {
			resType := res.Addr.Type
			providerType := res.ProviderConfig.ProviderConfig.Type

			providerSchema := schemas.Providers[providerType]
			if providerSchema == nil {
				return nil, fmt.Errorf("missing schema for %q", providerType)
			}

			var resSchema *configschema.Block
			switch res.Addr.Mode {
			case addrs.ManagedResourceMode:
				resSchema = providerSchema.ResourceTypes[resType]
			case addrs.DataResourceMode:
				resSchema = providerSchema.DataSources[resType]
			}

			if resSchema == nil {
				return nil, fmt.Errorf("missing resource schema for %q in %q", resType, providerType)
			}

			for key, i := range res.Instances {
				flatmap, err := shimmedAttributes(i.Current, resSchema.ImpliedType())
				if err != nil {
					return nil, fmt.Errorf("error decoding state for %q: %s", resType, err)
				}

				resState := &terraform.ResourceState{
					Type: resType,
					Primary: &terraform.InstanceState{
						ID:         flatmap["id"],
						Attributes: flatmap,
						Tainted:    i.Current.Status == states.ObjectTainted,
					},
				}
				if i.Current.SchemaVersion != 0 {
					resState.Primary.Meta = map[string]interface{}{
						"schema_version": i.Current.SchemaVersion,
					}
				}

				for _, dep := range i.Current.Dependencies {
					resState.Dependencies = append(resState.Dependencies, dep.String())
				}

				// convert the indexes to the old style flapmap indexes
				idx := ""
				switch key.(type) {
				case addrs.IntKey:
					// don't add numeric index values to resources with a count of 0
					if len(res.Instances) > 1 {
						idx = fmt.Sprintf(".%d", key)
					}
				case addrs.StringKey:
					idx = "." + key.String()
				}

				mod.Resources[res.Addr.String()+idx] = resState

				// add any deposed instances
				for _, dep := range i.Deposed {
					flatmap, err := shimmedAttributes(dep, resSchema.ImpliedType())
					if err != nil {
						return nil, fmt.Errorf("error decoding deposed state for %q: %s", resType, err)
					}

					deposed := &terraform.InstanceState{
						ID:         flatmap["id"],
						Attributes: flatmap,
						Tainted:    dep.Status == states.ObjectTainted,
					}
					if dep.SchemaVersion != 0 {
						deposed.Meta = map[string]interface{}{
							"schema_version": dep.SchemaVersion,
						}
					}

					resState.Deposed = append(resState.Deposed, deposed)
				}
			}
		}
	}

	return state, nil
}

func shimmedAttributes(instance *states.ResourceInstanceObjectSrc, ty cty.Type) (map[string]string, error) {
	flatmap := instance.AttrsFlat

	// if we have json attrs, they need to be decoded
	if flatmap == nil {
		rio, err := instance.Decode(ty)
		if err != nil {
			return nil, err
		}

		flatmap = hcl2shim.FlatmapValueFromHCL2(rio.Value)
	}
	return flatmap, nil
}

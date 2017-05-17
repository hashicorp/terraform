package terraform

import (
	"fmt"
	"os"
	"strings"

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/hilmapstructure"
)

// Variables returns the fully loaded set of variables to use with
// ContextOpts and NewContext, loading any additional variables from
// the environment or any other sources.
//
// The given module tree doesn't need to be loaded.
func Variables(
	m *module.Tree,
	override map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Variables are loaded in the following sequence. Each additional step
	// will override conflicting variable keys from prior steps:
	//
	//   * Take default values from config
	//   * Take values from TF_VAR_x env vars
	//   * Take values specified in the "override" param which is usually
	//     from -var, -var-file, etc.
	//

	// First load from the config
	for _, v := range m.Config().Variables {
		// If the var has no default, ignore
		if v.Default == nil {
			continue
		}

		// If the type isn't a string, we use it as-is since it is a rich type
		if v.Type() != config.VariableTypeString {
			result[v.Name] = v.Default
			continue
		}

		// v.Default has already been parsed as HCL but it may be an int type
		switch typedDefault := v.Default.(type) {
		case string:
			if typedDefault == "" {
				continue
			}
			result[v.Name] = typedDefault
		case int, int64:
			result[v.Name] = fmt.Sprintf("%d", typedDefault)
		case float32, float64:
			result[v.Name] = fmt.Sprintf("%f", typedDefault)
		case bool:
			result[v.Name] = fmt.Sprintf("%t", typedDefault)
		default:
			panic(fmt.Sprintf(
				"Unknown default var type: %T\n\n"+
					"THIS IS A BUG. Please report it.",
				v.Default))
		}
	}

	// Load from env vars
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, VarEnvPrefix) {
			continue
		}

		// Strip off the prefix and get the value after the first "="
		idx := strings.Index(v, "=")
		k := v[len(VarEnvPrefix):idx]
		v = v[idx+1:]

		// Override the configuration-default values. Note that *not* finding the variable
		// in configuration is OK, as we don't want to preclude people from having multiple
		// sets of TF_VAR_whatever in their environment even if it is a little weird.
		for _, schema := range m.Config().Variables {
			if schema.Name != k {
				continue
			}

			varType := schema.Type()
			varVal, err := parseVariableAsHCL(k, v, varType)
			if err != nil {
				return nil, err
			}

			switch varType {
			case config.VariableTypeMap:
				if err := varSetMap(result, k, varVal); err != nil {
					return nil, err
				}
			default:
				result[k] = varVal
			}
		}
	}

	// Load from overrides
	for k, v := range override {
		for _, schema := range m.Config().Variables {
			if schema.Name != k {
				continue
			}

			switch schema.Type() {
			case config.VariableTypeList:
				result[k] = v
			case config.VariableTypeMap:
				if err := varSetMap(result, k, v); err != nil {
					return nil, err
				}
			case config.VariableTypeString:
				// Convert to a string and set. We don't catch any errors
				// here because the validation step later should catch
				// any type errors.
				var strVal string
				if err := hilmapstructure.WeakDecode(v, &strVal); err == nil {
					result[k] = strVal
				} else {
					result[k] = v
				}
			default:
				panic(fmt.Sprintf(
					"Unhandled var type: %T\n\n"+
						"THIS IS A BUG. Please report it.",
					schema.Type()))
			}
		}
	}

	return result, nil
}

// varSetMap sets or merges the map in "v" with the key "k" in the
// "current" set of variables. This is just a private function to remove
// duplicate logic in Variables
func varSetMap(current map[string]interface{}, k string, v interface{}) error {
	existing, ok := current[k]
	if !ok {
		current[k] = v
		return nil
	}

	existingMap, ok := existing.(map[string]interface{})
	if !ok {
		panic(fmt.Sprintf("%q is not a map, this is a bug in Terraform.", k))
	}

	switch typedV := v.(type) {
	case []map[string]interface{}:
		for newKey, newVal := range typedV[0] {
			existingMap[newKey] = newVal
		}
	case map[string]interface{}:
		for newKey, newVal := range typedV {
			existingMap[newKey] = newVal
		}
	default:
		return fmt.Errorf("variable %q should be type map, got %s", k, hclTypeName(v))
	}
	return nil
}

package config

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/ast"
)

// hclConfigurable is an implementation of configurable that knows
// how to turn HCL configuration into a *Config object.
type hclConfigurable struct {
	File   string
	Object *ast.ObjectNode
}

func (t *hclConfigurable) Config() (*Config, error) {
	validKeys := map[string]struct{}{
		"output":   struct{}{},
		"provider": struct{}{},
		"resource": struct{}{},
		"variable": struct{}{},
	}

	type hclVariable struct {
		Default     interface{}
		Description string
		Fields      []string `hcl:",decodedFields"`
	}

	var rawConfig struct {
		Variable map[string]*hclVariable
	}

	if err := hcl.DecodeAST(&rawConfig, t.Object); err != nil {
		return nil, err
	}

	// Start building up the actual configuration. We start with
	// variables.
	// TODO(mitchellh): Make function like loadVariablesHcl so that
	// duplicates aren't overriden
	config := new(Config)
	if len(rawConfig.Variable) > 0 {
		config.Variables = make([]*Variable, 0, len(rawConfig.Variable))
		for k, v := range rawConfig.Variable {
			// Defaults turn into a slice of map[string]interface{} and
			// we need to make sure to convert that down into the
			// proper type for Config.
			if ms, ok := v.Default.([]map[string]interface{}); ok {
				def := make(map[string]interface{})
				for _, m := range ms {
					for k, v := range m {
						def[k] = v
					}
				}

				v.Default = def
			}

			newVar := &Variable{
				Name:        k,
				Default:     v.Default,
				Description: v.Description,
			}

			config.Variables = append(config.Variables, newVar)
		}
	}

	// Build the provider configs
	if providers := t.Object.Get("provider", false); providers != nil {
		var err error
		config.ProviderConfigs, err = loadProvidersHcl(providers)
		if err != nil {
			return nil, err
		}
	}

	// Build the resources
	if resources := t.Object.Get("resource", false); resources != nil {
		var err error
		config.Resources, err = loadResourcesHcl(resources)
		if err != nil {
			return nil, err
		}
	}

	// Build the outputs
	if outputs := t.Object.Get("output", false); outputs != nil {
		var err error
		config.Outputs, err = loadOutputsHcl(outputs)
		if err != nil {
			return nil, err
		}
	}

	// Check for invalid keys
	for _, elem := range t.Object.Elem {
		k := elem.Key()
		if _, ok := validKeys[k]; ok {
			continue
		}

		config.unknownKeys = append(config.unknownKeys, k)
	}

	return config, nil
}

// loadFileHcl is a fileLoaderFunc that knows how to read HCL
// files and turn them into hclConfigurables.
func loadFileHcl(root string) (configurable, []string, error) {
	var obj *ast.ObjectNode = nil

	// Read the HCL file and prepare for parsing
	d, err := ioutil.ReadFile(root)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"Error reading %s: %s", root, err)
	}

	// Parse it
	obj, err = hcl.Parse(string(d))
	if err != nil {
		return nil, nil, fmt.Errorf(
			"Error parsing %s: %s", root, err)
	}

	// Start building the result
	result := &hclConfigurable{
		File:   root,
		Object: obj,
	}

	// Dive in, find the imports. This is disabled for now since
	// imports were removed prior to Terraform 0.1. The code is
	// remaining here commented for historical purposes.
	/*
		imports := obj.Get("import")
		if imports == nil {
			result.Object.Ref()
			return result, nil, nil
		}

		if imports.Type() != libucl.ObjectTypeString {
			imports.Close()

			return nil, nil, fmt.Errorf(
				"Error in %s: all 'import' declarations should be in the format\n"+
					"`import \"foo\"` (Got type %s)",
				root,
				imports.Type())
		}

		// Gather all the import paths
		importPaths := make([]string, 0, imports.Len())
		iter := imports.Iterate(false)
		for imp := iter.Next(); imp != nil; imp = iter.Next() {
			path := imp.ToString()
			if !filepath.IsAbs(path) {
				// Relative paths are relative to the Terraform file itself
				dir := filepath.Dir(root)
				path = filepath.Join(dir, path)
			}

			importPaths = append(importPaths, path)
			imp.Close()
		}
		iter.Close()
		imports.Close()

		result.Object.Ref()
	*/

	return result, nil, nil
}

// LoadOutputsHcl recurses into the given HCL object and turns
// it into a mapping of outputs.
func loadOutputsHcl(ns []ast.Node) ([]*Output, error) {
	objects := hclObjectMap(ns)
	if len(objects) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Output, 0, len(objects))
	for n, o := range objects {
		var config map[string]interface{}

		if err := hcl.DecodeAST(&config, o); err != nil {
			return nil, err
		}

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for output %s: %s",
				n,
				err)
		}

		result = append(result, &Output{
			Name:      n,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}

// LoadProvidersHcl recurses into the given HCL object and turns
// it into a mapping of provider configs.
func loadProvidersHcl(ns []ast.Node) ([]*ProviderConfig, error) {
	objects := hclObjectMap(ns)
	if len(objects) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*ProviderConfig, 0, len(objects))
	for n, o := range objects {
		var config map[string]interface{}

		if err := hcl.DecodeAST(&config, o); err != nil {
			return nil, err
		}

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for provider config %s: %s",
				n,
				err)
		}

		result = append(result, &ProviderConfig{
			Name:      n,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}

// Given a handle to a HCL object, this recurses into the structure
// and pulls out a list of resources.
//
// The resulting resources may not be unique, but each resource
// represents exactly one resource definition in the HCL configuration.
// We leave it up to another pass to merge them together.
func loadResourcesHcl(ns []ast.Node) ([]*Resource, error) {
	typeMap := hclObjectMap(ns)

	// Where all the results will go
	var result []*Resource

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for t, rs := range typeMap {
		resourceMap := hclObjectMap([]ast.Node{rs})
		for k, o := range resourceMap {
			for _, o := range o.Elem {
				obj, ok := o.(ast.ObjectNode)
				if !ok {
					continue
				}

				var config map[string]interface{}
				if err := hcl.DecodeAST(&config, o); err != nil {
					return nil, fmt.Errorf(
						"Error reading config for %s[%s]: %s",
						t,
						k,
						err)
				}

				// Remove the fields we handle specially
				delete(config, "connection")
				delete(config, "count")
				delete(config, "depends_on")
				delete(config, "provisioner")

				rawConfig, err := NewRawConfig(config)
				if err != nil {
					return nil, fmt.Errorf(
						"Error reading config for %s[%s]: %s",
						t,
						k,
						err)
				}

				// If we have a count, then figure it out
				var count int = 1
				if os := obj.Get("count", false); os != nil {
					for _, o := range os {
						err = hcl.DecodeAST(&count, o)
						if err != nil {
							return nil, fmt.Errorf(
								"Error parsing count for %s[%s]: %s",
								t,
								k,
								err)
						}
					}
				}

				// If we have depends fields, then add those in
				var dependsOn []string
				if os := obj.Get("depends_on", false); os != nil {
					for _, o := range os {
						err := hcl.DecodeAST(&dependsOn, o)
						if err != nil {
							return nil, fmt.Errorf(
								"Error reading depends_on for %s[%s]: %s",
								t,
								k,
								err)
						}
					}
				}

				// If we have connection info, then parse those out
				var connInfo map[string]interface{}
				if os := obj.Get("connection", false); os != nil {
					for _, o := range os {
						err := hcl.DecodeAST(&connInfo, o)
						if err != nil {
							return nil, fmt.Errorf(
								"Error reading connection info for %s[%s]: %s",
								t,
								k,
								err)
						}
					}
				}

				// If we have provisioners, then parse those out
				var provisioners []*Provisioner
				if os := obj.Get("provisioner", false); os != nil {
					var err error
					provisioners, err = loadProvisionersHcl(os, connInfo)
					if err != nil {
						return nil, fmt.Errorf(
							"Error reading provisioners for %s[%s]: %s",
							t,
							k,
							err)
					}
				}

				result = append(result, &Resource{
					Name:         k,
					Type:         t,
					Count:        count,
					RawConfig:    rawConfig,
					Provisioners: provisioners,
					DependsOn:    dependsOn,
				})
			}
		}
	}

	return result, nil
}

func loadProvisionersHcl(ns []ast.Node, connInfo map[string]interface{}) ([]*Provisioner, error) {
	pos := make([]ast.AssignmentNode, 0, len(ns))

	// Accumulate all the actual provisioner configuration objects. We
	// have to iterate twice here:
	//
	//  1. The first iteration is of the list of `provisioner` blocks.
	//  2. The second iteration is of the dictionary within the
	//      provisioner which will have only one element which is the
	//      type of provisioner to use along with tis config.
	//
	// In JSON it looks kind of like this:
	//
	//   [
	//     {
	//       "shell": {
	//         ...
	//       }
	//     }
	//   ]
	//
	for _, n := range ns {
		obj, ok := n.(ast.ObjectNode)
		if !ok {
			continue
		}

		for _, elem := range obj.Elem {
			pos = append(pos, elem)
		}
	}

	// Short-circuit if there are no items
	if len(pos) == 0 {
		return nil, nil
	}

	result := make([]*Provisioner, 0, len(pos))
	for _, po := range pos {
		obj, ok := po.Value.(ast.ObjectNode)
		if !ok {
			continue
		}

		var config map[string]interface{}
		if err := hcl.DecodeAST(&config, obj); err != nil {
			return nil, err
		}

		// Delete the "connection" section, handle seperately
		delete(config, "connection")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, err
		}

		// Check if we have a provisioner-level connection
		// block that overrides the resource-level
		var subConnInfo map[string]interface{}
		if os := obj.Get("connection", false); os != nil {
			for _, o := range os {
				err := hcl.DecodeAST(&subConnInfo, o)
				if err != nil {
					return nil, err
				}
			}
		}

		// Inherit from the resource connInfo any keys
		// that are not explicitly overriden.
		if connInfo != nil && subConnInfo != nil {
			for k, v := range connInfo {
				if _, ok := subConnInfo[k]; !ok {
					subConnInfo[k] = v
				}
			}
		} else if subConnInfo == nil {
			subConnInfo = connInfo
		}

		// Parse the connInfo
		connRaw, err := NewRawConfig(subConnInfo)
		if err != nil {
			return nil, err
		}

		result = append(result, &Provisioner{
			Type:      po.Key(),
			RawConfig: rawConfig,
			ConnInfo:  connRaw,
		})
	}

	return result, nil
}

func hclObjectMap(ns []ast.Node) map[string]ast.ListNode {
	objects := make(map[string]ast.ListNode)

	for _, n := range ns {
		ns := []ast.Node{n}
		if ln, ok := n.(ast.ListNode); ok {
			ns = ln.Elem
		}

		for _, n := range ns {
			obj, ok := n.(ast.ObjectNode)
			if !ok {
				continue
			}

			for _, elem := range obj.Elem {
				val, ok := objects[elem.Key()]
				if !ok {
					val = ast.ListNode{}
				}

				val.Elem = append(val.Elem, elem.Value)
				objects[elem.Key()] = val
			}
		}
	}

	return objects
}

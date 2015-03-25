package config

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	hclobj "github.com/hashicorp/hcl/hcl"
)

// hclConfigurable is an implementation of configurable that knows
// how to turn HCL configuration into a *Config object.
type hclConfigurable struct {
	File   string
	Object *hclobj.Object
}

func (t *hclConfigurable) Config() (*Config, error) {
	validKeys := map[string]struct{}{
		"atlas":    struct{}{},
		"module":   struct{}{},
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

	if err := hcl.DecodeObject(&rawConfig, t.Object); err != nil {
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

	// Get Atlas configuration
	if atlas := t.Object.Get("atlas", false); atlas != nil {
		var err error
		config.Atlas, err = loadAtlasHcl(atlas)
		if err != nil {
			return nil, err
		}
	}

	// Build the modules
	if modules := t.Object.Get("module", false); modules != nil {
		var err error
		config.Modules, err = loadModulesHcl(modules)
		if err != nil {
			return nil, err
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
	for _, elem := range t.Object.Elem(true) {
		k := elem.Key
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
	var obj *hclobj.Object = nil

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

// Given a handle to a HCL object, this transforms it into the Atlas
// configuration.
func loadAtlasHcl(obj *hclobj.Object) (*AtlasConfig, error) {
	var config AtlasConfig
	if err := hcl.DecodeObject(&config, obj); err != nil {
		return nil, fmt.Errorf(
			"Error reading atlas config: %s",
			err)
	}

	return &config, nil
}

// Given a handle to a HCL object, this recurses into the structure
// and pulls out a list of modules.
//
// The resulting modules may not be unique, but each module
// represents exactly one module definition in the HCL configuration.
// We leave it up to another pass to merge them together.
func loadModulesHcl(os *hclobj.Object) ([]*Module, error) {
	var allNames []*hclobj.Object

	// See loadResourcesHcl for why this exists. Don't touch this.
	for _, o1 := range os.Elem(false) {
		// Iterate the inner to get the list of types
		for _, o2 := range o1.Elem(true) {
			// Iterate all of this type to get _all_ the types
			for _, o3 := range o2.Elem(false) {
				allNames = append(allNames, o3)
			}
		}
	}

	// Where all the results will go
	var result []*Module

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, obj := range allNames {
		k := obj.Key

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, obj); err != nil {
			return nil, fmt.Errorf(
				"Error reading config for %s: %s",
				k,
				err)
		}

		// Remove the fields we handle specially
		delete(config, "source")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for %s: %s",
				k,
				err)
		}

		// If we have a count, then figure it out
		var source string
		if o := obj.Get("source", false); o != nil {
			err = hcl.DecodeObject(&source, o)
			if err != nil {
				return nil, fmt.Errorf(
					"Error parsing source for %s: %s",
					k,
					err)
			}
		}

		result = append(result, &Module{
			Name:      k,
			Source:    source,
			RawConfig: rawConfig,
		})
	}

	return result, nil
}

// LoadOutputsHcl recurses into the given HCL object and turns
// it into a mapping of outputs.
func loadOutputsHcl(os *hclobj.Object) ([]*Output, error) {
	objects := make(map[string]*hclobj.Object)

	// Iterate over all the "output" blocks and get the keys along with
	// their raw configuration objects. We'll parse those later.
	for _, o1 := range os.Elem(false) {
		for _, o2 := range o1.Elem(true) {
			objects[o2.Key] = o2
		}
	}

	if len(objects) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Output, 0, len(objects))
	for n, o := range objects {
		var config map[string]interface{}

		if err := hcl.DecodeObject(&config, o); err != nil {
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
func loadProvidersHcl(os *hclobj.Object) ([]*ProviderConfig, error) {
	objects := make(map[string]*hclobj.Object)

	// Iterate over all the "provider" blocks and get the keys along with
	// their raw configuration objects. We'll parse those later.
	for _, o1 := range os.Elem(false) {
		for _, o2 := range o1.Elem(true) {
			objects[o2.Key] = o2
		}
	}

	if len(objects) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*ProviderConfig, 0, len(objects))
	for n, o := range objects {
		var config map[string]interface{}

		if err := hcl.DecodeObject(&config, o); err != nil {
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
func loadResourcesHcl(os *hclobj.Object) ([]*Resource, error) {
	var allTypes []*hclobj.Object

	// HCL object iteration is really nasty. Below is likely to make
	// no sense to anyone approaching this code. Luckily, it is very heavily
	// tested. If working on a bug fix or feature, we recommend writing a
	// test first then doing whatever you want to the code below. If you
	// break it, the tests will catch it. Likewise, if you change this,
	// MAKE SURE you write a test for your change, because its fairly impossible
	// to reason about this mess.
	//
	// Functionally, what the code does below is get the libucl.Objects
	// for all the TYPES, such as "aws_security_group".
	for _, o1 := range os.Elem(false) {
		// Iterate the inner to get the list of types
		for _, o2 := range o1.Elem(true) {
			// Iterate all of this type to get _all_ the types
			for _, o3 := range o2.Elem(false) {
				allTypes = append(allTypes, o3)
			}
		}
	}

	// Where all the results will go
	var result []*Resource

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, t := range allTypes {
		for _, obj := range t.Elem(true) {
			k := obj.Key

			var config map[string]interface{}
			if err := hcl.DecodeObject(&config, obj); err != nil {
				return nil, fmt.Errorf(
					"Error reading config for %s[%s]: %s",
					t.Key,
					k,
					err)
			}

			// Remove the fields we handle specially
			delete(config, "connection")
			delete(config, "count")
			delete(config, "depends_on")
			delete(config, "provisioner")
			delete(config, "lifecycle")

			rawConfig, err := NewRawConfig(config)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading config for %s[%s]: %s",
					t.Key,
					k,
					err)
			}

			// If we have a count, then figure it out
			var count string = "1"
			if o := obj.Get("count", false); o != nil {
				err = hcl.DecodeObject(&count, o)
				if err != nil {
					return nil, fmt.Errorf(
						"Error parsing count for %s[%s]: %s",
						t.Key,
						k,
						err)
				}
			}
			countConfig, err := NewRawConfig(map[string]interface{}{
				"count": count,
			})
			if err != nil {
				return nil, err
			}
			countConfig.Key = "count"

			// If we have depends fields, then add those in
			var dependsOn []string
			if o := obj.Get("depends_on", false); o != nil {
				err := hcl.DecodeObject(&dependsOn, o)
				if err != nil {
					return nil, fmt.Errorf(
						"Error reading depends_on for %s[%s]: %s",
						t.Key,
						k,
						err)
				}
			}

			// If we have connection info, then parse those out
			var connInfo map[string]interface{}
			if o := obj.Get("connection", false); o != nil {
				err := hcl.DecodeObject(&connInfo, o)
				if err != nil {
					return nil, fmt.Errorf(
						"Error reading connection info for %s[%s]: %s",
						t.Key,
						k,
						err)
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
						t.Key,
						k,
						err)
				}
			}

			// Check if the resource should be re-created before
			// destroying the existing instance
			var lifecycle ResourceLifecycle
			if o := obj.Get("lifecycle", false); o != nil {
				err = hcl.DecodeObject(&lifecycle, o)
				if err != nil {
					return nil, fmt.Errorf(
						"Error parsing lifecycle for %s[%s]: %s",
						t.Key,
						k,
						err)
				}
			}

			result = append(result, &Resource{
				Name:         k,
				Type:         t.Key,
				RawCount:     countConfig,
				RawConfig:    rawConfig,
				Provisioners: provisioners,
				DependsOn:    dependsOn,
				Lifecycle:    lifecycle,
			})
		}
	}

	return result, nil
}

func loadProvisionersHcl(os *hclobj.Object, connInfo map[string]interface{}) ([]*Provisioner, error) {
	pos := make([]*hclobj.Object, 0, int(os.Len()))

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
	for _, o1 := range os.Elem(false) {
		for _, o2 := range o1.Elem(true) {

			switch o1.Type {
			case hclobj.ValueTypeList:
				for _, o3 := range o2.Elem(true) {
					pos = append(pos, o3)
				}
			case hclobj.ValueTypeObject:
				pos = append(pos, o2)
			}
		}
	}

	// Short-circuit if there are no items
	if len(pos) == 0 {
		return nil, nil
	}

	result := make([]*Provisioner, 0, len(pos))
	for _, po := range pos {
		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, po); err != nil {
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
		if o := po.Get("connection", false); o != nil {
			err := hcl.DecodeObject(&subConnInfo, o)
			if err != nil {
				return nil, err
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
			Type:      po.Key,
			RawConfig: rawConfig,
			ConnInfo:  connRaw,
		})
	}

	return result, nil
}

/*
func hclObjectMap(os *hclobj.Object) map[string]ast.ListNode {
	objects := make(map[string][]*hclobj.Object)

	for _, o := range os.Elem(false) {
		for _, elem := range o.Elem(true) {
			val, ok := objects[elem.Key]
			if !ok {
				val = make([]*hclobj.Object, 0, 1)
			}

			val = append(val, elem)
			objects[elem.Key] = val
		}
	}

	return objects
}
*/

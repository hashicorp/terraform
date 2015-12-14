package config

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
	"github.com/mitchellh/mapstructure"
)

// hclConfigurable is an implementation of configurable that knows
// how to turn HCL configuration into a *Config object.
type hclConfigurable struct {
	File string
	Root *ast.File
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

	// Top-level item should be the object list
	list, ok := t.Root.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("error parsing: file doesn't contain a root object")
	}

	if err := hcl.DecodeObject(&rawConfig, list); err != nil {
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
	if atlas := list.Filter("atlas"); len(atlas.Items) > 0 {
		var err error
		config.Atlas, err = loadAtlasHcl(atlas)
		if err != nil {
			return nil, err
		}
	}

	// Build the modules
	if modules := list.Filter("module"); len(modules.Items) > 0 {
		var err error
		config.Modules, err = loadModulesHcl(modules)
		if err != nil {
			return nil, err
		}
	}

	// Build the provider configs
	if providers := list.Filter("provider"); len(providers.Items) > 0 {
		var err error
		config.ProviderConfigs, err = loadProvidersHcl(providers)
		if err != nil {
			return nil, err
		}
	}

	// Build the resources
	if resources := list.Filter("resource"); len(resources.Items) > 0 {
		var err error
		config.Resources, err = loadResourcesHcl(resources)
		if err != nil {
			return nil, err
		}
	}

	// Build the outputs
	if outputs := list.Filter("output"); len(outputs.Items) > 0 {
		var err error
		config.Outputs, err = loadOutputsHcl(outputs)
		if err != nil {
			return nil, err
		}
	}

	// Check for invalid keys
	for _, item := range list.Items {
		if len(item.Keys) == 0 {
			// Not sure how this would happen, but let's avoid a panic
			continue
		}

		k := item.Keys[0].Token.Value().(string)
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
	// Read the HCL file and prepare for parsing
	d, err := ioutil.ReadFile(root)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"Error reading %s: %s", root, err)
	}

	// Parse it
	hclRoot, err := hcl.Parse(string(d))
	if err != nil {
		return nil, nil, fmt.Errorf(
			"Error parsing %s: %s", root, err)
	}

	// Start building the result
	result := &hclConfigurable{
		File: root,
		Root: hclRoot,
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
func loadAtlasHcl(list *ast.ObjectList) (*AtlasConfig, error) {
	if len(list.Items) > 1 {
		return nil, fmt.Errorf("only one 'atlas' block allowed")
	}

	// Get our one item
	item := list.Items[0]

	var config AtlasConfig
	if err := hcl.DecodeObject(&config, item.Val); err != nil {
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
func loadModulesHcl(list *ast.ObjectList) ([]*Module, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Where all the results will go
	var result []*Module

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, item := range list.Items {
		k := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("module '%s': should be an object", k)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
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
		if o := listVal.Filter("source"); len(o.Items) > 0 {
			err = hcl.DecodeObject(&source, o.Items[0].Val)
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
func loadOutputsHcl(list *ast.ObjectList) ([]*Output, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Output, 0, len(list.Items))
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
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
func loadProvidersHcl(list *ast.ObjectList) ([]*ProviderConfig, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*ProviderConfig, 0, len(list.Items))
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("module '%s': should be an object", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		delete(config, "alias")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for provider config %s: %s",
				n,
				err)
		}

		// If we have an alias field, then add those in
		var alias string
		if a := listVal.Filter("alias"); len(a.Items) > 0 {
			err := hcl.DecodeObject(&alias, a.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading alias for provider[%s]: %s",
					n,
					err)
			}
		}

		result = append(result, &ProviderConfig{
			Name:      n,
			Alias:     alias,
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
func loadResourcesHcl(list *ast.ObjectList) ([]*Resource, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Where all the results will go
	var result []*Resource

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, item := range list.Items {
		if len(item.Keys) != 2 {
			return nil, fmt.Errorf(
				"position %s: resource must be followed by exactly two strings, a type and a name",
				item.Pos())
		}

		t := item.Keys[0].Token.Value().(string)
		k := item.Keys[1].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("resources %s[%s]: should be an object", t, k)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
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
		delete(config, "provider")
		delete(config, "lifecycle")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for %s[%s]: %s",
				t,
				k,
				err)
		}

		// If we have a count, then figure it out
		var count string = "1"
		if o := listVal.Filter("count"); len(o.Items) > 0 {
			err = hcl.DecodeObject(&count, o.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error parsing count for %s[%s]: %s",
					t,
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
		if o := listVal.Filter("depends_on"); len(o.Items) > 0 {
			err := hcl.DecodeObject(&dependsOn, o.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading depends_on for %s[%s]: %s",
					t,
					k,
					err)
			}
		}

		// If we have connection info, then parse those out
		var connInfo map[string]interface{}
		if o := listVal.Filter("connection"); len(o.Items) > 0 {
			err := hcl.DecodeObject(&connInfo, o.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading connection info for %s[%s]: %s",
					t,
					k,
					err)
			}
		}

		// If we have provisioners, then parse those out
		var provisioners []*Provisioner
		if os := listVal.Filter("provisioner"); len(os.Items) > 0 {
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

		// If we have a provider, then parse it out
		var provider string
		if o := listVal.Filter("provider"); len(o.Items) > 0 {
			err := hcl.DecodeObject(&provider, o.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading provider for %s[%s]: %s",
					t,
					k,
					err)
			}
		}

		// Check if the resource should be re-created before
		// destroying the existing instance
		var lifecycle ResourceLifecycle
		if o := listVal.Filter("lifecycle"); len(o.Items) > 0 {
			var raw map[string]interface{}
			if err = hcl.DecodeObject(&raw, o.Items[0].Val); err != nil {
				return nil, fmt.Errorf(
					"Error parsing lifecycle for %s[%s]: %s",
					t,
					k,
					err)
			}

			if err := mapstructure.WeakDecode(raw, &lifecycle); err != nil {
				return nil, fmt.Errorf(
					"Error parsing lifecycle for %s[%s]: %s",
					t,
					k,
					err)
			}
		}

		result = append(result, &Resource{
			Name:         k,
			Type:         t,
			RawCount:     countConfig,
			RawConfig:    rawConfig,
			Provisioners: provisioners,
			Provider:     provider,
			DependsOn:    dependsOn,
			Lifecycle:    lifecycle,
		})
	}

	return result, nil
}

func loadProvisionersHcl(list *ast.ObjectList, connInfo map[string]interface{}) ([]*Provisioner, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Provisioner, 0, len(list.Items))
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("provisioner '%s': should be an object", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		// Delete the "connection" section, handle separately
		delete(config, "connection")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, err
		}

		// Check if we have a provisioner-level connection
		// block that overrides the resource-level
		var subConnInfo map[string]interface{}
		if o := listVal.Filter("connection"); len(o.Items) > 0 {
			err := hcl.DecodeObject(&subConnInfo, o.Items[0].Val)
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
			Type:      n,
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

package config

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/go-multierror"
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
		"atlas":     struct{}{},
		"data":      struct{}{},
		"module":    struct{}{},
		"output":    struct{}{},
		"provider":  struct{}{},
		"resource":  struct{}{},
		"terraform": struct{}{},
		"variable":  struct{}{},
	}

	// Top-level item should be the object list
	list, ok := t.Root.Node.(*ast.ObjectList)
	if !ok {
		return nil, fmt.Errorf("error parsing: file doesn't contain a root object")
	}

	// Start building up the actual configuration.
	config := new(Config)

	// Terraform config
	if o := list.Filter("terraform"); len(o.Items) > 0 {
		var err error
		config.Terraform, err = loadTerraformHcl(o)
		if err != nil {
			return nil, err
		}
	}

	// Build the variables
	if vars := list.Filter("variable"); len(vars.Items) > 0 {
		var err error
		config.Variables, err = loadVariablesHcl(vars)
		if err != nil {
			return nil, err
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
	{
		var err error
		managedResourceConfigs := list.Filter("resource")
		dataResourceConfigs := list.Filter("data")

		config.Resources = make(
			[]*Resource, 0,
			len(managedResourceConfigs.Items)+len(dataResourceConfigs.Items),
		)

		managedResources, err := loadManagedResourcesHcl(managedResourceConfigs)
		if err != nil {
			return nil, err
		}
		dataResources, err := loadDataResourcesHcl(dataResourceConfigs)
		if err != nil {
			return nil, err
		}

		config.Resources = append(config.Resources, dataResources...)
		config.Resources = append(config.Resources, managedResources...)
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

// Given a handle to a HCL object, this transforms it into the Terraform config
func loadTerraformHcl(list *ast.ObjectList) (*Terraform, error) {
	if len(list.Items) > 1 {
		return nil, fmt.Errorf("only one 'terraform' block allowed per module")
	}

	// Get our one item
	item := list.Items[0]

	// This block should have an empty top level ObjectItem.  If there are keys
	// here, it's likely because we have a flattened JSON object, and we can
	// lift this into a nested ObjectList to decode properly.
	if len(item.Keys) > 0 {
		item = &ast.ObjectItem{
			Val: &ast.ObjectType{
				List: &ast.ObjectList{
					Items: []*ast.ObjectItem{item},
				},
			},
		}
	}

	// We need the item value as an ObjectList
	var listVal *ast.ObjectList
	if ot, ok := item.Val.(*ast.ObjectType); ok {
		listVal = ot.List
	} else {
		return nil, fmt.Errorf("terraform block: should be an object")
	}

	// NOTE: We purposely don't validate unknown HCL keys here so that
	// we can potentially read _future_ Terraform version config (to
	// still be able to validate the required version).
	//
	// We should still keep track of unknown keys to validate later, but
	// HCL doesn't currently support that.

	var config Terraform
	if err := hcl.DecodeObject(&config, item.Val); err != nil {
		return nil, fmt.Errorf(
			"Error reading terraform config: %s",
			err)
	}

	// If we have provisioners, then parse those out
	if os := listVal.Filter("backend"); len(os.Items) > 0 {
		var err error
		config.Backend, err = loadTerraformBackendHcl(os)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading backend config for terraform block: %s",
				err)
		}
	}

	return &config, nil
}

// Loads the Backend configuration from an object list.
func loadTerraformBackendHcl(list *ast.ObjectList) (*Backend, error) {
	if len(list.Items) > 1 {
		return nil, fmt.Errorf("only one 'backend' block allowed")
	}

	// Get our one item
	item := list.Items[0]

	// Verify the keys
	if len(item.Keys) != 1 {
		return nil, fmt.Errorf(
			"position %s: 'backend' must be followed by exactly one string: a type",
			item.Pos())
	}

	typ := item.Keys[0].Token.Value().(string)

	// Decode the raw config
	var config map[string]interface{}
	if err := hcl.DecodeObject(&config, item.Val); err != nil {
		return nil, fmt.Errorf(
			"Error reading backend config: %s",
			err)
	}

	rawConfig, err := NewRawConfig(config)
	if err != nil {
		return nil, fmt.Errorf(
			"Error reading backend config: %s",
			err)
	}

	b := &Backend{
		Type:      typ,
		RawConfig: rawConfig,
	}
	b.Hash = b.Rehash()

	return b, nil
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
		return nil, fmt.Errorf(
			"'output' must be followed by exactly one string: a name")
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Output, 0, len(list.Items))
	for _, item := range list.Items {
		n := item.Keys[0].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("output '%s': should be an object", n)
		}

		var config map[string]interface{}
		if err := hcl.DecodeObject(&config, item.Val); err != nil {
			return nil, err
		}

		// Delete special keys
		delete(config, "depends_on")

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for output %s: %s",
				n,
				err)
		}

		// If we have depends fields, then add those in
		var dependsOn []string
		if o := listVal.Filter("depends_on"); len(o.Items) > 0 {
			err := hcl.DecodeObject(&dependsOn, o.Items[0].Val)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading depends_on for output %q: %s",
					n,
					err)
			}
		}

		result = append(result, &Output{
			Name:      n,
			RawConfig: rawConfig,
			DependsOn: dependsOn,
		})
	}

	return result, nil
}

// LoadVariablesHcl recurses into the given HCL object and turns
// it into a list of variables.
func loadVariablesHcl(list *ast.ObjectList) ([]*Variable, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, fmt.Errorf(
			"'variable' must be followed by exactly one strings: a name")
	}

	// hclVariable is the structure each variable is decoded into
	type hclVariable struct {
		DeclaredType string `hcl:"type"`
		Default      interface{}
		Description  string
		Fields       []string `hcl:",decodedFields"`
	}

	// Go through each object and turn it into an actual result.
	result := make([]*Variable, 0, len(list.Items))
	for _, item := range list.Items {
		// Clean up items from JSON
		unwrapHCLObjectKeysFromJSON(item, 1)

		// Verify the keys
		if len(item.Keys) != 1 {
			return nil, fmt.Errorf(
				"position %s: 'variable' must be followed by exactly one strings: a name",
				item.Pos())
		}

		n := item.Keys[0].Token.Value().(string)
		if !NameRegexp.MatchString(n) {
			return nil, fmt.Errorf(
				"position %s: 'variable' name must match regular expression: %s",
				item.Pos(), NameRegexp)
		}

		// Check for invalid keys
		valid := []string{"type", "default", "description"}
		if err := checkHCLKeys(item.Val, valid); err != nil {
			return nil, multierror.Prefix(err, fmt.Sprintf(
				"variable[%s]:", n))
		}

		// Decode into hclVariable to get typed values
		var hclVar hclVariable
		if err := hcl.DecodeObject(&hclVar, item.Val); err != nil {
			return nil, err
		}

		// Defaults turn into a slice of map[string]interface{} and
		// we need to make sure to convert that down into the
		// proper type for Config.
		if ms, ok := hclVar.Default.([]map[string]interface{}); ok {
			def := make(map[string]interface{})
			for _, m := range ms {
				for k, v := range m {
					def[k] = v
				}
			}

			hclVar.Default = def
		}

		// Build the new variable and do some basic validation
		newVar := &Variable{
			Name:         n,
			DeclaredType: hclVar.DeclaredType,
			Default:      hclVar.Default,
			Description:  hclVar.Description,
		}
		if err := newVar.ValidateTypeAndDefault(); err != nil {
			return nil, err
		}

		result = append(result, newVar)
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
// and pulls out a list of data sources.
//
// The resulting data sources may not be unique, but each one
// represents exactly one data definition in the HCL configuration.
// We leave it up to another pass to merge them together.
func loadDataResourcesHcl(list *ast.ObjectList) ([]*Resource, error) {
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
				"position %s: 'data' must be followed by exactly two strings: a type and a name",
				item.Pos())
		}

		t := item.Keys[0].Token.Value().(string)
		k := item.Keys[1].Token.Value().(string)

		var listVal *ast.ObjectList
		if ot, ok := item.Val.(*ast.ObjectType); ok {
			listVal = ot.List
		} else {
			return nil, fmt.Errorf("data sources %s[%s]: should be an object", t, k)
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
		delete(config, "depends_on")
		delete(config, "provider")
		delete(config, "count")

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

		result = append(result, &Resource{
			Mode:         DataResourceMode,
			Name:         k,
			Type:         t,
			RawCount:     countConfig,
			RawConfig:    rawConfig,
			Provider:     provider,
			Provisioners: []*Provisioner{},
			DependsOn:    dependsOn,
			Lifecycle:    ResourceLifecycle{},
		})
	}

	return result, nil
}

// Given a handle to a HCL object, this recurses into the structure
// and pulls out a list of managed resources.
//
// The resulting resources may not be unique, but each resource
// represents exactly one "resource" block in the HCL configuration.
// We leave it up to another pass to merge them together.
func loadManagedResourcesHcl(list *ast.ObjectList) ([]*Resource, error) {
	list = list.Children()
	if len(list.Items) == 0 {
		return nil, nil
	}

	// Where all the results will go
	var result []*Resource

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, item := range list.Items {
		// GH-4385: We detect a pure provisioner resource and give the user
		// an error about how to do it cleanly.
		if len(item.Keys) == 4 && item.Keys[2].Token.Value().(string) == "provisioner" {
			return nil, fmt.Errorf(
				"position %s: provisioners in a resource should be wrapped in a list\n\n"+
					"Example: \"provisioner\": [ { \"local-exec\": ... } ]",
				item.Pos())
		}

		// Fix up JSON input
		unwrapHCLObjectKeysFromJSON(item, 2)

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
			if len(o.Items) > 1 {
				return nil, fmt.Errorf(
					"%s[%s]: Multiple lifecycle blocks found, expected one",
					t, k)
			}

			// Check for invalid keys
			valid := []string{"create_before_destroy", "ignore_changes", "prevent_destroy"}
			if err := checkHCLKeys(o.Items[0].Val, valid); err != nil {
				return nil, multierror.Prefix(err, fmt.Sprintf(
					"%s[%s]:", t, k))
			}

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
			Mode:         ManagedResourceMode,
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

		// Parse the "when" value
		when := ProvisionerWhenCreate
		if v, ok := config["when"]; ok {
			switch v {
			case "create":
				when = ProvisionerWhenCreate
			case "destroy":
				when = ProvisionerWhenDestroy
			default:
				return nil, fmt.Errorf(
					"position %s: 'provisioner' when must be 'create' or 'destroy'",
					item.Pos())
			}
		}

		// Parse the "on_failure" value
		onFailure := ProvisionerOnFailureFail
		if v, ok := config["on_failure"]; ok {
			switch v {
			case "continue":
				onFailure = ProvisionerOnFailureContinue
			case "fail":
				onFailure = ProvisionerOnFailureFail
			default:
				return nil, fmt.Errorf(
					"position %s: 'provisioner' on_failure must be 'continue' or 'fail'",
					item.Pos())
			}
		}

		// Delete fields we special case
		delete(config, "connection")
		delete(config, "when")
		delete(config, "on_failure")

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
			When:      when,
			OnFailure: onFailure,
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

func checkHCLKeys(node ast.Node, valid []string) error {
	var list *ast.ObjectList
	switch n := node.(type) {
	case *ast.ObjectList:
		list = n
	case *ast.ObjectType:
		list = n.List
	default:
		return fmt.Errorf("cannot check HCL keys of type %T", n)
	}

	validMap := make(map[string]struct{}, len(valid))
	for _, v := range valid {
		validMap[v] = struct{}{}
	}

	var result error
	for _, item := range list.Items {
		key := item.Keys[0].Token.Value().(string)
		if _, ok := validMap[key]; !ok {
			result = multierror.Append(result, fmt.Errorf(
				"invalid key: %s", key))
		}
	}

	return result
}

// unwrapHCLObjectKeysFromJSON cleans up an edge case that can occur when
// parsing JSON as input: if we're parsing JSON then directly nested
// items will show up as additional "keys".
//
// For objects that expect a fixed number of keys, this breaks the
// decoding process. This function unwraps the object into what it would've
// looked like if it came directly from HCL by specifying the number of keys
// you expect.
//
// Example:
//
// { "foo": { "baz": {} } }
//
// Will show up with Keys being: []string{"foo", "baz"}
// when we really just want the first two. This function will fix this.
func unwrapHCLObjectKeysFromJSON(item *ast.ObjectItem, depth int) {
	if len(item.Keys) > depth && item.Keys[0].Token.JSON {
		for len(item.Keys) > depth {
			// Pop off the last key
			n := len(item.Keys)
			key := item.Keys[n-1]
			item.Keys[n-1] = nil
			item.Keys = item.Keys[:n-1]

			// Wrap our value in a list
			item.Val = &ast.ObjectType{
				List: &ast.ObjectList{
					Items: []*ast.ObjectItem{
						&ast.ObjectItem{
							Keys: []*ast.ObjectKey{key},
							Val:  item.Val,
						},
					},
				},
			}
		}
	}
}

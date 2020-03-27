package tfconfig

import (
	"io/ioutil"
	"strings"

	legacyhcl "github.com/hashicorp/hcl"
	legacyast "github.com/hashicorp/hcl/hcl/ast"
)

func loadModuleLegacyHCL(dir string) (*Module, Diagnostics) {
	// This implementation is intentionally more quick-and-dirty than the
	// main loader. In particular, it doesn't bother to keep careful track
	// of multiple error messages because we always fall back on returning
	// the main parser's error message if our fallback parsing produces
	// an error, and thus the errors here are not seen by the end-caller.
	mod := newModule(dir)

	primaryPaths, diags := dirFiles(dir)
	if diags.HasErrors() {
		return mod, diagnosticsHCL(diags)
	}

	for _, filename := range primaryPaths {
		src, err := ioutil.ReadFile(filename)
		if err != nil {
			return mod, diagnosticsErrorf("Error reading %s: %s", filename, err)
		}

		hclRoot, err := legacyhcl.Parse(string(src))
		if err != nil {
			return mod, diagnosticsErrorf("Error parsing %s: %s", filename, err)
		}

		list, ok := hclRoot.Node.(*legacyast.ObjectList)
		if !ok {
			return mod, diagnosticsErrorf("Error parsing %s: no root object", filename)
		}

		for _, item := range list.Filter("terraform").Items {
			if len(item.Keys) > 0 {
				item = &legacyast.ObjectItem{
					Val: &legacyast.ObjectType{
						List: &legacyast.ObjectList{
							Items: []*legacyast.ObjectItem{item},
						},
					},
				}
			}

			type TerraformBlock struct {
				RequiredVersion string `hcl:"required_version"`
			}
			var block TerraformBlock
			err = legacyhcl.DecodeObject(&block, item.Val)
			if err != nil {
				return nil, diagnosticsErrorf("terraform block: %s", err)
			}

			if block.RequiredVersion != "" {
				mod.RequiredCore = append(mod.RequiredCore, block.RequiredVersion)
			}
		}

		if vars := list.Filter("variable"); len(vars.Items) > 0 {
			vars = vars.Children()
			type VariableBlock struct {
				Type        string `hcl:"type"`
				Default     interface{}
				Description string
				Fields      []string `hcl:",decodedFields"`
			}

			for _, item := range vars.Items {
				unwrapLegacyHCLObjectKeysFromJSON(item, 1)

				if len(item.Keys) != 1 {
					return nil, diagnosticsErrorf("variable block at %s has no label", item.Pos())
				}

				name := item.Keys[0].Token.Value().(string)

				var block VariableBlock
				err := legacyhcl.DecodeObject(&block, item.Val)
				if err != nil {
					return nil, diagnosticsErrorf("invalid variable block at %s: %s", item.Pos(), err)
				}

				// Clean up legacy HCL decoding ambiguity by unwrapping list of maps
				if ms, ok := block.Default.([]map[string]interface{}); ok {
					def := make(map[string]interface{})
					for _, m := range ms {
						for k, v := range m {
							def[k] = v
						}
					}
					block.Default = def
				}

				v := &Variable{
					Name:        name,
					Type:        block.Type,
					Description: block.Description,
					Default:     block.Default,
					Pos:         sourcePosLegacyHCL(item.Pos(), filename),
				}
				if _, exists := mod.Variables[name]; exists {
					return nil, diagnosticsErrorf("duplicate variable block for %q", name)
				}
				mod.Variables[name] = v

			}
		}

		if outputs := list.Filter("output"); len(outputs.Items) > 0 {
			outputs = outputs.Children()
			type OutputBlock struct {
				Description string
			}

			for _, item := range outputs.Items {
				unwrapLegacyHCLObjectKeysFromJSON(item, 1)

				if len(item.Keys) != 1 {
					return nil, diagnosticsErrorf("output block at %s has no label", item.Pos())
				}

				name := item.Keys[0].Token.Value().(string)

				var block OutputBlock
				err := legacyhcl.DecodeObject(&block, item.Val)
				if err != nil {
					return nil, diagnosticsErrorf("invalid output block at %s: %s", item.Pos(), err)
				}

				o := &Output{
					Name:        name,
					Description: block.Description,
					Pos:         sourcePosLegacyHCL(item.Pos(), filename),
				}
				if _, exists := mod.Outputs[name]; exists {
					return nil, diagnosticsErrorf("duplicate output block for %q", name)
				}
				mod.Outputs[name] = o
			}
		}

		for _, blockType := range []string{"resource", "data"} {
			if resources := list.Filter(blockType); len(resources.Items) > 0 {
				resources = resources.Children()
				type ResourceBlock struct {
					Provider string
				}

				for _, item := range resources.Items {
					unwrapLegacyHCLObjectKeysFromJSON(item, 2)

					if len(item.Keys) != 2 {
						return nil, diagnosticsErrorf("resource block at %s has wrong label count", item.Pos())
					}

					typeName := item.Keys[0].Token.Value().(string)
					name := item.Keys[1].Token.Value().(string)
					var mode ResourceMode
					var rMap map[string]*Resource
					switch blockType {
					case "resource":
						mode = ManagedResourceMode
						rMap = mod.ManagedResources
					case "data":
						mode = DataResourceMode
						rMap = mod.DataResources
					}

					var block ResourceBlock
					err := legacyhcl.DecodeObject(&block, item.Val)
					if err != nil {
						return nil, diagnosticsErrorf("invalid resource block at %s: %s", item.Pos(), err)
					}

					var providerName, providerAlias string
					if dotPos := strings.IndexByte(block.Provider, '.'); dotPos != -1 {
						providerName = block.Provider[:dotPos]
						providerAlias = block.Provider[dotPos+1:]
					} else {
						providerName = block.Provider
					}
					if providerName == "" {
						providerName = resourceTypeDefaultProviderName(typeName)
					}

					r := &Resource{
						Mode: mode,
						Type: typeName,
						Name: name,
						Provider: ProviderRef{
							Name:  providerName,
							Alias: providerAlias,
						},
						Pos: sourcePosLegacyHCL(item.Pos(), filename),
					}
					key := r.MapKey()
					if _, exists := rMap[key]; exists {
						return nil, diagnosticsErrorf("duplicate resource block for %q", key)
					}
					rMap[key] = r
				}
			}

		}

		if moduleCalls := list.Filter("module"); len(moduleCalls.Items) > 0 {
			moduleCalls = moduleCalls.Children()
			type ModuleBlock struct {
				Source  string
				Version string
			}

			for _, item := range moduleCalls.Items {
				unwrapLegacyHCLObjectKeysFromJSON(item, 1)

				if len(item.Keys) != 1 {
					return nil, diagnosticsErrorf("module block at %s has no label", item.Pos())
				}

				name := item.Keys[0].Token.Value().(string)

				var block ModuleBlock
				err := legacyhcl.DecodeObject(&block, item.Val)
				if err != nil {
					return nil, diagnosticsErrorf("module block at %s: %s", item.Pos(), err)
				}

				mc := &ModuleCall{
					Name:    name,
					Source:  block.Source,
					Version: block.Version,
					Pos:     sourcePosLegacyHCL(item.Pos(), filename),
				}
				// it's possible this module call is from an override file
				if origMod, exists := mod.ModuleCalls[name]; exists {
					if mc.Source == "" {
						mc.Source = origMod.Source
					}
				}
				mod.ModuleCalls[name] = mc
			}
		}

		if providerConfigs := list.Filter("provider"); len(providerConfigs.Items) > 0 {
			providerConfigs = providerConfigs.Children()
			type ProviderBlock struct {
				Version string
			}

			for _, item := range providerConfigs.Items {
				unwrapLegacyHCLObjectKeysFromJSON(item, 1)

				if len(item.Keys) != 1 {
					return nil, diagnosticsErrorf("provider block at %s has no label", item.Pos())
				}

				name := item.Keys[0].Token.Value().(string)

				var block ProviderBlock
				err := legacyhcl.DecodeObject(&block, item.Val)
				if err != nil {
					return nil, diagnosticsErrorf("invalid provider block at %s: %s", item.Pos(), err)
				}
				// Even if there wasn't an explicit version required, we still
				// need an entry in our map to signal the unversioned dependency.
				if _, exists := mod.RequiredProviders[name]; !exists {
					mod.RequiredProviders[name] = &ProviderRequirement{}
				}

				if block.Version != "" {
					mod.RequiredProviders[name].VersionConstraints = append(mod.RequiredProviders[name].VersionConstraints, block.Version)
				}
			}
		}
	}

	return mod, nil
}

// unwrapLegacyHCLObjectKeysFromJSON cleans up an edge case that can occur when
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
func unwrapLegacyHCLObjectKeysFromJSON(item *legacyast.ObjectItem, depth int) {
	if len(item.Keys) > depth && item.Keys[0].Token.JSON {
		for len(item.Keys) > depth {
			// Pop off the last key
			n := len(item.Keys)
			key := item.Keys[n-1]
			item.Keys[n-1] = nil
			item.Keys = item.Keys[:n-1]

			// Wrap our value in a list
			item.Val = &legacyast.ObjectType{
				List: &legacyast.ObjectList{
					Items: []*legacyast.ObjectItem{
						&legacyast.ObjectItem{
							Keys: []*legacyast.ObjectKey{key},
							Val:  item.Val,
						},
					},
				},
			}
		}
	}
}

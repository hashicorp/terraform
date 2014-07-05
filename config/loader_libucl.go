package config

import (
	"fmt"
	"path/filepath"

	"github.com/mitchellh/go-libucl"
)

// Put the parse flags we use for libucl in a constant so we can get
// equally behaving parsing everywhere.
const libuclParseFlags = libucl.ParserKeyLowercase

// libuclConfigurable is an implementation of configurable that knows
// how to turn libucl configuration into a *Config object.
type libuclConfigurable struct {
	Object *libucl.Object
}

func (t *libuclConfigurable) Close() error {
	return t.Object.Close()
}

func (t *libuclConfigurable) Config() (*Config, error) {
	type LibuclVariable struct {
		Default     string
		Description string
		Fields      []string `libucl:",decodedFields"`
	}

	var rawConfig struct {
		Variable map[string]*LibuclVariable
	}

	if err := t.Object.Decode(&rawConfig); err != nil {
		return nil, err
	}

	// Start building up the actual configuration. We start with
	// variables.
	config := new(Config)
	config.Variables = make(map[string]*Variable)
	for k, v := range rawConfig.Variable {
		defaultSet := false
		for _, f := range v.Fields {
			if f == "Default" {
				defaultSet = true
				break
			}
		}

		config.Variables[k] = &Variable{
			Default:     v.Default,
			Description: v.Description,
			defaultSet:  defaultSet,
		}
	}

	// Build the provider configs
	providers := t.Object.Get("provider")
	if providers != nil {
		var err error
		config.ProviderConfigs, err = loadProvidersLibucl(providers)
		providers.Close()
		if err != nil {
			return nil, err
		}
	}

	// Build the resources
	resources := t.Object.Get("resource")
	if resources != nil {
		var err error
		config.Resources, err = loadResourcesLibucl(resources)
		resources.Close()
		if err != nil {
			return nil, err
		}
	}

	// Build the outputs
	if outputs := t.Object.Get("output"); outputs != nil {
		var err error
		config.Outputs, err = loadOutputsLibucl(outputs)
		outputs.Close()
		if err != nil {
			return nil, err
		}
	}

	return config, nil
}

// loadFileLibucl is a fileLoaderFunc that knows how to read libucl
// files and turn them into libuclConfigurables.
func loadFileLibucl(root string) (configurable, []string, error) {
	var obj *libucl.Object = nil

	// Parse and store the object. We don't use a defer here so that
	// we clear resources right away rather than stack them up all the
	// way through our recursive calls.
	parser := libucl.NewParser(libuclParseFlags)
	err := parser.AddFile(root)
	if err == nil {
		obj = parser.Object()
		defer obj.Close()
	}
	parser.Close()

	// If there was an error, return early
	if err != nil {
		return nil, nil, err
	}

	// Start building the result
	result := &libuclConfigurable{
		Object: obj,
	}

	// Otherwise, dive in, find the imports.
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
	return result, importPaths, nil
}

// LoadOutputsLibucl recurses into the given libucl object and turns
// it into a mapping of outputs.
func loadOutputsLibucl(o *libucl.Object) (map[string]*Output, error) {
	objects := make(map[string]*libucl.Object)

	// Iterate over all the "output" blocks and get the keys along with
	// their raw configuration objects. We'll parse those later.
	iter := o.Iterate(false)
	for o1 := iter.Next(); o1 != nil; o1 = iter.Next() {
		iter2 := o1.Iterate(true)
		for o2 := iter2.Next(); o2 != nil; o2 = iter2.Next() {
			objects[o2.Key()] = o2
			defer o2.Close()
		}

		o1.Close()
		iter2.Close()
	}
	iter.Close()

	// Go through each object and turn it into an actual result.
	result := make(map[string]*Output)
	for n, o := range objects {
		var config map[string]interface{}

		if err := o.Decode(&config); err != nil {
			return nil, err
		}

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for output %s: %s",
				n,
				err)
		}

		result[n] = &Output{
			Name:      n,
			RawConfig: rawConfig,
		}
	}

	return result, nil
}

// LoadProvidersLibucl recurses into the given libucl object and turns
// it into a mapping of provider configs.
func loadProvidersLibucl(o *libucl.Object) (map[string]*ProviderConfig, error) {
	objects := make(map[string]*libucl.Object)

	// Iterate over all the "provider" blocks and get the keys along with
	// their raw configuration objects. We'll parse those later.
	iter := o.Iterate(false)
	for o1 := iter.Next(); o1 != nil; o1 = iter.Next() {
		iter2 := o1.Iterate(true)
		for o2 := iter2.Next(); o2 != nil; o2 = iter2.Next() {
			objects[o2.Key()] = o2
			defer o2.Close()
		}

		o1.Close()
		iter2.Close()
	}
	iter.Close()

	// Go through each object and turn it into an actual result.
	result := make(map[string]*ProviderConfig)
	for n, o := range objects {
		var config map[string]interface{}

		if err := o.Decode(&config); err != nil {
			return nil, err
		}

		rawConfig, err := NewRawConfig(config)
		if err != nil {
			return nil, fmt.Errorf(
				"Error reading config for provider config %s: %s",
				n,
				err)
		}

		result[n] = &ProviderConfig{
			RawConfig: rawConfig,
		}
	}

	return result, nil
}

// Given a handle to a libucl object, this recurses into the structure
// and pulls out a list of resources.
//
// The resulting resources may not be unique, but each resource
// represents exactly one resource definition in the libucl configuration.
// We leave it up to another pass to merge them together.
func loadResourcesLibucl(o *libucl.Object) ([]*Resource, error) {
	var allTypes []*libucl.Object

	// Libucl object iteration is really nasty. Below is likely to make
	// no sense to anyone approaching this code. Luckily, it is very heavily
	// tested. If working on a bug fix or feature, we recommend writing a
	// test first then doing whatever you want to the code below. If you
	// break it, the tests will catch it. Likewise, if you change this,
	// MAKE SURE you write a test for your change, because its fairly impossible
	// to reason about this mess.
	//
	// Functionally, what the code does below is get the libucl.Objects
	// for all the TYPES, such as "aws_security_group".
	iter := o.Iterate(false)
	for o1 := iter.Next(); o1 != nil; o1 = iter.Next() {
		// Iterate the inner to get the list of types
		iter2 := o1.Iterate(true)
		for o2 := iter2.Next(); o2 != nil; o2 = iter2.Next() {
			// Iterate all of this type to get _all_ the types
			iter3 := o2.Iterate(false)
			for o3 := iter3.Next(); o3 != nil; o3 = iter3.Next() {
				allTypes = append(allTypes, o3)
			}

			o2.Close()
			iter3.Close()
		}

		o1.Close()
		iter2.Close()
	}
	iter.Close()

	// Where all the results will go
	var result []*Resource

	// Now go over all the types and their children in order to get
	// all of the actual resources.
	for _, t := range allTypes {
		// Release the resources for this raw type since we don't need it.
		// Note that this makes it unsafe now to use allTypes again.
		defer t.Close()

		iter := t.Iterate(true)
		defer iter.Close()
		for r := iter.Next(); r != nil; r = iter.Next() {
			defer r.Close()

			var config map[string]interface{}
			if err := r.Decode(&config); err != nil {
				return nil, fmt.Errorf(
					"Error reading config for %s[%s]: %s",
					t.Key(),
					r.Key(),
					err)
			}

			// Remove the "count" from the config, since we treat that special
			delete(config, "count")

			rawConfig, err := NewRawConfig(config)
			if err != nil {
				return nil, fmt.Errorf(
					"Error reading config for %s[%s]: %s",
					t.Key(),
					r.Key(),
					err)
			}

			// If we have a count, then figure it out
			var count int = 1
			if o := r.Get("count"); o != nil {
				err = o.Decode(&count)
				o.Close()
				if err != nil {
					return nil, fmt.Errorf(
						"Error parsing count for %s[%s]: %s",
						t.Key(),
						r.Key(),
						err)
				}
			}

			result = append(result, &Resource{
				Name:      r.Key(),
				Type:      t.Key(),
				Count:     count,
				RawConfig: rawConfig,
			})
		}
	}

	return result, nil
}

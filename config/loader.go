package config

import (
	"fmt"

	"github.com/mitchellh/go-libucl"
)

// Put the parse flags we use for libucl in a constant so we can get
// equally behaving parsing everywhere.
const libuclParseFlags = libucl.ParserKeyLowercase

// libuclImportTree represents a tree structure of the imports from the
// configuration files along with the raw libucl objects from those files.
type libuclImportTree struct {
	Path     string
	Object   *libucl.Object
	Children []*libuclImportTree
}

// libuclConfigTree represents a tree structure of the loaded configurations
// of all the Terraform files.
type libuclConfigTree struct {
	Path     string
	Config   *Config
	Children []*libuclConfigTree
}

// Load loads the Terraform configuration from a given file.
func Load(path string) (*Config, error) {
	importTree, err := loadTreeLibucl(path)
	if err != nil {
		return nil, err
	}

	configTree, err := importTree.ConfigTree()
	if err != nil {
		return nil, err
	}

	return configTree.Config, nil
}

func loadTreeLibucl(root string) (*libuclImportTree, error) {
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
		return nil, err
	}

	// Start building the result
	result := &libuclImportTree{
		Path:   root,
		Object: obj,
	}

	// Otherwise, dive in, find the imports.
	imports := obj.Get("import")
	if imports == nil {
		result.Object.Ref()
		return result, nil
	}

	if imports.Type() != libucl.ObjectTypeString {
		imports.Close()

		return nil, fmt.Errorf(
			"Error in %s: all 'import' declarations should be in the format\n"+
				"`import \"foo\"` (Got type %s)",
			root,
			imports.Type())
	}

	// Gather all the import paths
	importPaths := make([]string, 0, imports.Len())
	iter := imports.Iterate(false)
	for imp := iter.Next(); imp != nil; imp = iter.Next() {
		importPaths = append(importPaths, imp.ToString())
		imp.Close()
	}
	iter.Close()
	imports.Close()

	// Load them all
	result.Children = make([]*libuclImportTree, len(importPaths))
	for i, path := range importPaths {
		imp, err := loadTreeLibucl(path)
		if err != nil {
			return nil, err
		}

		result.Children[i] = imp
	}

	result.Object.Ref()
	return result, nil
}

// Given a handle to a libucl object, this recurses into the structure
// and pulls out a list of resources.
//
// The resulting resources may not be unique, but each resource
// represents exactly one resource definition in the libucl configuration.
// We leave it up to another pass to merge them together.
func loadResourcesLibucl(o *libucl.Object) ([]Resource, error) {
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
	var result []Resource

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

			result = append(result, Resource{
				Name:   r.Key(),
				Type:   t.Key(),
				Config: config,
			})
		}
	}

	return result, nil
}

func (t *libuclImportTree) ConfigTree() (*libuclConfigTree, error) {
	var rawConfig struct {
		Variable map[string]Variable
	}

	if err := t.Object.Decode(&rawConfig); err != nil {
		return nil, fmt.Errorf(
			"Error decoding %s: %s",
			t.Path,
			err)
	}

	// Start building up the actual configuration. We first
	// copy the fields that can be directly assigned.
	config := new(Config)
	config.Variables = rawConfig.Variable

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

	// Build our result
	result := &libuclConfigTree{
		Path:   t.Path,
		Config: config,
	}

	return result, nil
}

// Helper for parsing a single libucl-formatted file into
// the given structure.
func parseFile(path string, result interface{}) error {
	parser := libucl.NewParser(libuclParseFlags)
	defer parser.Close()

	if err := parser.AddFile(path); err != nil {
		return err
	}

	root := parser.Object()
	defer root.Close()

	if err := root.Decode(result); err != nil {
		return err
	}

	return nil
}

package config

import (
	"fmt"

	"github.com/mitchellh/go-libucl"
)

// Put the parse flags we use for libucl in a constant so we can get
// equally behaving parsing everywhere.
const libuclParseFlags = libucl.ParserKeyLowercase

// Load loads the Terraform configuration from a given file.
func Load(path string) (*Config, error) {
	var rawConfig struct {
		Variable map[string]Variable
		Object   *libucl.Object `libucl:",object"`
	}

	// Parse the libucl file into the raw format
	if err := parseFile(path, &rawConfig); err != nil {
		return nil, err
	}

	// Make sure we close the raw object
	defer rawConfig.Object.Close()

	// Start building up the actual configuration. We first
	// copy the fields that can be directly assigned.
	config := new(Config)
	config.Variables = rawConfig.Variable

	// Build the resources
	resources := rawConfig.Object.Get("resource")
	if resources != nil {
		defer resources.Close()

		var err error
		config.Resources, err = loadResourcesLibucl(resources)
		if err != nil {
			return nil, err
		}
	}

	return config, nil
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

package variables

import (
	"fmt"
	"io/ioutil"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/go-homedir"
)

// FlagFile is a flag.Value implementation for parsing user variables
// from the command line in the form of files. i.e. '-var-file=foo'
type FlagFile map[string]interface{}

func (v *FlagFile) String() string {
	return ""
}

func (v *FlagFile) Set(raw string) error {
	vs, err := loadKVFile(raw)
	if err != nil {
		return err
	}

	*v = Merge(*v, vs)
	return nil
}

func loadKVFile(rawPath string) (map[string]interface{}, error) {
	path, err := homedir.Expand(rawPath)
	if err != nil {
		return nil, fmt.Errorf(
			"Error expanding path: %s", err)
	}

	// Read the HCL file and prepare for parsing
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf(
			"Error reading %s: %s", path, err)
	}

	// Parse it
	obj, err := hcl.Parse(string(d))
	if err != nil {
		return nil, fmt.Errorf(
			"Error parsing %s: %s", path, err)
	}

	var result map[string]interface{}
	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, fmt.Errorf(
			"Error decoding Terraform vars file: %s\n\n"+
				"The vars file should be in the format of `key = \"value\"`.\n"+
				"Decoding errors are usually caused by an invalid format.",
			err)
	}

	err = flattenMultiMaps(result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

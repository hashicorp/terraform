package command

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/go-homedir"
)

// FlagKV is a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value'.
type FlagKV map[string]string

func (v *FlagKV) String() string {
	return ""
}

func (v *FlagKV) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx == -1 {
		return fmt.Errorf("No '=' value in arg: %s", raw)
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	key, value := raw[0:idx], raw[idx+1:]
	(*v)[key] = value
	return nil
}

// FlagKVFile is a flag.Value implementation for parsing user variables
// from the command line in the form of files. i.e. '-var-file=foo'
type FlagKVFile map[string]string

func (v *FlagKVFile) String() string {
	return ""
}

func (v *FlagKVFile) Set(raw string) error {
	vs, err := loadKVFile(raw)
	if err != nil {
		return err
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	for key, value := range vs {
		(*v)[key] = value
	}

	return nil
}

func loadKVFile(rawPath string) (map[string]string, error) {
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

	var result map[string]string
	if err := hcl.DecodeObject(&result, obj); err != nil {
		return nil, fmt.Errorf(
			"Error decoding Terraform vars file: %s\n\n"+
				"The vars file should be in the format of `key = \"value\"`.\n"+
				"Decoding errors are usually caused by an invalid format.",
			err)
	}

	return result, nil
}

// FlagStringSlice is a flag.Value implementation for parsing targets from the
// command line, e.g. -target=aws_instance.foo -target=aws_vpc.bar

type FlagStringSlice []string

func (v *FlagStringSlice) String() string {
	return ""
}
func (v *FlagStringSlice) Set(raw string) error {
	*v = append(*v, raw)

	return nil
}

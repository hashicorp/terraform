package command

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl"
	"github.com/mitchellh/go-homedir"
)

// FlagTypedKVis a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value', where value is
// a type intended for use as a Terraform variable
type FlagTypedKV map[string]interface{}

func (v *FlagTypedKV) String() string {
	return ""
}

func (v *FlagTypedKV) Set(raw string) error {
	key, value, err := parseVarFlagAsHCL(raw)
	if err != nil {
		return err
	}

	if *v == nil {
		*v = make(map[string]interface{})
	}

	(*v)[key] = value
	return nil
}

// FlagStringKV is a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value', where value is
// only ever a primitive.
type FlagStringKV map[string]string

func (v *FlagStringKV) String() string {
	return ""
}

func (v *FlagStringKV) Set(raw string) error {
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
type FlagKVFile map[string]interface{}

func (v *FlagKVFile) String() string {
	return ""
}

func (v *FlagKVFile) Set(raw string) error {
	vs, err := loadKVFile(raw)
	if err != nil {
		return err
	}

	if *v == nil {
		*v = make(map[string]interface{})
	}

	for key, value := range vs {
		(*v)[key] = value
	}

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

var (
	// This regular expression is how we check if a value for a variable
	// matches what we'd expect a rich HCL value to be. For example: {
	// definitely signals a map. If a value DOESN'T match this, we return
	// it as a raw string.
	varFlagHCLRe = regexp.MustCompile(`^["\[\{]`)
)

// parseVarFlagAsHCL parses the value of a single variable as would have been specified
// on the command line via -var or in an environment variable named TF_VAR_x, where x is
// the name of the variable. In order to get around the restriction of HCL requiring a
// top level object, we prepend a sentinel key, decode the user-specified value as its
// value and pull the value back out of the resulting map.
func parseVarFlagAsHCL(input string) (string, interface{}, error) {
	idx := strings.Index(input, "=")
	if idx == -1 {
		return "", nil, fmt.Errorf("No '=' value in variable: %s", input)
	}
	probablyName := input[0:idx]
	value := input[idx+1:]
	trimmed := strings.TrimSpace(value)

	// If the value is a simple number, don't parse it as hcl because the
	// variable type may actually be a string, and HCL will convert it to the
	// numberic value. We could check this in the validation later, but the
	// conversion may alter the string value.
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return probablyName, value, nil
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return probablyName, value, nil
	}

	// HCL will also parse hex as a number
	if strings.HasPrefix(trimmed, "0x") {
		if _, err := strconv.ParseInt(trimmed[2:], 16, 64); err == nil {
			return probablyName, value, nil
		}
	}

	// If the value is a boolean value, also convert it to a simple string
	// since Terraform core doesn't accept primitives as anything other
	// than string for now.
	if _, err := strconv.ParseBool(trimmed); err == nil {
		return probablyName, value, nil
	}

	parsed, err := hcl.Parse(input)
	if err != nil {
		// If it didn't parse as HCL, we check if it doesn't match our
		// whitelist of TF-accepted HCL types for inputs. If not, then
		// we let it through as a raw string.
		if !varFlagHCLRe.MatchString(trimmed) {
			return probablyName, value, nil
		}

		// This covers flags of the form `foo=bar` which is not valid HCL
		// At this point, probablyName is actually the name, and the remainder
		// of the expression after the equals sign is the value.
		if regexp.MustCompile(`Unknown token: \d+:\d+ IDENT`).Match([]byte(err.Error())) {
			return probablyName, value, nil
		}

		return "", nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL: %s", probablyName, input, err)
	}

	var decoded map[string]interface{}
	if hcl.DecodeObject(&decoded, parsed); err != nil {
		return "", nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL: %s", probablyName, input, err)
	}

	// Cover cases such as key=
	if len(decoded) == 0 {
		return probablyName, "", nil
	}

	if len(decoded) > 1 {
		return "", nil, fmt.Errorf("Cannot parse value for variable %s (%q) as valid HCL. Only one value may be specified.", probablyName, input)
	}

	err = flattenMultiMaps(decoded)
	if err != nil {
		return probablyName, "", err
	}

	var k string
	var v interface{}
	for k, v = range decoded {
		break
	}
	return k, v, nil
}

// Variables don't support any type that can be configured via multiple
// declarations of the same HCL map, so any instances of
// []map[string]interface{} are either a single map that can be flattened, or
// are invalid config.
func flattenMultiMaps(m map[string]interface{}) error {
	for k, v := range m {
		switch v := v.(type) {
		case []map[string]interface{}:
			switch {
			case len(v) > 1:
				return fmt.Errorf("multiple map declarations not supported for variables")
			case len(v) == 1:
				m[k] = v[0]
			}
		}
	}
	return nil
}

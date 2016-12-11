package variables

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl"
)

// ParseInput parses a manually inputed variable to a richer value.
//
// This will turn raw input into rich types such as `[]` to a real list or
// `{}` to a real map. This function should be used to parse any manual untyped
// input for variables in order to provide a consistent experience.
func ParseInput(value string) (interface{}, error) {
	trimmed := strings.TrimSpace(value)

	// If the value is a simple number, don't parse it as hcl because the
	// variable type may actually be a string, and HCL will convert it to the
	// numberic value. We could check this in the validation later, but the
	// conversion may alter the string value.
	if _, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return value, nil
	}
	if _, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return value, nil
	}

	// HCL will also parse hex as a number
	if strings.HasPrefix(trimmed, "0x") {
		if _, err := strconv.ParseInt(trimmed[2:], 16, 64); err == nil {
			return value, nil
		}
	}

	// If the value is a boolean value, also convert it to a simple string
	// since Terraform core doesn't accept primitives as anything other
	// than string for now.
	if _, err := strconv.ParseBool(trimmed); err == nil {
		return value, nil
	}

	parsed, err := hcl.Parse(fmt.Sprintf("foo=%s", trimmed))
	if err != nil {
		// If it didn't parse as HCL, we check if it doesn't match our
		// whitelist of TF-accepted HCL types for inputs. If not, then
		// we let it through as a raw string.
		if !varFlagHCLRe.MatchString(trimmed) {
			return value, nil
		}

		// This covers flags of the form `foo=bar` which is not valid HCL
		// At this point, probablyName is actually the name, and the remainder
		// of the expression after the equals sign is the value.
		if regexp.MustCompile(`Unknown token: \d+:\d+ IDENT`).Match([]byte(err.Error())) {
			return value, nil
		}

		return nil, fmt.Errorf(
			"Cannot parse value for variable (%q) as valid HCL: %s",
			value, err)
	}

	var decoded map[string]interface{}
	if hcl.DecodeObject(&decoded, parsed); err != nil {
		return nil, fmt.Errorf(
			"Cannot parse value for variable (%q) as valid HCL: %s",
			value, err)
	}

	// Cover cases such as key=
	if len(decoded) == 0 {
		return "", nil
	}

	if len(decoded) > 1 {
		return nil, fmt.Errorf(
			"Cannot parse value for variable (%q) as valid HCL. "+
				"Only one value may be specified.",
			value)
	}

	err = flattenMultiMaps(decoded)
	if err != nil {
		return "", err
	}

	return decoded["foo"], nil
}

var (
	// This regular expression is how we check if a value for a variable
	// matches what we'd expect a rich HCL value to be. For example: {
	// definitely signals a map. If a value DOESN'T match this, we return
	// it as a raw string.
	varFlagHCLRe = regexp.MustCompile(`^["\[\{]`)
)

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

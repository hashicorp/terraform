package external

import (
	"fmt"
	"os/exec"
)

// validateProgramAttr is a validation function for the "program" attribute we
// accept as input on our resources.
//
// The attribute is assumed to be specified in schema as a list of strings.
func validateProgramAttr(v interface{}) error {
	args := v.([]interface{})
	if len(args) < 1 {
		return fmt.Errorf("'program' list must contain at least one element")
	}

	for i, vI := range args {
		if _, ok := vI.(string); !ok {
			return fmt.Errorf(
				"'program' element %d is %T; a string is required",
				i, vI,
			)
		}
	}

	// first element is assumed to be an executable command, possibly found
	// using the PATH environment variable.
	_, err := exec.LookPath(args[0].(string))
	if err != nil {
		return fmt.Errorf("can't find external program %q", args[0])
	}

	return nil
}

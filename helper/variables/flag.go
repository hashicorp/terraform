package variables

import (
	"fmt"
	"strings"
)

// Flag a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value', where value is
// a type intended for use as a Terraform variable.
type Flag map[string]interface{}

func (v *Flag) String() string {
	return ""
}

func (v *Flag) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx == -1 {
		return fmt.Errorf("No '=' value in arg: %s", raw)
	}

	key, input := raw[0:idx], raw[idx+1:]

	// Trim the whitespace on the key
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("No key to left '=' in arg: %s", raw)
	}

	value, err := ParseInput(input)
	if err != nil {
		return err
	}

	*v = Merge(*v, map[string]interface{}{key: value})
	return nil
}

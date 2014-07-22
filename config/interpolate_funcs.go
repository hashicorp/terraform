package config

import (
	"fmt"
	"strings"
)

// Funcs is the mapping of built-in functions for configuration.
var Funcs map[string]InterpolationFunc

func init() {
	Funcs = map[string]InterpolationFunc{
		"lookup": interpolationFuncLookup,
	}
}

// interpolationFuncLookup implements the "lookup" function that allows
// dynamic lookups of map types within a Terraform configuration.
func interpolationFuncLookup(
	vs map[string]string, args ...string) (string, error) {
	if len(args) != 2 {
		return "", fmt.Errorf(
			"lookup expects 2 arguments, got %d", len(args))
	}

	k := fmt.Sprintf("var.%s", strings.Join(args, "."))
	v, ok := vs[k]
	if !ok {
		return "", fmt.Errorf(
			"lookup in '%s' failed to find '%s'",
			args[0], args[1])
	}

	return v, nil
}

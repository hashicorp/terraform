package config

import (
	"github.com/hashicorp/hil/ast"
)

// Funcs used to return a mapping of built-in functions for configuration.
//
// However, these function implementations are no longer used. To find the
// current function implementations, refer to ../lang/functions.go  instead.
func Funcs() map[string]ast.Function {
	return nil
}

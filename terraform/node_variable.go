package terraform

import (
	"github.com/hashicorp/terraform/addrs"
)

// NodeVariable is a interface implemented by node types that represent a
// single variable.
type NodeVariable interface {
	// variableAddr is implemented by types in node_root_variable.go and
	// node_module_variable.go.
	variableAddr() addrs.AbsInputVariableInstance
}

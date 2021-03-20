package debug

import (
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/instances"
	"github.com/hashicorp/terraform/lang"
)

// RuntimeContext is an interface which must be implemented by the Terraform
// language runtime in order to allow the debugger engine to retrieve
// information about the current execution context.
//
// If the language runtime itself is making concurrent calls to its object
// of type Interface then the Runtime it provides to the Interface methods
// must have concurrency-safe implementations of these methods.
type RuntimeContext interface {
	// Scope returns a Scope object which can evaluate expressions in the
	// given module instance, with optional "self" and repetition values.
	//
	// If self is non-nil then it must refer to an object in the given module
	// instance.
	Scope(
		modInst addrs.ModuleInstance,
		self addrs.Referenceable,
		keyData instances.RepetitionData,
	) *lang.Scope
}

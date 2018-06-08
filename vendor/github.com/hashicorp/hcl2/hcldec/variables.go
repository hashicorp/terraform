package hcldec

import (
	"github.com/hashicorp/hcl2/hcl"
)

// Variables processes the given body with the given spec and returns a
// list of the variable traversals that would be required to decode
// the same pairing of body and spec.
//
// This can be used to conditionally populate the variables in the EvalContext
// passed to Decode, for applications where a static scope is insufficient.
//
// If the given body is not compliant with the given schema, the result may
// be incomplete, but that's assumed to be okay because the eventual call
// to Decode will produce error diagnostics anyway.
func Variables(body hcl.Body, spec Spec) []hcl.Traversal {
	schema := ImpliedSchema(spec)

	content, _, _ := body.PartialContent(schema)

	var vars []hcl.Traversal

	if vs, ok := spec.(specNeedingVariables); ok {
		vars = append(vars, vs.variablesNeeded(content)...)
	}
	spec.visitSameBodyChildren(func(s Spec) {
		if vs, ok := s.(specNeedingVariables); ok {
			vars = append(vars, vs.variablesNeeded(content)...)
		}
	})

	return vars
}

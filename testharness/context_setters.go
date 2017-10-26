package testharness

import (
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// A contextSetter is something passed to the first argument of a "describe"
// call in a spec to establish what any sub-testers are testing. Internally,
// these objects produce derived Context objects representing what the user
// passed in.
//
// Method Context must at least return a new context with a new name compared
// to the given parent, even if an error is returned. This changed name will
// be used to report any errors to the user. If the returned diagnostics
// contains no errors then it will be used to run tests within the associated
// "describe" body.
type contextSetter interface {
	Context(parent *Context, subject *Subject) (*Context, tfdiags.Diagnostics)
}

// contextSetter implementations
var (
	_ contextSetter = simpleContextSetter("")
	_ contextSetter = (*resourceContextSetter)(nil)
)

// A simpleContextSetter is a contextSetter that actually provides no
// additional contextual objects, but records the user-supplied name of
// something being tested.
type simpleContextSetter string

func (s simpleContextSetter) Context(parent *Context, subject *Subject) (*Context, tfdiags.Diagnostics) {
	return parent.WithNameSuffix(string(s)), nil
}

// A resourceContextSetter is a contextSetter that uses a resource from the
// state as its context.
type resourceContextSetter struct {
	Addr *terraform.ResourceAddress

	DefRange tfdiags.SourceRange
}

func (s resourceContextSetter) Context(parent *Context, subject *Subject) (*Context, tfdiags.Diagnostics) {
	// TODO: Set the resource object too
	return parent.WithNameSuffix(s.Addr.String()), nil
}

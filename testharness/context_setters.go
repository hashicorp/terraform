package testharness

import (
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

// A contextSetter is something passed to the first argument of a "describe"
// call in a spec to establish what any sub-testers are testing. Internally,
// these objects produce derived Context objects representing what the user
// passed in.

// Method AppendContext must append zero or more new contexts to the given
// slice (which may be nil) and return the result. Each context appended
// represents a distinct context in which to run any downstream tests.
type contextSetter interface {
	AppendContexts(parent *Context, subject *Subject, ctxs []*Context) ([]*Context, tfdiags.Diagnostics)
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

func (s simpleContextSetter) AppendContexts(parent *Context, subject *Subject, ctxs []*Context) ([]*Context, tfdiags.Diagnostics) {
	return append(ctxs, parent.WithNameSuffix(string(s))), nil
}

// A resourceContextSetter is a contextSetter that uses a resource from the
// state as its context.
type resourceContextSetter struct {
	Addr *terraform.ResourceAddress

	DefRange tfdiags.SourceRange
}

func (s *resourceContextSetter) AppendContexts(parent *Context, subject *Subject, ctxs []*Context) ([]*Context, tfdiags.Diagnostics) {
	// TODO: Set the resource object too
	// TODO: If the resource address refers to a resource block with multiple
	// instances (e.g. "count" is set) then generate one context for each
	// of the instances matched.
	return append(ctxs, parent.WithNameSuffix(s.Addr.String())), nil
}

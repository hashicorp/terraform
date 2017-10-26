package testharness

import (
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
)

// Tester is an interface implemented by objects that can run tests.
type Tester interface {
	Test() // TODO: Flesh out the arguments for this
}

// Testers is a slice of Tester.
type Testers []Tester

// describe represents a single "describe" call in a test specification.
//
// describe implements Tester.
type describe struct {
	Described contextSetter
	BodyFn    *lua.LFunction

	DefRange tfdiags.SourceRange
}

func (t *describe) Test() {
	// TODO: Implement this once we figure out what the Tester interface
	// really contains.
}

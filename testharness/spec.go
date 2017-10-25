package testharness

import (
	lua "github.com/yuin/gopher-lua"
)

// Spec is the main type in this package, representing a whole test
// specification that has been loaded from a set of Lua files.
type Spec struct {
	scenarios map[string]*Scenario
	testers   Testers
	lstate    *lua.LState
}

// Scenarios returns all of the scenarios associated with the receiving Spec.
//
// The caller must treat the returned map as read-only.
func (s *Spec) Scenarios() map[string]*Scenario {
	return s.scenarios
}

// Scenario returns the scenario with the given name, or nil if there is no
// such scenario defined.
func (s *Spec) Scenario(name string) *Scenario {
	return s.scenarios[name]
}

type Tester interface {
	Test() // TODO: Flesh out the arguments for this
}

type Testers []Tester

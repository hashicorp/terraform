package luacty

import (
	lua "github.com/yuin/gopher-lua"
)

// Converter is the main type in this package, providing the conversion
// functionality in both directions.
//
// A converter is specific to a givan lua.LState because it uses that state
// to create new values and to interact with the Lua stack during operations.
type Converter struct {
	lstate    *lua.LState
	metatable *lua.LTable
}

// NewConverter creates and returns a new Converter for the given Lua state.
func NewConverter(L *lua.LState) *Converter {
	c := &Converter{
		lstate: L,
	}
	c.metatable = c.ctyMetatable()
	return c
}

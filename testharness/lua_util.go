package testharness

import (
	"github.com/hashicorp/hcl2/hcl"
	lua "github.com/yuin/gopher-lua"
)

func callingRange(L *lua.LState, level int) *hcl.Range {
	d, ok := L.GetStack(level)
	if !ok {
		return nil
	}
	L.GetInfo("Sl", d, nil)
	return &hcl.Range{
		Filename: d.Source,
		Start:    hcl.Pos{Line: d.CurrentLine, Column: 1, Byte: -1},
		End:      hcl.Pos{Line: d.CurrentLine, Column: 1, Byte: -1},
	}
}

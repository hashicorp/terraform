package testharness

import (
	"fmt"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/hashicorp/terraform/tfdiags"
	lua "github.com/yuin/gopher-lua"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/gopherlua-cty/luacty"
)

type Scenario struct {
	Name      string
	Variables map[string]cty.Value

	DefRange tfdiags.SourceRange
}

type ScenariosBuilder struct {
	Scenarios map[string]*Scenario
	Diags     tfdiags.Diagnostics
}

func (b *ScenariosBuilder) luaScenarioFunc(L *lua.LState) int {
	name := L.OptString(1, "")
	fn := L.OptFunction(2, nil)

	if name == "" {
		b.Diags = b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid scenario declaration",
			Detail:   "A \"scenario\" call must have a non-empty name as its first argument.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}
	if fn == nil {
		b.Diags = b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid scenario declaration",
			Detail:   "A \"scenario\" call must have a definition function as its second argument.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}
	if L.GetTop() != 2 {
		b.Diags = b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid scenario declaration",
			Detail:   "The \"scenario\" function expects two arguments: a name and a definition function.",
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	if b.Scenarios == nil {
		b.Scenarios = make(map[string]*Scenario)
	}

	var rng hcl.Range
	if rngPtr := callingRange(L, 1); rngPtr != nil {
		rng = *rngPtr
	}

	b.Scenarios[name] = &Scenario{
		Name:     name,
		DefRange: tfdiags.SourceRangeFromHCL(rng),
	}

	variables := func(L *lua.LState) int {
		table := L.OptTable(1, nil)
		if table == nil {
			b.Diags = b.Diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid scenario variables declaration",
				Detail:   "The \"variables\" function expects one argument: a table of variable values.",
				Subject:  callingRange(L, 1),
			})
			return 0
		}

		if b.Scenarios[name].Variables != nil {
			b.Diags = b.Diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid duplicate variables declaration",
				Detail:   "The variables for this scenario were already declared. Only one \"variables\" call is allowed per \"scenario\" definition.",
				Subject:  callingRange(L, 1),
			})
			return 0
		}

		conv := luacty.NewConverter(L)
		vars := make(map[string]cty.Value)
		b.Scenarios[name].Variables = vars

		table.ForEach(func(key, value lua.LValue) {
			if !lua.LVCanConvToString(key) {
				b.Diags = b.Diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid variable name",
					Detail:   fmt.Sprintf("The value %q is not a valid variable name.", key.String()),
					Subject:  callingRange(L, 1),
				})
				return
			}

			keyStr := lua.LVAsString(key)

			valCty, err := conv.ToCtyValue(value, cty.DynamicPseudoType)
			if err != nil {
				b.Diags = b.Diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid variable value",
					Detail:   fmt.Sprintf("Unsuitable value for variable %q: %s.", keyStr, err),
					Subject:  callingRange(L, 1),
				})
				return
			}

			vars[keyStr] = valCty
		})

		return 0
	}

	defEnv := L.NewTable()
	defEnv.RawSet(lua.LString("variables"), L.NewFunction(variables))
	L.SetFEnv(fn, defEnv)

	L.Push(fn)
	err := L.PCall(0, 0, nil)
	if err != nil {
		b.Diags = b.Diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Error in scenario definition function",
			Detail:   fmt.Sprintf("Error occured in the definition function for scenario %q: %s.", name, err),
			Subject:  callingRange(L, 1),
		})
		return 0
	}

	if b.Scenarios[name].Variables == nil {
		// Make sure there's always a non-nil map here, for caller convenience.
		b.Scenarios[name].Variables = make(map[string]cty.Value)
	}

	return 0
}

package stressgen

import (
	"math/rand"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// GenerateConfigVariable uses the given random number generator to generate
// a random ConfigVariable object.
func GenerateConfigVariable(rnd *rand.Rand, ns *Namespace) *ConfigVariable {
	addr := addrs.InputVariable{Name: ns.GenerateShortName(rnd)}
	ret := &ConfigVariable{
		Addr: addr,
	}
	// TODO: Possibly populate the other optional fields too
	if includeType := decideBool(rnd, 75); includeType {
		// For now we're only generating string-typed variables because our
		// current focus is on testing the graph construction and walking
		// rather than on expression evaluation. Maybe we'll add some other
		// possibilities later.
		ret.TypeConstraint = cty.String
	}
	if optional := decideBool(rnd, 25); optional {
		defStr := ns.GenerateLongName(rnd)
		ret.DefaultValue = cty.StringVal(defStr)
		ret.CallerWillSet = decideBool(rnd, 15)
	} else {
		// If the variable is required then the caller must always set it.
		ret.CallerWillSet = true
	}

	declareConfigVariable(ret, ns)
	return ret
}

// declareConfigVariable creates the declaration of the given variable in the
// given namespace. This is shared by both GenerateConfigVariable and by
// ConfigVariable.GenerateModified.
func declareConfigVariable(v *ConfigVariable, ns *Namespace) {
	expr := NewConfigExprRef(v.Addr, nil)
	ns.DeclareReferenceable(expr)
}

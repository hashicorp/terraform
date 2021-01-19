package stressgen

import (
	"math/rand"

	"github.com/hashicorp/terraform/addrs"
)

// GenerateConfigOutput uses the given random number generator to generate
// a random ConfigOutput object.
func GenerateConfigOutput(rnd *rand.Rand, ns *Namespace) *ConfigOutput {
	addr := addrs.OutputValue{Name: ns.GenerateShortName(rnd)}
	valExpr := ns.GenerateExpression(rnd)
	ret := &ConfigOutput{
		Addr:  addr,
		Value: valExpr,
	}
	// TODO: Possibly populate the other optional fields too

	declareConfigOutput(ret, ns)
	return ret
}

// declareConfigOutput creates the declaration of the given output in the
// given namespace. This is shared by both GenerateConfigOutput and by
// ConfigOutput.GenerateModified.
func declareConfigOutput(o *ConfigOutput, ns *Namespace) {
	ns.DeclareOutputValue(o)
}

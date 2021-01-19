package stressgen

import (
	"math/rand"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// GenerateConfigModuleCall uses the given random number generator to generate
// a random ConfigModuleCall object.
func GenerateConfigModuleCall(rnd *rand.Rand, parentNS *Namespace) *ConfigModuleCall {
	addr := addrs.ModuleCall{Name: parentNS.GenerateShortName(rnd)}
	childNS := parentNS.ChildNamespace(addr.Name)
	ret := &ConfigModuleCall{
		Addr:           addr,
		Arguments:      make(map[addrs.InputVariable]ConfigExpr),
		ChildNamespace: childNS,
	}

	ret.ForEachExpr, ret.CountExpr = generateRepetitionArgs(rnd, parentNS)

	objCount := rnd.Intn(25)
	objs := make([]ConfigObject, 0, objCount+1) // +1 for the boilerplate object

	// We always need a boilerplate object.
	boilerplate := &ConfigBoilerplate{
		ModuleAddr: childNS.ModuleAddr,
		Providers: map[string]addrs.Provider{
			"stressful": addrs.MustParseProviderSourceString("terraform.io/stresstest/stressful"),
		},
	}
	objs = append(objs, boilerplate)

	for i := 0; i < objCount; i++ {
		obj := GenerateConfigObject(rnd, childNS)
		objs = append(objs, obj)

		if cv, ok := obj.(*ConfigVariable); ok && cv.CallerWillSet {
			// The expression comes from parentNS here because the arguments
			// are defined in the calling module, not the called module.
			chosenExpr := parentNS.GenerateExpression(rnd)
			ret.Arguments[cv.Addr] = chosenExpr
		}
	}

	ret.Objects = objs

	declareConfigModuleCall(ret, childNS)
	return ret
}

// declareConfigModuleCall creates the declaration of the given module call in
// the given namespace. This is shared by both GenerateConfigModuleCall and by
// ConfigModuleCall.GenerateModified.
func declareConfigModuleCall(mc *ConfigModuleCall, ns *Namespace) {
	// In the case were we're generating a count expression, we can't know
	// until instantiation how many instances there will be, so we don't
	// declare any referencables in that case. That's not ideal, but we
	// accept the compromise because we can still generate references for
	// for_each and those two mechanisms share a lot of supporting code
	// in common. Having the number of instances for count be able to vary
	// between instantiations is also an interesting thing to test, even
	// though we can't guarantee to generate valid references in that case.
	if mc.CountExpr != nil {
		return
	}

	switch {
	case mc.ForEachExpr != nil:
		for keyStr := range mc.ForEachExpr.Exprs {
			for name := range mc.ChildNamespace.OutputValues {
				moduleInstAddr := addrs.ModuleCallInstance{
					Call: mc.Addr,
					Key:  addrs.StringKey(keyStr),
				}
				ref := NewConfigExprRef(moduleInstAddr, cty.GetAttrPath(name))
				ns.DeclareReferenceable(ref)
			}
		}
	default:
		for name := range mc.ChildNamespace.OutputValues {
			moduleInstAddr := addrs.ModuleCallInstance{
				Call: mc.Addr,
				Key:  addrs.NoKey,
			}
			ref := NewConfigExprRef(moduleInstAddr, cty.GetAttrPath(name))
			ns.DeclareReferenceable(ref)
		}
	}
}

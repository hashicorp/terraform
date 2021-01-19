package stressgen

import (
	"math/rand"

	"github.com/hashicorp/terraform/addrs"
	"github.com/zclconf/go-cty/cty"
)

// GenerateConfigResource uses the given random number generator to call either
// GenerateConfigManagedResource or GenerageConfigDataResource and then return
// the resulting object.
//
// Despite of the very general return type, GenerateConfigResource is
// guaranteed to return either *ConfigManagedResource or *ConfigDataResource.
func GenerateConfigResource(rnd *rand.Rand, ns *Namespace) ConfigObject {
	// data resources are slightly less likely than managed resources, just
	// because there are more different mechanisms and steps involved in
	// managed resources and thus more permutations to possibly test.
	generateData := decideBool(rnd, 40)
	switch {
	case generateData:
		return GenerateConfigDataResource(rnd, ns)
	default:
		return GenerateConfigManagedResource(rnd, ns)
	}
}

// GenerateConfigManagedResource uses the given random number generator to
// generate a random ConfigManagedResource object.
func GenerateConfigManagedResource(rnd *rand.Rand, ns *Namespace) *ConfigManagedResource {
	// We always generate resource associated with the stressful provider,
	// because that's the only provider we have available in our stresstest
	// cases.
	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "stressful",
		Name: ns.GenerateShortName(rnd),
	}
	// TODO: Currently we also assume only one available provider configuration
	// for the stressful provider, but it'd be good to also exercise the
	// codepaths related to non-default providers and explicitly passing
	// providers between modules, which would require Namespace to track which
	// provider aliases are available and for this function to potentially
	// generate non-default provider references.

	common := ConfigResource{
		Addr:      addr,
		Arguments: make(map[string]ConfigExpr),
	}
	common.ForEachExpr, common.CountExpr = generateRepetitionArgs(rnd, ns)
	common.Arguments["name"] = ns.GenerateExpression(rnd)
	if decideBool(rnd, 5) {
		common.Arguments["force_replace"] = ns.GenerateExpression(rnd)
	}

	ret := &ConfigManagedResource{
		ConfigResource: common,
	}

	// We have a relatively low likelihood of generating create_before_destroy,
	// because this flag automatically propagates to other resources in the
	// dependency chain and so turning this on for one resource will typically
	// turn it on for all (or, at least, most) other resources.
	ret.CreateBeforeDestroy = decideBool(rnd, 5)

	declareConfigManagedResource(ret, ns)
	return ret
}

// declareConfigManagedResource creates the declaration of the given managed
// resource in the given namespace. This is shared by both
// GenerateConfigManagedResource and by ConfigManagedResource.GenerateModified.
func declareConfigManagedResource(mc *ConfigManagedResource, ns *Namespace) {
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
			instAddr := mc.Addr.Instance(addrs.StringKey(keyStr))
			ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("name")))
			ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("computed_name")))
		}
	default:
		instAddr := mc.Addr.Instance(addrs.NoKey)
		ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("name")))
		ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("computed_name")))
	}
}

// GenerateConfigDataResource uses the given random number generator to
// generate a random ConfigDataResource object.
func GenerateConfigDataResource(rnd *rand.Rand, ns *Namespace) *ConfigDataResource {
	// We always generate resource associated with the stressful provider,
	// because that's the only provider we have available in our stresstest
	// cases.
	addr := addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "stressful",
		Name: ns.GenerateShortName(rnd),
	}
	// TODO: Currently we also assume only one available provider configuration
	// for the stressful provider, but it'd be good to also exercise the
	// codepaths related to non-default providers and explicitly passing
	// providers between modules, which would require Namespace to track which
	// provider aliases are available and for this function to potentially
	// generate non-default provider references.

	common := ConfigResource{
		Addr:      addr,
		Arguments: make(map[string]ConfigExpr),
	}
	common.ForEachExpr, common.CountExpr = generateRepetitionArgs(rnd, ns)
	common.Arguments["in"] = ns.GenerateExpression(rnd)

	ret := &ConfigDataResource{
		ConfigResource: common,
	}

	declareConfigDataResource(ret, ns)
	return ret
}

// declareConfigDataResource creates the declaration of the given data
// resource in the given namespace. This is shared by both
// GenerateConfigDataResource and by ConfigDataResource.GenerateModified.
func declareConfigDataResource(mc *ConfigDataResource, ns *Namespace) {
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
			instAddr := mc.Addr.Instance(addrs.StringKey(keyStr))
			ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("in")))
			ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("out")))
		}
	default:
		instAddr := mc.Addr.Instance(addrs.NoKey)
		ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("in")))
		ns.DeclareReferenceable(NewConfigExprRef(instAddr, cty.GetAttrPath("out")))
	}
}

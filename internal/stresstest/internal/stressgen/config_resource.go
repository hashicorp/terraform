package stressgen

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/stresstest/internal/stressprovider"
	"github.com/hashicorp/terraform/states"
)

// ConfigResource represents the common parts across both ConfigManagedResource
// and ConfigDataResource.
type ConfigResource struct {
	Addr addrs.Resource

	// ForEachExpr and CountExpr are mutually exclusive and, if set, represent
	// either a "for_each" or "count" meta-argument. Both might be nil, in
	// which case the module call is single-instanced and has neither argument.
	ForEachExpr *ConfigExprForEach
	CountExpr   *ConfigExprCount

	// Arguments are the argument values that the resource configuration will
	// explicitly include. It doesn't include optional arguments which the
	// random generator has chosen to omit in order to let the provider decide
	// the value.
	Arguments map[string]ConfigExpr
}

// ConfigManagedResource is an implementation of ConfigObject representing
// a "resource" block in the configuration.
type ConfigManagedResource struct {
	ConfigResource

	CreateBeforeDestroy bool
}

var _ ConfigObject = (*ConfigManagedResource)(nil)

// ConfigDataResource is an implementation of ConfigObject representing a
// "data" block in the configuration.
type ConfigDataResource struct {
	ConfigResource
}

var _ ConfigObject = (*ConfigDataResource)(nil)

// ConfigResourceInstance represents the common parts across both
// ConfigManagedResourceInstance and ConfigDataResourceInstance.
//
// Due to the collision of terminology here this is a bit confusing and so
// worth defining further: an instance of a resource config represents the
// values defined in a particular "resource"/"data" block in the calling module,
// which is different from the idea of a "resource instance" in Terraform Core,
// which represents the potentially-many "copies" of the resource as a
// result of using "count" or "for_each" in the call.
type ConfigResourceInstance struct {
	Addr addrs.AbsResource

	// InstanceKeys tracks the instance keys we're expecting our declared
	// resource to have based on how it's using "count", "for_each", or neither.
	// For "count" this will contain zero or more int keys, while for "for_each"
	// it will contain zero or more string keys. For neither, it always contains
	// a single element which is addrs.NoKey.
	InstanceKeys []addrs.InstanceKey
}

// ConfigManagedResourceInstance represents a binding between a
// ConfigManagedResource and a particular calling module instance.
//
// See the documentation for ConfigResourceInstance to see how the meaning of
// this type differs from the Terraform Core idea of a "resource instance".
type ConfigManagedResourceInstance struct {
	ConfigResourceInstance
	Obj *ConfigManagedResource

	// ExpectedNames is the expected value for both the "name" and
	// "computed_name" attributes in the final state for each instances.
	ExpectedNames map[addrs.InstanceKey]cty.Value

	// ExpectedForceReplaces is the expected value for the "force_replace"
	// attribute in the final state for each instance.
	ExpectedForceReplaces map[addrs.InstanceKey]cty.Value
}

var _ ConfigObjectInstance = (*ConfigManagedResourceInstance)(nil)

// ConfigDataResourceInstance represents a binding between a
// ConfigDataResource and a particular calling module instance.
//
// See the documentation for ConfigResourceInstance to see how the meaning of
// this type differs from the Terraform Core idea of a "resource instance".
type ConfigDataResourceInstance struct {
	ConfigResourceInstance
	Obj *ConfigDataResource

	// ExpectedValues is the expected value for both the "in" and "out"
	// attributes in the final state for each instance.
	ExpectedValues map[addrs.InstanceKey]cty.Value
}

var _ ConfigObjectInstance = (*ConfigDataResourceInstance)(nil)

// DisplayName implements ConfigObject.DisplayName.
func (o *ConfigResource) DisplayName() string {
	return o.Addr.String()
}

// makeConfigBlock is some common functionality for generating resource
// config blocks, called by both ConfigManagedResource.AppendConfig and
// ConfigDataResource.AppendConfig.
func (o *ConfigResource) makeConfigBlock(typeName string) *hclwrite.Block {
	block := hclwrite.NewBlock(typeName, []string{o.Addr.Type, o.Addr.Name})
	body := block.Body()

	if haveMetaArgs := appendRepetitionMetaArgs(body, o.ForEachExpr, o.CountExpr); haveMetaArgs && len(o.Arguments) > 0 {
		body.AppendNewline()
	}

	for name, expr := range o.Arguments {
		body.SetAttributeRaw(name, expr.BuildExpr().BuildTokens(nil))
	}

	return block
}

// AppendConfig implements ConfigObject.AppendConfig.
func (o *ConfigManagedResource) AppendConfig(to *hclwrite.Body) {
	block := o.makeConfigBlock("resource")
	if o.CreateBeforeDestroy { // any other lifecycle flags in future will also need to be included here
		block.Body().AppendNewline()
		lcBlock := block.Body().AppendNewBlock("lifecycle", nil)
		lcBody := lcBlock.Body()
		if o.CreateBeforeDestroy {
			lcBody.SetAttributeValue("create_before_destroy", cty.True)
		}
	}
	to.AppendBlock(block)
}

// AppendConfig implements ConfigObject.AppendConfig.
func (o *ConfigDataResource) AppendConfig(to *hclwrite.Body) {
	block := o.makeConfigBlock("data")
	to.AppendBlock(block)
}

// GenerateModified implements ConfigObject.GenerateModified.
func (o *ConfigManagedResource) GenerateModified(rnd *rand.Rand, ns *Namespace) ConfigObject {
	declareConfigManagedResource(o, ns)
	return o
}

// GenerateModified implements ConfigObject.GenerateModified.
func (o *ConfigDataResource) GenerateModified(rnd *rand.Rand, ns *Namespace) ConfigObject {
	declareConfigDataResource(o, ns)
	return o
}

// instantiate is the shared common part of both
// ConfigManagedResource.Instantiate and ConfigDataResource.Instantiate.
func (o *ConfigResource) instantiate(reg *Registry) ConfigResourceInstance {
	// Instantiating the call is also the point where we finally expand
	// out the potentially-multiple instances of the resource itself that
	// can be caused by using "count" or "for_each" arguments. Each
	// resource instance has its own separate expected state object, because
	// the exact values used for references may vary between calling module
	// instances.
	instanceKeys := instanceKeysForRepetitionMetaArgs(reg, o.ForEachExpr, o.CountExpr)

	return ConfigResourceInstance{
		Addr:         o.Addr.Absolute(reg.ModuleAddr),
		InstanceKeys: instanceKeys,
	}
}

// Instantiate implements ConfigObject.Instantiate.
func (o *ConfigManagedResource) Instantiate(reg *Registry) ConfigObjectInstance {
	common := o.instantiate(reg)
	ret := &ConfigManagedResourceInstance{
		ConfigResourceInstance: common,
	}

	ret.ExpectedNames = make(map[addrs.InstanceKey]cty.Value, len(common.InstanceKeys))
	ret.ExpectedForceReplaces = make(map[addrs.InstanceKey]cty.Value, len(common.InstanceKeys))
	for _, key := range common.InstanceKeys {
		instAddr := o.Addr.Instance(key)
		expectedName := o.Arguments["name"].ExpectedValue(reg)
		expectedForceReplace := cty.NullVal(cty.String)
		if expr, ok := o.Arguments["force_replace"]; ok {
			expectedForceReplace = expr.ExpectedValue(reg)
		}
		ret.ExpectedNames[key] = expectedName
		ret.ExpectedForceReplaces[key] = expectedForceReplace
		reg.RegisterRefValue(instAddr, cty.ObjectVal(map[string]cty.Value{
			"name":          expectedName,
			"computed_name": expectedName,
		}))
	}

	return ret
}

// Instantiate implements ConfigObject.Instantiate.
func (o *ConfigDataResource) Instantiate(reg *Registry) ConfigObjectInstance {
	common := o.instantiate(reg)
	ret := &ConfigDataResourceInstance{
		ConfigResourceInstance: common,
	}

	ret.ExpectedValues = make(map[addrs.InstanceKey]cty.Value, len(common.InstanceKeys))
	for _, key := range common.InstanceKeys {
		instAddr := o.Addr.Instance(key)
		expectedValue := o.Arguments["in"].ExpectedValue(reg)
		ret.ExpectedValues[key] = expectedValue
		reg.RegisterRefValue(instAddr, cty.ObjectVal(map[string]cty.Value{
			"in":  expectedValue,
			"out": expectedValue,
		}))
	}

	return ret
}

// DisplayName implements ConfigObjectInstance.DisplayName.
func (o *ConfigResourceInstance) DisplayName() string {
	return o.Addr.String()
}

// Object implements ConfigObjectInstance.Object.
func (o *ConfigManagedResourceInstance) Object() ConfigObject {
	return o.Obj
}

// Object implements ConfigObjectInstance.Object.
func (o *ConfigDataResourceInstance) Object() ConfigObject {
	return o.Obj
}

// checkState is the common CheckState functionality shared across both
// ConfigManagedResourceInstance.CheckState and
// ConfigDataResourceInstance.CheckState.
//
// This only checks the new state, because the idea of a "prior state" is
// only relevant to managed resources, and so must be checked elsewhere if
// needed.
func (o *ConfigResourceInstance) checkState(state *states.State, f func(addrs.InstanceKey, *states.ResourceInstanceObjectSrc) []error) []error {
	log.Printf("Checking %s", o.DisplayName())

	var errs []error
	for _, key := range o.InstanceKeys {
		instanceAddr := o.Addr.Instance(key)
		log.Printf("Checking %s", instanceAddr)

		gotState := state.ResourceInstance(instanceAddr)
		if gotState == nil {
			errs = append(errs, fmt.Errorf("%s: instance is missing from state", instanceAddr))
			continue
		}

		// We're testing happy paths, so all instances should end up
		// having a current object that is "ready".
		obj := gotState.Current
		if obj == nil {
			errs = append(errs, fmt.Errorf("%s: instance has no current object", instanceAddr))
			continue
		}
		if status := obj.Status; status != states.ObjectReady {
			errs = append(errs, fmt.Errorf("%s: current object is %#v", instanceAddr, status))
		}

		// Deposed objects should only show up if a destroy fails, and we're
		// trying to exercise only happy paths here, so nothing should
		// ever appear as deposed.
		if len(gotState.Deposed) != 0 {
			errs = append(errs, fmt.Errorf("%s: instance has deposed objects", instanceAddr))
		}

		errs = append(errs, f(key, obj)...)
	}

	return errs
}

// CheckState implements ConfigObjectInstance.CheckState.
func (o *ConfigManagedResourceInstance) CheckState(prior, new *states.State) []error {
	return o.checkState(new, func(key addrs.InstanceKey, stateSrc *states.ResourceInstanceObjectSrc) []error {
		instanceAddr := o.Addr.Instance(key)
		var errs []error

		state, err := stateSrc.Decode(stressprovider.ManagedResourceTypeSchema.Block.ImpliedType())
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid state data: %s", instanceAddr, err))
			return errs
		}
		gotData := state.Value

		if got, want := gotData.GetAttr("name"), o.ExpectedNames[key]; !want.RawEquals(got) {
			errs = append(errs, ErrUnexpected{
				Message: fmt.Sprintf("%s: wrong 'name'", instanceAddr),
				Got:     got,
				Want:    want,
			})
		}
		if got, want := gotData.GetAttr("computed_name"), o.ExpectedNames[key]; !want.RawEquals(got) {
			errs = append(errs, ErrUnexpected{
				Message: fmt.Sprintf("%s: wrong 'computed_name'", instanceAddr),
				Got:     got,
				Want:    want,
			})
		}
		if got, want := gotData.GetAttr("force_replace"), o.ExpectedForceReplaces[key]; !want.RawEquals(got) {
			errs = append(errs, ErrUnexpected{
				Message: fmt.Sprintf("%s: wrong 'force_replace'", instanceAddr),
				Got:     got,
				Want:    want,
			})
		}
		return errs
	})
}

// CheckState implements ConfigObjectInstance.CheckState.
func (o *ConfigDataResourceInstance) CheckState(prior, new *states.State) []error {
	return o.checkState(new, func(key addrs.InstanceKey, stateSrc *states.ResourceInstanceObjectSrc) []error {
		instanceAddr := o.Addr.Instance(key)
		var errs []error

		state, err := stateSrc.Decode(stressprovider.DataResourceTypeSchema.Block.ImpliedType())
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: invalid state data: %s", instanceAddr, err))
			return errs
		}
		gotData := state.Value

		if got, want := gotData.GetAttr("in"), o.ExpectedValues[key]; !want.RawEquals(got) {
			errs = append(errs, ErrUnexpected{
				Message: fmt.Sprintf("%s: wrong 'in'", instanceAddr),
				Got:     got,
				Want:    want,
			})
		}
		if got, want := gotData.GetAttr("out"), o.ExpectedValues[key]; !want.RawEquals(got) {
			errs = append(errs, ErrUnexpected{
				Message: fmt.Sprintf("%s: wrong 'out'", instanceAddr),
				Got:     got,
				Want:    want,
			})
		}
		// TODO: Also test provider_value, to make sure it matches the
		// value associated with the selected provider configuration.
		// (At the time of writing this, we don't have explicit provider
		// configurations randomly generated yet.)
		return errs
	})
}

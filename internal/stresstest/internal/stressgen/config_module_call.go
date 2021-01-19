package stressgen

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/states"
)

// ConfigModuleCall is an implementation of ConfigObject representing the
// declaration of a child module call.
type ConfigModuleCall struct {
	Addr addrs.ModuleCall

	// ChildNamespace is the namespace representing the static contents of the
	// child module.
	ChildNamespace *Namespace

	// ForEachExpr and CountExpr are mutually exclusive and, if set, represent
	// either a "for_each" or "count" meta-argument. Both might be nil, in
	// which case the module call is single-instanced and has neither argument.
	ForEachExpr *ConfigExprForEach
	CountExpr   *ConfigExprCount

	// Even if a module uses count / for_each to declare multiple instances,
	// all of those instances always share the same configuration, and so
	// Objects describes that shared configuration.
	// ConfigModuleCallInstance.ObjectInstances then captures the
	// per-module-instance instantiations of those.
	Objects []ConfigObject

	// Arguments are the variable values that this call will expicitly set
	// inside its module block. It doesn't include input variables that
	// the child module has declared as optional and which the call will
	// just leave to take on their default values.
	Arguments map[addrs.InputVariable]ConfigExpr
}

var _ ConfigObject = (*ConfigModuleCall)(nil)

// ConfigModuleCallInstance represents the binding of a ConfigModuleCall to
// a particular module instance.
//
// Due to the collision of terminology here this is a bit confusing and so
// worth defining further: an instance of a module _call_ represents the
// values defined in a particular "module" block in the calling module,
// which is different from the idea of a "module instance" in Terraform Core,
// which represents the potentially-many "copies" of the definitions inside
// that module as a result of using "count" or "for_each" in the call.
type ConfigModuleCallInstance struct {
	CallerAddr addrs.ModuleInstance
	CallAddr   addrs.ModuleCall
	Obj        *ConfigModuleCall

	// InstanceKeys tracks the instance keys we're expecting our declared
	// module to have based on how it's using "count", "for_each", or neither.
	// For "count" this will contain zero or more int keys, while for "for_each"
	// it will contain zero or more string keys. For neither, it always contains
	// a single element which is addrs.NoKey.
	InstanceKeys []addrs.InstanceKey

	// ObjectInstances tracks for each of the instances of the module the
	// instances for each of the objects declared in Obj.Objects. The slices
	// in this map should always be the same length as Objects and
	// the indices should correlate.
	ObjectInstances map[addrs.InstanceKey][]ConfigObjectInstance

	// InstanceRegistries tracks the child Registry we used for evaluating
	// each of the different instances of the module.
	InstanceRegistries map[addrs.InstanceKey]*Registry
}

var _ ConfigObjectInstance = (*ConfigModuleCallInstance)(nil)

// DisplayName implements ConfigObject.DisplayName.
func (o *ConfigModuleCall) DisplayName() string {
	return o.Addr.String()
}

// AppendConfig implements ConfigObject.AppendConfig.
func (o *ConfigModuleCall) AppendConfig(to *hclwrite.Body) {
	block := hclwrite.NewBlock("module", []string{o.Addr.Name})
	body := block.Body()
	body.SetAttributeValue("source", cty.StringVal("./"+o.Addr.Name))
	appendRepetitionMetaArgs(body, o.ForEachExpr, o.CountExpr)
	if len(o.Arguments) > 0 {
		body.AppendNewline()
	}

	for addr, expr := range o.Arguments {
		body.SetAttributeRaw(addr.Name, expr.BuildExpr().BuildTokens(nil))
	}

	to.AppendBlock(block)
}

// GenerateModified implements ConfigObject.GenerateModified.
func (o *ConfigModuleCall) GenerateModified(rnd *rand.Rand, ns *Namespace) ConfigObject {
	return o
}

// Instantiate implements ConfigObject.Instantiate.
func (o *ConfigModuleCall) Instantiate(reg *Registry) ConfigObjectInstance {
	// Instantiating the call is also the point where we finally expand
	// out the potentially-multiple instances of the module itself that
	// can be caused by using "count" or "for_each" arguments. Each
	// module instance has its own separate instances of the configuration
	// objects, to allow for each one to potentially take different variable
	// values.

	instanceKeys := instanceKeysForRepetitionMetaArgs(reg, o.ForEachExpr, o.CountExpr)
	instInsts := make(map[addrs.InstanceKey][]ConfigObjectInstance, len(instanceKeys))
	regs := make(map[addrs.InstanceKey]*Registry, len(instanceKeys))
	for _, key := range instanceKeys {
		childReg := reg.NewChild(addrs.ModuleInstanceStep{
			Name:        o.Addr.Name,
			InstanceKey: key,
		})
		regs[key] = childReg

		// Before we instantiate the objects we must make sure that any of
		// the variables we're expected to set have expected values recorded
		// in the registry, so other objects can derive from them.
		for addr, expr := range o.Arguments {
			// Arguments are evaluated in the parent module, so we're
			// using reg rather than childReg here...
			v := expr.ExpectedValue(reg)

			// ...but the actual variable value belongs to the child registry,
			// so that variable declarations in the child can access them.
			childReg.RegisterVariableValue(addr, v)
		}

		insts := make([]ConfigObjectInstance, len(o.Objects))
		for i, obj := range o.Objects {
			insts[i] = obj.Instantiate(childReg)
		}
		instInsts[key] = insts
	}

	return &ConfigModuleCallInstance{
		CallerAddr:         reg.ModuleAddr,
		CallAddr:           o.Addr,
		Obj:                o,
		InstanceKeys:       instanceKeys,
		ObjectInstances:    instInsts,
		InstanceRegistries: regs,
	}
}

// DisplayName implements ConfigObjectInstance.DisplayName.
func (o *ConfigModuleCallInstance) DisplayName() string {
	if o.CallerAddr.IsRoot() {
		return o.CallAddr.String()
	}
	return o.CallerAddr.String() + "." + o.CallAddr.String()
}

// Object implements ConfigObjectInstance.Object.
func (o *ConfigModuleCallInstance) Object() ConfigObject {
	return o.Obj
}

// CheckState implements ConfigObjectInstance.CheckState.
func (o *ConfigModuleCallInstance) CheckState(prior, new *states.State) []error {
	log.Printf("Checking %s", o.DisplayName())

	var errs []error

	// Each of the module instances we've declared should now have corresponding
	// container objects in the state.
	for _, key := range o.InstanceKeys {
		instanceAddr := o.CallerAddr.Child(o.CallAddr.Name, key)
		log.Printf("Checking %s", instanceAddr)
		if ms := new.Module(instanceAddr); ms == nil {
			errs = append(errs, fmt.Errorf("no module state for %s", instanceAddr))
		}
	}

	// We also need to delegate to all of our child module object instances
	// and give them a chance to raise errors.
	for _, insts := range o.ObjectInstances {
		for _, inst := range insts {
			errs = append(errs, inst.CheckState(prior, new)...)
		}
	}

	return errs
}

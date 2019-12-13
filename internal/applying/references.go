package applying

import (
	"github.com/hashicorp/terraform/addrs"
)

// configReferences is a higher-level representation of []addrs.Referenceable
// describing a set of references from one object to another that are
// implied by configuration.
//
// This can also include dependencies from the state, which were originally
// recorded from configuration in an earlier operation.
//
// This type only captures information about references that require edges
// to be created in the dependency graph.
type configReferences struct {
	InputVariables   map[string]addrs.AbsInputVariableInstance
	LocalValues      map[string]addrs.AbsLocalValue
	OutputValues     map[string]addrs.AbsOutputValue
	AllModuleOutputs map[string]addrs.ModuleInstance
	Resources        map[string]addrs.AbsResource
}

func findConfigReferences(moduleAddr addrs.ModuleInstance, refAddrs []addrs.Referenceable) configReferences {
	ret := configReferences{
		InputVariables:   make(map[string]addrs.AbsInputVariableInstance),
		LocalValues:      make(map[string]addrs.AbsLocalValue),
		OutputValues:     make(map[string]addrs.AbsOutputValue),
		AllModuleOutputs: make(map[string]addrs.ModuleInstance),
		Resources:        make(map[string]addrs.AbsResource),
	}
	for _, addr := range refAddrs {
		switch addr := addr.(type) {
		case addrs.InputVariable:
			absAddr := moduleAddr.InputVariable(addr.Name)
			ret.InputVariables[absAddr.String()] = absAddr
		case addrs.LocalValue:
			absAddr := addr.Absolute(moduleAddr)
			ret.LocalValues[absAddr.String()] = absAddr
		case addrs.ModuleCallOutput:
			childModuleAddr := addr.Call.ModuleInstance(moduleAddr)
			absAddr := childModuleAddr.OutputValue(addr.Name)
			ret.OutputValues[absAddr.String()] = absAddr
		case addrs.ModuleCall:
			absAddr := moduleAddr.Child(addr.Name, addrs.NoKey)
			ret.AllModuleOutputs[absAddr.String()] = absAddr
		case addrs.ModuleCallInstance:
			absAddr := moduleAddr.Child(addr.Call.Name, addr.Key)
			ret.AllModuleOutputs[absAddr.String()] = absAddr
		case addrs.Resource:
			absAddr := addr.Absolute(moduleAddr)
			ret.Resources[absAddr.String()] = absAddr
		case addrs.ResourceInstance:
			absAddr := addr.ContainingResource().Absolute(moduleAddr)
			ret.Resources[absAddr.String()] = absAddr
		}
	}
	return ret
}

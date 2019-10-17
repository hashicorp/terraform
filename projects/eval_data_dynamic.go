package projects

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/projects/projectconfigs"
	"github.com/hashicorp/terraform/projects/projectlang"
)

type dynamicEvalData struct {
	config           *projectconfigs.Config
	workspaceOutputs map[addrs.ProjectWorkspace]map[addrs.OutputValue]cty.Value
	contextValues    map[addrs.ProjectContextValue]cty.Value
}

var _ projectlang.DynamicEvaluateData = (*dynamicEvalData)(nil)

func (d *dynamicEvalData) BaseDir() string {
	return d.config.ProjectRoot
}

func (d *dynamicEvalData) LocalValueExpr(addr addrs.LocalValue) hcl.Expression {
	return d.config.Locals[addr.Name].Value
}

func (d *dynamicEvalData) ContextValue(addr addrs.ProjectContextValue) cty.Value {
	return d.contextValues[addr]
}

func (d *dynamicEvalData) WorkspaceConfigValue(addr addrs.ProjectWorkspaceConfig) cty.Value {
	noKeyAddr := addr.Instance(addrs.NoKey)
	if noKeyOutputs, exists := d.workspaceOutputs[noKeyAddr]; exists {
		attrs := make(map[string]cty.Value, len(noKeyOutputs))
		for outputAddr, v := range noKeyOutputs {
			attrs[outputAddr.Name] = v
		}
		return cty.ObjectVal(attrs)
	}
	objs := make(map[string]cty.Value)
	for instAddr, outputs := range d.workspaceOutputs {
		if instAddr.Config() != addr {
			continue
		}
		attrs := make(map[string]cty.Value, len(outputs))
		for outputAddr, v := range outputs {
			attrs[outputAddr.Name] = v
		}
		objs[string(instAddr.Key.(addrs.StringKey))] = cty.ObjectVal(attrs)
	}
	return cty.ObjectVal(objs)
}

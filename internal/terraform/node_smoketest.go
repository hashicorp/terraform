package terraform

import (
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeExpandSmokeTest struct {
	addr   addrs.ConfigSmokeTest
	config *configs.SmokeTest

	makeInstance func(addrs.AbsSmokeTest, *configs.SmokeTest) dag.Vertex
}

var _ GraphNodeModulePath = (*nodeExpandSmokeTest)(nil)
var _ GraphNodeDynamicExpandable = (*nodeExpandSmokeTest)(nil)

func (n *nodeExpandSmokeTest) Name() string {
	return n.addr.String() + " (expand)"
}

// GraphNodeModulePath implementation
func (n *nodeExpandSmokeTest) ModulePath() addrs.Module {
	return n.addr.Module
}

// GraphNodeDynamicExpandable implementation
func (n *nodeExpandSmokeTest) DynamicExpand(tfCtx EvalContext) (*Graph, error) {
	exp := tfCtx.InstanceExpander()
	modInsts := exp.ExpandModule(n.ModulePath())

	var g Graph
	for _, modAddr := range modInsts {
		testAddr := n.addr.SmokeTest.Absolute(modAddr)
		inst := n.makeInstance(testAddr, n.config)
		g.Add(inst)
	}

	return &g, nil
}

type nodeSmokeTestPre struct {
	addr   addrs.AbsSmokeTest
	config *configs.SmokeTest
}

var _ GraphNodeModuleInstance = (*nodeSmokeTestPre)(nil)
var _ GraphNodeExecutable = (*nodeSmokeTestPre)(nil)

func (n *nodeSmokeTestPre) Name() string {
	return n.addr.String() + " (preconditions)"
}

// GraphNodeModuleInstance implementation
func (n *nodeSmokeTestPre) Path() addrs.ModuleInstance {
	return n.addr.Module
}

// GraphNodeExecutable implementation
func (n *nodeSmokeTestPre) Execute(tfCtx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	return diags
}

type nodeSmokeTestPost struct {
	addr   addrs.AbsSmokeTest
	config *configs.SmokeTest
}

var _ GraphNodeModuleInstance = (*nodeSmokeTestPost)(nil)
var _ GraphNodeExecutable = (*nodeSmokeTestPost)(nil)

func (n *nodeSmokeTestPost) Name() string {
	return n.addr.String() + " (postconditions)"
}

// GraphNodeModuleInstance implementation
func (n *nodeSmokeTestPost) Path() addrs.ModuleInstance {
	return n.addr.Module
}

// GraphNodeExecutable implementation
func (n *nodeSmokeTestPost) Execute(tfCtx EvalContext, op walkOperation) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	return diags
}

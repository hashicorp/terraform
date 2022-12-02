package terraform

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

type nodeExpandSmokeTest struct {
	typeName string
	addr     addrs.ConfigSmokeTest
	config   *configs.SmokeTest

	makeInstance  func(addrs.AbsSmokeTest, *configs.SmokeTest) dag.Vertex
	reportObjects bool
}

var _ GraphNodeModulePath = (*nodeExpandSmokeTest)(nil)
var _ GraphNodeDynamicExpandable = (*nodeExpandSmokeTest)(nil)

func (n *nodeExpandSmokeTest) Name() string {
	return fmt.Sprintf("%s (%s)", n.addr.String(), n.typeName)
}

// GraphNodeModulePath implementation
func (n *nodeExpandSmokeTest) ModulePath() addrs.Module {
	return n.addr.Module
}

// GraphNodeDynamicExpandable implementation
func (n *nodeExpandSmokeTest) DynamicExpand(tfCtx EvalContext) (*Graph, error) {
	exp := tfCtx.InstanceExpander()
	modInsts := exp.ExpandModule(n.ModulePath())

	instAddrs := addrs.MakeSet[addrs.Checkable]()
	var g Graph
	for _, modAddr := range modInsts {
		testAddr := n.addr.SmokeTest.Absolute(modAddr)
		log.Printf("[TRACE] nodeExpandSmokeTest: Node for %s", testAddr)
		instAddrs.Add(testAddr)
		inst := n.makeInstance(testAddr, n.config)
		g.Add(inst)
	}
	addRootNodeToGraph(&g)

	if n.reportObjects {
		// We must report all of our instances so that the checkState will expect
		// reports from us later.
		checkState := tfCtx.Checks()
		checkState.ReportCheckableObjects(n.addr, instAddrs)
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
	if op != walkApply {
		// We only actually evaluate smoke test preconditions during apply.
		//
		// TODO: Should we try to evaluate them during planning too, and
		// just let them be unknown if we don't have enough information?
		// That means we will need to remember which preconditions we
		// already checked during plan and avoid re-reporting them during
		// apply, though.
		return nil
	}

	return evalCheckRules(
		addrs.SmokeTestPrecondition,
		n.config.Preconditions,
		tfCtx,
		n.addr, EvalDataForNoInstanceKey,
		tfdiags.Warning, // FIXME: Should only report errors as diagnostics, and failures only via the check state
	)
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

	if op != walkApply {
		// We only actually evaluate smoke test postconditions during apply.
		//
		// TODO: Should we try to evaluate them during planning too, and
		// just let them be unknown if we don't have enough information?
		// That means we will need to remember which postconditions we
		// already checked during plan and avoid re-reporting them during
		// apply, though.
		return nil
	}

	// Before we take any real actions, we'll check if any of our preconditions
	// or data resources already failed and just skip if so, since the
	// postconditions are likely to be invalid in that case.
	checkState := tfCtx.Checks()
	precondStatus := checkState.ObjectCheckStatusByConditionType(n.addr, addrs.SmokeTestPrecondition)
	if precondStatus == checks.StatusFail || precondStatus == checks.StatusError {
		// If it's already failing then it can't get any more successful.
		log.Printf("[TRACE] nodeSmokeTestPost: Skipping %s because its preconditions failed", n.addr)
		return diags
	}
	dataStatus := checkState.ObjectCheckStatusByConditionType(n.addr, addrs.SmokeTestDataResource)
	if dataStatus == checks.StatusFail || dataStatus == checks.StatusError {
		// If it's already failing then it can't get any more successful.
		log.Printf("[TRACE] nodeSmokeTestPost: Skipping %s because its data resources failed", n.addr)
		return diags
	}

	moreDiags := evalCheckRules(
		addrs.SmokeTestPostcondition,
		n.config.Postconditions,
		tfCtx,
		n.addr, EvalDataForNoInstanceKey,
		tfdiags.Warning, // FIXME: Should only report errors as diagnostics, and failures only via the check state
	)
	diags = diags.Append(moreDiags)

	return diags
}

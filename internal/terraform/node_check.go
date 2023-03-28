package terraform

import (
	"log"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/checks"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/lang"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

var (
	_ GraphNodeModulePath = (*nodeReportCheck)(nil)
	_ GraphNodeExecutable = (*nodeReportCheck)(nil)
)

// nodeReportCheck calls the ReportCheckableObjects function for our assertions
// within the check blocks.
//
// We need this to happen before the checks are actually verified and before any
// nested data blocks, so the creator of this structure should make sure this
// node is a parent of any nested data blocks.
//
// This needs to be separate to nodeExpandCheck, because the actual checks
// should happen after referenced data blocks rather than before.
type nodeReportCheck struct {
	addr addrs.ConfigCheck
}

func (n *nodeReportCheck) ModulePath() addrs.Module {
	return n.addr.Module
}

func (n *nodeReportCheck) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {
	exp := ctx.InstanceExpander()
	modInsts := exp.ExpandModule(n.ModulePath())

	instAddrs := addrs.MakeSet[addrs.Checkable]()
	for _, modAddr := range modInsts {
		instAddrs.Add(n.addr.Check.Absolute(modAddr))
	}
	ctx.Checks().ReportCheckableObjects(n.addr, instAddrs)
	return nil
}

func (n *nodeReportCheck) Name() string {
	return n.addr.String() + " (report)"
}

var (
	_ GraphNodeModulePath        = (*nodeExpandCheck)(nil)
	_ GraphNodeDynamicExpandable = (*nodeExpandCheck)(nil)
	_ GraphNodeReferencer        = (*nodeExpandCheck)(nil)
)

// nodeExpandCheck creates child nodes that actually execute the assertions for
// a given check block.
//
// This must happen after any other nodes/resources/data sources that are
// referenced, so we implement GraphNodeReferencer.
//
// This needs to be separate to nodeReportCheck as nodeReportCheck must happen
// first, while nodeExpandCheck must execute after any referenced blocks.
type nodeExpandCheck struct {
	addr   addrs.ConfigCheck
	config *configs.Check

	makeInstance func(addrs.AbsCheck, *configs.Check) dag.Vertex
}

func (n *nodeExpandCheck) ModulePath() addrs.Module {
	return n.addr.Module
}

func (n *nodeExpandCheck) DynamicExpand(ctx EvalContext) (*Graph, error) {
	exp := ctx.InstanceExpander()
	modInsts := exp.ExpandModule(n.ModulePath())

	var g Graph
	for _, modAddr := range modInsts {
		testAddr := n.addr.Check.Absolute(modAddr)
		log.Printf("[TRACE] nodeExpandCheck: Node for %s", testAddr)
		g.Add(n.makeInstance(testAddr, n.config))
	}
	addRootNodeToGraph(&g)

	return &g, nil
}

func (n *nodeExpandCheck) References() []*addrs.Reference {
	var refs []*addrs.Reference
	for _, assert := range n.config.Asserts {
		condition, _ := lang.ReferencesInExpr(assert.Condition)
		message, _ := lang.ReferencesInExpr(assert.ErrorMessage)
		refs = append(refs, condition...)
		refs = append(refs, message...)
	}
	return refs
}

func (n *nodeExpandCheck) Name() string {
	return n.addr.String() + " (expand)"
}

var (
	_ GraphNodeModuleInstance = (*nodeCheckAssert)(nil)
	_ GraphNodeExecutable     = (*nodeCheckAssert)(nil)
)

type nodeCheckAssert struct {
	addr   addrs.AbsCheck
	config *configs.Check

	// We only want to actually execute the checks during the plan and apply
	// operations, but we still want to validate our config during
	// other operations.
	executeChecks bool

	// We only want to report the checks during select operations.
	//
	// For example, when a plan is auto approved we won't pollute the output
	// with check results from the plan that can't be used by the user.
	raiseChecks bool
}

func (n *nodeCheckAssert) ModulePath() addrs.Module {
	return n.Path().Module()
}

func (n *nodeCheckAssert) Path() addrs.ModuleInstance {
	return n.addr.Module
}

func (n *nodeCheckAssert) Execute(ctx EvalContext, _ walkOperation) tfdiags.Diagnostics {

	// We only want to actually execute the checks during specific
	// operations, such as plan and applies.
	if n.executeChecks {
		if status := ctx.Checks().ObjectCheckStatus(n.addr); status == checks.StatusFail || status == checks.StatusError {
			// This check is already failing, so we won't try and evaluate it.
			// This typically means there was an error in a data block within
			// the check block.
			return nil
		}

		severity := CheckSeverityWarning
		if !n.raiseChecks {
			severity = CheckSeveritySkip
		}

		return evalCheckRules(
			addrs.CheckAssertion,
			n.config.Asserts,
			ctx,
			n.addr,
			EvalDataForNoInstanceKey,
			severity)
	}

	// Otherwise let's still validate the config and references and return
	// diagnostics if references do not exist etc.
	var diags tfdiags.Diagnostics
	for _, assert := range n.config.Asserts {
		_, _, moreDiags := validateCheckRule(addrs.CheckAssertion, assert, ctx, n.addr, EvalDataForNoInstanceKey)
		diags = diags.Append(moreDiags)
	}
	return diags
}

func (n *nodeCheckAssert) Name() string {
	return n.addr.String() + " (assertions)"
}

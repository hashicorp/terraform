// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// nodeExpandPlannableResource represents an addrs.ConfigResource and implements
// DynamicExpand to a subgraph containing all of the addrs.AbsResourceInstance
// resulting from both the containing module and resource-specific expansion.
type nodeExpandPlannableResource struct {
	*NodeAbstractResource

	// forceCreateBeforeDestroy might be set via our GraphNodeDestroyerCBD
	// during graph construction, if dependencies require us to force this
	// on regardless of what the configuration says.
	forceCreateBeforeDestroy bool

	// skipRefresh indicates that we should skip refreshing individual instances
	skipRefresh bool

	preDestroyRefresh bool

	// skipPlanChanges indicates we should skip trying to plan change actions
	// for any instances.
	skipPlanChanges bool

	// minimalRefresh indicates that we should run an initial plan for each instance prior to refreshing:
	//   - If the plan returns a no-op, then the instance won't be refreshed.
	//   - If the plan returns a change (anything but no-op), the instance will be refreshed and another plan will be run.
	minimalRefresh bool

	// forceReplace are resource instance addresses where the user wants to
	// force generating a replace action. This set isn't pre-filtered, so
	// it might contain addresses that have nothing to do with the resource
	// that this node represents, which the node itself must therefore ignore.
	forceReplace []addrs.AbsResourceInstance

	// We attach dependencies to the Resource during refresh, since the
	// instances are instantiated during DynamicExpand.
	// FIXME: These would be better off converted to a generic Set data
	// structure in the future, as we need to compare for equality and take the
	// union of multiple groups of dependencies.
	dependencies []addrs.ConfigResource

	excludes []addrs.Targetable
}

var (
	_ GraphNodeCreateBeforeDestroy  = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeDynamicExpandable    = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeReferenceable        = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeReferencer           = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeImportReferencer     = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeConfigResource       = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeAttachResourceConfig = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeAttachDependencies   = (*nodeExpandPlannableResource)(nil)
	_ GraphNodeTargetable           = (*nodeExpandPlannableResource)(nil)
)

func (n *nodeExpandPlannableResource) Name() string {
	return n.NodeAbstractResource.Name() + " (expand)"
}

// GraphNodeAttachDependencies
func (n *nodeExpandPlannableResource) AttachDependencies(deps []addrs.ConfigResource) {
	n.dependencies = deps
}

// GraphNodeAttachExcludes
func (n *nodeExpandPlannableResource) AttachExcludes(excludes []addrs.Targetable) {
	n.excludes = excludes
}

// GraphNodeDestroyerCBD
func (n *nodeExpandPlannableResource) CreateBeforeDestroy() bool {
	if n.forceCreateBeforeDestroy {
		return true
	}

	// If we have no config, we just assume no
	if n.Config == nil || n.Config.Managed == nil {
		return false
	}

	return n.Config.Managed.CreateBeforeDestroy
}

// GraphNodeDestroyerCBD
func (n *nodeExpandPlannableResource) ForceCreateBeforeDestroy() {
	n.forceCreateBeforeDestroy = true
}

func (n *nodeExpandPlannableResource) DynamicExpand(ctx EvalContext) (*Graph, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// First, make sure the count and the foreach don't refer to the same
	// resource. The config maybe nil if we are generating configuration, or
	// deleting a resource.
	if n.Config != nil {
		diags = diags.Append(validateMetaSelfRef(n.Addr.Resource, n.Config.Count))
		diags = diags.Append(validateMetaSelfRef(n.Addr.Resource, n.Config.ForEach))
		if diags.HasErrors() {
			return nil, diags
		}
	}

	// Expand the current module.
	expander := ctx.InstanceExpander()
	moduleInstances := expander.ExpandModule(n.Addr.Module, false)

	// Expand the imports for this resource.
	imports, unknownImports, importDiags := n.expandResourceImports(ctx, ctx.Deferrals().DeferralAllowed())
	diags = diags.Append(importDiags)

	// if allowUnknown was set to false in expandResourceImports, we should
	// not have any unknown imports.
	if !ctx.Deferrals().DeferralAllowed() && unknownImports.Len() > 0 {
		panic("unexpected unknown imports")
	}

	pem := expander.UnknownModuleInstances(n.Addr.Module, false)
	g, expandDiags := n.dynamicExpand(ctx, moduleInstances, pem, imports, unknownImports)
	diags = diags.Append(expandDiags)
	return g, diags
}

// Import blocks are expanded in conjunction with their associated resource block.
func (n *nodeExpandPlannableResource) expandResourceImports(ctx EvalContext, allowUnknown bool) (addrs.Map[addrs.AbsResourceInstance, cty.Value], addrs.Map[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]], tfdiags.Diagnostics) {
	// Imports maps the target address to an import ID.
	knownImports := addrs.MakeMap[addrs.AbsResourceInstance, cty.Value]()
	unknownImports := addrs.MakeMap[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]]()
	var diags tfdiags.Diagnostics

	if len(n.importTargets) == 0 {
		return knownImports, unknownImports, diags
	}

	state := ctx.State()

	for _, imp := range n.importTargets {
		if imp.Config == nil {
			// if we have a legacy addr, it was supplied on the commandline so
			// there is nothing to expand
			if !imp.LegacyAddr.Equal(addrs.AbsResourceInstance{}) {
				knownImports.Put(imp.LegacyAddr, cty.StringVal(imp.LegacyID))
				return knownImports, unknownImports, diags
			}

			// legacy import tests may have no configuration
			log.Printf("[WARN] no configuration for import target %#v", imp)
			continue
		}

		// "to" here needs the containing resource
		// but I don't know how to work that out at this point (with expansion)
		// do I need to get all possible expansions for the imp.RelModule and then only use the one for this actual node?
		allMods := ctx.InstanceExpander().AllInstances().InstancesForModule(imp.RelModule, false)
		for _, mod := range allMods {
			// get the context from the module the import was defined in
			ctx = evalContextForModuleInstance(ctx, mod)
			state = ctx.State()
			if imp.Config.ForEach == nil {
				traversal, hds := hcl.AbsTraversalForExpr(imp.Config.To)
				diags = diags.Append(hds)
				to, tds := addrs.ParseAbsResourceInstance(traversal)
				diags = diags.Append(tds)
				if diags.HasErrors() {
					return knownImports, unknownImports, diags
				}
				// add the module that the import block was configured in to the resource addr
				to.Module = append(mod, to.Module...)

				diags = diags.Append(validateImportTargetExpansion(n.Config, to, imp.Config.To))

				var importID cty.Value
				var evalDiags tfdiags.Diagnostics
				if imp.Config.ID != nil {
					importID, evalDiags = evaluateImportIdExpression(imp.Config.ID, ctx, EvalDataForNoInstanceKey, allowUnknown)
				} else if imp.Config.Identity != nil {
					providerSchema, err := ctx.ProviderSchema(n.ResolvedProvider)
					if err != nil {
						diags = diags.Append(err)
						return knownImports, unknownImports, diags
					}
					schema := providerSchema.SchemaForResourceAddr(to.Resource.Resource)

					importID, evalDiags = evaluateImportIdentityExpression(imp.Config.Identity, schema.Identity, ctx, EvalDataForNoInstanceKey, allowUnknown)
				} else {
					// Should never happen
					return knownImports, unknownImports, diags
				}

				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return knownImports, unknownImports, diags
				}

				// If we already have an import statement for this resource instance, it
				// must have come from a parent module, because duplicate import blocks in
				// the same module result in an error.
				//
				// The import block in the parent module overrides the block in the child module.
				if knownImports.Has(to) {
					continue
				}

				knownImports.Put(to, importID)

				log.Printf("[TRACE] expandResourceImports: found single import target %s", to)
				continue
			}

			forEachData, known, forEachDiags := newForEachEvaluator(imp.Config.ForEach, ctx, allowUnknown).ImportValues()
			diags = diags.Append(forEachDiags)
			if forEachDiags.HasErrors() {
				return knownImports, unknownImports, diags
			}

			if !known {
				// Then we need to parse the target address as a PartialResource
				// instead of a known resource.
				addr, evalDiags := evalImportUnknownToExpression(imp.Config.To)
				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return knownImports, unknownImports, diags
				}

				// We're going to work out which instances this import block might
				// target actually already exist.
				knownInstances := addrs.MakeSet[addrs.AbsResourceInstance]()

				cfg := addr.ConfigResource()
				modInsts := state.ModuleInstances(append(imp.RelModule, cfg.Module...))
				for _, modInst := range modInsts {
					abs := cfg.Absolute(modInst)
					resource := state.Resource(cfg.Absolute(modInst))
					if resource == nil {
						// Then we are creating every instance of this resource.
						continue
					}

					for inst := range resource.Instances {
						knownInstances.Add(abs.Instance(inst))
					}
				}

				unknownImports.Put(addr, knownInstances)
				continue
			}

			for _, keyData := range forEachData {
				var evalDiags tfdiags.Diagnostics
				res, evalDiags := evalImportToExpression(imp.Config.To, keyData)
				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return knownImports, unknownImports, diags
				}
				// add the module that the import block was configured in to the resource addr
				res.Module = append(mod, res.Module...)

				diags = diags.Append(validateImportTargetExpansion(n.Config, res, imp.Config.To))

				var importID cty.Value
				if imp.Config.ID != nil {
					importID, evalDiags = evaluateImportIdExpression(imp.Config.ID, ctx, keyData, allowUnknown)
				} else if imp.Config.Identity != nil {
					providerSchema, err := ctx.ProviderSchema(n.ResolvedProvider)
					if err != nil {
						diags = diags.Append(err)
						return knownImports, unknownImports, diags
					}
					schema := providerSchema.SchemaForResourceAddr(res.Resource.Resource)

					importID, evalDiags = evaluateImportIdentityExpression(imp.Config.Identity, schema.Identity, ctx, keyData, allowUnknown)
				} else {
					// Should never happen
					return knownImports, unknownImports, diags
				}

				diags = diags.Append(evalDiags)
				if diags.HasErrors() {
					return knownImports, unknownImports, diags
				}

				// If we already have an import statement for this resource instance, it
				// must have come from a parent module, because duplicate import blocks in
				// the same module result in an error.
				//
				// The import block in the parent module overrides the block in the child module.
				if knownImports.Has(res) {
					continue
				}

				knownImports.Put(res, importID)
				log.Printf("[TRACE] expandResourceImports: expanded import target %s", res)
			}
		}
	}

	// filter out any known import which already exist in state
	for _, el := range knownImports.Elements() {
		// if the resource exists in state, but not config, we will not remove
		// the target and instead let validateImportTargets return the proper
		// "missing config" error
		if n.Config != nil {
			if state.ResourceInstance(el.Key) != nil {
				log.Printf("[DEBUG] expandResourceImports: skipping import address %s already in state", el.Key)
				knownImports.Remove(el.Key)
			}
		}
	}

	return knownImports, unknownImports, diags
}

func (n *nodeExpandPlannableResource) concreteResource(ctx EvalContext, knownImports addrs.Map[addrs.AbsResourceInstance, cty.Value], unknownImports addrs.Map[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]], skipPlanChanges bool) func(*NodeAbstractResourceInstance) dag.Vertex {
	return func(a *NodeAbstractResourceInstance) dag.Vertex {
		var m *NodePlannableResourceInstance

		// If we're in legacy import mode (the import CLI command), we only need
		// to return the import node, not a plannable resource node.
		for _, importTarget := range n.importTargets {
			if importTarget.LegacyAddr.Equal(a.Addr) {

				// If we're in the legacy import mode, then we should never
				// see unknown imports. So, it's fine to just look at the known
				// imports here.
				idValue := knownImports.Get(importTarget.LegacyAddr)

				return &graphNodeImportState{
					Addr:             importTarget.LegacyAddr,
					ID:               idValue.AsString(),
					ResolvedProvider: n.ResolvedProvider,
				}
			}
		}

		// Add the config and state since we don't do that via transforms
		a.Config = n.Config
		a.ResolvedProvider = n.ResolvedProvider
		a.Schema = n.Schema
		a.ProvisionerSchemas = n.ProvisionerSchemas
		a.ProviderMetas = n.ProviderMetas
		a.dependsOn = n.dependsOn
		a.Dependencies = n.dependencies
		a.preDestroyRefresh = n.preDestroyRefresh
		a.generateConfigPath = n.generateConfigPath
		a.actionTriggers = n.actionTriggers

		m = &NodePlannableResourceInstance{
			NodeAbstractResourceInstance: a,

			// By the time we're walking, we've figured out whether we need
			// to force on CreateBeforeDestroy due to dependencies on other
			// nodes that have it.
			ForceCreateBeforeDestroy: n.CreateBeforeDestroy(),
			excludes:                 n.excludes,
			skipRefresh:              n.skipRefresh,
			skipPlanChanges:          skipPlanChanges,
			minimalRefresh:           n.minimalRefresh,
			forceReplace:             slices.ContainsFunc(n.forceReplace, a.Addr.Equal),
		}

		if importID, ok := knownImports.GetOk(a.Addr); ok {
			m.importTarget = importTarget{
				target:       importID,
				importConfig: n.importTargets[0].Config,
			}
		} else {
			// We're going to check now if this resource instance *might* be
			// targeted by one of the unknown imports. If it is, we'll set the
			// import target to an unknown value so that the import operation
			// will be deferred.
			for _, unknownImport := range unknownImports.Elems {
				if unknownImport.Key.MatchesInstance(a.Addr) {
					if unknownImport.Value.Has(a.Addr) {
						// This means that this particular instance already
						// exists within the state. `import` blocks that target
						// instances that already exist are ignored by
						// Terraform. This means that even if this unknown
						// import does eventually resolve to this instance then
						// it would be ignored anyway. So for this instance we
						// won't set the import target.
						continue
					}

					m.importTarget = importTarget{target: cty.UnknownVal(cty.String)}
				}
			}
		}

		return m
	}
}

func (n *nodeExpandPlannableResource) concreteResourceOrphan(a *NodeAbstractResourceInstance) dag.Vertex {
	// Add the config and state since we don't do that via transforms
	a.Config = n.Config
	a.ResolvedProvider = n.ResolvedProvider
	a.Schema = n.Schema
	a.ProvisionerSchemas = n.ProvisionerSchemas
	a.ProviderMetas = n.ProviderMetas
	a.actionTriggers = n.actionTriggers

	return &NodePlannableResourceInstanceOrphan{
		NodeAbstractResourceInstance: a,
		// -minimal-refresh optimizes to skip refreshing when destroying / deleting instances
		skipRefresh:     n.skipRefresh || n.minimalRefresh,
		skipPlanChanges: n.skipPlanChanges,
		excludes:        n.excludes,
	}
}

func (n *nodeExpandPlannableResource) validForceReplaceTargets(instanceAddrs []addrs.AbsResourceInstance) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	for _, candidateAddr := range n.forceReplace {
		if candidateAddr.Resource.Key == addrs.NoKey {
			if n.Addr.Resource.Equal(candidateAddr.Resource.Resource) {
				switch {
				case len(instanceAddrs) == 0:
					// In this case there _are_ no instances to replace, so
					// there isn't any alternative address for us to suggest.
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Incompletely-matched force-replace resource instance",
						fmt.Sprintf(
							"Your force-replace request for %s doesn't match any resource instances because this resource doesn't have any instances.",
							candidateAddr,
						),
					))
				case len(instanceAddrs) == 1:
					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Incompletely-matched force-replace resource instance",
						fmt.Sprintf(
							"Your force-replace request for %s doesn't match any resource instances because it lacks an instance key.\n\nTo force replacement of the single declared instance, use the following option instead:\n  -replace=%q",
							candidateAddr, instanceAddrs[0],
						),
					))
				default:
					var possibleValidOptions strings.Builder
					for _, addr := range instanceAddrs {
						fmt.Fprintf(&possibleValidOptions, "\n  -replace=%q", addr)
					}

					diags = diags.Append(tfdiags.Sourceless(
						tfdiags.Warning,
						"Incompletely-matched force-replace resource instance",
						fmt.Sprintf(
							"Your force-replace request for %s doesn't match any resource instances because it lacks an instance key.\n\nTo force replacement of particular instances, use one or more of the following options instead:%s",
							candidateAddr, possibleValidOptions.String(),
						),
					))
				}
			}
		}
	}

	return diags
}

func (n *nodeExpandPlannableResource) dynamicExpand(ctx EvalContext, knownModules []addrs.ModuleInstance, partialModules addrs.Set[addrs.PartialExpandedModule], knownImports addrs.Map[addrs.AbsResourceInstance, cty.Value], unknownImports addrs.Map[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]]) (*Graph, tfdiags.Diagnostics) {
	var g Graph
	var diags tfdiags.Diagnostics

	knownResources := addrs.MakeSet[addrs.AbsResourceInstance]()
	partialResources := addrs.MakeSet[addrs.PartialExpandedResource]()
	maybeOrphanResources := addrs.MakeSet[addrs.AbsResourceInstance]()

	for _, moduleAddr := range knownModules {
		resourceAddr := n.Addr.Resource.Absolute(moduleAddr)
		resources, partials, maybeOrphans, moreDiags := n.expandKnownModule(ctx, resourceAddr, knownImports, unknownImports, &g)
		diags = diags.Append(moreDiags)

		if diags.HasErrors() {
			return nil, diags
		}

		// Track all the resources we know about.
		knownResources = knownResources.Union(resources)
		partialResources = partialResources.Union(partials)
		maybeOrphanResources = maybeOrphanResources.Union(maybeOrphans)
	}

	for _, moduleAddr := range partialModules {
		resourceAddr := moduleAddr.Resource(n.Addr.Resource)
		partialResources.Add(resourceAddr)

		// And add a node to the graph for this resource.
		g.Add(&nodePlannablePartialExpandedResource{
			addr:              resourceAddr,
			config:            n.Config,
			resolvedProvider:  n.ResolvedProvider,
			skipPlanChanges:   n.skipPlanChanges,
			preDestroyRefresh: n.preDestroyRefresh,
		})
	}

	func() {
		ss := ctx.PrevRunState()
		if ss == nil {
			return // No previous state, so nothing to do here.
		}
		state := ss.Lock()
		defer ss.Unlock()

	Resources:
		for _, res := range state.Resources(n.Addr) {

			for _, knownModule := range knownModules {
				if knownModule.Equal(res.Addr.Module) {
					// Then we handled this resource as part of the known
					// modules processing.
					continue Resources
				}
			}

			for _, partialResource := range partialResources {
				if partialResource.MatchesResource(res.Addr) {

					for key := range res.Instances {
						// Then each of the instances is a "maybe orphan"
						// instance, and we need to add a node for that.
						maybeOrphanResources.Add(res.Addr.Instance(key))
						g.Add(n.concreteResource(ctx, addrs.MakeMap[addrs.AbsResourceInstance, cty.Value](), addrs.MakeMap[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]](), true)(NewNodeAbstractResourceInstance(res.Addr.Instance(key))))
					}

					// Move onto the next resource.
					continue Resources
				}
			}

			// Otherwise, everything in here is just a simple orphaned instance.

			for key := range res.Instances {
				inst := res.Addr.Instance(key)
				abs := NewNodeAbstractResourceInstance(inst)
				abs.AttachResourceState(res)
				g.Add(n.concreteResourceOrphan(abs))
			}
		}
	}()

	// We might expect an address because it's in an import block, but have no
	// config and aren't generating any. This isn't caught during validation
	// because generateConfigPath is only a plan option.
	missingImportConfig := false

	// We need to ensure that all of the expanded import targets are actually
	// present in the configuration, because we can't import something that
	// doesn't exist.
	for _, addr := range knownImports.Keys() {
		expectedAddr := false
		if knownResources.Has(addr) {
			expectedAddr = true
		}

		for _, partialAddr := range partialResources {
			if partialAddr.MatchesInstance(addr) {
				// This is a partial-expanded address, so we can't yet know
				// whether it's in the configuration or not, and so we'll
				// defer dealing with it to a future round.
				expectedAddr = true
				break
			}
		}

		if expectedAddr && n.Config == nil && n.generateConfigPath == "" {
			missingImportConfig = true
			continue
		}

		if !expectedAddr {
			// If we get here then the import target is not in the configuration
			// at all, and so we'll report an error.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration for import target does not exist",
				fmt.Sprintf("The configuration for the given import %s does not exist. All target instances must have an associated configuration to be imported.", addr),
			))
		}
	}

	// We'll also perform the same kind of validation on our unknown imports.
	// This will be less precise because we don't have the full state to
	// compare against, but we can at least check that the import targets are
	// in the configuration.
	for _, elem := range unknownImports.Elems {
		unknownImport := elem.Key
		expectedAddr := false

		for _, resource := range knownResources {
			if unknownImport.MatchesInstance(resource) {
				// This is in the configuration so we can skip it.
				expectedAddr = true
			}
		}

		for _, partialResource := range partialResources {
			// If the partial resource is a subset of the unknown import, or
			// vice versa, then it *might* match up one day once everything
			// is resolved so we'll allow it for now.
			if partialResource.MatchesPartial(unknownImport) {
				expectedAddr = true
			}
			if unknownImport.MatchesPartial(partialResource) {
				expectedAddr = true
			}
		}

		if expectedAddr && n.Config == nil && n.generateConfigPath == "" {
			missingImportConfig = true
			continue
		}

		if !expectedAddr {
			// If we get here then the import target is not in the configuration
			// at all, and so we'll report an error.
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Configuration for import target does not exist",
				fmt.Sprintf("The configuration for the given import %s does not exist. All target instances must have an associated configuration to be imported.", unknownImport),
			))
		}
	}

	if missingImportConfig {
		// We expect the address because it's in an import block, but we
		// have no config and aren't generating any. This isn't caught
		// during validation because generateConfigPath is only a plan
		// option. If we got this far however, it means this node is
		// eligible for config generation, so suggest it to the user.
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Configuration for import target does not exist",
			Detail:   fmt.Sprintf("The configuration for the given import target %s does not exist. If you wish to automatically generate config for this resource, use the -generate-config-out option within terraform plan. Otherwise, make sure the target resource exists within your configuration. For example:\n\n  terraform plan -generate-config-out=generated.tf", n.Addr),
			Subject:  n.importTargets[0].Config.To.Range().Ptr(),
		})
	}

	// If this is a resource that participates in custom condition checks
	// (i.e. it has preconditions or postconditions) then the check state
	// wants to know the addresses of the checkable objects so that it can
	// treat them as unknown status if we encounter an error before actually
	// visiting the checks.
	if checkState := ctx.Checks(); checkState.ConfigHasChecks(n.NodeAbstractResource.Addr) {
		checkables := addrs.MakeSet[addrs.Checkable]()
		for _, addr := range knownResources {
			checkables.Add(addr)
		}
		for _, addr := range maybeOrphanResources {
			checkables.Add(addr)
		}

		checkState.ReportCheckableObjects(n.NodeAbstractResource.Addr, checkables)
	}

	return &g, diags
}

func (n *nodeExpandPlannableResource) expandKnownModule(globalCtx EvalContext, resAddr addrs.AbsResource, knownImports addrs.Map[addrs.AbsResourceInstance, cty.Value], unknownImports addrs.Map[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]], g *Graph) (addrs.Set[addrs.AbsResourceInstance], addrs.Set[addrs.PartialExpandedResource], addrs.Set[addrs.AbsResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	moduleCtx := evalContextForModuleInstance(globalCtx, resAddr.Module)

	moreDiags := n.recordResourceData(moduleCtx, resAddr)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, nil, nil, diags
	}

	expander := moduleCtx.InstanceExpander()
	_, knownInstKeys, haveUnknownKeys := expander.ResourceInstanceKeys(resAddr)

	knownResources := addrs.MakeSet[addrs.AbsResourceInstance]()
	partialResources := addrs.MakeSet[addrs.PartialExpandedResource]()

	for _, key := range knownInstKeys {
		knownResources.Add(resAddr.Instance(key))
	}
	if haveUnknownKeys {
		partialResources.Add(resAddr.Module.UnexpandedResource(resAddr.Resource))
	}

	mustHaveIndex := len(knownInstKeys) != 1 || haveUnknownKeys
	if len(knownInstKeys) == 1 && knownInstKeys[0] != addrs.NoKey {
		mustHaveIndex = true
	}
	if mustHaveIndex {
		var instanceAddrs []addrs.AbsResourceInstance
		for _, key := range knownInstKeys {
			instanceAddrs = append(instanceAddrs, resAddr.Instance(key))
		}
		diags = diags.Append(n.validForceReplaceTargets(instanceAddrs))
	}

	instGraph, maybeOrphanResources, instDiags := n.knownModuleSubgraph(moduleCtx, resAddr, knownInstKeys, haveUnknownKeys, knownImports, unknownImports)
	diags = diags.Append(instDiags)
	if instDiags.HasErrors() {
		return nil, nil, nil, diags
	}
	g.Subsume(&instGraph.AcyclicGraph.Graph)
	return knownResources, partialResources, maybeOrphanResources, diags
}

func (n *nodeExpandPlannableResource) knownModuleSubgraph(ctx EvalContext, addr addrs.AbsResource, knownInstKeys []addrs.InstanceKey, haveUnknownKeys bool, knownImports addrs.Map[addrs.AbsResourceInstance, cty.Value], unknownImports addrs.Map[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]]) (*Graph, addrs.Set[addrs.AbsResourceInstance], tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if n.Config == nil && n.generateConfigPath != "" && knownImports.Len() == 0 {
		// We're generating configuration, but there's nothing to import, which
		// means the import block must have expanded to zero instances.
		// the instance expander will always return a single instance because
		// we have assumed there will eventually be a configuration for this
		// resource, so return here before we add that to the graph.
		return &Graph{}, nil, diags
	}

	// Our graph transformers require access to the full state, so we'll
	// temporarily lock it while we work on this.
	state := ctx.State().Lock()
	defer ctx.State().Unlock()

	maybeOrphans := addrs.MakeSet[addrs.AbsResourceInstance]()

	steps := []GraphTransformer{

		DynamicTransformer(func(graph *Graph) error {
			// We'll add a node for all the known instance keys.
			for _, key := range knownInstKeys {
				graph.Add(n.concreteResource(ctx, knownImports, unknownImports, n.skipPlanChanges)(NewNodeAbstractResourceInstance(addr.Instance(key))))
			}
			return nil
		}),

		DynamicTransformer(func(graph *Graph) error {
			// We'll add a node if there are unknown instance keys.
			if haveUnknownKeys {
				addr := addr.Module.UnexpandedResource(addr.Resource)

				graph.Add(&nodePlannablePartialExpandedResource{
					addr:              addr,
					config:            n.Config,
					resolvedProvider:  n.ResolvedProvider,
					skipPlanChanges:   n.skipPlanChanges,
					preDestroyRefresh: n.preDestroyRefresh,
				})
			}
			return nil
		}),

		DynamicTransformer(func(graph *Graph) error {
			// Ephemeral resources don't need to be accounted for in this transform,
			// since they are not in the state.
			if addr.Resource.Mode == addrs.EphemeralResourceMode {
				return nil
			}

			// We'll add nodes for any orphaned resources.
			rs := state.Resource(addr)
			if rs == nil {
				return nil
			}
		Instances:
			for key, inst := range rs.Instances {
				if inst.Current == nil {
					continue
				}

				for _, knownKey := range knownInstKeys {
					if knownKey == key {
						// Then we have a known instance, so we can skip this
						// one - it's definitely not an orphan.
						continue Instances
					}
				}

				if haveUnknownKeys {
					// Then this is a "maybe orphan" instance. It isn't mapped
					// to a known instance but we have unknown keys so we don't
					// know for sure that it's been deleted.
					maybeOrphans.Add(addr.Instance(key))
					graph.Add(n.concreteResource(ctx, addrs.MakeMap[addrs.AbsResourceInstance, cty.Value](), addrs.MakeMap[addrs.PartialExpandedResource, addrs.Set[addrs.AbsResourceInstance]](), true)(NewNodeAbstractResourceInstance(addr.Instance(key))))
					continue
				}

				// If none of the above, then this is definitely an orphan.
				graph.Add(n.concreteResourceOrphan(NewNodeAbstractResourceInstance(addr.Instance(key))))
			}

			return nil
		}),

		// Attach the state
		&AttachStateTransformer{State: state},

		// Targeting
		&TargetsTransformer{Targets: n.Targets},
	}

	b := &BasicGraphBuilder{
		Steps: steps,
		Name:  "nodeExpandPlannableResource",
	}
	graph, graphDiags := b.Build(addr.Module)
	diags = diags.Append(graphDiags)
	return graph, maybeOrphans, diags
}

// transformDynamic is a helper struct that wraps a single function, allowing
// us to transform a graph dynamically.
type transformDynamic struct {
	Transformer func(*Graph) error
}

// DynamicTransformer returns a GraphTransformer that will apply the given
// function to the graph during the dynamic expansion phase.
func DynamicTransformer(f func(*Graph) error) GraphTransformer {
	return &transformDynamic{Transformer: f}
}

// implements GraphTransformer
func (t *transformDynamic) Transform(g *Graph) error {
	return t.Transformer(g)
}

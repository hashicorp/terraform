package projects

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/projects/projectconfigs"
	"github.com/hashicorp/terraform/projects/projectlang"
	"github.com/hashicorp/terraform/tfdiags"
)

// Project represents a single Terraform project, which is a container for
// zero or more workspaces.
type Project struct {
	config *projectconfigs.Config

	workspaceEachVals map[addrs.ProjectWorkspace]cty.Value
}

// FindProject finds the project that contains the given start directory, if
// any.
func FindProject(startDir string) (*Project, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	rootDir, err := projectconfigs.FindProjectRoot(startDir)
	if err != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Couldn't find project root",
			Detail:   fmt.Sprintf("Can't find the root directory of the current project: %s.", err),
		})
		return nil, diags
	}

	config, moreDiags := projectconfigs.Load(rootDir)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}
	project, moreDiags := newProject(config)
	diags = diags.Append(moreDiags)
	return project, diags
}

func newProject(config *projectconfigs.Config) (*Project, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	evalData := newStaticEvalData(config)
	eachVals := make(map[addrs.ProjectWorkspace]cty.Value)

	var wsWithForEach []*projectconfigs.Workspace
	var usWithForEach []*projectconfigs.Upstream
	var forEachExprs []hcl.Expression
	for _, ws := range config.Workspaces {
		if ws.ForEach != nil {
			wsWithForEach = append(wsWithForEach, ws)
			forEachExprs = append(forEachExprs, ws.ForEach)
		} else {
			eachVals[ws.InstanceAddr(addrs.NoKey)] = cty.NilVal
		}
	}
	for _, us := range config.Upstreams {
		if us.ForEach != nil {
			usWithForEach = append(usWithForEach, us)
			forEachExprs = append(forEachExprs, us.ForEach)
		} else {
			eachVals[us.InstanceAddr(addrs.NoKey)] = cty.NilVal
		}
	}

	forEachVals, moreDiags := projectlang.StaticEvaluateExprs(forEachExprs, evalData)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

Vals:
	for i, val := range forEachVals {
		if val.IsNull() {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each value",
				Detail:   "The for_each argument cannot be null.",
				Subject:  forEachExprs[i].Range().Ptr(),
			})
			continue
		}
		var baseAddr addrs.ProjectWorkspaceConfig
		if i < len(wsWithForEach) {
			baseAddr = wsWithForEach[i].Addr()
		} else {
			baseAddr = usWithForEach[i-len(wsWithForEach)].Addr()
		}

		if !(val.Type().IsSetType() || val.Type().IsMapType() || val.Type().IsObjectType()) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each value",
				Detail:   "The for_each argument value must be a map or a set of strings.",
				Subject:  forEachExprs[i].Range().Ptr(),
			})
			continue
		}
		if val.Type().IsSetType() && !val.Type().ElementType().Equals(cty.String) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid for_each value",
				Detail:   "If the for_each argument value is a set then it must be a set of strings.",
				Subject:  forEachExprs[i].Range().Ptr(),
			})
			continue
		}

		for it := val.ElementIterator(); it.Next(); {
			keyVal, val := it.Element()
			if val.Type().IsSetType() {
				keyVal = val
			}
			if keyVal.IsNull() {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each value",
					Detail:   "A for_each set value must not contain null values.",
					Subject:  forEachExprs[i].Range().Ptr(),
				})
				continue Vals
			}
			key := keyVal.AsString()
			if !hclsyntax.ValidIdentifier(key) {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid for_each key",
					Detail:   "Cannot use %q has a workspace instance key. Instance keys must be a letter followed by zero or more letters, underscores, or dashes.",
					Subject:  forEachExprs[i].Range().Ptr(),
				})
				continue Vals
			}
			addr := baseAddr.Instance(addrs.StringKey(key))
			eachVals[addr] = val
		}
	}

	if diags.HasErrors() {
		return nil, diags
	}

	return &Project{
		config:            config,
		workspaceEachVals: eachVals,
	}, diags
}

// AllWorkspaceAddrs returns the addresses of all of the workspaces defined
// in the project.
//
// This expands each "workspace" block in the configuration to zero or more
// actual workspaces, based on the for_each expression. The result contains
// one element per workspace.
func (p *Project) AllWorkspaceAddrs() []addrs.ProjectWorkspace {
	ret := make([]addrs.ProjectWorkspace, 0, len(p.workspaceEachVals))
	for addr := range p.workspaceEachVals {
		if addr.Rel != addrs.ProjectWorkspaceCurrent {
			continue
		}
		ret = append(ret, addr)
	}
	sort.Stable(sortWorkspaceAddrs(ret))
	return ret
}

// WorkspaceConfigDependencies returns the addresses of the workspace
// configurations that the workspace with the given address depends on.
//
// This is intended to support callers that want to calculate a dependency
// graph of workspaces e.g. to automate applying them all in a particular
// sequence. If a workspace is applied without first applying its dependencies
// then the apply could either fail or operate based on stale information.
//
// Note that dependencies are between workspace configurations, not individual
// workspaces. This is because we allow workspace instance keys to be selected
// dynamically, but the references between workspace configurations themselves
// are required to be static.
func (p *Project) WorkspaceConfigDependencies(ws addrs.ProjectWorkspaceConfig) []addrs.ProjectWorkspaceConfig {
	var exprs []hcl.Expression
	if ws.Rel != addrs.ProjectWorkspaceCurrent {
		// Only workspaces in the current project can have dependencies in
		// the context of this project. (Upstream workspaces might have
		// dependencies too, but you'd need to ask the upstream project
		// configuration what they are.)
		return nil
	}
	cfg := p.config.Workspaces[ws.Name]
	if cfg == nil {
		return nil
	}

	if cfg.Variables != nil {
		exprs = append(exprs, cfg.Variables)
	}
	if cfg.ConfigSource != nil {
		exprs = append(exprs, cfg.ConfigSource)
	}
	if cfg.Remote != nil {
		exprs = append(exprs, cfg.Remote)
	}
	// TODO: Also cfg.StateStorage, but that requires a schema for the state storage implementation

	evalData := newStaticEvalData(p.config)
	return projectlang.RequiredWorkspaceConfigsForExprs(exprs, evalData)
}

// WorkspaceDependencies is a helper wrapper around WorkspaceConfigDependencies
// that adapts it to work with individual workspaces rather than workspace
// configurations.
//
// Because dependencies are derived from configuration, they are always between
// workspace configurations rather than individual workspaces. However, since
// we know the full set of workspaces we can indirectly determine the full set
// of workspaces that are needed for another workspace by just listing out
// all of the workspace instances belonging to the required workspace
// configurations.
//
// BUG(ma): This currently only includes current-project workspaces in the
// result, because we don't yet have any function to get a list of
// resolved upstream workspace instances.
func (p *Project) WorkspaceDependencies(ws addrs.ProjectWorkspace) []addrs.ProjectWorkspace {
	cfgAddrs := p.WorkspaceConfigDependencies(ws.Config())
	wsAddrs := p.AllWorkspaceAddrs() // FIXME: This doesn't include upstreams
	var ret []addrs.ProjectWorkspace
	for _, cfgAddr := range cfgAddrs {
		for _, wsAddr := range wsAddrs {
			if wsAddr.Config() == cfgAddr {
				ret = append(ret, wsAddr)
			}
		}
	}
	return ret
}

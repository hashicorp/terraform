package projects

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/projects/projectconfigs"
	"github.com/hashicorp/terraform/projects/projectlang"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/tfdiags"
)

// Workspace represents a single selectable workspace, each of which has its
// own configuration directory, state, and input variables.
type Workspace struct {
	project   *Project
	addr      addrs.ProjectWorkspace
	configDir string
	variables map[addrs.InputVariable]cty.Value

	// TODO: Also the remote config or state storage config, but for now
	// we're just forcing local state as a prototype.
}

// LoadWorkspace instantiates a specific workspace from the receiving project.
//
// In order to do so, it resolves any references in the workspace configuration
// that the workspace belongs to, which might involve retrieving the output
// values from other workspaces, evaluating local values, etc.
//
// This can be a relatively expensive operation in a project that has lots of
// remote workspaces, so it should be used carefully. A caller that needs to
// work with many workspaces at once might be better off using
// LoadAllWorkspaces, which is able to optimize its work to ensure that each
// workspace is only accessed once, and also avoids producing the same errors
// multiple times if a given expression in the configuration contributes to
// multiple workspaces.
func (m *ProjectManager) LoadWorkspace(addr addrs.ProjectWorkspace) (*Workspace, tfdiags.Diagnostics) {
	wss, diags := m.loadWorkspaces([]addrs.ProjectWorkspace{addr})
	return wss[addr], diags
}

// LoadAllWorkspaces is like LoadWorkspace but loads all of the project's
// workspaces at once, as efficiently as possible.
//
// This is a better alternative to LoadWorkspace for callers that need to work
// with all or most of the workspaces in a project at once, because it's able
// to optimize its work and avoid duplicate calls. However, it's not suitable
// for callers that intend to work only with a single workspace because it is
// likely to fetch more data than necessary and will fail if the current user
// does not have access to any of the workspace outputs, even if those
// outputs would not normally be needed to process a particular selected
// workspace.
func (m *ProjectManager) LoadAllWorkspaces() (map[addrs.ProjectWorkspace]*Workspace, tfdiags.Diagnostics) {
	return m.loadWorkspaces(m.project.AllWorkspaceAddrs())
}

// loadWorkspaces is the common implementation of both LoadWorkspace and
// LoadAllWorkspaces, which loads all of the workspaces requested in the
// given address slice, and as a side-effect fetches outputs for any other
// workspaces that the given ones depend on, fetching each workspace at most
// once.
//
// The resulting map includes Workspace objects only for the requested
// addresses, even if others needed to be fetched in order to complete the
// operation.
func (m *ProjectManager) loadWorkspaces(wantedAddrs []addrs.ProjectWorkspace) (map[addrs.ProjectWorkspace]*Workspace, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	if len(wantedAddrs) == 0 {
		return nil, nil
	}
	ret := make(map[addrs.ProjectWorkspace]*Workspace, len(wantedAddrs))

	// First we'll follow the dependencies of what's requested to expand out
	// our full list of workspaces to fetch. Along the way we'll keep track
	// of the dependencies we found so we can use them for a topological sort
	// below. The keys in "deps" after this loop represent our full set of
	// needed workspaces.
	deps := make(map[addrs.ProjectWorkspace][]addrs.ProjectWorkspace)
	for _, needAddr := range wantedAddrs {
		deps[needAddr] = m.project.WorkspaceDependencies(needAddr)
	}
	for {
		// We'll keep iterating until we stop finding new workspaces.
		// This must converge eventually because the number of workspaces
		// is finite itself, and so the worst case is that we load all of
		// the workspaces.
		new := false
		for _, needAddrs := range deps {
			for _, needAddr := range needAddrs {
				if _, exists := deps[needAddr]; exists {
					continue
				}
				new = true
				deps[needAddr] = m.project.WorkspaceDependencies(needAddr)
			}
		}

		if !new {
			break
		}
	}

	// NOTE: Strictly speaking we only need to fetch the outputs for the
	// workspaces that are referenced by the ones requested, not for the
	// ones requested directly. However, for the sake of simplicity we'll
	// fetch all of them here. In a real implementation we'd likely want to
	// optimize this in a number of ways, including avoiding fetching things
	// we don't need to fetch _and_ doing our fetches as concurrently as
	// possible.

	workspaces := make(map[addrs.ProjectWorkspace]*Workspace)
	workspaceOutputs := make(map[addrs.ProjectWorkspace]map[addrs.OutputValue]cty.Value)
	dependents := make(map[addrs.ProjectWorkspace][]addrs.ProjectWorkspace)
	inDegree := make(map[addrs.ProjectWorkspace]int)
	for addr, needAddrs := range deps {
		for _, needAddr := range needAddrs {
			inDegree[addr]++
			dependents[needAddr] = append(dependents[needAddr], addr)
		}
	}
	var queue []addrs.ProjectWorkspace
	for addr := range deps {
		if inDegree[addr] == 0 {
			queue = append(queue, addr)
		}
	}
	for len(queue) > 0 {
		var addr addrs.ProjectWorkspace
		addr, queue = queue[0], queue[1:] // dequeue next item
		if val := workspaces[addr]; val != nil {
			continue // Already dealt with this one
		}
		delete(inDegree, addr)
		log.Printf("[TRACE] projects.ProjectManager.loadWorkspaces: evaluating configuration for workspace %s", addr)
		switch addr.Rel {
		case addrs.ProjectWorkspaceCurrent:
			cfg := m.project.config.Workspaces[addr.Name]
			if cfg == nil {
				panic(fmt.Sprintf("no config for %s", addr))
			}
			workspace, moreDiags := m.initCurrentProjectWorkspace(addr, cfg, workspaceOutputs)
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				// Downstream references are likely to fail too if we failed
				// to init this project, so we'll just bail out early.
				return nil, diags
			}
			workspaces[addr] = workspace
			outputs, moreDiags := workspace.LatestOutputValues()
			diags = diags.Append(moreDiags)
			if moreDiags.HasErrors() {
				return nil, diags
			}
			workspaceOutputs[addr] = outputs
		case addrs.ProjectWorkspaceUpstream:
			// TODO: Skipping this for now, since we're just prototyping.
			panic("upstream workspaces not yet supported")
		default:
			panic("unsupported workspace relationship")
		}

		for _, referrerAddr := range dependents[addr] {
			inDegree[referrerAddr]--
			if inDegree[referrerAddr] < 1 {
				queue = append(queue, referrerAddr)
			}
		}
	}

	for _, wantAddr := range wantedAddrs {
		ret[wantAddr] = workspaces[wantAddr]
	}

	return ret, diags
}

func (m *ProjectManager) initCurrentProjectWorkspace(addr addrs.ProjectWorkspace, cfg *projectconfigs.Workspace, otherWorkspaceOutputs map[addrs.ProjectWorkspace]map[addrs.OutputValue]cty.Value) (*Workspace, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Workspace{
		addr:    addr,
		project: m.project,
	}

	// This is a little more convoluted than we'd ideally like because we want
	// to evaluate all of our expressions in a single call into projectlang so
	// that we don't produce redundant diagnostic messages, but we don't always
	// have explicit values for all of the arguments.
	const configExpr = 0
	const variablesExpr = 1
	exprs := []hcl.Expression{
		configExpr:    hcl.StaticExpr(cty.NullVal(cty.String), cfg.DeclRange.ToHCL()),
		variablesExpr: hcl.StaticExpr(cty.EmptyObjectVal, cfg.DeclRange.ToHCL()),
	}

	if cfg.ConfigSource != nil {
		exprs[configExpr] = cfg.ConfigSource
	}
	if cfg.Variables != nil {
		exprs[variablesExpr] = cfg.Variables
	}

	data := &dynamicEvalData{
		config:           m.project.config,
		workspaceOutputs: otherWorkspaceOutputs,
		contextValues:    m.ContextValues(),
	}
	each := projectlang.NoEach
	if addr.Key != addrs.NoKey {
		each.Key = addr.Key
		each.Value = m.project.workspaceEachVals[addr]
	}

	vals, moreDiags := projectlang.DynamicEvaluateExprs(
		exprs,
		data, each,
	)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	if !vals[configExpr].IsNull() {
		err := gocty.FromCtyValue(vals[configExpr], &ret.configDir)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid workspace configuration root",
				Detail:   fmt.Sprintf("Invalid root module path for workspace: %s.", tfdiags.FormatError(err)),
				Subject:  exprs[configExpr].Range().Ptr(),
			})
		}
	} else {
		ret.configDir = "."
	}

	if obj := vals[variablesExpr]; !obj.IsNull() {
		if !(obj.Type().IsObjectType() || obj.Type().IsMapType()) {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid workspace variables",
				Detail:   "Invalid value for \"variables\" argument: must be a mapping from variable name to corresponding variable value.",
				Subject:  exprs[variablesExpr].Range().Ptr(),
			})
		} else {
			raw := obj.AsValueMap()
			ret.variables = make(map[addrs.InputVariable]cty.Value, len(raw))
			for n, v := range raw {
				ret.variables[addrs.InputVariable{Name: n}] = v
			}
		}
	}

	// TODO: Also the remote state storage config

	return ret, diags
}

// Addr returns the address of the workspace represented by the reciever.
func (w *Workspace) Addr() addrs.ProjectWorkspace {
	return w.addr
}

// Project returns the project object that this workspace belongs to.
func (w *Workspace) Project() *Project {
	return w.project
}

// ConfigDir returns the path to the directory containing the root module
// for this workspace, relative to the project's root directory.
func (w *Workspace) ConfigDir() string {
	return w.configDir
}

// InputVariables returns the configured input variable values for the
// workspace.
func (w *Workspace) InputVariables() map[addrs.InputVariable]cty.Value {
	return w.variables
}

// StateMgr returns a configured state manager object for this workspace,
// or returns user-oriented diagnistics messages explaining why it cannot.
//
// The configuration for the state storage is validated for syntax as part of
// instantiating the workspace, so errors from this method will generally
// describe "dynamic" problems, such as being unable to connect to a remote
// server identified in the configuration.
func (w *Workspace) StateMgr() (statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	// FIXME: For initial prototyping we're forcing local state at fixed
	// paths on disk. Eventually this should respect the "remote" or
	// "state_storage" settings in the workspace configuration.

	stateDir := filepath.Join(w.project.config.ProjectRoot, ".terraform", "workspaces2-prototype-state")
	stateFilePath := filepath.Join(stateDir, w.addr.StringCompact()+".tfstate")

	err := os.MkdirAll(stateDir, os.ModePerm)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Can't create local state directory",
			fmt.Sprintf("Error creating %s: %s.", stateDir, err),
		))
		return nil, diags
	}

	return statemgr.NewFilesystem(stateFilePath), nil
}

// LatestOutputValues returns the output values recorded at the end of the
// most recent operation against this workspace.
//
// The returned diagnostics might contain errors if the storage for the output
// values is currently unreachable for some reason. In that case, the
// returned map is invalid and must not be used.
func (w *Workspace) LatestOutputValues() (map[addrs.OutputValue]cty.Value, tfdiags.Diagnostics) {
	// For the moment we're still getting outputs from the state directly.
	// Ideally we'd switch to using a specialized API for this when the
	// state is mastered in Terraform Cloud or Enterprise, so that we can
	// apply separate access controls to whole state vs. outputs.

	var diags tfdiags.Diagnostics
	stateMgr, moreDiags := w.StateMgr()
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}

	err := stateMgr.RefreshState()
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Failed to fetch workspace state",
			fmt.Sprintf("Could not retrieve the latest state snapshot for workspace %s in order to determine its latest output values: %s.", w.addr.StringCompact(), err),
		))
		return nil, diags
	}

	state := stateMgr.State()
	if state == nil {
		// FIXME: In order to produce an error message when workspaces are
		// established in the wrong order, the caller will need some way to
		// distinguish between no snapshots yet at all (state == nil) and
		// a snapshot without any outputs. For now these are indistinguishable,
		// so a reference to a workspace not yet established will just produce
		// a useless downstream error about the output not being present.
		return nil, diags
	}
	raw := state.RootModule().OutputValues
	ret := make(map[addrs.OutputValue]cty.Value, len(raw))
	for k, v := range raw {
		ret[addrs.OutputValue{Name: k}] = v.Value
	}

	return ret, nil
}

type sortWorkspaceAddrs []addrs.ProjectWorkspace

var _ sort.Interface = sortWorkspaceAddrs(nil)

func (s sortWorkspaceAddrs) Len() int {
	return len(s)
}

func (s sortWorkspaceAddrs) Less(i, j int) bool {
	switch {
	case s[i].Rel != s[j].Rel:
		return s[i].Rel == addrs.ProjectWorkspaceCurrent
	case s[i].Name != s[j].Name:
		return s[i].Name < s[j].Name
	case s[i].Key != s[j].Key:
		return addrs.InstanceKeyLess(s[i].Key, s[j].Key)
	default:
		return false
	}
}

func (s sortWorkspaceAddrs) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

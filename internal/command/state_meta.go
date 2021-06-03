package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"

	backendLocal "github.com/hashicorp/terraform/internal/backend/local"
)

// StateMeta is the meta struct that should be embedded in state subcommands.
type StateMeta struct {
	Meta
}

// State returns the state for this meta. This gets the appropriate state from
// the backend, but changes the way that backups are done. This configures
// backups to be timestamped rather than just the original state path plus a
// backup path.
func (c *StateMeta) State() (statemgr.Full, error) {
	var realState statemgr.Full
	backupPath := c.backupPath
	stateOutPath := c.statePath

	// use the specified state
	if c.statePath != "" {
		realState = statemgr.NewFilesystem(c.statePath)
	} else {
		// Load the backend
		b, backendDiags := c.Backend(nil)
		if backendDiags.HasErrors() {
			return nil, backendDiags.Err()
		}

		workspace, err := c.Workspace()
		if err != nil {
			return nil, err
		}

		// Check remote Terraform version is compatible
		remoteVersionDiags := c.remoteBackendVersionCheck(b, workspace)
		c.showDiagnostics(remoteVersionDiags)
		if remoteVersionDiags.HasErrors() {
			return nil, fmt.Errorf("Error checking remote Terraform version")
		}

		// Get the state
		s, err := b.StateMgr(workspace)
		if err != nil {
			return nil, err
		}

		// Get a local backend
		localRaw, backendDiags := c.Backend(&BackendOpts{ForceLocal: true})
		if backendDiags.HasErrors() {
			// This should never fail
			panic(backendDiags.Err())
		}
		localB := localRaw.(*backendLocal.Local)
		_, stateOutPath, _ = localB.StatePaths(workspace)
		if err != nil {
			return nil, err
		}

		realState = s
	}

	// We always backup state commands, so set the back if none was specified
	// (the default is "-", but some tests bypass the flag parsing).
	if backupPath == "-" || backupPath == "" {
		// Determine the backup path. stateOutPath is set to the resulting
		// file where state is written (cached in the case of remote state)
		backupPath = fmt.Sprintf(
			"%s.%d%s",
			stateOutPath,
			time.Now().UTC().Unix(),
			DefaultBackupExtension)
	}

	// If the backend is local (which it should always be, given our asserting
	// of it above) we can now enable backups for it.
	if lb, ok := realState.(*statemgr.Filesystem); ok {
		lb.SetBackupPath(backupPath)
	}

	return realState, nil
}

func (c *StateMeta) lookupResourceInstanceAddr(state *states.State, allowMissing bool, addrStr string) ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	target, diags := addrs.ParseTargetStr(addrStr)
	if diags.HasErrors() {
		return nil, diags
	}

	targetAddr := target.Subject
	var ret []addrs.AbsResourceInstance
	switch addr := targetAddr.(type) {
	case addrs.ModuleInstance:
		// Matches all instances within the indicated module and all of its
		// descendent modules.

		// found is used to identify cases where the selected module has no
		// resources, but one or more of its submodules does.
		found := false
		ms := state.Module(addr)
		if ms != nil {
			found = true
			ret = append(ret, c.collectModuleResourceInstances(ms)...)
		}
		for _, cms := range state.Modules {
			if !addr.Equal(cms.Addr) {
				if addr.IsAncestor(cms.Addr) || addr.TargetContains(cms.Addr) {
					found = true
					ret = append(ret, c.collectModuleResourceInstances(cms)...)
				}
			}
		}

		if found == false && !allowMissing {
			diags = diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Unknown module",
				fmt.Sprintf(`The current state contains no module at %s. If you've just added this module to the configuration, you must run "terraform apply" first to create the module's entry in the state.`, addr),
			))
		}

	case addrs.AbsResource:
		// Matches all instances of the specific selected resource.
		rs := state.Resource(addr)
		if rs == nil {
			if !allowMissing {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Unknown resource",
					fmt.Sprintf(`The current state contains no resource %s. If you've just added this resource to the configuration, you must run "terraform apply" first to create the resource's entry in the state.`, addr),
				))
			}
			break
		}
		ret = append(ret, c.collectResourceInstances(addr.Module, rs)...)
	case addrs.AbsResourceInstance:
		is := state.ResourceInstance(addr)
		if is == nil {
			if !allowMissing {
				diags = diags.Append(tfdiags.Sourceless(
					tfdiags.Error,
					"Unknown resource instance",
					fmt.Sprintf(`The current state contains no resource instance %s. If you've just added its resource to the configuration or have changed the count or for_each arguments, you must run "terraform apply" first to update the resource's entry in the state.`, addr),
				))
			}
			break
		}
		ret = append(ret, addr)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Less(ret[j])
	})

	return ret, diags
}

func (c *StateMeta) lookupSingleStateObjectAddr(state *states.State, addrStr string) (addrs.Targetable, tfdiags.Diagnostics) {
	target, diags := addrs.ParseTargetStr(addrStr)
	if diags.HasErrors() {
		return nil, diags
	}
	return target.Subject, diags
}

func (c *StateMeta) lookupResourceInstanceAddrs(state *states.State, addrStrs ...string) ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	var ret []addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics
	for _, addrStr := range addrStrs {
		moreAddrs, moreDiags := c.lookupResourceInstanceAddr(state, false, addrStr)
		ret = append(ret, moreAddrs...)
		diags = diags.Append(moreDiags)
	}
	return ret, diags
}

func (c *StateMeta) lookupAllResourceInstanceAddrs(state *states.State) ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	var ret []addrs.AbsResourceInstance
	var diags tfdiags.Diagnostics
	for _, ms := range state.Modules {
		ret = append(ret, c.collectModuleResourceInstances(ms)...)
	}
	sort.Slice(ret, func(i, j int) bool {
		return ret[i].Less(ret[j])
	})
	return ret, diags
}

func (c *StateMeta) collectModuleResourceInstances(ms *states.Module) []addrs.AbsResourceInstance {
	var ret []addrs.AbsResourceInstance
	for _, rs := range ms.Resources {
		ret = append(ret, c.collectResourceInstances(ms.Addr, rs)...)
	}
	return ret
}

func (c *StateMeta) collectResourceInstances(moduleAddr addrs.ModuleInstance, rs *states.Resource) []addrs.AbsResourceInstance {
	var ret []addrs.AbsResourceInstance
	for key := range rs.Instances {
		ret = append(ret, rs.Addr.Instance(key))
	}
	return ret
}

func (c *StateMeta) lookupAllResources(state *states.State) ([]*states.Resource, tfdiags.Diagnostics) {
	var ret []*states.Resource
	var diags tfdiags.Diagnostics
	for _, ms := range state.Modules {
		for _, resource := range ms.Resources {
			ret = append(ret, resource)
		}
	}
	return ret, diags
}

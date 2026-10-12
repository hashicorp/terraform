// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
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
func (c *StateMeta) State(view arguments.ViewType) (statemgr.Full, error) {
	var realState statemgr.Full
	backupPath := c.backupPath
	stateOutPath := c.statePath

	// use the specified state
	if c.statePath != "" {
		realState = statemgr.NewFilesystem(c.statePath)
	} else {

		// Load the backend
		b, diags := c.backend(".", view)
		if diags.HasErrors() {
			return nil, diags.Err()
		}

		workspace, err := c.Workspace()
		if err != nil {
			return nil, err
		}

		// Check remote Terraform version is compatible
		remoteVersionDiags := c.remoteVersionCheck(b, workspace)
		c.showDiagnostics(remoteVersionDiags)
		if remoteVersionDiags.HasErrors() {
			return nil, fmt.Errorf("Error checking remote Terraform version")
		}

		// Get the state
		s, sDiags := b.StateMgr(workspace)
		if sDiags.HasErrors() {
			return nil, sDiags.Err()
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

// parseStateAddr parses an address given to a state command.
//
// A module address is parsed as a target pattern, so a module call without an
// instance key selects every instance of that module, as it always has, and
// [*] may be used to select every instance explicitly.
//
// A resource or resource instance address is parsed as a concrete address, so
// it can't accidentally select objects in more than one module instance. A
// module call without an instance key in a resource address only refers to
// the instance of a module call without count or for_each, and wildcards are
// not allowed.
func parseStateAddr(addrStr string) (addrs.Targetable, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	traversal, parseDiags := hclsyntax.ParseTraversalPartial([]byte(addrStr), "", hcl.InitialPos)
	diags = diags.Append(parseDiags)
	if parseDiags.HasErrors() {
		return nil, diags
	}

	pattern, moreDiags := addrs.ParseTarget(traversal)
	diags = diags.Append(moreDiags)
	if moreDiags.HasErrors() {
		return nil, diags
	}
	if _, isModule := pattern.ConfigAddr().(addrs.Module); isModule {
		return pattern, diags
	}

	for _, step := range traversal {
		if _, ok := step.(hcl.TraverseSplat); ok {
			return nil, diags.Append(tfdiags.Sourceless(
				tfdiags.Error,
				"Invalid resource address",
				fmt.Sprintf("The address %s contains a wildcard. Wildcards are only supported in module addresses, where they select every instance of a module. A resource address must refer to a single resource or resource instance.", addrStr),
			))
		}
	}

	addr, moreDiags := addrs.ParseAbsTargetable(traversal)
	diags = diags.Append(moreDiags)
	return addr, diags
}

func (c *StateMeta) lookupResourceInstanceAddr(state *states.State, allowMissing bool, addrStr string) ([]addrs.AbsResourceInstance, tfdiags.Diagnostics) {
	target, diags := parseStateAddr(addrStr)
	if diags.HasErrors() {
		return nil, diags
	}

	var ret []addrs.AbsResourceInstance
	switch addr := target.(type) {
	case addrs.TargetPattern:
		// Matches all instances within the selected modules and all of their
		// descendant modules.

		// found is used to identify cases where a selected module has no
		// resources, but one or more of its submodules does.
		found := false
		for _, ms := range state.Modules {
			if addr.Contains(ms.Addr) {
				found = true
				ret = append(ret, c.collectModuleResourceInstances(ms)...)
			}
		}

		if !found && !allowMissing {
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
	return addrs.ParseAbsTargetableStr(addrStr)
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

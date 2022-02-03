package componentstree

import (
	"fmt"
	"path"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/ngaddrs"
	"github.com/hashicorp/terraform/internal/terraform-ng/internal/tfcomponents"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// LoadComponentsTree uses the given loader function to load first the given
// components file and then any other components files it refers to (directly
// or indirectly) via component group declarations.
//
// If the returned diagnostics have no errors, the returned node is the root
// node of the tree. If the diagnostics includes errors, node will be an
// incomplete tree that may be suitable for cautious analysis.
func LoadComponentsTree(sourceAddr addrs.ModuleSource, loader ConfigLoader) (*Node, tfdiags.Diagnostics) {
	// This is a kinda-arbitrarily-chosen capacity to give us some breathing
	// room in this buffer for reasonably-shallow trees without allocation.
	path := make([]ngaddrs.ComponentGroupCall, 0, 4)
	return loadComponentsTree(path, nil, sourceAddr, loader)
}

func loadComponentsTree(path []ngaddrs.ComponentGroupCall, parent *Node, sourceAddr addrs.ModuleSource, loader ConfigLoader) (*Node, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics
	ret := &Node{
		Parent:     parent,
		CallPath:   make([]ngaddrs.ComponentGroupCall, len(path)), // we'll copy the data in below
		SourceAddr: sourceAddr,
	}
	copy(ret.CallPath, path)
	if parent != nil {
		ret.Root = parent.Root
	} else {
		ret.Root = ret
	}

	config, moreDiags := loader.LoadConfig(path, sourceAddr)
	diags = diags.Append(moreDiags)
	if config == nil {
		return ret, diags
	}
	ret.Config = config

	childNodes := make(map[ngaddrs.ComponentGroupCall]*Node, len(config.Groups))
	ret.Children = childNodes
	for _, call := range config.Groups {
		if call.SourceAddr == nil {
			// Suggests that the given address was invalid, in which case there
			// should already be an error about it in "diags" from our loader
			// call above.
			continue
		}

		callAddr := call.CallAddr()

		// The loader needs an absolute source address because otherwise it
		// wouldn't know what to resolve the address relative to.
		childSourceAddr := call.SourceAddr
		childSourceAddr, err := addrs.ResolveRelativeModuleSource(moduleAddrContainingComponentConfigAddr(sourceAddr), childSourceAddr)
		if err != nil {
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid component group configuration address",
				Detail:   fmt.Sprintf("Component group %q has invalid configuration source: %s.", call.Name, err),
				Subject:  call.SourceAddrRange.ToHCL().Ptr(),
			})
			continue
		}

		path := append(path, callAddr)
		childNode, moreDiags := loadComponentsTree(path, ret, childSourceAddr, loader)
		diags = diags.Append(moreDiags)
		// NOTE: a nil entry in the map indicates a call so invalid that
		// nothing could be loaded at all.
		childNodes[callAddr] = childNode
	}

	return ret, diags
}

type ConfigLoader interface {
	LoadConfig(path []ngaddrs.ComponentGroupCall, sourceAddr addrs.ModuleSource) (*tfcomponents.Config, tfdiags.Diagnostics)
}

// HACK: Because we're currently mildly abusing addrs.ModuleSource to represent
// the locations of .tfcomponents files, but addrs.ModuleSource is really
// designed to represent _module directories_, this little adapter allows us
// to find the address of the directory that the given tfcomponents address
// belongs to, which should therefore be a reasonable base address to use
// with addrs.ResolveRelativeModuleSource.
//
// This sort of address-munging shenanigans should usually be inside the addrs
// package to avoid spreading that logic out, but since we're just prototyping
// here we'll do it locally. If we do something like this in a real
// implementation then we should add a new address type to represent
// .tfcomponents file source locations.
func moduleAddrContainingComponentConfigAddr(addr addrs.ModuleSource) addrs.ModuleSource {
	switch addr := addr.(type) {
	case addrs.ModuleSourceLocal:
		return addrs.ModuleSourceLocal(path.Dir(string(addr)))
	case addrs.ModuleSourceRegistry:
		return addrs.ModuleSourceRegistry{
			PackageAddr: addr.PackageAddr,
			Subdir:      path.Dir(addr.Subdir),
		}
	case addrs.ModuleSourceRemote:
		return addrs.ModuleSourceRemote{
			PackageAddr: addr.PackageAddr,
			Subdir:      path.Dir(addr.Subdir),
		}
	default:
		panic(fmt.Sprintf("unsupported address type %T", addr))
	}
}

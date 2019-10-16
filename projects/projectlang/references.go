package projectlang

import (
	"sort"

	"github.com/hashicorp/hcl/v2"

	"github.com/hashicorp/terraform/addrs"
)

// RequiredWorkspaceConfigsForExprs finds the workspace configurations that the
// given expressions refer to, either directly via a workspace.foo or
// upstream.foo expression or indirectly via a local value.
//
// This does not include indirect dependencies via other workspaces. In other
// words, it finds the direct "parent" workspaces for a workspace, but not
// indirect "ancestor" workspaces".
//
// The given StaticEvaluateData will be used to obtain expressions for local
// values in order to flatten any indirect references via them.
func RequiredWorkspaceConfigsForExprs(exprs []hcl.Expression, data StaticEvaluateData) []addrs.ProjectWorkspaceConfig {
	s := make(workspaceConfigAddrSet)
	needLocals := make(map[addrs.LocalValue]struct{})

	for _, expr := range exprs {
		for _, ref := range findReferencesInExpr(expr) {
			switch addr := ref.Subject.(type) {
			case addrs.ProjectWorkspaceConfig:
				s.Add(addr)
			case addrs.LocalValue:
				needLocals[addr] = struct{}{}
			}
		}
	}

	for {
		// This is a kinda "brute force" approach where we'll
		// unnecessarily re-evaluate the same local values over and over.
		// It should be possible to recognize ones we already dealt with
		// and skip them but since we don't expect there to be a large
		// number of locals in a project configuration this should be good
		// enough for now.
		foundMoreLocals := false
		for addr := range needLocals {
			expr := data.LocalValueExpr(addr)
			for _, ref := range findReferencesInExpr(expr) {
				switch addr := ref.Subject.(type) {
				case addrs.ProjectWorkspaceConfig:
					s.Add(addr)
				case addrs.LocalValue:
					if _, exists := needLocals[addr]; !exists {
						needLocals[addr] = struct{}{}
						foundMoreLocals = true
					}
				}
			}
		}

		if !foundMoreLocals {
			break
		}
	}

	return s.List()
}

func findReferencesInExpr(expr hcl.Expression) []*addrs.ProjectConfigReference {
	traversals := expr.Variables()
	if len(traversals) == 0 {
		return nil
	}
	ret := make([]*addrs.ProjectConfigReference, 0, len(traversals))
	for _, traversal := range expr.Variables() {
		ref, diags := addrs.ParseProjectConfigRef(traversal)
		if diags.HasErrors() {
			continue
		}
		ret = append(ret, ref)
	}
	return ret
}

type workspaceConfigAddrSet map[addrs.ProjectWorkspaceConfig]struct{}

func (s workspaceConfigAddrSet) Has(addr addrs.ProjectWorkspaceConfig) bool {
	_, ok := s[addr]
	return ok
}

func (s workspaceConfigAddrSet) Add(addr addrs.ProjectWorkspaceConfig) {
	s[addr] = struct{}{}
}

func (s workspaceConfigAddrSet) Remove(addr addrs.ProjectWorkspaceConfig) {
	delete(s, addr)
}

func (s workspaceConfigAddrSet) List() []addrs.ProjectWorkspaceConfig {
	if len(s) == 0 {
		return nil
	}
	ret := make([]addrs.ProjectWorkspaceConfig, 0, len(s))
	for addr := range s {
		ret = append(ret, addr)
	}

	// Ensure that the result is always consistently ordered, so that any
	// derived behavior is also consistent.
	sort.Slice(ret, func(i, j int) bool {
		a, b := ret[i], ret[j]
		switch {
		case a.Rel != b.Rel:
			return a.Rel < b.Rel
		case a.Name != b.Name:
			return a.Name < b.Name
		default:
			return false
		}
	})

	return ret
}

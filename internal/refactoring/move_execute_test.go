package refactoring

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/states"
)

func TestApplyMoves(t *testing.T) {
	// TODO: Renable this once we're ready to implement the intended behaviors
	// it is describing.
	t.Skip("ApplyMoves is not yet fully implemented")

	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("example.com/foo/bar"),
	}
	rootNoKeyResourceAddr := [...]addrs.AbsResourceInstance{
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
	}
	rootIntKeyResourceAddr := [...]addrs.AbsResourceInstance{
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
		addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),
	}

	tests := map[string]struct {
		Stmts []MoveStatement
		State *states.State

		WantResults       map[addrs.UniqueKey]MoveResult
		WantInstanceAddrs []string
	}{
		"no moves and empty state": {
			[]MoveStatement{},
			states.NewState(),
			nil,
			nil,
		},
		"no moves": {
			[]MoveStatement{},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					rootNoKeyResourceAddr[0],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			nil,
			[]string{
				`foo.from`,
			},
		},
		"single move of whole singleton resource": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					rootNoKeyResourceAddr[0],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			map[addrs.UniqueKey]MoveResult{
				rootNoKeyResourceAddr[0].UniqueKey(): {
					From: rootNoKeyResourceAddr[0],
					To:   rootNoKeyResourceAddr[1],
				},
				rootNoKeyResourceAddr[1].UniqueKey(): {
					From: rootNoKeyResourceAddr[1],
					To:   rootNoKeyResourceAddr[1],
				},
			},
			[]string{
				`foo.to`,
			},
		},
		"single move of whole 'count' resource": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					rootIntKeyResourceAddr[0],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			map[addrs.UniqueKey]MoveResult{
				rootNoKeyResourceAddr[0].UniqueKey(): {
					From: rootIntKeyResourceAddr[0],
					To:   rootIntKeyResourceAddr[1],
				},
				rootNoKeyResourceAddr[1].UniqueKey(): {
					From: rootIntKeyResourceAddr[0],
					To:   rootIntKeyResourceAddr[1],
				},
			},
			[]string{
				`foo.to[0]`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var stmtsBuf strings.Builder
			for _, stmt := range test.Stmts {
				fmt.Fprintf(&stmtsBuf, "- from: %s\n  to:   %s", stmt.From, stmt.To)
			}
			t.Logf("move statements:\n%s", stmtsBuf.String())

			t.Logf("resource instances in prior state:\n%s", spew.Sdump(allResourceInstanceAddrsInState(test.State)))

			state := test.State.DeepCopy() // don't modify the test case in-place
			gotResults := ApplyMoves(test.Stmts, state)

			if diff := cmp.Diff(test.WantResults, gotResults); diff != "" {
				t.Errorf("wrong results\n%s", diff)
			}

			gotInstAddrs := allResourceInstanceAddrsInState(state)
			if diff := cmp.Diff(test.WantInstanceAddrs, gotInstAddrs); diff != "" {
				t.Errorf("wrong resource instances in final state\n%s", diff)
			}
		})
	}
}

func testMoveStatement(t *testing.T, module string, from string, to string) MoveStatement {
	t.Helper()

	moduleAddr := addrs.RootModule
	if len(module) != 0 {
		moduleAddr = addrs.Module(strings.Split(module, "."))
	}

	fromTraversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(from), "from", hcl.InitialPos)
	if hclDiags.HasErrors() {
		t.Fatalf("invalid 'from' argument: %s", hclDiags.Error())
	}
	fromAddr, diags := addrs.ParseMoveEndpoint(fromTraversal)
	if diags.HasErrors() {
		t.Fatalf("invalid 'from' argument: %s", diags.Err().Error())
	}
	toTraversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(to), "to", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("invalid 'to' argument: %s", hclDiags.Error())
	}
	toAddr, diags := addrs.ParseMoveEndpoint(toTraversal)
	if diags.HasErrors() {
		t.Fatalf("invalid 'from' argument: %s", diags.Err().Error())
	}

	fromInModule, toInModule := addrs.UnifyMoveEndpoints(moduleAddr, fromAddr, toAddr)
	if fromInModule == nil || toInModule == nil {
		t.Fatalf("incompatible endpoints")
	}

	return MoveStatement{
		From: fromInModule,
		To:   toInModule,

		// DeclRange not populated because it's unimportant for our tests
	}
}

func allResourceInstanceAddrsInState(state *states.State) []string {
	var ret []string
	for _, ms := range state.Modules {
		for _, rs := range ms.Resources {
			for key := range rs.Instances {
				ret = append(ret, rs.Addr.Instance(key).String())
			}
		}
	}
	sort.Strings(ret)
	return ret
}

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
	providerAddr := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("example.com/foo/bar"),
	}

	moduleBoo, _ := addrs.ParseModuleInstanceStr("module.boo")
	moduleBarKey, _ := addrs.ParseModuleInstanceStr("module.bar[0]")

	instAddrs := map[string]addrs.AbsResourceInstance{
		"foo.from": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),

		"foo.mid": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "mid",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),

		"foo.to": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),

		"foo.from[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),

		"foo.to[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.IntKey(0)).Absolute(addrs.RootModuleInstance),

		"module.boo.foo.from": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.NoKey).Absolute(moduleBoo),

		"module.boo.foo.mid": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "mid",
		}.Instance(addrs.NoKey).Absolute(moduleBoo),

		"module.boo.foo.to": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.NoKey).Absolute(moduleBoo),

		"module.boo.foo.from[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.IntKey(0)).Absolute(moduleBoo),

		"module.boo.foo.to[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.IntKey(0)).Absolute(moduleBoo),

		"module.bar[0].foo.from": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.NoKey).Absolute(moduleBarKey),

		"module.bar[0].foo.mid": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "mid",
		}.Instance(addrs.NoKey).Absolute(moduleBarKey),

		"module.bar[0].foo.to": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.NoKey).Absolute(moduleBarKey),

		"module.bar[0].foo.from[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "from",
		}.Instance(addrs.IntKey(0)).Absolute(moduleBarKey),

		"module.bar[0].foo.to[0]": addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "foo",
			Name: "to",
		}.Instance(addrs.IntKey(0)).Absolute(moduleBarKey),
	}

	emptyResults := MoveResults{
		Changes: map[addrs.UniqueKey]MoveSuccess{},
		Blocked: map[addrs.UniqueKey]MoveBlocked{},
	}

	tests := map[string]struct {
		Stmts []MoveStatement
		State *states.State

		WantResults       MoveResults
		WantInstanceAddrs []string
	}{
		"no moves and empty state": {
			[]MoveStatement{},
			states.NewState(),
			emptyResults,
			nil,
		},
		"no moves": {
			[]MoveStatement{},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			emptyResults,
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
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["foo.to"].UniqueKey(): {
						From: instAddrs["foo.from"],
						To:   instAddrs["foo.to"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
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
					instAddrs["foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["foo.to[0]"].UniqueKey(): {
						From: instAddrs["foo.from[0]"],
						To:   instAddrs["foo.to[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`foo.to[0]`,
			},
		},
		"chained move of whole singleton resource": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.mid"),
				testMoveStatement(t, "", "foo.mid", "foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["foo.to"].UniqueKey(): {
						From: instAddrs["foo.from"],
						To:   instAddrs["foo.to"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`foo.to`,
			},
		},

		"move whole resource into module": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "module.boo.foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.boo.foo.to[0]"].UniqueKey(): {
						From: instAddrs["foo.from[0]"],
						To:   instAddrs["module.boo.foo.to[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.boo.foo.to[0]`,
			},
		},

		"move resource instance between modules": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.boo.foo.from[0]", "module.bar[0].foo.to[0]"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.bar[0].foo.to[0]"].UniqueKey(): {
						From: instAddrs["module.boo.foo.from[0]"],
						To:   instAddrs["module.bar[0].foo.to[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"move whole single module to indexed module": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.bar[0].foo.from[0]"].UniqueKey(): {
						From: instAddrs["module.boo.foo.from[0]"],
						To:   instAddrs["module.bar[0].foo.from[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.bar[0].foo.from[0]`,
			},
		},

		"move whole module to indexed module and move instance chained": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]"),
				testMoveStatement(t, "bar", "foo.from[0]", "foo.to[0]"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.bar[0].foo.to[0]"].UniqueKey(): {
						From: instAddrs["module.boo.foo.from[0]"],
						To:   instAddrs["module.bar[0].foo.to[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"move instance to indexed module and instance chained": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.boo.foo.from[0]", "module.bar[0].foo.from[0]"),
				testMoveStatement(t, "bar", "foo.from[0]", "foo.to[0]"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.from[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.bar[0].foo.to[0]"].UniqueKey(): {
						From: instAddrs["module.boo.foo.from[0]"],
						To:   instAddrs["module.bar[0].foo.to[0]"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"move module instance to already-existing module instance": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.bar[0].foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.to[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				// Nothing moved, because the module.b address is already
				// occupied by another module.
				Changes: map[addrs.UniqueKey]MoveSuccess{},
				Blocked: map[addrs.UniqueKey]MoveBlocked{
					instAddrs["module.bar[0].foo.from"].Module.UniqueKey(): {
						Wanted: instAddrs["module.boo.foo.to[0]"].Module,
						Actual: instAddrs["module.bar[0].foo.from"].Module,
					},
				},
			},
			[]string{
				`module.bar[0].foo.from`,
				`module.boo.foo.to[0]`,
			},
		},

		"move resource to already-existing resource": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
				s.SetResourceInstanceCurrent(
					instAddrs["foo.to"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				// Nothing moved, because the from.to address is already
				// occupied by another resource.
				Changes: map[addrs.UniqueKey]MoveSuccess{},
				Blocked: map[addrs.UniqueKey]MoveBlocked{
					instAddrs["foo.from"].ContainingResource().UniqueKey(): {
						Wanted: instAddrs["foo.to"].ContainingResource(),
						Actual: instAddrs["foo.from"].ContainingResource(),
					},
				},
			},
			[]string{
				`foo.from`,
				`foo.to`,
			},
		},

		"move resource instance to already-existing resource instance": {
			[]MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to[0]"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
				s.SetResourceInstanceCurrent(
					instAddrs["foo.to[0]"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				// Nothing moved, because the from.to[0] address is already
				// occupied by another resource instance.
				Changes: map[addrs.UniqueKey]MoveSuccess{},
				Blocked: map[addrs.UniqueKey]MoveBlocked{
					instAddrs["foo.from"].UniqueKey(): {
						Wanted: instAddrs["foo.to[0]"],
						Actual: instAddrs["foo.from"],
					},
				},
			},
			[]string{
				`foo.from`,
				`foo.to[0]`,
			},
		},
		"move resource and containing module": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]"),
				testMoveStatement(t, "boo", "foo.from", "foo.to"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.boo.foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.bar[0].foo.to"].UniqueKey(): {
						From: instAddrs["module.boo.foo.from"],
						To:   instAddrs["module.bar[0].foo.to"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.bar[0].foo.to`,
			},
		},

		"move module and then move resource into it": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo"),
				testMoveStatement(t, "", "foo.from", "module.boo.foo.from"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.bar[0].foo.to"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{
					instAddrs["module.boo.foo.from"].UniqueKey(): {
						instAddrs["foo.from"],
						instAddrs["module.boo.foo.from"],
					},
					instAddrs["module.boo.foo.to"].UniqueKey(): {
						instAddrs["module.bar[0].foo.to"],
						instAddrs["module.boo.foo.to"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{},
			},
			[]string{
				`module.boo.foo.from`,
				`module.boo.foo.to`,
			},
		},

		"module move collides with resource move": {
			[]MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo"),
				testMoveStatement(t, "", "foo.from", "module.boo.foo.from"),
			},
			states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					instAddrs["module.bar[0].foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
				s.SetResourceInstanceCurrent(
					instAddrs["foo.from"],
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					providerAddr,
				)
			}),
			MoveResults{
				Changes: map[addrs.UniqueKey]MoveSuccess{

					instAddrs["module.boo.foo.from"].UniqueKey(): {
						instAddrs["module.bar[0].foo.from"],
						instAddrs["module.boo.foo.from"],
					},
				},
				Blocked: map[addrs.UniqueKey]MoveBlocked{
					instAddrs["foo.from"].ContainingResource().UniqueKey(): {
						Actual: instAddrs["foo.from"].ContainingResource(),
						Wanted: instAddrs["module.boo.foo.from"].ContainingResource(),
					},
				},
			},
			[]string{
				`foo.from`,
				`module.boo.foo.from`,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var stmtsBuf strings.Builder
			for _, stmt := range test.Stmts {
				fmt.Fprintf(&stmtsBuf, "• from: %s\n  to:   %s\n", stmt.From, stmt.To)
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

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

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
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states"
)

func TestApplyMoves(t *testing.T) {
	barProviderAddress := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("example.com/foo/bar"),
	}
	fooProviderAddress := addrs.AbsProviderConfig{
		Module:   addrs.RootModule,
		Provider: addrs.MustParseProviderSourceString("example.com/bar/foo"),
	}

	mustParseInstAddr := func(s string) addrs.AbsResourceInstance {
		addr, err := addrs.ParseAbsResourceInstanceStr(s)
		if err != nil {
			t.Fatal(err)
		}
		return addr
	}

	emptyResults := makeMoveResults()

	tests := map[string]struct {
		Stmts []MoveStatement
		State *states.State

		// We only need providers if we are doing a cross-resource type move
		// so most of the test cases don't need this.
		Providers map[addrs.Provider]providers.Factory

		WantResults       MoveResults
		WantDiags         []string
		WantInstanceAddrs []string
	}{
		"no moves and empty state": {
			Stmts:             []MoveStatement{},
			State:             states.NewState(),
			Providers:         nil,
			WantResults:       emptyResults,
			WantInstanceAddrs: nil,
		},
		"no moves": {
			Stmts: []MoveStatement{},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: emptyResults,
			WantInstanceAddrs: []string{
				`foo.from`,
			},
		},
		"single move of whole singleton resource": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("foo.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("foo.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`foo.to`,
			},
		},
		"single move of whole 'count' resource": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("foo.to[0]"), MoveSuccess{
						From: mustParseInstAddr("foo.from[0]"),
						To:   mustParseInstAddr("foo.to[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`foo.to[0]`,
			},
		},
		"chained move of whole singleton resource": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.mid", &barProviderAddress),
				testMoveStatement(t, "", "foo.mid", "foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("foo.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("foo.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`foo.to`,
			},
		},

		"move whole resource into module": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "module.boo.foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.boo.foo.to[0]"), MoveSuccess{
						From: mustParseInstAddr("foo.from[0]"),
						To:   mustParseInstAddr("module.boo.foo.to[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.boo.foo.to[0]`,
			},
		},

		"move resource instance between modules": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo.foo.from[0]", "module.bar[0].foo.to[0]", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.to[0]"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from[0]"),
						To:   mustParseInstAddr("module.bar[0].foo.to[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"module move with child module": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar", nil),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.module.hoo.foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar.foo.from"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from"),
						To:   mustParseInstAddr("module.bar.foo.from"),
					}),
					addrs.MakeMapElem(mustParseInstAddr("module.bar.module.hoo.foo.from"), MoveSuccess{
						From: mustParseInstAddr("module.boo.module.hoo.foo.from"),
						To:   mustParseInstAddr("module.bar.module.hoo.foo.from"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar.foo.from`,
				`module.bar.module.hoo.foo.from`,
			},
		},

		"move whole single module to indexed module": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]", nil),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.from[0]"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from[0]"),
						To:   mustParseInstAddr("module.bar[0].foo.from[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.from[0]`,
			},
		},

		"move whole module to indexed module and move instance chained": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]", nil),
				testMoveStatement(t, "bar", "foo.from[0]", "foo.to[0]", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.to[0]"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from[0]"),
						To:   mustParseInstAddr("module.bar[0].foo.to[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"move instance to indexed module and instance chained": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo.foo.from[0]", "module.bar[0].foo.from[0]", &barProviderAddress),
				testMoveStatement(t, "bar", "foo.from[0]", "foo.to[0]", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.to[0]"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from[0]"),
						To:   mustParseInstAddr("module.bar[0].foo.to[0]"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.to[0]`,
			},
		},

		"move module instance to already-existing module instance": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo", nil),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.bar[0].foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.to[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				// Nothing moved, because the module.b address is already
				// occupied by another module.
				Changes: emptyResults.Changes,
				Blocked: addrs.MakeMap(
					addrs.MakeMapElem[addrs.AbsMoveable](
						mustParseInstAddr("module.bar[0].foo.from").Module,
						MoveBlocked{
							Wanted: mustParseInstAddr("module.boo.foo.to[0]").Module,
							Actual: mustParseInstAddr("module.bar[0].foo.from").Module,
						},
					),
				),
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.from`,
				`module.boo.foo.to[0]`,
			},
		},

		"move resource to already-existing resource": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.to"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				// Nothing moved, because the from.to address is already
				// occupied by another resource.
				Changes: emptyResults.Changes,
				Blocked: addrs.MakeMap(
					addrs.MakeMapElem[addrs.AbsMoveable](
						mustParseInstAddr("foo.from").ContainingResource(),
						MoveBlocked{
							Wanted: mustParseInstAddr("foo.to").ContainingResource(),
							Actual: mustParseInstAddr("foo.from").ContainingResource(),
						},
					),
				),
			},
			WantInstanceAddrs: []string{
				`foo.from`,
				`foo.to`,
			},
		},

		"move resource instance to already-existing resource instance": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "foo.to[0]", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.to[0]"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				// Nothing moved, because the from.to[0] address is already
				// occupied by another resource instance.
				Changes: emptyResults.Changes,
				Blocked: addrs.MakeMap(
					addrs.MakeMapElem[addrs.AbsMoveable](
						mustParseInstAddr("foo.from"),
						MoveBlocked{
							Wanted: mustParseInstAddr("foo.to[0]"),
							Actual: mustParseInstAddr("foo.from"),
						},
					),
				),
			},
			WantInstanceAddrs: []string{
				`foo.from`,
				`foo.to[0]`,
			},
		},
		"move resource and containing module": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.boo", "module.bar[0]", nil),
				testMoveStatement(t, "boo", "foo.from", "foo.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.boo.foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.to"), MoveSuccess{
						From: mustParseInstAddr("module.boo.foo.from"),
						To:   mustParseInstAddr("module.bar[0].foo.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].foo.to`,
			},
		},

		"move module and then move resource into it": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo", nil),
				testMoveStatement(t, "", "foo.from", "module.boo.foo.from", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.bar[0].foo.to"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.boo.foo.from"), MoveSuccess{
						mustParseInstAddr("foo.from"),
						mustParseInstAddr("module.boo.foo.from"),
					}),
					addrs.MakeMapElem(mustParseInstAddr("module.boo.foo.to"), MoveSuccess{
						mustParseInstAddr("module.bar[0].foo.to"),
						mustParseInstAddr("module.boo.foo.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.boo.foo.from`,
				`module.boo.foo.to`,
			},
		},

		"move resources into module and then move module": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "module.boo.foo.to", &barProviderAddress),
				testMoveStatement(t, "", "bar.from", "module.boo.bar.to", &barProviderAddress),
				testMoveStatement(t, "", "module.boo", "module.bar[0]", nil),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("bar.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].foo.to"), MoveSuccess{
						mustParseInstAddr("foo.from"),
						mustParseInstAddr("module.bar[0].foo.to"),
					}),
					addrs.MakeMapElem(mustParseInstAddr("module.bar[0].bar.to"), MoveSuccess{
						mustParseInstAddr("bar.from"),
						mustParseInstAddr("module.bar[0].bar.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`module.bar[0].bar.to`,
				`module.bar[0].foo.to`,
			},
		},

		"module move collides with resource move": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "module.bar[0]", "module.boo", nil),
				testMoveStatement(t, "", "foo.from", "module.boo.foo.from", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("module.bar[0].foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("module.boo.foo.from"), MoveSuccess{
						mustParseInstAddr("module.bar[0].foo.from"),
						mustParseInstAddr("module.boo.foo.from"),
					}),
				),
				Blocked: addrs.MakeMap(
					addrs.MakeMapElem[addrs.AbsMoveable](
						mustParseInstAddr("foo.from").ContainingResource(),
						MoveBlocked{
							Actual: mustParseInstAddr("foo.from").ContainingResource(),
							Wanted: mustParseInstAddr("module.boo.foo.from").ContainingResource(),
						},
					),
				),
			},
			WantInstanceAddrs: []string{
				`foo.from`,
				`module.boo.foo.from`,
			},
		},

		"cross resource type move unsupported": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "bar.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			Providers: map[addrs.Provider]providers.Factory{
				barProviderAddress.Provider: func() (providers.Interface, error) {
					return &mockProvider{
						moveResourceState: false,
						moveResourceError: nil,
					}, nil
				},
			},
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("bar.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("bar.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantDiags: []string{
				"(Error) Unsupported `moved` across resource types:The provider \"example.com/foo/bar\" does not support moved operations across resource types and providers.",
			},
			WantInstanceAddrs: []string{
				`bar.to`,
			},
		},
		"cross resource type move errors": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "bar.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			Providers: map[addrs.Provider]providers.Factory{
				barProviderAddress.Provider: func() (providers.Interface, error) {
					return &mockProvider{
						moveResourceState: true,
						moveResourceError: fmt.Errorf("provider can't move between those resource types"),
					}, nil
				},
			},
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("bar.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("bar.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantDiags: []string{
				"(Error) expected error:provider can't move between those resource types",
			},
			WantInstanceAddrs: []string{
				`bar.to`,
			},
		},
		"cross resource type move": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "bar.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					barProviderAddress,
				)
			}),
			Providers: map[addrs.Provider]providers.Factory{
				barProviderAddress.Provider: func() (providers.Interface, error) {
					return &mockProvider{
						moveResourceState: true,
						moveResourceError: nil,
					}, nil
				},
			},
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("bar.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("bar.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`bar.to`,
			},
		},
		"cross provider move": {
			Stmts: []MoveStatement{
				testMoveStatement(t, "", "foo.from", "bar.to", &barProviderAddress),
			},
			State: states.BuildState(func(s *states.SyncState) {
				s.SetResourceInstanceCurrent(
					mustParseInstAddr("foo.from"),
					&states.ResourceInstanceObjectSrc{
						Status:    states.ObjectReady,
						AttrsJSON: []byte(`{}`),
					},
					fooProviderAddress,
				)
			}),
			Providers: map[addrs.Provider]providers.Factory{
				barProviderAddress.Provider: func() (providers.Interface, error) {
					return &mockProvider{
						moveResourceState: true,
						moveResourceError: nil,
					}, nil
				},
				fooProviderAddress.Provider: func() (providers.Interface, error) {
					return &mockProvider{}, nil
				},
			},
			WantResults: MoveResults{
				Changes: addrs.MakeMap(
					addrs.MakeMapElem(mustParseInstAddr("bar.to"), MoveSuccess{
						From: mustParseInstAddr("foo.from"),
						To:   mustParseInstAddr("bar.to"),
					}),
				),
				Blocked: emptyResults.Blocked,
			},
			WantInstanceAddrs: []string{
				`bar.to`,
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
			gotResults, diags := ApplyMoves(test.Stmts, state, test.Providers)

			var actualDiags []string
			for _, diag := range diags {
				actualDiags = append(actualDiags, fmt.Sprintf("(%s) %s:%s", diag.Severity(), diag.Description().Summary, diag.Description().Detail))
			}
			if diff := cmp.Diff(test.WantDiags, actualDiags); diff != "" {
				t.Errorf("wrong diagnostics\n%s", diff)
			}

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

func testMoveStatement(t *testing.T, module string, from string, to string, provider *addrs.AbsProviderConfig) MoveStatement {
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

	stmt := MoveStatement{
		From: fromInModule,
		To:   toInModule,

		// DeclRange not populated because it's unimportant for our tests
	}

	if provider != nil {
		// Only set the provider for resource type moves.
		stmt.Provider = provider
	}

	return stmt
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

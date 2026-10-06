// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"slices"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestPartialExpandedResourceIsTargetedBy(t *testing.T) {

	tcs := []struct {
		per    string
		target string
		want   bool
	}{
		{
			"test.a",
			"test.a",
			true,
		},
		{
			"test.a",
			"test.a[0]",
			true,
		},
		{
			"test.a[*]",
			"test.a",
			true,
		},
		{
			"test.a[*]",
			"test.a[0]",
			true,
		},
		{
			"test.a[*]",
			"test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[1].test.a",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod.test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a[*]",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod[1].test.a[*]",
			"module.mod[\"key\"].test.a[0]",
			false,
		},
		{
			"module.mod[*].test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod[0].test.a",
			true,
		},
		{
			"module.mod[*].test.a",
			"module.mod[\"key\"].test.a",
			true,
		},
		{
			// a resource in the root module doesn't target a resource with
			// the same name in a child module
			"module.mod[*].test.a",
			"test.a",
			false,
		},
		{
			"module.mod[*].test.a",
			"test.a[0]",
			false,
		},
		{
			"module.mod[*].module.child[*].test.a",
			"module.mod.test.a",
			false,
		},
		{
			"module.mod[*].module.child[*].test.a",
			"module.mod",
			true,
		},
		{
			"module.mod[0].test.a[*]",
			"module.mod[*].test.a",
			true,
		},
		{
			"module.mod[0].test.a[*]",
			"module.mod[*]",
			true,
		},
		{
			"module.mod[0].module.child[*].test.a",
			"module.mod[*].module.child[*].test.a",
			true,
		},
		{
			"module.mod[0].module.child[*].test.a",
			"module.mod[1].module.child[*].test.a",
			false,
		},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("PartialResource(%q).IsTargetedBy(%q)", tc.per, tc.target), func(t *testing.T) {
			per := mustParsePartialResourceInstanceStr(tc.per).PartialResource()
			target := mustParseTargetPattern(tc.target)

			got := per.IsTargetedBy(target)
			if got != tc.want {
				t.Errorf("PartialResource(%q).IsTargetedBy(%q): got %v; want %v", tc.per, tc.target, got, tc.want)
			}
		})
	}

	configResourceTargets := []struct {
		per    string
		target ConfigResource
		want   bool
	}{
		{
			"module.mod[*].test.a",
			Module{"mod"}.Resource(ManagedResourceMode, "test", "a"),
			true,
		},
		{
			"module.mod[*].test.a",
			RootModule.Resource(ManagedResourceMode, "test", "a"),
			false,
		},
		{
			"module.mod[*].test.a",
			Module{"mod", "child"}.Resource(ManagedResourceMode, "test", "a"),
			false,
		},
	}
	for _, tc := range configResourceTargets {
		t.Run(fmt.Sprintf("PartialResource(%q).IsTargetedBy(ConfigResource(%q))", tc.per, tc.target), func(t *testing.T) {
			per := mustParsePartialResourceInstanceStr(tc.per).PartialResource()
			got := per.IsTargetedBy(tc.target)
			if got != tc.want {
				t.Errorf("PartialResource(%q).IsTargetedBy(ConfigResource(%q)): got %v; want %v", tc.per, tc.target, got, tc.want)
			}
		})
	}
}

func TestParsePartialExpandedModule(t *testing.T) {

	// these functions are a bit weird, as the normal parsing supported by
	// HCL can't put unknown values into the instance keys. So we need to
	// build the traversals in the same way the thing that is calling these
	// functions does.

	tcs := []struct {
		traversal func(t *testing.T) (string, hcl.Traversal)
		want      string
		remain    int
	}{
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.mod"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want:   "module.mod",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.mod[0]"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				// Hack the key into an unknown value.
				traversal[2] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.mod[*]", traversal
			},
			want:   "module.mod[*]",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want:   "module.child.module.grandchild",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child[0].module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				return addr, traversal
			},
			want:   "module.child[0].module.grandchild",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child[0].module.grandchild"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[2] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child[*].module.grandchild", traversal
			},
			want:   "module.child[*].module.grandchild[*]",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild[0]"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[4] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child.module.grandchild[*]", traversal
			},
			want:   "module.child.module.grandchild[*]",
			remain: 0,
		},
		{
			traversal: func(t *testing.T) (string, hcl.Traversal) {
				addr := "module.child.module.grandchild[0].resource_type.resource_name"
				traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
				if len(diags) > 0 {
					t.Fatalf("unexpected diagnostics: %v", diags)
				}
				traversal[4] = hcl.TraverseIndex{
					Key: cty.UnknownVal(cty.Number),
				}
				return "module.child.module.grandchild[*].resource_type.resource_name", traversal
			},
			want:   "module.child.module.grandchild[*]",
			remain: 2,
		},
	}

	for _, tc := range tcs {
		addr, traversal := tc.traversal(t)
		t.Run(addr, func(t *testing.T) {
			module, rest, diags := ParsePartialExpandedModule(traversal)
			if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			if got := module.String(); got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
			if len(rest) != tc.remain {
				t.Errorf("got %d remaining traversals; want %d", len(rest), tc.remain)
			}
		})
	}

}

func TestParsePartialExpandedResource(t *testing.T) {

	tcs := []struct {
		addr   string
		want   string
		remain int
	}{
		{
			addr:   "resource_type.resource_name",
			want:   "resource_type.resource_name[*]",
			remain: 0,
		},
		{
			addr: "module.mod.resource_type.resource_name",
			want: "module.mod.resource_type.resource_name[*]",
		},
		{
			addr:   "resource_type.resource_name[0]",
			want:   "resource_type.resource_name[*]",
			remain: 0,
		},
		{
			addr:   "resource_type.resource_name[0].attr",
			want:   "resource_type.resource_name[*]",
			remain: 1,
		},
		{
			addr:   "resource.resource_type.resource_name",
			want:   "resource_type.resource_name[*]",
			remain: 0,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.addr, func(t *testing.T) {
			traversal, traversalDiags := hclsyntax.ParseTraversalAbs([]byte(tc.addr), "", hcl.InitialPos)
			if len(traversalDiags) > 0 {
				t.Fatalf("unexpected diagnostics: %v", traversalDiags)
			}

			partial, rest, diags := ParsePartialExpandedResource(traversal)
			if len(diags) > 0 {
				t.Fatalf("unexpected diagnostics: %s", diags)
			}

			if got := partial.String(); got != tc.want {
				t.Errorf("got %s; want %s", got, tc.want)
			}
			if len(rest) != tc.remain {
				t.Errorf("got %d remaining traversals; want %d", len(rest), tc.remain)
			}
		})
	}
}

func mustParsePartialResourceInstanceStr(s string) AbsResourceInstance {
	r, diags := ParsePartialResourceInstanceStr(s)
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings().Error())
	}
	return r
}

func TestPartialExpandedModule(t *testing.T) {
	pem := mustParsePartialExpandedModule("module.a[0].module.b[*].module.c[*]")

	if got, want := pem.String(), "module.a[0].module.b[*].module.c[*]"; got != want {
		t.Errorf("wrong String\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := pem.LevelsKnown(), 1; got != want {
		t.Errorf("wrong LevelsKnown: got %d, want %d", got, want)
	}
	if got, want := pem.KnownPrefix(), mustParseModuleInstanceStr("module.a[0]"); !got.Equal(want) {
		t.Errorf("wrong KnownPrefix\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := pem.UnexpandedSuffix(), []ModuleCall{{Name: "b"}, {Name: "c"}}; !slices.Equal(got, want) {
		t.Errorf("wrong UnexpandedSuffix\ngot:  %#v\nwant: %#v", got, want)
	}
	if got, want := pem.FirstUnexpandedCall().String(), "module.a[0].module.b"; got != want {
		t.Errorf("wrong FirstUnexpandedCall\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := pem.Module().String(), "module.a.module.b.module.c"; got != want {
		t.Errorf("wrong Module\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := pem.UnknownModuleInstance().String(), "module.a[0].module.b[*].module.c[*]"; got != want {
		t.Errorf("wrong UnknownModuleInstance\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := pem.Child(ModuleCall{Name: "d"}).String(), "module.a[0].module.b[*].module.c[*].module.d[*]"; got != want {
		t.Errorf("wrong Child\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := mustParseModuleInstanceStr("module.a[0]").UnexpandedChild(ModuleCall{Name: "b"}).String(), "module.a[0].module.b[*]"; got != want {
		t.Errorf("wrong UnexpandedChild\ngot:  %s\nwant: %s", got, want)
	}

	root := RootModuleInstance.UnexpandedChild(ModuleCall{Name: "a"})
	if got := root.KnownPrefix(); got != nil {
		t.Errorf("wrong KnownPrefix for unexpanded root call: got %#v, want nil", got)
	}
	if got, want := root.FirstUnexpandedCall().String(), "module.a"; got != want {
		t.Errorf("wrong FirstUnexpandedCall\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPartialExpandedModuleMatches(t *testing.T) {
	for _, tc := range []struct {
		pem   string
		other string
		want  bool
	}{
		{"module.a[*]", "module.a[0]", true},
		{"module.a[*]", "module.a", true},
		{"module.a[*]", "module.b[0]", false},
		{
			// only the module instances themselves match, not their
			// descendants
			"module.a[*]",
			"module.a[0].module.b[0]",
			false,
		},
		{"module.a[0].module.b[*]", `module.a[0].module.b["x"]`, true},
		{"module.a[0].module.b[*]", `module.a[1].module.b["x"]`, false},
		{"module.a.module.b[*]", "module.a.module.b[0]", true},
		{"module.a.module.b[*]", "module.a[0].module.b[0]", false},

		// matching other partial-expanded modules
		{"module.a[*].module.b[*]", "module.a[0].module.b[*]", true},
		{"module.a[0].module.b[*]", "module.a[0].module.b[*]", true},
		{"module.a[0].module.b[*]", "module.a[*].module.b[*]", false},
		{"module.a[0].module.b[*]", "module.a[1].module.b[*]", false},
	} {
		t.Run(fmt.Sprintf("%s-matches-%s", tc.pem, tc.other), func(t *testing.T) {
			pem := mustParsePartialExpandedModule(tc.pem)
			other := mustParsePartialModuleInstance(tc.other)

			if got := pem.MatchesPartial(other.PartialModule()); got != tc.want {
				t.Errorf("MatchesPartial: got %t, want %t", got, tc.want)
			}
			isPartial := slices.ContainsFunc(other, func(step ModuleInstanceStep) bool {
				return step.InstanceKey == WildcardKey
			})
			if !isPartial {
				if got := pem.MatchesInstance(other); got != tc.want {
					t.Errorf("MatchesInstance: got %t, want %t", got, tc.want)
				}
			}
		})
	}
}

func TestModuleInstancePartialModule(t *testing.T) {
	for _, tc := range []struct {
		addr string
		want string
	}{
		{"module.a", "module.a"},
		{"module.a[*]", "module.a[*]"},
		{"module.a[0].module.b[*]", "module.a[0].module.b[*]"},
		{"module.a[*].module.b[*]", "module.a[*].module.b[*]"},
		{
			// once a step is not expanded, none of the steps after it can be
			// expanded either
			"module.a[*].module.b[0]",
			"module.a[*].module.b[*]",
		},
	} {
		t.Run(tc.addr, func(t *testing.T) {
			got := mustParsePartialModuleInstance(tc.addr).PartialModule()
			if got.String() != tc.want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, tc.want)
			}
			if !got.UnknownModuleInstance().PartialModule().MatchesPartial(got) {
				t.Errorf("%s does not round-trip through UnknownModuleInstance", got)
			}
		})
	}
}

func TestPartialExpandedResourceMatches(t *testing.T) {
	for _, tc := range []struct {
		per   string
		other string
		want  bool
	}{
		{"test.a[*]", "test.a[0]", true},
		{"test.a[*]", "test.a", true},
		{"test.a[*]", "test.b[0]", false},
		{"test.a[*]", "data.test.a[0]", false},
		{"module.m[*].test.a[*]", "module.m[0].test.a[1]", true},
		{"module.m[*].test.a[*]", "test.a[1]", false},
		{"module.m[*].test.a[*]", "module.m[0].module.n.test.a", false},
		{"module.m[0].test.a[*]", "module.m[1].test.a[0]", false},
		{"module.m[0].module.n[*].test.a[*]", `module.m[0].module.n["x"].test.a`, true},

		// matching other partial-expanded resources
		{"module.m[*].test.a[*]", "module.m[0].test.a[*]", true},
		{"module.m[*].test.a[*]", "module.m[*].test.a[*]", true},
		{"module.m[0].test.a[*]", "module.m[*].test.a[*]", false},
	} {
		t.Run(fmt.Sprintf("%s-matches-%s", tc.per, tc.other), func(t *testing.T) {
			per := mustParsePartialResourceInstanceStr(tc.per).PartialResource()
			other := mustParsePartialResourceInstanceStr(tc.other)

			if got := per.MatchesPartial(other.PartialResource()); got != tc.want {
				t.Errorf("MatchesPartial: got %t, want %t", got, tc.want)
			}
			if other.Resource.Key != WildcardKey {
				if got := per.MatchesInstance(other); got != tc.want {
					t.Errorf("MatchesInstance: got %t, want %t", got, tc.want)
				}
				if got := per.MatchesResource(other.ContainingResource()); got != tc.want {
					t.Errorf("MatchesResource: got %t, want %t", got, tc.want)
				}
			}
		})
	}
}

func TestPartialExpandedResource(t *testing.T) {
	partial := mustParsePartialResourceInstanceStr("module.m[0].module.n[*].test.a[*]").PartialResource()
	known := mustParsePartialResourceInstanceStr("module.m[0].test.a[*]").PartialResource()

	if got, want := partial.String(), "module.m[0].module.n[*].test.a[*]"; got != want {
		t.Errorf("wrong String\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := partial.UnknownResourceInstance().String(), "module.m[0].module.n[*].test.a[*]"; got != want {
		t.Errorf("wrong UnknownResourceInstance\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := partial.ConfigResource().String(), "module.m.module.n.test.a"; got != want {
		t.Errorf("wrong ConfigResource\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := partial.KnownModuleInstancePrefix().String(), "module.m[0]"; got != want {
		t.Errorf("wrong KnownModuleInstancePrefix\ngot:  %s\nwant: %s", got, want)
	}
	if _, ok := partial.AbsResource(); ok {
		t.Errorf("partial-expanded module has an AbsResource")
	}
	if _, ok := partial.ModuleInstance(); ok {
		t.Errorf("partial-expanded module has a ModuleInstance")
	}
	if got, ok := partial.PartialExpandedModule(); !ok || got.String() != "module.m[0].module.n[*]" {
		t.Errorf("wrong PartialExpandedModule: got %s, %t", got, ok)
	}

	if got, ok := known.AbsResource(); !ok || got.String() != "module.m[0].test.a" {
		t.Errorf("wrong AbsResource: got %s, %t", got, ok)
	}
	if got, ok := known.ModuleInstance(); !ok || got.String() != "module.m[0]" {
		t.Errorf("wrong ModuleInstance: got %s, %t", got, ok)
	}
	if _, ok := known.PartialExpandedModule(); ok {
		t.Errorf("fully-expanded module has a PartialExpandedModule")
	}
	if got, want := known.UniqueKey(), mustParseAbsResourceInstanceStr("module.m[0].test.a").ContainingResource().UniqueKey(); got != want {
		t.Errorf("wrong UniqueKey: got %#v, want %#v", got, want)
	}
}

func TestPartialExpandedAction(t *testing.T) {
	act := Action{Type: "act", Name: "a"}
	partial := mustParsePartialExpandedModule("module.m[0].module.n[*]").Action(act)
	known := mustParseModuleInstanceStr("module.m[0]").UnexpandedAction(act)

	if got, want := partial.String(), "module.m[0].module.n[*].action.act.a[*]"; got != want {
		t.Errorf("wrong String\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := partial.UnknownActionInstance().String(), "module.m[0].module.n[*].action.act.a[*]"; got != want {
		t.Errorf("wrong UnknownActionInstance\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := partial.ConfigAction().String(), "module.m.module.n.action.act.a"; got != want {
		t.Errorf("wrong ConfigAction\ngot:  %s\nwant: %s", got, want)
	}
	if _, ok := partial.AbsAction(); ok {
		t.Errorf("partial-expanded module has an AbsAction")
	}
	if got, ok := partial.PartialExpandedModule(); !ok || got.String() != "module.m[0].module.n[*]" {
		t.Errorf("wrong PartialExpandedModule: got %s, %t", got, ok)
	}
	if got, ok := known.AbsAction(); !ok || got.String() != "module.m[0].action.act.a" {
		t.Errorf("wrong AbsAction: got %s, %t", got, ok)
	}

	for _, tc := range []struct {
		a, b PartialExpandedAction
		want bool
	}{
		{partial, partial, true},
		{partial, mustParsePartialExpandedModule("module.m[0].module.n[*]").Action(act), true},
		{known, known, true},
		{partial, known, false},
		{partial, mustParsePartialExpandedModule("module.m[1].module.n[*]").Action(act), false},
		{partial, mustParsePartialExpandedModule("module.m[0].module.n[*]").Action(Action{Type: "act", Name: "b"}), false},
	} {
		if got := tc.a.Equal(tc.b); got != tc.want {
			t.Errorf("%s.Equal(%s): got %t, want %t", tc.a, tc.b, got, tc.want)
		}
	}

	for _, tc := range []struct {
		pea   PartialExpandedAction
		other AbsAction
		want  bool
	}{
		{partial, mustParseModuleInstanceStr(`module.m[0].module.n["x"]`).Action("act", "a"), true},
		{partial, mustParseModuleInstanceStr(`module.m[1].module.n["x"]`).Action("act", "a"), false},
		{partial, mustParseModuleInstanceStr(`module.m[0].module.n["x"]`).Action("act", "b"), false},
		{partial, mustParseModuleInstanceStr("module.m[0]").Action("act", "a"), false},
		{known, mustParseModuleInstanceStr("module.m[0]").Action("act", "a"), true},
		{known, mustParseModuleInstanceStr("module.m[1]").Action("act", "a"), false},
	} {
		if got := tc.pea.MatchesAction(tc.other); got != tc.want {
			t.Errorf("%s.MatchesAction(%s): got %t, want %t", tc.pea, tc.other, got, tc.want)
		}
	}
}

// mustParsePartialModuleInstance parses a module instance address which may
// use the [*] wildcard for any instance key.
func mustParsePartialModuleInstance(s string) ModuleInstance {
	traversal, hclDiags := hclsyntax.ParseTraversalAbsPattern([]byte(s), "", hcl.InitialPos)
	if hclDiags.HasErrors() {
		panic(hclDiags.Error())
	}
	addr, remain, diags := parseModuleInstancePrefix(traversal, wildcardInstanceKeys)
	if diags.HasErrors() {
		panic(diags.ErrWithWarnings().Error())
	}
	if len(remain) != 0 {
		panic(fmt.Sprintf("%s is not a module instance address", s))
	}
	return addr
}

// mustParsePartialExpandedModule parses a partial-expanded module address,
// using [*] for each instance key which is not known.
func mustParsePartialExpandedModule(s string) PartialExpandedModule {
	return mustParsePartialModuleInstance(s).PartialModule()
}

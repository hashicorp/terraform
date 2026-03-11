// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"strings"
	"testing"
)

func TestGraph(t *testing.T) {
	a := LocalValue{Name: "a"}
	b := LocalValue{Name: "b"}
	c := LocalValue{Name: "c"}
	d := LocalValue{Name: "d"}

	g := NewDirectedGraph[LocalValue]()

	g.AddDependency(d, c)
	g.AddDependency(d, b)
	g.AddDependency(c, b)
	g.AddDependency(b, a)

	t.Run("StringForComparison", func(t *testing.T) {
		gotStr := strings.TrimSpace(g.StringForComparison())
		wantStr := strings.TrimSpace(`
local.a
local.b
  local.a
local.c
  local.b
local.d
  local.b
  local.c
`)
		if gotStr != wantStr {
			t.Errorf("wrong string representation\ngot:\n%s\n\nwant:\n%s", gotStr, wantStr)
		}
	})

	t.Run("direct dependencies of a", func(t *testing.T) {
		deps := g.DirectDependenciesOf(a)
		if got, want := len(deps), 0; got != want {
			t.Errorf("a has %d dependencies, but should have %d", got, want)
		}
	})
	t.Run("direct dependencies of b", func(t *testing.T) {
		deps := g.DirectDependenciesOf(b)
		if got, want := len(deps), 1; got != want {
			t.Errorf("b has %d dependencies, but should have %d", got, want)
		}
		if !deps.Has(a) {
			t.Errorf("b does not depend on a, but should")
		}
	})
	t.Run("direct dependencies of d", func(t *testing.T) {
		deps := g.DirectDependenciesOf(d)
		if got, want := len(deps), 2; got != want {
			t.Errorf("d has %d dependencies, but should have %d", got, want)
		}
		if !deps.Has(b) {
			t.Errorf("d does not depend on b, but should")
		}
		if !deps.Has(c) {
			t.Errorf("d does not depend on c, but should")
		}
	})
	t.Run("direct dependents of a", func(t *testing.T) {
		depnts := g.DirectDependentsOf(a)
		if got, want := len(depnts), 1; got != want {
			t.Errorf("a has %d dependents, but should have %d", got, want)
		}
		if !depnts.Has(b) {
			t.Errorf("b does not depend on a, but should")
		}
	})
	t.Run("direct dependents of b", func(t *testing.T) {
		depnts := g.DirectDependentsOf(b)
		if got, want := len(depnts), 2; got != want {
			t.Errorf("b has %d dependents, but should have %d", got, want)
		}
		if !depnts.Has(c) {
			t.Errorf("c does not depend on b, but should")
		}
		if !depnts.Has(d) {
			t.Errorf("d does not depend on b, but should")
		}
	})
	t.Run("direct dependents of d", func(t *testing.T) {
		depnts := g.DirectDependentsOf(d)
		if got, want := len(depnts), 0; got != want {
			t.Errorf("d has %d dependents, but should have %d", got, want)
		}
	})
	t.Run("transitive dependencies of a", func(t *testing.T) {
		deps := g.TransitiveDependenciesOf(a)
		if got, want := len(deps), 0; got != want {
			t.Errorf("a has %d transitive dependencies, but should have %d", got, want)
		}
	})
	t.Run("transitive dependencies of b", func(t *testing.T) {
		deps := g.TransitiveDependenciesOf(b)
		if got, want := len(deps), 1; got != want {
			t.Errorf("b has %d transitive dependencies, but should have %d", got, want)
		}
		if !deps.Has(a) {
			t.Errorf("b does not depend on a, but should")
		}
	})
	t.Run("transitive dependencies of d", func(t *testing.T) {
		deps := g.TransitiveDependenciesOf(d)
		if got, want := len(deps), 3; got != want {
			t.Errorf("d has %d transitive dependencies, but should have %d", got, want)
		}
		if !deps.Has(a) {
			t.Errorf("d does not depend on a, but should")
		}
		if !deps.Has(b) {
			t.Errorf("d does not depend on b, but should")
		}
		if !deps.Has(c) {
			t.Errorf("d does not depend on c, but should")
		}
	})
	t.Run("transitive dependents of a", func(t *testing.T) {
		depnts := g.TransitiveDependentsOf(a)
		if got, want := len(depnts), 3; got != want {
			t.Errorf("a has %d transitive dependents, but should have %d", got, want)
		}
		if !depnts.Has(b) {
			t.Errorf("b does not depend on a, but should")
		}
		if !depnts.Has(c) {
			t.Errorf("c does not depend on a, but should")
		}
		if !depnts.Has(d) {
			t.Errorf("d does not depend on a, but should")
		}
	})
	t.Run("transitive dependents of b", func(t *testing.T) {
		depnts := g.TransitiveDependentsOf(b)
		if got, want := len(depnts), 2; got != want {
			t.Errorf("b has %d transitive dependents, but should have %d", got, want)
		}
		if !depnts.Has(c) {
			t.Errorf("c does not depend on b, but should")
		}
		if !depnts.Has(d) {
			t.Errorf("d does not depend on b, but should")
		}
	})
	t.Run("transitive dependents of d", func(t *testing.T) {
		depnts := g.TransitiveDependentsOf(d)
		if got, want := len(depnts), 0; got != want {
			t.Errorf("d has %d transitive dependents, but should have %d", got, want)
		}
	})
}

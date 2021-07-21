package configs

import (
	"io/ioutil"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestResourceUnusedAttrs(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		rootDir := "testdata/valid-modules/unused-attrs"
		parser := NewParser(nil)
		mod, diags := parser.LoadConfigDir(rootDir)
		if diags.HasErrors() {
			t.Fatalf("unexpected parse errors: %s", diags.Error())
		}

		rc := mod.ResourceByAddr(addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "a",
			Name: "b",
		})
		if rc == nil {
			t.Fatalf("missing resource a.b")
		}

		if !rc.Managed.UnusedSet {
			t.Errorf("unused argument is not set, but should be")
		}

		var got []string
		for _, traversal := range rc.Managed.Unused {
			got = append(got, tfdiags.FormatHCLTraversal(traversal))
		}
		sort.Strings(got)

		want := []string{
			".bar[0]",
			".baz.boop",
			`.bleep["bloop"].blah`,
			".foo",
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong traversals\n%s", diff)
		}
	})

	t.Run("override", func(t *testing.T) {
		// In this test case, a normal file sets some "unused" traversals,
		// but then an override file overrides unused to [], which should
		// therefore effectively unset all of the traversals from "unused".

		rootDir := "testdata/valid-modules/unused-attrs-override"
		parser := NewParser(nil)
		mod, diags := parser.LoadConfigDir(rootDir)
		if diags.HasErrors() {
			t.Fatalf("unexpected parse errors: %s", diags.Error())
		}

		rc := mod.ResourceByAddr(addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "a",
			Name: "b",
		})
		if rc == nil {
			t.Fatalf("missing resource a.b")
		}

		if !rc.Managed.UnusedSet {
			t.Errorf("unused argument is not set, but should be")
		}

		var got []string
		for _, traversal := range rc.Managed.Unused {
			got = append(got, tfdiags.FormatHCLTraversal(traversal))
		}
		sort.Strings(got)

		// We want no traversals at all, because of the override file.
		var want []string

		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("wrong traversals\n%s", diff)
		}

	})

	t.Run("experimental", func(t *testing.T) {
		// This is a temporary test making sure that the "unused" attribute
		// is only available with the appropriate experiment enabled. If we
		// later stabilize this feature as-is then we could potentially just
		// drop this test entirely.
		src, err := ioutil.ReadFile("testdata/error-files/unused_attrs_experiment.tf")
		if err != nil {
			t.Fatal(err)
		}

		parser := testParser(map[string]string{
			"mod/unused_attrs_experiment.tf": string(src),
		})

		_, diags := parser.LoadConfigDir("mod")

		const wantSummary = `The "unused" argument is experimental`
		foundMessage := false
		for _, diag := range diags {
			if diag.Summary == wantSummary {
				foundMessage = true
				break
			}
		}
		if !foundMessage {
			t.Errorf("didn't see the expected error\nwant summary: %s\ngot: %s", wantSummary, diags.Error())
		}
	})
}

package discovery

import (
	"fmt"
	"strings"
	"testing"

	"github.com/blang/semver"
)

func TestPluginMetaSetManipulation(t *testing.T) {
	metas := []PluginMeta{
		{
			Name:    "foo",
			Version: "1.0.0",
			Path:    "test-foo",
		},
		{
			Name:    "bar",
			Version: "2.0.0",
			Path:    "test-bar",
		},
		{
			Name:    "baz",
			Version: "2.0.0",
			Path:    "test-bar",
		},
	}
	s := make(PluginMetaSet)

	if count := s.Count(); count != 0 {
		t.Fatalf("set has Count %d before any items added", count)
	}

	// Can we add metas?
	for _, p := range metas {
		s.Add(p)
		if !s.Has(p) {
			t.Fatalf("%q not in set after adding it", p.Name)
		}
	}

	if got, want := s.Count(), len(metas); got != want {
		t.Fatalf("set has Count %d after all items added; want %d", got, want)
	}

	// Can we still retrieve earlier ones after we added later ones?
	for _, p := range metas {
		if !s.Has(p) {
			t.Fatalf("%q not in set after all adds", p.Name)
		}
	}

	// Can we remove metas?
	for _, p := range metas {
		s.Remove(p)
		if s.Has(p) {
			t.Fatalf("%q still in set after removing it", p.Name)
		}
	}

	if count := s.Count(); count != 0 {
		t.Fatalf("set has Count %d after all items removed", count)
	}
}

func TestPluginMetaSetValidateVersions(t *testing.T) {
	metas := []PluginMeta{
		{
			Name:    "foo",
			Version: "1.0.0",
			Path:    "test-foo",
		},
		{
			Name:    "bar",
			Version: "0.0.1",
			Path:    "test-bar",
		},
		{
			Name:    "baz",
			Version: "bananas",
			Path:    "test-bar",
		},
	}
	s := make(PluginMetaSet)

	for _, p := range metas {
		s.Add(p)
	}

	valid, invalid := s.ValidateVersions()
	if count := valid.Count(); count != 2 {
		t.Errorf("valid set has %d metas; want 2", count)
	}
	if count := invalid.Count(); count != 1 {
		t.Errorf("valid set has %d metas; want 1", count)
	}

	if !valid.Has(metas[0]) {
		t.Errorf("'foo' not in valid set")
	}
	if !valid.Has(metas[1]) {
		t.Errorf("'bar' not in valid set")
	}
	if !invalid.Has(metas[2]) {
		t.Errorf("'baz' not in invalid set")
	}

	if invalid.Has(metas[0]) {
		t.Errorf("'foo' in invalid set")
	}
	if invalid.Has(metas[1]) {
		t.Errorf("'bar' in invalid set")
	}
	if valid.Has(metas[2]) {
		t.Errorf("'baz' in valid set")
	}

}

func TestPluginMetaSetWithName(t *testing.T) {
	tests := []struct {
		metas     []PluginMeta
		name      string
		wantCount int
	}{
		{
			[]PluginMeta{},
			"foo",
			0,
		},
		{
			[]PluginMeta{
				{
					Name:    "foo",
					Version: "0.0.1",
					Path:    "foo",
				},
			},
			"foo",
			1,
		},
		{
			[]PluginMeta{
				{
					Name:    "foo",
					Version: "0.0.1",
					Path:    "foo",
				},
			},
			"bar",
			0,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("Test%02d", i), func(t *testing.T) {
			s := make(PluginMetaSet)
			for _, p := range test.metas {
				s.Add(p)
			}
			filtered := s.WithName(test.name)
			if gotCount := filtered.Count(); gotCount != test.wantCount {
				t.Errorf("got count %d in %#v; want %d", gotCount, filtered, test.wantCount)
			}
		})
	}
}

func TestPluginMetaSetByName(t *testing.T) {
	metas := []PluginMeta{
		{
			Name:    "foo",
			Version: "1.0.0",
			Path:    "test-foo",
		},
		{
			Name:    "foo",
			Version: "2.0.0",
			Path:    "test-foo-2",
		},
		{
			Name:    "bar",
			Version: "0.0.1",
			Path:    "test-bar",
		},
		{
			Name:    "baz",
			Version: "1.2.0",
			Path:    "test-bar",
		},
	}
	s := make(PluginMetaSet)

	for _, p := range metas {
		s.Add(p)
	}

	byName := s.ByName()
	if got, want := len(byName), 3; got != want {
		t.Errorf("%d keys in ByName map; want %d", got, want)
	}
	if got, want := len(byName["foo"]), 2; got != want {
		t.Errorf("%d metas for 'foo'; want %d", got, want)
	}
	if got, want := len(byName["bar"]), 1; got != want {
		t.Errorf("%d metas for 'bar'; want %d", got, want)
	}
	if got, want := len(byName["baz"]), 1; got != want {
		t.Errorf("%d metas for 'baz'; want %d", got, want)
	}

	if !byName["foo"].Has(metas[0]) {
		t.Errorf("%#v missing from 'foo' set", metas[0])
	}
	if !byName["foo"].Has(metas[1]) {
		t.Errorf("%#v missing from 'foo' set", metas[1])
	}
	if !byName["bar"].Has(metas[2]) {
		t.Errorf("%#v missing from 'bar' set", metas[2])
	}
	if !byName["baz"].Has(metas[3]) {
		t.Errorf("%#v missing from 'baz' set", metas[3])
	}
}

func TestPluginMetaSetNewest(t *testing.T) {
	tests := []struct {
		versions []string
		want     string
	}{
		{
			[]string{
				"0.0.1",
			},
			"0.0.1",
		},
		{
			[]string{
				"0.0.1",
				"0.0.2",
			},
			"0.0.2",
		},
		{
			[]string{
				"1.0.0",
				"1.0.0-beta1",
			},
			"1.0.0",
		},
		{
			[]string{
				"0.0.1",
				"1.0.0",
			},
			"1.0.0",
		},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.versions, "|"), func(t *testing.T) {
			s := make(PluginMetaSet)
			for _, version := range test.versions {
				s.Add(PluginMeta{
					Name:    "foo",
					Version: version,
					Path:    "foo-V" + version,
				})
			}

			newest := s.Newest()
			if newest.Version != test.want {
				t.Errorf("version is %q; want %q", newest.Version, test.want)
			}
		})
	}
}

func TestPluginMetaSetConstrainVersions(t *testing.T) {
	metas := []PluginMeta{
		{
			Name:    "foo",
			Version: "1.0.0",
			Path:    "test-foo",
		},
		{
			Name:    "foo",
			Version: "2.0.0",
			Path:    "test-foo-2",
		},
		{
			Name:    "foo",
			Version: "3.0.0",
			Path:    "test-foo-2",
		},
		{
			Name:    "bar",
			Version: "0.0.5",
			Path:    "test-bar",
		},
		{
			Name:    "baz",
			Version: "0.0.1",
			Path:    "test-bar",
		},
	}
	s := make(PluginMetaSet)

	for _, p := range metas {
		s.Add(p)
	}

	byName := s.ConstrainVersions(map[string]semver.Range{
		"foo": semver.MustParseRange(">=2.0.0"),
		"bar": semver.MustParseRange(">=0.0.0"),
		"baz": semver.MustParseRange(">=1.0.0"),
		"fun": semver.MustParseRange(">5.0.0"),
	})
	if got, want := len(byName), 3; got != want {
		t.Errorf("%d keys in map; want %d", got, want)
	}

	if got, want := len(byName["foo"]), 2; got != want {
		t.Errorf("%d metas for 'foo'; want %d", got, want)
	}
	if got, want := len(byName["bar"]), 1; got != want {
		t.Errorf("%d metas for 'bar'; want %d", got, want)
	}
	if got, want := len(byName["baz"]), 0; got != want {
		t.Errorf("%d metas for 'baz'; want %d", got, want)
	}
	// "fun" is not in the map at all, because we have no metas for that name

	if !byName["foo"].Has(metas[1]) {
		t.Errorf("%#v missing from 'foo' set", metas[1])
	}
	if !byName["foo"].Has(metas[2]) {
		t.Errorf("%#v missing from 'foo' set", metas[2])
	}
	if !byName["bar"].Has(metas[3]) {
		t.Errorf("%#v missing from 'bar' set", metas[3])
	}

}

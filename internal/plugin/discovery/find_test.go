// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestFindPluginPaths(t *testing.T) {
	got := findPluginPaths(
		"foo",
		[]string{
			"testdata/current-style-plugins/mockos_mockarch",
			"testdata/legacy-style-plugins",
			"testdata/non-existent",
			"testdata/not-a-dir",
		},
	)
	want := []string{
		filepath.Join("testdata", "current-style-plugins", "mockos_mockarch", "terraform-foo-bar_v0.0.1"),
		filepath.Join("testdata", "current-style-plugins", "mockos_mockarch", "terraform-foo-bar_v1.0.0.exe"),
		// un-versioned plugins are still picked up, even in current-style paths
		filepath.Join("testdata", "current-style-plugins", "mockos_mockarch", "terraform-foo-missing-version"),
		filepath.Join("testdata", "legacy-style-plugins", "terraform-foo-bar"),
		filepath.Join("testdata", "legacy-style-plugins", "terraform-foo-baz"),
	}

	// Turn the paths back into relative paths, since we don't care exactly
	// where this code is present on the system for the sake of this test.
	baseDir, err := os.Getwd()
	if err != nil {
		// Should never happen
		panic(err)
	}
	for i, absPath := range got {
		if !filepath.IsAbs(absPath) {
			t.Errorf("got non-absolute path %s", absPath)
		}

		got[i], err = filepath.Rel(baseDir, absPath)
		if err != nil {
			t.Fatalf("Can't make %s relative to current directory %s", absPath, baseDir)
		}
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}

func TestResolvePluginPaths(t *testing.T) {
	got := ResolvePluginPaths([]string{
		"/example/mockos_mockarch/terraform-foo-bar_v0.0.1",
		"/example/mockos_mockarch/terraform-foo-baz_v0.0.1",
		"/example/mockos_mockarch/terraform-foo-baz_v1.0.0",
		"/example/mockos_mockarch/terraform-foo-baz_v2.0.0_x4",
		"/example/mockos_mockarch/terraform-foo-upper_V2.0.0_X4",
		"/example/terraform-foo-bar",
		"/example/mockos_mockarch/terraform-foo-bar_vbananas",
		"/example/mockos_mockarch/terraform-foo-bar_v",
		"/example/mockos_mockarch/terraform-foo-windowsthing1_v1.0.0.exe",
		"/example/mockos_mockarch/terraform-foo-windowsthing2_v1.0.0_x4.exe",
		"/example/mockos_mockarch/terraform-foo-windowsthing3.exe",
		"/example2/mockos_mockarch/terraform-foo-bar_v0.0.1",
	})

	want := []PluginMeta{
		{
			Name:    "bar",
			Version: "0.0.1",
			Path:    "/example/mockos_mockarch/terraform-foo-bar_v0.0.1",
		},
		{
			Name:    "baz",
			Version: "0.0.1",
			Path:    "/example/mockos_mockarch/terraform-foo-baz_v0.0.1",
		},
		{
			Name:    "baz",
			Version: "1.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-baz_v1.0.0",
		},
		{
			Name:    "baz",
			Version: "2.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-baz_v2.0.0_x4",
		},
		{
			Name:    "upper",
			Version: "2.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-upper_V2.0.0_X4",
		},
		{
			Name:    "bar",
			Version: "0.0.0",
			Path:    "/example/terraform-foo-bar",
		},
		{
			Name:    "bar",
			Version: "bananas",
			Path:    "/example/mockos_mockarch/terraform-foo-bar_vbananas",
		},
		{
			Name:    "bar",
			Version: "",
			Path:    "/example/mockos_mockarch/terraform-foo-bar_v",
		},
		{
			Name:    "windowsthing1",
			Version: "1.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-windowsthing1_v1.0.0.exe",
		},
		{
			Name:    "windowsthing2",
			Version: "1.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-windowsthing2_v1.0.0_x4.exe",
		},
		{
			Name:    "windowsthing3",
			Version: "0.0.0",
			Path:    "/example/mockos_mockarch/terraform-foo-windowsthing3.exe",
		},
	}

	for p := range got {
		t.Logf("got %#v", p)
	}

	if got, want := got.Count(), len(want); got != want {
		t.Errorf("got %d items; want %d", got, want)
	}

	for _, wantMeta := range want {
		if !got.Has(wantMeta) {
			t.Errorf("missing meta %#v", wantMeta)
		}
	}
}

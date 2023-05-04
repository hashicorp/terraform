// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package discovery

import (
	"testing"
)

func TestLocalPluginCache(t *testing.T) {
	cache := NewLocalPluginCache("testdata/plugin-cache")

	foo1Path := cache.CachedPluginPath("provider", "foo", VersionStr("v0.0.1").MustParse())
	if foo1Path == "" {
		t.Errorf("foo v0.0.1 not found; should have been found")
	}

	foo2Path := cache.CachedPluginPath("provider", "foo", VersionStr("v0.0.2").MustParse())
	if foo2Path != "" {
		t.Errorf("foo v0.0.2 found at %s; should not have been found", foo2Path)
	}

	baz1Path := cache.CachedPluginPath("provider", "baz", VersionStr("v0.0.1").MustParse())
	if baz1Path != "" {
		t.Errorf("baz v0.0.1 found at %s; should not have been found", baz1Path)
	}

	baz2Path := cache.CachedPluginPath("provider", "baz", VersionStr("v0.0.2").MustParse())
	if baz1Path != "" {
		t.Errorf("baz v0.0.2 found at %s; should not have been found", baz2Path)
	}
}

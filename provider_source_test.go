// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestProviderSource_skipsImplicitLocalDirWhenItMatchesPluginCacheDir(t *testing.T) {
	home := t.TempDir()
	pluginsDir := filepath.Join(home, ".terraform.d", "plugins")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatalf("failed to create plugins dir: %s", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(home, "xdg-data-dirs-does-not-exist"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data-home-does-not-exist"))

	src, diags := providerSource(nil, disco.NewWithCredentialsSource(nil), pluginsDir)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err())
	}

	multi, ok := src.(getproviders.MultiSource)
	if !ok {
		t.Fatalf("unexpected source type %T", src)
	}
	if len(multi) != 1 {
		t.Fatalf("expected only the registry source when plugin_cache_dir matches the implicit search dir, got %d selectors", len(multi))
	}
}

func TestProviderSource_keepsImplicitLocalDirWhenPluginCacheDirDiffers(t *testing.T) {
	home := t.TempDir()
	pluginsDir := filepath.Join(home, ".terraform.d", "plugins")
	cacheDir := filepath.Join(home, ".terraform.d", "cache")
	if err := os.MkdirAll(pluginsDir, 0o755); err != nil {
		t.Fatalf("failed to create plugins dir: %s", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_DIRS", filepath.Join(home, "xdg-data-dirs-does-not-exist"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "xdg-data-home-does-not-exist"))

	src, diags := providerSource(nil, disco.NewWithCredentialsSource(nil), cacheDir)
	if diags.HasErrors() {
		t.Fatalf("unexpected diagnostics: %s", diags.Err())
	}

	multi, ok := src.(getproviders.MultiSource)
	if !ok {
		t.Fatalf("unexpected source type %T", src)
	}
	if len(multi) != 2 {
		t.Fatalf("expected implicit local dir plus registry when plugin_cache_dir differs, got %d selectors", len(multi))
	}
}

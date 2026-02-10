// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
)

// TestSourceBundleParser_LoadConfigDir_WithRelativePath tests that when
// LocalPathForSource returns a relative path, LoadConfigDir correctly uses
// it as-is without attempting to convert it.
func TestSourceBundleParser_LoadConfigDir_WithRelativePath(t *testing.T) {
	// Use the basics-bundle from stacks testdata which has a component with has a relative source.
	bundlePath := "../stacks/stackconfig/testdata/basics-bundle"
	bundle, err := sourcebundle.OpenDir(bundlePath)
	if err != nil {
		t.Fatalf("failed to open source bundle: %s", err)
	}

	source := sourceaddrs.MustParseSource("../stacks/stackconfig/testdata/basics-bundle").(sourceaddrs.FinalSource)

	// Create a SourceBundleParser and load the config directory.
	parser := NewSourceBundleParser(bundle)
	mod, diags := parser.LoadConfigDir(source)

	if diags.HasErrors() {
		t.Fatalf("unexpected errors: %s", diags.Error())
	}

	if mod == nil {
		t.Fatal("expected non-nil module")
	}

	// Verify that the SourceDir is set and that it's a relative path.
	if mod.SourceDir == "" {
		t.Error("expected SourceDir to be set, but it was empty")
	}
	if filepath.IsAbs(mod.SourceDir) {
		t.Errorf("expected SourceDir to be relative, but got absolute path: %s", mod.SourceDir)
	}
}

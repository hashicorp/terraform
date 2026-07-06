// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-slug/sourceaddrs"
	"github.com/hashicorp/go-slug/sourcebundle"
)

func TestSourceBundleParserConcurrentLoadConfigDir(t *testing.T) {
	bundleDir := testSourceBundleDir(t)
	bundle, err := sourcebundle.OpenDir(bundleDir)
	if err != nil {
		t.Fatalf("failed to open source bundle: %s", err)
	}

	source := sourceaddrs.MustParseSource("git::https://example.com/root.git").(sourceaddrs.FinalSource)
	parser := NewSourceBundleParser(bundle)

	const (
		loaders    = 8
		iterations = 20
	)

	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(loaders)

	for i := 0; i < loaders; i++ {
		go func() {
			defer workers.Done()
			<-start

			for i := 0; i < iterations; i++ {
				_, diags := parser.LoadConfigDir(source)
				if diags.HasErrors() {
					t.Errorf("unexpected diagnostics: %s", diags.Error())
					return
				}
			}
		}()
	}

	close(start)
	workers.Wait()
}

func testSourceBundleDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	moduleDir := filepath.Join(dir, "root")
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("failed to create module dir: %s", err)
	}

	manifest := `{
	  "terraform_source_bundle": 1,
	  "packages": [
	    {
	      "source": "git::https://example.com/root.git",
	      "local": "root",
	      "meta": {}
	    }
	  ],
	  "registry": []
	}`
	if err := os.WriteFile(filepath.Join(dir, "terraform-sources.json"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("failed to write bundle manifest: %s", err)
	}

	resourceBody := strings.Repeat("x", 4096)
	for i := 0; i < 40; i++ {
		filename := filepath.Join(moduleDir, fmt.Sprintf("file%02d.tf", i))
		content := fmt.Sprintf("resource \"terraform_data\" \"example_%02d\" {\n  input = \"%s\"\n}\n", i, resourceBody)
		if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write test source file: %s", err)
		}
	}

	return dir
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/e2e"
)

// The tests in this file are for the "terraform providers mirror" command,
// which is tested in an e2etest mode rather than a unit test mode because it
// interacts directly with Terraform Registry and the full details of that are
// tricky to mock. Such a mock is _possible_, but we're using e2etest as a
// compromise for now to keep these tests relatively simple.
func TestTerraformProvidersMirror(t *testing.T) {
	// This test reaches out to releases.hashicorp.com to download the
	// template and null providers, so it can only run if network access is
	// allowed.
	skipIfCannotAccessNetwork(t)

	for _, test := range []struct {
		name string
		args []string
		err  string
	}{
		{
			name: "terraform-providers-mirror",
			args: []string{"-platform=linux_amd64", "-platform=windows_386"},
		},
		{
			name: "terraform-providers-mirror-with-lock-file",
			args: []string{"-platform=linux_amd64", "-platform=windows_386"},
		},
		{
			// should ignore lock file
			name: "terraform-providers-mirror-with-broken-lock-file",
			args: []string{"-platform=linux_amd64", "-platform=windows_386", "-lock-file=false"},
		},
		{
			name: "terraform-providers-mirror-with-broken-lock-file",
			args: []string{"-platform=linux_amd64", "-platform=windows_386", "-lock-file=true"},
			err:  "Inconsistent dependency lock file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			outputDir := t.TempDir()
			t.Logf("creating mirror directory in %s", outputDir)

			fixturePath := filepath.Join("testdata", test.name)
			tf := e2e.NewBinary(t, terraformBin, fixturePath)

			args := []string{"providers", "mirror"}
			args = append(args, test.args...)
			args = append(args, outputDir)

			stdout, stderr, err := tf.Run(args...)
			if test.err != "" {
				if !strings.Contains(stderr, test.err) {
					t.Fatalf("expected error %q, got %q\n", test.err, stderr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %s\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
			}

			// The test fixture includes exact version constraints for the two
			// providers it depends on so that the following should remain stable.
			// In the (unlikely) event that these particular versions of these
			// providers are removed from the registry, this test will start to fail.
			want := []string{
				"registry.terraform.io/hashicorp/null/2.1.0.json",
				"registry.terraform.io/hashicorp/null/index.json",
				"registry.terraform.io/hashicorp/null/terraform-provider-null_2.1.0_linux_amd64.zip",
				"registry.terraform.io/hashicorp/null/terraform-provider-null_2.1.0_windows_386.zip",
				"registry.terraform.io/hashicorp/template/2.1.1.json",
				"registry.terraform.io/hashicorp/template/index.json",
				"registry.terraform.io/hashicorp/template/terraform-provider-template_2.1.1_linux_amd64.zip",
				"registry.terraform.io/hashicorp/template/terraform-provider-template_2.1.1_windows_386.zip",
			}
			var got []string
			walkErr := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil // we only care about leaf files for this test
				}
				relPath, err := filepath.Rel(outputDir, path)
				if err != nil {
					return err
				}
				got = append(got, filepath.ToSlash(relPath))
				return nil
			})
			if walkErr != nil {
				t.Fatal(walkErr)
			}
			sort.Strings(got)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("unexpected files in result\n%s", diff)
			}

		})
	}
}

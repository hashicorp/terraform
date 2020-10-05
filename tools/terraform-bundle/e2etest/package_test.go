package e2etest

import (
	"archive/zip"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/e2e"
)

func TestPackage_empty(t *testing.T) {
	t.Parallel()
	// This test reaches out to releases.hashicorp.com to download the
	// template provider, so it can only run if network access is allowed.
	// We intentionally don't try to stub this here, because there's already
	// a stubbed version of this in the "command" package and so the goal here
	// is to test the interaction with the real repository.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "empty")
	tfBundle := e2e.NewBinary(bundleBin, fixturePath)
	defer tfBundle.Close()

	stdout, stderr, err := tfBundle.Run("package", "terraform-bundle.hcl")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "Fetching Terraform 0.13.0 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.13.0-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
}

func TestPackage_manyProviders(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download providers, so
	// it can only run if network access is allowed. We intentionally don't try
	// to stub this here, because there's already a stubbed version of this in
	// the "command" package and so the goal here is to test the interaction
	// with the real repository.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "many-providers")
	tfBundle := e2e.NewBinary(bundleBin, fixturePath)
	defer tfBundle.Close()

	stdout, stderr, err := tfBundle.Run("package", "terraform-bundle.hcl")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	// Here we have to check each provider separately
	// because it's internally held in a map (i.e. not guaranteed order)

	if !strings.Contains(stdout, `- Finding hashicorp/aws versions matching "~> 2.26.0"...
- Installing hashicorp/aws v2.26.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Finding hashicorp/kubernetes versions matching "1.8.0"...
- Installing hashicorp/kubernetes v1.8.0...
- Finding hashicorp/kubernetes versions matching "1.8.1"...
- Installing hashicorp/kubernetes v1.8.1...
- Finding hashicorp/kubernetes versions matching "1.9.0"...
- Installing hashicorp/kubernetes v1.9.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Finding hashicorp/null versions matching "2.1.0"...
- Installing hashicorp/null v2.1.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, "Fetching Terraform 0.13.0 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.13.0-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	// check the contents of the created zipfile
	files, err := ioutil.ReadDir(tfBundle.WorkDir())
	if err != nil {
		t.Fatalf("error reading workdir: %s", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "terraform_0.13.0-bundle") {
			read, err := zip.OpenReader(filepath.Join(tfBundle.WorkDir(), file.Name()))
			if err != nil {
				t.Fatalf("Failed to open archive: %s", err)
			}
			defer read.Close()

			expectedFiles := map[string]struct{}{
				"terraform":                                   {},
				testProviderBinaryPath("null", "2.1.0"):       {},
				testProviderBinaryPath("aws", "2.26.0"):       {},
				testProviderBinaryPath("kubernetes", "1.8.0"): {},
				testProviderBinaryPath("kubernetes", "1.8.1"): {},
				testProviderBinaryPath("kubernetes", "1.9.0"): {},
			}
			extraFiles := make(map[string]struct{})

			for _, file := range read.File {
				if _, exists := expectedFiles[file.Name]; exists {
					if !file.FileInfo().Mode().IsRegular() {
						t.Errorf("Expected file is not a regular file: %s", file.Name)
					}
					delete(expectedFiles, file.Name)
				} else {
					extraFiles[file.Name] = struct{}{}
				}
			}
			if len(expectedFiles) != 0 {
				t.Errorf("missing expected file(s): %#v", expectedFiles)
			}
			if len(extraFiles) != 0 {
				t.Errorf("found extra unexpected file(s): %#v", extraFiles)
			}
		}
	}
}

func TestPackage_localProviders(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download terrafrom, so
	// it can only run if network access is allowed. The providers are installed
	// from the local cache.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "local-providers")
	tfBundle := e2e.NewBinary(bundleBin, fixturePath)
	defer tfBundle.Close()

	// we explicitly specify the platform so that tests can find the local binary under the expected directory
	stdout, stderr, err := tfBundle.Run("package", "-os=darwin", "-arch=amd64", "terraform-bundle.hcl")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	// Here we have to check each provider separately
	// because it's internally held in a map (i.e. not guaranteed order)
	if !strings.Contains(stdout, "Fetching Terraform 0.13.0 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.13.0-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	// check the contents of the created zipfile
	files, err := ioutil.ReadDir(tfBundle.WorkDir())
	if err != nil {
		t.Fatalf("error reading workdir: %s", err)
	}
	for _, file := range files {
		if strings.Contains(file.Name(), "terraform_0.13.0-bundle") {
			read, err := zip.OpenReader(filepath.Join(tfBundle.WorkDir(), file.Name()))
			if err != nil {
				t.Fatalf("Failed to open archive: %s", err)
			}
			defer read.Close()

			expectedFiles := map[string]struct{}{
				"terraform": {},
				"plugins/example.com/myorg/mycloud/0.1.0/darwin_amd64/terraform-provider-mycloud": {},
			}
			extraFiles := make(map[string]struct{})

			for _, file := range read.File {
				if _, exists := expectedFiles[file.Name]; exists {
					if !file.FileInfo().Mode().IsRegular() {
						t.Errorf("Expected file is not a regular file: %s", file.Name)
					}
					delete(expectedFiles, file.Name)
				} else {
					extraFiles[file.Name] = struct{}{}
				}
			}
			if len(expectedFiles) != 0 {
				t.Errorf("missing expected file(s): %#v", expectedFiles)
			}
			if len(extraFiles) != 0 {
				t.Errorf("found extra unexpected file(s): %#v", extraFiles)
			}
		}
	}
}

// testProviderBinaryPath takes a provider name (assumed to be a hashicorp
// provider) and version and returns the expected binary path, relative to the
// archive, for the plugin.
func testProviderBinaryPath(provider, version string) string {
	os := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf(
		"plugins/registry.terraform.io/hashicorp/%s/%s/%s_%s/terraform-provider-%s_v%s_x4",
		provider, version, os, arch, provider, version,
	)
}

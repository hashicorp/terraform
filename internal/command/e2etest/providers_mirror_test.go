package e2etest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
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

	outputDir, err := ioutil.TempDir("", "terraform-e2etest-providers-mirror")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outputDir)
	t.Logf("creating mirror directory in %s", outputDir)

	fixturePath := filepath.Join("testdata", "terraform-providers-mirror")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, stderr, err := tf.Run("providers", "mirror", "-platform=linux_amd64", "-platform=windows_386", outputDir)
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
}

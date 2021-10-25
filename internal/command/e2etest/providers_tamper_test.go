package e2etest

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
)

// TestProviderTampering tests various ways that the provider plugins in the
// local cache directory might be modified after an initial "terraform init",
// which other Terraform commands which use those plugins should catch and
// report early.
func TestProviderTampering(t *testing.T) {
	// General setup: we'll do a one-off init of a test directory as our
	// starting point, and then we'll clone that result for each test so
	// that we can save the cost of a repeated re-init with the same
	// provider.
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// null provider, so it can only run if network access is allowed.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "provider-tampering-base")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "Installing hashicorp/null v") {
		t.Errorf("null provider download message is missing from init output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	seedDir := tf.WorkDir()
	const providerVersion = "3.1.0" // must match the version in the fixture config
	pluginDir := ".terraform/providers/registry.terraform.io/hashicorp/null/" + providerVersion + "/" + getproviders.CurrentPlatform.String()
	pluginExe := pluginDir + "/terraform-provider-null_v" + providerVersion + "_x5"
	if getproviders.CurrentPlatform.OS == "windows" {
		pluginExe += ".exe" // ugh
	}

	t.Run("cache dir totally gone", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		err := os.RemoveAll(filepath.Join(workDir, ".terraform"))
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("plan")
		if err == nil {
			t.Fatalf("unexpected plan success\nstdout:\n%s", stdout)
		}
		if want := `registry.terraform.io/hashicorp/null: there is no package for registry.terraform.io/hashicorp/null 3.1.0 cached in .terraform/providers`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `terraform init`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}

		// Running init as suggested resolves the problem
		_, stderr, err = tf.Run("init")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}
		_, stderr, err = tf.Run("plan")
		if err != nil {
			t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
		}
	})
	t.Run("cache dir totally gone, explicit backend", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		err := ioutil.WriteFile(filepath.Join(workDir, "backend.tf"), []byte(localBackendConfig), 0600)
		if err != nil {
			t.Fatal(err)
		}

		err = os.RemoveAll(filepath.Join(workDir, ".terraform"))
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("plan")
		if err == nil {
			t.Fatalf("unexpected plan success\nstdout:\n%s", stdout)
		}
		if want := `Initial configuration of the requested backend "local"`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `terraform init`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}

		// Running init as suggested resolves the problem
		_, stderr, err = tf.Run("init")
		if err != nil {
			t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
		}
		_, stderr, err = tf.Run("plan")
		if err != nil {
			t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
		}
	})
	t.Run("null plugin package modified before plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		err := ioutil.WriteFile(filepath.Join(workDir, pluginExe), []byte("tamper"), 0600)
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("plan")
		if err == nil {
			t.Fatalf("unexpected plan success\nstdout:\n%s", stdout)
		}
		if want := `registry.terraform.io/hashicorp/null: the cached package for registry.terraform.io/hashicorp/null 3.1.0 (in .terraform/providers) does not match any of the checksums recorded in the dependency lock file`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `terraform init`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
	t.Run("version constraint changed in config before plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		err := ioutil.WriteFile(filepath.Join(workDir, "provider-tampering-base.tf"), []byte(`
			terraform {
				required_providers {
					null = {
						source  = "hashicorp/null"
						version = "1.0.0"
					}
				}
			}
		`), 0600)
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("plan")
		if err == nil {
			t.Fatalf("unexpected plan success\nstdout:\n%s", stdout)
		}
		if want := `provider registry.terraform.io/hashicorp/null: locked version selection 3.1.0 doesn't match the updated version constraints "1.0.0"`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `terraform init -upgrade`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
	t.Run("lock file modified before plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		// NOTE: We're just emptying out the lock file here because that's
		// good enough for what we're trying to assert. The leaf codepath
		// that generates this family of errors has some different variations
		// of this error message for otehr sorts of inconsistency, but those
		// are tested more thoroughly over in the "configs" package, which is
		// ultimately responsible for that logic.
		err := ioutil.WriteFile(filepath.Join(workDir, ".terraform.lock.hcl"), []byte(``), 0600)
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("plan")
		if err == nil {
			t.Fatalf("unexpected plan success\nstdout:\n%s", stdout)
		}
		if want := `provider registry.terraform.io/hashicorp/null: required by this configuration but no version is selected`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `terraform init`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
	t.Run("lock file modified after plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		_, stderr, err := tf.Run("plan", "-out", "tfplan")
		if err != nil {
			t.Fatalf("unexpected plan failure\nstderr:\n%s", stderr)
		}

		err = os.Remove(filepath.Join(workDir, ".terraform.lock.hcl"))
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("apply", "tfplan")
		if err == nil {
			t.Fatalf("unexpected apply success\nstdout:\n%s", stdout)
		}
		if want := `provider registry.terraform.io/hashicorp/null: required by this configuration but no version is selected`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
		if want := `Create a new plan from the updated configuration.`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
	t.Run("plugin cache dir entirely removed after plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		_, stderr, err := tf.Run("plan", "-out", "tfplan")
		if err != nil {
			t.Fatalf("unexpected plan failure\nstderr:\n%s", stderr)
		}

		err = os.RemoveAll(filepath.Join(workDir, ".terraform"))
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("apply", "tfplan")
		if err == nil {
			t.Fatalf("unexpected apply success\nstdout:\n%s", stdout)
		}
		if want := `registry.terraform.io/hashicorp/null: there is no package for registry.terraform.io/hashicorp/null 3.1.0 cached in .terraform/providers`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
	t.Run("null plugin package modified after plan", func(t *testing.T) {
		tf := e2e.NewBinary(terraformBin, seedDir)
		defer tf.Close()
		workDir := tf.WorkDir()

		_, stderr, err := tf.Run("plan", "-out", "tfplan")
		if err != nil {
			t.Fatalf("unexpected plan failure\nstderr:\n%s", stderr)
		}

		err = ioutil.WriteFile(filepath.Join(workDir, pluginExe), []byte("tamper"), 0600)
		if err != nil {
			t.Fatal(err)
		}

		stdout, stderr, err := tf.Run("apply", "tfplan")
		if err == nil {
			t.Fatalf("unexpected apply success\nstdout:\n%s", stdout)
		}
		if want := `registry.terraform.io/hashicorp/null: the cached package for registry.terraform.io/hashicorp/null 3.1.0 (in .terraform/providers) does not match any of the checksums recorded in the dependency lock file`; !strings.Contains(stderr, want) {
			t.Errorf("missing expected error message\nwant substring: %s\ngot:\n%s", want, stderr)
		}
	})
}

const localBackendConfig = `
terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}
`

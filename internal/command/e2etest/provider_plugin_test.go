// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
	"github.com/hashicorp/terraform/internal/getproviders/providerreqs"
	"github.com/hashicorp/terraform/internal/grpcwrap"
	tfplugin "github.com/hashicorp/terraform/internal/plugin6"
	simple "github.com/hashicorp/terraform/internal/provider-simple-v6"
	proto "github.com/hashicorp/terraform/internal/tfplugin6"
)

// TestProviderProtocols verifies that Terraform can execute provider plugins
// with both supported protocol versions.
func TestProviderProtocols(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}
	t.Parallel()

	tf := e2e.NewBinary(t, terraformBin, "testdata/provider-plugin")

	// In order to do a decent end-to-end test for this case we will need a real
	// enough provider plugin to try to run and make sure we are able to
	// actually run it. Here will build the simple and simple6 (built with
	// protocol v6) providers.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	simpleProvider := filepath.Join(tf.WorkDir(), "terraform-provider-simple")
	simpleProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simpleProvider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(tf.Path(hashiDir, "simple/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simpleProviderExe, tf.Path(hashiDir, "simple/0.0.1/", platform, "terraform-provider-simple")); err != nil {
		t.Fatal(err)
	}

	//// INIT
	_, stderr, err := tf.Run("init", "-plugin-dir=cache")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan", "-out=tfplan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	//// APPLY
	stdout, stderr, err := tf.Run("apply", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.") {
		t.Fatalf("wrong output:\nstdout:%s\nstderr%s", stdout, stderr)
	}

	/// DESTROY
	stdout, stderr, err = tf.Run("destroy", "-auto-approve")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 2 destroyed") {
		t.Fatalf("wrong destroy output\nstdout:%s\nstderr:%s", stdout, stderr)
	}
}

// TestProviderInstall_dev_override verifies provider plugin installation behaviour
// when a dev_override is in use.
func TestProviderInstall_dev_override(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := "testdata/provider-plugin" // Reused

	// In temp dir create a plugin cache to be used in the test cases.
	// The cache is supplied to commands using the -plugin-dir init flag.
	// There are 4 providers total:
	// - simple provider with versions 1.0.0 and 2.0.0 available
	// - simple6 provider with versions 1.0.0 and 2.0.0 available
	td := t.TempDir()
	providerVersionOld := "1.0.0"
	providerVersionNew := "2.0.0"
	platform := getproviders.CurrentPlatform.String()
	absolutePathToCache := filepath.Join(td, "cache")
	simple5Provider := filepath.Join(td, "terraform-provider-simple")
	simple5ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simple5Provider)
	for _, v := range []string{providerVersionOld, providerVersionNew} {
		dir := filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", v, platform)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		// Create an executable copy of the simple5ProviderExe file per version in the cache dir
		data, err := os.ReadFile(simple5ProviderExe)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "terraform-provider-simple"), data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	simple6Provider := filepath.Join(td, "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
	for _, v := range []string{providerVersionOld, providerVersionNew} {
		dir := filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple6", v, platform)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		// Create an executable copy of the simple6ProviderExe file per version in the cache dir
		data, err := os.ReadFile(simple6ProviderExe)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "terraform-provider-simple6"), data, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Get hashes of 3 of the 4 providers
	// These are used when creating or asserting against lock files.
	var simple5v1_0_0Hash providerreqs.Hash
	var simple6v1_0_0Hash providerreqs.Hash
	var simple5v2_0_0Hash providerreqs.Hash
	var err error
	loc := getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", providerVersionOld, platform))
	if simple5v1_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}
	loc = getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", providerVersionNew, platform))
	if simple5v2_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}
	loc = getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple6", providerVersionOld, platform))
	if simple6v1_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}

	// Set up a reused CLI configuration file that sets simple6 as a dev_override,
	// Tests will use this via the TF_CLI_CONFIG_FILE environment variable.
	cliCfg := fmt.Sprintf(`provider_installation {

  dev_overrides {
    "hashicorp/simple6" = "%s"
  }

  # For all other providers, install them directly from their origin provider
  # registries as normal. If you omit this, Terraform will _only_ use
  # the dev_overrides block, and so no other providers will be available.
  direct {}
}
`, filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple6", providerVersionOld, platform))
	cliConfigFilePath := filepath.Join(td, "dev_override.tfrc")
	if err := os.WriteFile(cliConfigFilePath, []byte(cliCfg), 0644); err != nil {
		t.Fatalf("err: %s", err)
	}

	t.Run("dev_override not installed during init when provider not present in dependency lock file", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// There is no lock file present at start
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if err == nil {
			t.Fatal("expected error due to file not existing, got no error")
		}
		if !os.IsNotExist(err) {
			t.Fatalf("expected error due to file not existing, got different error: %s", err)
		}

		// The simple6 provider is a dev_override
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + cliConfigFilePath)

		// The init process should succeed.
		stdout, stderr, err := tf.Run("init", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lockfile is created
		// simple provider is installed using the latest, 2.0.0 version,
		// but the dev_override simple6 provider is not added to the lockfile.
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)

		expectedLockFileContent := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "2.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v2_0_0Hash,
		)
		if diff := cmp.Diff(expectedLockFileContent, string(buf)); diff != "" {
			t.Errorf("unexpected difference in lock file content: %s", diff)
		}
	})

	t.Run("dev_override causes provider to be removed from dependency lock file during init", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Lockfile contains both simple and simple6 providers already
		priorLockFile := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}

provider "registry.terraform.io/hashicorp/simple6" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v1_0_0Hash,
			simple6v1_0_0Hash,
		)
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		if err := os.WriteFile(lockFile, []byte(priorLockFile), 0644); err != nil {
			t.Fatalf("error writing prior lock file: %s", err)
		}

		// The simple6 provider is a dev_override
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + cliConfigFilePath)

		// The init process should succeed.
		stdout, stderr, err := tf.Run("init", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lockfile has been altered to remove the simple6 provider
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)
		expectedLockFile := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v1_0_0Hash,
		)
		if diff := cmp.Diff(expectedLockFile, string(buf)); diff != "" {
			t.Fatalf("unexpected difference in lock file content: %s", diff)
		}
	})

	t.Run("dev_override also causes provider to be removed from dependency lock file during init -upgrade", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Lockfile contains both simple and simple6 providers already
		priorLockFile := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}

provider "registry.terraform.io/hashicorp/simple6" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v1_0_0Hash,
			simple6v1_0_0Hash,
		)
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		if err := os.WriteFile(lockFile, []byte(priorLockFile), 0644); err != nil {
			t.Fatalf("error writing prior lock file: %s", err)
		}

		// The simple6 provider is a dev_override
		tf.AddEnv("TF_CLI_CONFIG_FILE=" + cliConfigFilePath)

		// The init -upgrade process should succeed.
		stdout, stderr, err := tf.Run("init", "-upgrade", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lockfile shows evidence of upgrade process
		// simple provider is upgraded to the newer 2.0.0 version,
		// but the dev_override simple6 provider is removed from the lockfile.
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)

		expectedLockFileContent := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "2.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v2_0_0Hash,
		)
		if diff := cmp.Diff(expectedLockFileContent, string(buf)); diff != "" {
			t.Errorf("unexpected difference in lock file content: %s", diff)
		}
	})
}

// TestProviderInstall_reattached verifies provider plugin installation behaviour
// when a reattached/unmanaged provider is in use.
func TestProviderInstall_reattached(t *testing.T) {
	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}

	fixturePath := "testdata/provider-plugin" // Reused

	// In temp dir create a plugin cache to be used in the test cases.
	// The cache is supplied to commands using the -plugin-dir init flag.
	// There are 4 providers total:
	// - simple provider with versions 1.0.0 and 2.0.0 available
	// - simple6 provider with versions 1.0.0 and 2.0.0 available
	td := t.TempDir()
	providerVersionOld := "1.0.0"
	providerVersionNew := "2.0.0"
	platform := getproviders.CurrentPlatform.String()
	absolutePathToCache := filepath.Join(td, "cache")
	simple5Provider := filepath.Join(td, "terraform-provider-simple")
	simple5ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple/main", simple5Provider)
	for _, v := range []string{providerVersionOld, providerVersionNew} {
		dir := filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", v, platform)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		// Create an executable copy of the simple5ProviderExe file per version in the cache dir
		data, err := os.ReadFile(simple5ProviderExe)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "terraform-provider-simple"), data, 0755); err != nil {
			t.Fatal(err)
		}
	}
	simple6Provider := filepath.Join(td, "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)
	for _, v := range []string{providerVersionOld, providerVersionNew} {
		dir := filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple6", v, platform)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		// Create an executable copy of the simple6ProviderExe file per version in the cache dir
		data, err := os.ReadFile(simple6ProviderExe)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "terraform-provider-simple6"), data, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Get hashes of 3 of the 4 providers
	// These are used when creating or asserting against lock files.
	var simple5v1_0_0Hash providerreqs.Hash
	var simple6v1_0_0Hash providerreqs.Hash
	var simple5v2_0_0Hash providerreqs.Hash
	var err error
	loc := getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", providerVersionOld, platform))
	if simple5v1_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}
	loc = getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple", providerVersionNew, platform))
	if simple5v2_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}
	loc = getproviders.PackageLocalDir(filepath.Join(absolutePathToCache, "registry.terraform.io/hashicorp", "simple6", providerVersionOld, platform))
	if simple6v1_0_0Hash, err = getproviders.PackageHash(loc); err != nil {
		t.Fatal(err)
	}

	// Launch a separate simple6 provider process to be re-used as a reattached provider.
	// Tests will use this via the TF_REATTACH_PROVIDERS environment variable.
	reattachConfig := reattachedProviderForTest(t, addrs.NewDefaultProvider("simple6"), 6)

	t.Run("reattached provider not installed when provider not present in dependency lock file", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// There is no lock file present at start
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		_, err := os.Stat(lockFile)
		if err == nil {
			t.Fatal("expected error due to file not existing, got no error")
		}
		if !os.IsNotExist(err) {
			t.Fatalf("expected error due to file not existing, got different error: %s", err)
		}

		// The simple6 provider is reattached/unmanaged
		tf.AddEnv("TF_REATTACH_PROVIDERS=" + reattachConfig)

		// The init process should succeed.
		stdout, stderr, err := tf.Run("init", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lock file should have been created
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)

		// We expect the lock file to not contain the simple6 provider that's being reattached/unmanaged,
		// because that provider is skipped during the installation process.
		// The simple (v5) provider is installed as usual, pulling in the latest version.
		expectedLockFileContent := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "2.0.0"
  hashes = [
    "%s",
  ]
}`, simple5v2_0_0Hash)

		if diff := cmp.Diff(expectedLockFileContent, string(buf)); diff != "" {
			t.Errorf("unexpected difference in lock file content: %s", diff)
		}
	})

	t.Run("reattached providers do NOT cause provider to be removed from dependency lock file during init", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Lockfile contains both simple and simple6 providers already
		priorLockFile := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}

provider "registry.terraform.io/hashicorp/simple6" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v1_0_0Hash,
			simple6v1_0_0Hash,
		)
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		if err := os.WriteFile(lockFile, []byte(priorLockFile), 0644); err != nil {
			t.Fatalf("error writing prior lock file: %s", err)
		}

		// The simple6 provider is reattached/unmanaged
		tf.AddEnv("TF_REATTACH_PROVIDERS=" + reattachConfig)

		// The init process should succeed.
		stdout, stderr, err := tf.Run("init", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lockfile is unchanged despite use of a reattached/unmanaged simple6 provider
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)
		if diff := cmp.Diff(priorLockFile, string(buf)); diff != "" {
			t.Fatalf("unexpected difference in lock file content: %s", diff)
		}
	})

	t.Run("reattached providers are unchanged in the dependency lock file during init -upgrade", func(t *testing.T) {
		terraformBin := e2e.GoBuild("github.com/hashicorp/terraform", "terraform")
		tf := e2e.NewBinary(t, terraformBin, fixturePath)

		// Lockfile contains both simple and simple6 providers already at older version 1.0.0
		priorLockFile := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}

provider "registry.terraform.io/hashicorp/simple6" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v1_0_0Hash,
			simple6v1_0_0Hash,
		)
		lockFile := filepath.Join(tf.WorkDir(), ".terraform.lock.hcl")
		if err := os.WriteFile(lockFile, []byte(priorLockFile), 0644); err != nil {
			t.Fatalf("error writing prior lock file: %s", err)
		}

		// The simple6 provider is reattached/unmanaged
		tf.AddEnv("TF_REATTACH_PROVIDERS=" + reattachConfig)

		// The init -upgrade process should succeed.
		stdout, stderr, err := tf.Run("init", "-upgrade", fmt.Sprintf("-plugin-dir=%s", absolutePathToCache))
		if err != nil {
			t.Fatalf("unexpected error: %s\nstdout: %s\nstderr: %s", err, stdout, stderr)
		}

		// Lockfile shows evidence of upgrade process
		// simple provider is upgraded to the newer 2.0.0 version,
		// but the reattached simple6 provider is unchanged due to being reattached.
		buf, err := os.ReadFile(lockFile)
		if err != nil {
			t.Fatalf("unexpected error accessing lock file: %s", err)
		}
		buf = bytes.TrimSpace(buf)

		expectedLockFileContent := fmt.Sprintf(`# This file is maintained automatically by "terraform init".
# Manual edits may be lost in future updates.

provider "registry.terraform.io/hashicorp/simple" {
  version = "2.0.0"
  hashes = [
    "%s",
  ]
}

provider "registry.terraform.io/hashicorp/simple6" {
  version = "1.0.0"
  hashes = [
    "%s",
  ]
}`,
			simple5v2_0_0Hash,
			simple6v1_0_0Hash,
		)
		if diff := cmp.Diff(expectedLockFileContent, string(buf)); diff != "" {
			t.Errorf("unexpected difference in lock file content: %s", diff)
		}
	})
}

// reattachedProviderForTest launches a provider process and returns a reattach config string
// that can be used as the value for the TF_REATTACH_PROVIDERS environment variable in tests.
// Cleanup of the provider process is handled internally.
func reattachedProviderForTest(t *testing.T, provider addrs.Provider, protocol int) string {
	t.Helper()
	if !slices.Contains([]int{5, 6}, protocol) {
		t.Fatalf("test setup tried to create a provider using protocol version %d, which is unsupported. Choose between 5 and 6.", protocol)
	}

	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})
	server := &providerServer{
		ProviderServer: grpcwrap.Provider6(simple.Provider()),
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go plugin.Serve(&plugin.ServeConfig{
		Logger: hclog.New(&hclog.LoggerOptions{
			Name:   "plugintest",
			Level:  hclog.Trace,
			Output: io.Discard,
		}),
		Test: &plugin.ServeTestConfig{
			Context:          ctx,
			ReattachConfigCh: reattachCh,
			CloseCh:          closeCh,
		},
		GRPCServer: plugin.DefaultGRPCServer,
		VersionedPlugins: map[int]plugin.PluginSet{
			6: {
				"provider": &tfplugin.GRPCProviderPlugin{
					GRPCProvider: func() proto.ProviderServer {
						return server
					},
				},
			},
		},
	})
	config := <-reattachCh
	if config == nil {
		t.Fatalf("no reattach config received")
	}
	reattachStr, err := json.Marshal(map[string]reattachConfig{
		provider.String(): {
			Protocol:        string(config.Protocol),
			ProtocolVersion: 6,
			Pid:             config.Pid,
			Test:            true,
			Addr: reattachConfigAddr{
				Network: config.Addr.Network(),
				String:  config.Addr.String(),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(reattachStr)
}

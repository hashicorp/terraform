package e2etest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/e2e"
)

func TestInitProviders(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template provider, so it can only run if network access is allowed.
	// We intentionally don't try to stub this here, because there's already
	// a stubbed version of this in the "command" package and so the goal here
	// is to test the interaction with the real repository.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "template-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "Terraform has been successfully initialized!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, "- Downloading plugin for provider \"template\" (terraform-providers/template)") {
		t.Errorf("provider download message is missing from output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	if !strings.Contains(stdout, "* provider.template: version = ") {
		t.Errorf("provider pinning recommendation is missing from output:\n%s", stdout)
	}

}

func TestInitProvidersInternal(t *testing.T) {
	t.Parallel()

	// This test should _not_ reach out anywhere because the "terraform"
	// provider is internal to the core terraform binary.

	fixturePath := filepath.Join("test-fixtures", "terraform-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "Terraform has been successfully initialized!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if strings.Contains(stdout, "Downloading plugin for provider") {
		// Shouldn't have downloaded anything with this config, because the
		// provider is built in.
		t.Errorf("provider download message appeared in output:\n%s", stdout)
	}

}

func TestInitProviders_pluginCache(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to access plugin
	// metadata, and download the null plugin, though the template plugin
	// should come from local cache.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "plugin-cache")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// Our fixture dir has a generic os_arch dir, which we need to customize
	// to the actual OS/arch where this test is running in order to get the
	// desired result.
	fixtMachineDir := tf.Path("cache/os_arch")
	wantMachineDir := tf.Path("cache", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	os.Rename(fixtMachineDir, wantMachineDir)

	cmd := tf.Cmd("init")
	cmd.Env = append(cmd.Env, "TF_PLUGIN_CACHE_DIR=./cache")
	cmd.Stdin = nil
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	stderr := cmd.Stderr.(*bytes.Buffer).String()
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s\n", stderr)
	}

	path := fmt.Sprintf(".terraform/plugins/%s_%s/terraform-provider-template_v2.1.0_x4", runtime.GOOS, runtime.GOARCH)
	content, err := tf.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read installed plugin from %s: %s", path, err)
	}
	if strings.TrimSpace(string(content)) != "this is not a real plugin" {
		t.Errorf("template plugin was not installed from local cache")
	}

	if !tf.FileExists(fmt.Sprintf(".terraform/plugins/%s_%s/terraform-provider-null_v2.1.0_x4", runtime.GOOS, runtime.GOARCH)) {
		t.Errorf("null plugin was not installed")
	}

	if !tf.FileExists(fmt.Sprintf("cache/%s_%s/terraform-provider-null_v2.1.0_x4", runtime.GOOS, runtime.GOARCH)) {
		t.Errorf("null plugin is not in cache after install")
	}
}

func TestInit_fromModule(t *testing.T) {
	t.Parallel()

	// This test reaches out to registry.terraform.io and github.com to lookup
	// and fetch a module.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "empty")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	cmd := tf.Cmd("init", "-from-module=hashicorp/vault/aws")
	cmd.Stdin = nil
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	stderr := cmd.Stderr.(*bytes.Buffer).String()
	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	content, err := tf.ReadFile("main.tf")
	if err != nil {
		t.Fatalf("failed to read main.tf: %s", err)
	}
	if !bytes.Contains(content, []byte("vault")) {
		t.Fatalf("main.tf doesn't appear to be a vault configuration: \n%s", content)
	}
}

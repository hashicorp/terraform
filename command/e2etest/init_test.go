package e2etest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestInitProviders(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template provider, so it can only run if network access is allowed.
	// We intentionally don't try to stub this here, because there's already
	// a stubbed version of this in the "command" package and so the goal here
	// is to test the interaction with the real repository.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "template-provider")
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

	if !strings.Contains(stdout, "- Installing hashicorp/template v") {
		t.Errorf("provider download message is missing from output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	if !strings.Contains(stdout, "Terraform has created a lock file") {
		t.Errorf("lock file notification is missing from output:\n%s", stdout)
	}

}

func TestInitProvidersInternal(t *testing.T) {
	t.Parallel()

	// This test should _not_ reach out anywhere because the "terraform"
	// provider is internal to the core terraform binary.

	fixturePath := filepath.Join("testdata", "terraform-provider")
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

	if strings.Contains(stdout, "Installing hashicorp/terraform") {
		// Shouldn't have downloaded anything with this config, because the
		// provider is built in.
		t.Errorf("provider download message appeared in output:\n%s", stdout)
	}

	if strings.Contains(stdout, "Installing terraform.io/builtin/terraform") {
		// Shouldn't have downloaded anything with this config, because the
		// provider is built in.
		t.Errorf("provider download message appeared in output:\n%s", stdout)
	}
}

func TestInitProvidersVendored(t *testing.T) {
	t.Parallel()

	// This test will try to reach out to registry.terraform.io as one of the
	// possible installation locations for
	// hashicorp/null, where it will find that
	// versions do exist but will ultimately select the version that is
	// vendored due to the version constraint.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "vendored-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// Our fixture dir has a generic os_arch dir, which we need to customize
	// to the actual OS/arch where this test is running in order to get the
	// desired result.
	fixtMachineDir := tf.Path("terraform.d/plugins/registry.terraform.io/hashicorp/null/1.0.0+local/os_arch")
	wantMachineDir := tf.Path("terraform.d/plugins/registry.terraform.io/hashicorp/null/1.0.0+local/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	err := os.Rename(fixtMachineDir, wantMachineDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

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

	if !strings.Contains(stdout, "- Installing hashicorp/null v1.0.0+local") {
		t.Errorf("provider download message is missing from output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

}

func TestInitProvidersLocalOnly(t *testing.T) {
	t.Parallel()

	// This test should not reach out to the network if it is behaving as
	// intended. If it _does_ try to access an upstream registry and encounter
	// an error doing so then that's a legitimate test failure that should be
	// fixed. (If it incorrectly reaches out anywhere then it's likely to be
	// to the host "example.com", which is the placeholder domain we use in
	// the test fixture.)

	fixturePath := filepath.Join("testdata", "local-only-provider")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	// If you run this test on a workstation with a plugin-cache directory
	// configured, it will leave a bad directory behind and terraform init will
	// not work until you remove it.
	//
	// To avoid this, we will  "zero out" any existing cli config file.
	tf.AddEnv("TF_CLI_CONFIG_FILE=\"\"")
	defer tf.Close()

	// Our fixture dir has a generic os_arch dir, which we need to customize
	// to the actual OS/arch where this test is running in order to get the
	// desired result.
	fixtMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/os_arch")
	wantMachineDir := tf.Path("terraform.d/plugins/example.com/awesomecorp/happycloud/1.2.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	err := os.Rename(fixtMachineDir, wantMachineDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

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

	if !strings.Contains(stdout, "- Installing example.com/awesomecorp/happycloud v1.2.0") {
		t.Errorf("provider download message is missing from output:\n%s", stdout)
		t.Logf("(this can happen if you have a conflicting copy of the plugin in one of the global plugin search dirs)")
	}
}

func TestInitProvidersCustomMethod(t *testing.T) {
	t.Parallel()

	// This test should not reach out to the network if it is behaving as
	// intended. If it _does_ try to access an upstream registry and encounter
	// an error doing so then that's a legitimate test failure that should be
	// fixed. (If it incorrectly reaches out anywhere then it's likely to be
	// to the host "example.com", which is the placeholder domain we use in
	// the test fixture.)

	for _, configFile := range []string{"cliconfig.tfrc", "cliconfig.tfrc.json"} {
		t.Run(configFile, func(t *testing.T) {
			fixturePath := filepath.Join("testdata", "custom-provider-install-method")
			tf := e2e.NewBinary(terraformBin, fixturePath)
			defer tf.Close()

			// Our fixture dir has a generic os_arch dir, which we need to customize
			// to the actual OS/arch where this test is running in order to get the
			// desired result.
			fixtMachineDir := tf.Path("fs-mirror/example.com/awesomecorp/happycloud/1.2.0/os_arch")
			wantMachineDir := tf.Path("fs-mirror/example.com/awesomecorp/happycloud/1.2.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
			err := os.Rename(fixtMachineDir, wantMachineDir)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			// We'll use a local CLI configuration file taken from our fixture
			// directory so we can force a custom installation method config.
			tf.AddEnv("TF_CLI_CONFIG_FILE=" + tf.Path(configFile))

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

			if !strings.Contains(stdout, "- Installing example.com/awesomecorp/happycloud v1.2.0") {
				t.Errorf("provider download message is missing from output:\n%s", stdout)
			}
		})
	}
}

func TestInitProviders_pluginCache(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to access plugin
	// metadata, and download the null plugin, though the template plugin
	// should come from local cache.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "plugin-cache")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	// Our fixture dir has a generic os_arch dir, which we need to customize
	// to the actual OS/arch where this test is running in order to get the
	// desired result.
	fixtMachineDir := tf.Path("cache/registry.terraform.io/hashicorp/template/2.1.0/os_arch")
	wantMachineDir := tf.Path("cache/registry.terraform.io/hashicorp/template/2.1.0/", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
	err := os.Rename(fixtMachineDir, wantMachineDir)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	cmd := tf.Cmd("init")

	// convert the slashes if building for windows.
	p := filepath.FromSlash("./cache")
	cmd.Env = append(cmd.Env, "TF_PLUGIN_CACHE_DIR="+p)
	err = cmd.Run()
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	path := filepath.FromSlash(fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/template/2.1.0/%s_%s/terraform-provider-template_v2.1.0_x4", runtime.GOOS, runtime.GOARCH))
	content, err := tf.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read installed plugin from %s: %s", path, err)
	}
	if strings.TrimSpace(string(content)) != "this is not a real plugin" {
		t.Errorf("template plugin was not installed from local cache")
	}

	nullLinkPath := filepath.FromSlash(fmt.Sprintf(".terraform/providers/registry.terraform.io/hashicorp/null/2.1.0/%s_%s/terraform-provider-null_v2.1.0_x4", runtime.GOOS, runtime.GOARCH))
	if runtime.GOOS == "windows" {
		nullLinkPath = nullLinkPath + ".exe"
	}
	if !tf.FileExists(nullLinkPath) {
		t.Errorf("null plugin was not installed into %s", nullLinkPath)
	}

	nullCachePath := filepath.FromSlash(fmt.Sprintf("cache/registry.terraform.io/hashicorp/null/2.1.0/%s_%s/terraform-provider-null_v2.1.0_x4", runtime.GOOS, runtime.GOARCH))
	if runtime.GOOS == "windows" {
		nullCachePath = nullCachePath + ".exe"
	}
	if !tf.FileExists(nullCachePath) {
		t.Errorf("null plugin is not in cache after install. expected in: %s", nullCachePath)
	}
}

func TestInit_fromModule(t *testing.T) {
	t.Parallel()

	// This test reaches out to registry.terraform.io and github.com to lookup
	// and fetch a module.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "empty")
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

func TestInitProviderNotFound(t *testing.T) {
	t.Parallel()

	// This test will reach out to registry.terraform.io as one of the possible
	// installation locations for hashicorp/nonexist, which should not exist.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "provider-not-found")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	t.Run("registry provider not found", func(t *testing.T) {
		_, stderr, err := tf.Run("init", "-no-color")
		if err == nil {
			t.Fatal("expected error, got success")
		}

		oneLineStderr := strings.ReplaceAll(stderr, "\n", " ")
		if !strings.Contains(oneLineStderr, "provider registry registry.terraform.io does not have a provider named registry.terraform.io/hashicorp/nonexist") {
			t.Errorf("expected error message is missing from output:\n%s", stderr)
		}

		if !strings.Contains(oneLineStderr, "All modules should specify their required_providers") {
			t.Errorf("expected error message is missing from output:\n%s", stderr)
		}
	})

	t.Run("local provider not found", func(t *testing.T) {
		// The -plugin-dir directory must exist for the provider installer to search it.
		pluginDir := tf.Path("empty")
		if err := os.Mkdir(pluginDir, os.ModePerm); err != nil {
			t.Fatal(err)
		}

		_, stderr, err := tf.Run("init", "-no-color", "-plugin-dir="+pluginDir)
		if err == nil {
			t.Fatal("expected error, got success")
		}

		if !strings.Contains(stderr, "provider registry.terraform.io/hashicorp/nonexist was not\nfound in any of the search locations\n\n  - "+pluginDir) {
			t.Errorf("expected error message is missing from output:\n%s", stderr)
		}
	})

	t.Run("special characters enabled", func(t *testing.T) {
		_, stderr, err := tf.Run("init")
		if err == nil {
			t.Fatal("expected error, got success")
		}

		expectedErr := `╷
│ Error: Failed to query available provider packages
│` + ` ` + `
│ Could not retrieve the list of available versions for provider
│ hashicorp/nonexist: provider registry registry.terraform.io does not have a
│ provider named registry.terraform.io/hashicorp/nonexist
│ 
│ All modules should specify their required_providers so that external
│ consumers will get the correct providers when using a module. To see which
│ modules are currently depending on hashicorp/nonexist, run the following
│ command:
│     terraform providers
╵

`
		if stripAnsi(stderr) != expectedErr {
			t.Errorf("wrong output:\n%s", cmp.Diff(stripAnsi(stderr), expectedErr))
		}
	})
}

func TestInitProviderWarnings(t *testing.T) {
	t.Parallel()

	// This test will reach out to registry.terraform.io as one of the possible
	// installation locations for hashicorp/nonexist, which should not exist.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "provider-warnings")
	tf := e2e.NewBinary(terraformBin, fixturePath)
	defer tf.Close()

	stdout, _, err := tf.Run("init")
	if err == nil {
		t.Fatal("expected error, got success")
	}

	if !strings.Contains(stdout, "This provider is archived and no longer needed.") {
		t.Errorf("expected warning message is missing from output:\n%s", stdout)
	}

}

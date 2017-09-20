package e2etest

import (
	"path/filepath"
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

	if !strings.Contains(stdout, "- Downloading plugin for provider \"template\"") {
		t.Errorf("provider download message is missing from output:\n%s", stdout)
		t.Logf("(this can happen if you have a copy of the plugin in one of the global plugin search dirs)")
	}

	if !strings.Contains(stdout, "* provider.template: version = ") {
		t.Errorf("provider pinning recommendation is missing from output:\n%s", stdout)
	}

}

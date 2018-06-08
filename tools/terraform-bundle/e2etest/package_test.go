package e2etest

import (
	"path/filepath"
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

	fixturePath := filepath.Join("test-fixtures", "empty")
	tfBundle := e2e.NewBinary(bundleBin, fixturePath)
	defer tfBundle.Close()

	stdout, stderr, err := tfBundle.Run("package", "terraform-bundle.hcl")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "Fetching Terraform 0.10.1 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.10.1-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
}

func TestPackage_manyProviders(t *testing.T) {
	t.Parallel()

	// This test reaches out to releases.hashicorp.com to download the
	// template provider, so it can only run if network access is allowed.
	// We intentionally don't try to stub this here, because there's already
	// a stubbed version of this in the "command" package and so the goal here
	// is to test the interaction with the real repository.
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("test-fixtures", "many-providers")
	tfBundle := e2e.NewBinary(bundleBin, fixturePath)
	defer tfBundle.Close()

	stdout, stderr, err := tfBundle.Run("package", "terraform-bundle.hcl")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "Checking for available provider plugins on ") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	// Here we have to check each provider separately
	// because it's internally held in a map (i.e. not guaranteed order)

	if !strings.Contains(stdout, `- Resolving "aws" provider (~> 0.1)...
- Downloading plugin for provider "aws" (0.1.4)...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Resolving "kubernetes" provider (0.1.0)...
- Downloading plugin for provider "kubernetes" (0.1.0)...
- Resolving "kubernetes" provider (0.1.1)...
- Downloading plugin for provider "kubernetes" (0.1.1)...
- Resolving "kubernetes" provider (0.1.2)...
- Downloading plugin for provider "kubernetes" (0.1.2)...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Resolving "null" provider (0.1.0)...
- Downloading plugin for provider "null" (0.1.0)...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, "Fetching Terraform 0.10.1 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.10.1-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
}

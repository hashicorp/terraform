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

	if !strings.Contains(stdout, "Fetching Terraform 0.12.0 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.12.0-bundle") {
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

	if !strings.Contains(stdout, `- Resolving "aws" provider (~> 2.26.0)...
- Checking for provider plugin on https://releases.hashicorp.com...
- Downloading plugin for provider "aws" (hashicorp/aws) 2.26.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Resolving "kubernetes" provider (1.8.0)...
- Checking for provider plugin on https://releases.hashicorp.com...
- Downloading plugin for provider "kubernetes" (hashicorp/kubernetes) 1.8.0...
- Resolving "kubernetes" provider (1.8.1)...
- Checking for provider plugin on https://releases.hashicorp.com...
- Downloading plugin for provider "kubernetes" (hashicorp/kubernetes) 1.8.1...
- Resolving "kubernetes" provider (1.9.0)...
- Checking for provider plugin on https://releases.hashicorp.com...
- Downloading plugin for provider "kubernetes" (hashicorp/kubernetes) 1.9.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, `- Resolving "null" provider (2.1.0)...
- Checking for provider plugin on https://releases.hashicorp.com...
- Downloading plugin for provider "null" (hashicorp/null) 2.1.0...`) {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}

	if !strings.Contains(stdout, "Fetching Terraform 0.12.0 core package...") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Creating terraform_0.12.0-bundle") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "All done!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
}

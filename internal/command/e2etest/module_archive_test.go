// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package e2etest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestInitModuleArchive(t *testing.T) {
	t.Parallel()

	// this fetches a module archive from github
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "module-archive")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

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
}

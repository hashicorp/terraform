// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/mnptu/internal/e2e"
)

func TestInitModuleArchive(t *testing.T) {
	t.Parallel()

	// this fetches a module archive from github
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "module-archive")
	tf := e2e.NewBinary(t, mnptuBin, fixturePath)

	stdout, stderr, err := tf.Run("init")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if stderr != "" {
		t.Errorf("unexpected stderr output:\n%s", stderr)
	}

	if !strings.Contains(stdout, "mnptu has been successfully initialized!") {
		t.Errorf("success message is missing from output:\n%s", stdout)
	}
}

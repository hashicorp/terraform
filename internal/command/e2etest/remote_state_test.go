// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"path/filepath"
	"testing"

	"github.com/hashicorp/mnptu/internal/e2e"
)

func TestmnptuProviderRead(t *testing.T) {
	// Ensure the mnptu provider can correctly read a remote state

	t.Parallel()
	fixturePath := filepath.Join("testdata", "mnptu-provider")
	tf := e2e.NewBinary(t, mnptuBin, fixturePath)

	//// INIT
	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	//// PLAN
	_, stderr, err = tf.Run("plan")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}
}

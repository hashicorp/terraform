// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestForget(t *testing.T) {
	skipIfCannotAccessNetwork(t)

	fixturePath := filepath.Join("testdata", "forget-lifecycle")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	// zero out any existing cli config file by passing in an empty file.
	configFile := emptyConfigFileForTests(t, tf.WorkDir())
	tf.AddEnv(fmt.Sprintf("TF_CLI_CONFIG_FILE=%s", configFile))

	_, stderr, err := tf.Run("init")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	stdout, stderr, err := tf.Run("apply", "--auto-approve")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "Apply complete! Resources: 2 added, 0 changed, 0 destroyed.") {
		t.Errorf("missing expected output:\n%s", stdout)
	}

	stdout, stderr, err = tf.Run("destroy", "--auto-approve")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "Destroy complete! Resources: 0 destroyed.") {
		t.Errorf("missing expected output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "will no longer be managed by Terraform") {
		t.Fatalf("missing forget message from output")
	}

	// recreate
	_, stderr, err = tf.Run("apply", "--auto-approve")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	// taint
	_, stderr, err = tf.Run("taint", "module.child.random_pet.child")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	stdout, stderr, err = tf.Run("apply", "--auto-approve")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}
	if !strings.Contains(stdout, "Apply complete! Resources: 1 added, 0 changed, 0 destroyed.") {
		t.Errorf("missing expected output:\n%s", stdout)
	}
	if !strings.Contains(stdout, "will no longer be managed by Terraform") {
		t.Fatalf("missing forget message from output")
	}
}

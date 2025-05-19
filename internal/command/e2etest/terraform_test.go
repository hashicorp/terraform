// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

func TestTerraformProviderData(t *testing.T) {

	fixturePath := filepath.Join("testdata", "terraform-managed-data")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	_, stderr, err := tf.Run("init", "-input=false")
	if err != nil {
		t.Fatalf("unexpected init error: %s\nstderr:\n%s", err, stderr)
	}

	stdout, stderr, err := tf.Run("plan", "-out=tfplan", "-input=false")
	if err != nil {
		t.Fatalf("unexpected plan error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "4 to add, 0 to change, 0 to destroy") {
		t.Errorf("incorrect plan tally; want 4 to add:\n%s", stdout)
	}

	stdout, stderr, err = tf.Run("apply", "-input=false", "tfplan")
	if err != nil {
		t.Fatalf("unexpected apply error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, "Resources: 4 added, 0 changed, 0 destroyed") {
		t.Errorf("incorrect apply tally; want 4 added:\n%s", stdout)
	}

	state, err := tf.LocalState()
	if err != nil {
		t.Fatalf("failed to read state file: %s", err)
	}

	// we'll check the final output to validate the resources
	d := state.RootOutputValues["d"].Value
	input := d.GetAttr("input")
	output := d.GetAttr("output")
	if input.IsNull() {
		t.Fatal("missing input from resource d")
	}
	if !input.RawEquals(output) {
		t.Fatalf("input %#v does not equal output %#v\n", input, output)
	}
}

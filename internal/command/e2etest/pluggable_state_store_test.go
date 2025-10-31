// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestPrimary_stateStore_workspaceCmd(t *testing.T) {
	if v := os.Getenv("TF_TEST_EXPERIMENTS"); v == "" {
		t.Skip("can't run without enabling experiments in the executable terraform binary, enable with TF_TEST_EXPERIMENTS=1")
	}

	if !canRunGoBuild {
		// We're running in a separate-build-then-run context, so we can't
		// currently execute this test which depends on being able to build
		// new executable at runtime.
		//
		// (See the comment on canRunGoBuild's declaration for more information.)
		t.Skip("can't run without building a new provider executable")
	}
	t.Parallel()

	tf := e2e.NewBinary(t, terraformBin, "testdata/full-workflow-with-state-store-fs")

	// In order to test integration with PSS we need a provider plugin implementing a state store.
	// Here will build the simple6 (built with protocol v6) provider, which implements PSS.
	simple6Provider := filepath.Join(tf.WorkDir(), "terraform-provider-simple6")
	simple6ProviderExe := e2e.GoBuild("github.com/hashicorp/terraform/internal/provider-simple-v6/main", simple6Provider)

	// Move the provider binaries into a directory that we will point terraform
	// to using the -plugin-dir cli flag.
	platform := getproviders.CurrentPlatform.String()
	hashiDir := "cache/registry.terraform.io/hashicorp/"
	if err := os.MkdirAll(tf.Path(hashiDir, "simple6/0.0.1/", platform), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(simple6ProviderExe, tf.Path(hashiDir, "simple6/0.0.1/", platform, "terraform-provider-simple6")); err != nil {
		t.Fatal(err)
	}

	//// Init
	_, stderr, err := tf.Run("init", "-enable-pluggable-state-storage-experiment=true", "-plugin-dir=cache", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	//// Create Workspace
	newWorkspace := "foobar"
	stdout, stderr, err := tf.Run("workspace", "new", newWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	expectedMsg := fmt.Sprintf("Created and switched to workspace %q!", newWorkspace)
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}

	//// List Workspaces
	stdout, stderr, err = tf.Run("workspace", "list", "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	if !strings.Contains(stdout, newWorkspace) {
		t.Errorf("unexpected output, expected the new %q workspace to be listed present, but it's missing. Got:\n%s", newWorkspace, stdout)
	}

	//// Select Workspace
	selectedWorkspace := "default"
	stdout, stderr, err = tf.Run("workspace", "select", selectedWorkspace, "-no-color")
	if err != nil {
		t.Fatalf("unexpected error: %s\nstderr:\n%s", err, stderr)
	}

	expectedMsg = fmt.Sprintf("Switched to workspace %q.", selectedWorkspace)
	if !strings.Contains(stdout, expectedMsg) {
		t.Errorf("unexpected output, expected %q, but got:\n%s", expectedMsg, stdout)
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/internal/backend/local"
)

// DefaultDataDir is the default directory for storing local data.
const DefaultDataDir = ".terraform"

func TestMeta_Workspace_override(t *testing.T) {
	defer func(value string) {
		os.Setenv(WorkspaceNameEnvVar, value)
	}(os.Getenv(WorkspaceNameEnvVar))

	m := new(Meta)

	testCases := map[string]struct {
		workspace string
		err       error
	}{
		"": {
			"default",
			nil,
		},
		"development": {
			"development",
			nil,
		},
		"invalid name": {
			"",
			errInvalidWorkspaceNameEnvVar,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			os.Setenv(WorkspaceNameEnvVar, name)
			workspace, err := m.Workspace()
			if workspace != tc.workspace {
				t.Errorf("Unexpected workspace\n got: %s\nwant: %s\n", workspace, tc.workspace)
			}
			if err != tc.err {
				t.Errorf("Unexpected error\n got: %s\nwant: %s\n", err, tc.err)
			}
		})
	}
}

// If somehow an invalid workspace has been selected, the Meta.Workspace
// method should not return an error, to ensure that we don't break any
// existing workflows with invalid workspace names.
func TestMeta_Workspace_invalidSelected(t *testing.T) {
	td := t.TempDir()
	defer testChdir(t, td)()

	// this is an invalid workspace name
	workspace := "test workspace"

	// create the workspace directories
	if err := os.MkdirAll(filepath.Join(local.DefaultWorkspaceDir, workspace), 0755); err != nil {
		t.Fatal(err)
	}

	// create the workspace file to select it
	if err := os.MkdirAll(DefaultDataDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(DefaultDataDir, local.DefaultWorkspaceFile), []byte(workspace), 0644); err != nil {
		t.Fatal(err)
	}

	m := new(Meta)

	ws, err := m.Workspace()
	if ws != workspace {
		t.Errorf("Unexpected workspace\n got: %s\nwant: %s\n", ws, workspace)
	}
	if err != nil {
		t.Errorf("Unexpected error: %s", err)
	}
}

// testChdir changes the directory and returns a function to defer to
// revert the old cwd.
func testChdir(t *testing.T, new string) func() {
	t.Helper()

	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := os.Chdir(new); err != nil {
		t.Fatalf("err: %v", err)
	}

	return func() {
		// Re-run the function ignoring the defer result
		testChdir(t, old)
	}
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackmigrate

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/workdir"
)

type Meta struct {
	// WorkingDir is an object representing the "working directory" where we're
	// running commands. In the normal case this literally refers to the
	// working directory of the Terraform process, though this can take on
	// a more symbolic meaning when the user has overridden default behavior
	// to specify a different working directory or to override the special
	// data directory where we'll persist settings that must survive between
	// consecutive commands.
	WorkingDir *workdir.Dir
}

var errInvalidWorkspaceNameEnvVar = fmt.Errorf("Invalid workspace name set using %s", WorkspaceNameEnvVar)

// Workspace returns the name of the currently configured workspace, corresponding
// to the desired named state.
func (m *Meta) Workspace() (string, error) {
	current, overridden := m.WorkspaceOverridden()
	if overridden && !validWorkspaceName(current) {
		return "", errInvalidWorkspaceNameEnvVar
	}
	return current, nil
}

// WorkspaceOverridden returns the name of the currently configured workspace,
// corresponding to the desired named state, as well as a bool saying whether
// this was set via the TF_WORKSPACE environment variable.
func (m *Meta) WorkspaceOverridden() (string, bool) {
	if envVar := os.Getenv(WorkspaceNameEnvVar); envVar != "" {
		return envVar, true
	}

	envData, err := ioutil.ReadFile(filepath.Join(m.DataDir(), local.DefaultWorkspaceFile))
	current := string(bytes.TrimSpace(envData))
	if current == "" {
		current = backend.DefaultStateName
	}

	if err != nil && !os.IsNotExist(err) {
		// always return the default if we can't get a workspace name
		log.Printf("[ERROR] failed to read current workspace: %s", err)
	}

	return current, false
}

// fixupMissingWorkingDir is a compensation for various existing tests which
// directly construct incomplete "Meta" objects. Specifically, it deals with
// a test that omits a WorkingDir value by constructing one just-in-time.
//
// We shouldn't ever rely on this in any real codepath, because it doesn't
// take into account the various ways users can override our default
// directory selection behaviors.
func (m *Meta) fixupMissingWorkingDir() {
	if m.WorkingDir == nil {
		log.Printf("[WARN] This 'Meta' object is missing its WorkingDir, so we're creating a default one suitable only for tests")
		m.WorkingDir = workdir.NewDir(".")
	}
}

// DataDir returns the directory where local data will be stored.
// Defaults to DefaultDataDir in the current working directory.
func (m *Meta) DataDir() string {
	m.fixupMissingWorkingDir()
	return m.WorkingDir.DataDir()
}

// validWorkspaceName returns true is this name is valid to use as a workspace name.
// Since most named states are accessed via a filesystem path or URL, check if
// escaping the name would be required.
func validWorkspaceName(name string) bool {
	return name == url.PathEscape(name)
}

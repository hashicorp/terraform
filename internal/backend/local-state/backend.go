// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package local_state

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const (
	DefaultWorkspaceDir    = "terraform.tfstate.d"
	DefaultWorkspaceFile   = "environment"
	DefaultStateFilename   = "terraform.tfstate"
	DefaultBackupExtension = ".backup"
)

type Local struct {
	// The State* paths are set from the backend config, and may be left blank
	// to use the defaults. If the actual paths for the local backend state are
	// needed, use the StatePaths method.
	//
	// StatePath is the local path where state is read from.
	//
	// StateOutPath is the local path where the state will be written.
	// If this is empty, it will default to StatePath.
	//
	// StateBackupPath is the local path where a backup file will be written.
	// Set this to "-" to disable state backup.
	//
	// StateWorkspaceDir is the path to the folder containing data for
	// non-default workspaces. This defaults to DefaultWorkspaceDir if not set.
	StatePath         string
	StateOutPath      string
	StateBackupPath   string
	StateWorkspaceDir string

	// The OverrideState* paths are set based on per-operation CLI arguments
	// and will override what'd be built from the State* fields if non-empty.
	// While the interpretation of the State* fields depends on the active
	// workspace, the OverrideState* fields are always used literally.
	OverrideStatePath       string
	OverrideStateOutPath    string
	OverrideStateBackupPath string

	// We only want to create a single instance of a local state, so store them
	// here as they're loaded.
	states map[string]statemgr.Full
}

var _ backend.Backend = (*Local)(nil)

func New() *Local {
	return &Local{}
}

func (b *Local) ConfigSchema() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"path": {
				Type:     cty.String,
				Optional: true,
			},
			"workspace_dir": {
				Type:     cty.String,
				Optional: true,
			},
		},
	}
}

func (b *Local) PrepareConfig(obj cty.Value) (cty.Value, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	if val := obj.GetAttr("path"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local state file path",
				`The "path" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "path"}},
			))
		}
	}

	if val := obj.GetAttr("workspace_dir"); !val.IsNull() {
		p := val.AsString()
		if p == "" {
			diags = diags.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Invalid local workspace directory path",
				`The "workspace_dir" attribute value must not be empty.`,
				cty.Path{cty.GetAttrStep{Name: "workspace_dir"}},
			))
		}
	}

	return obj, diags
}

func (b *Local) Configure(obj cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if val := obj.GetAttr("path"); !val.IsNull() {
		p := val.AsString()
		b.StatePath = p
		b.StateOutPath = p
	} else {
		b.StatePath = DefaultStateFilename
		b.StateOutPath = DefaultStateFilename
	}

	if val := obj.GetAttr("workspace_dir"); !val.IsNull() {
		p := val.AsString()
		b.StateWorkspaceDir = p
	} else {
		b.StateWorkspaceDir = DefaultWorkspaceDir
	}

	return diags
}

func (b *Local) Workspaces() ([]string, error) {
	// the listing always start with "default"
	envs := []string{backend.DefaultStateName}

	entries, err := ioutil.ReadDir(b.stateWorkspaceDir())
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return envs, nil
	}
	if err != nil {
		return nil, err
	}

	var listed []string
	for _, entry := range entries {
		if entry.IsDir() {
			listed = append(listed, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(listed)
	envs = append(envs, listed...)

	return envs, nil
}

// DeleteWorkspace removes a workspace.
//
// The "default" workspace cannot be removed.
func (b *Local) DeleteWorkspace(name string, force bool) error {
	if name == "" {
		return errors.New("empty state name")
	}

	if name == backend.DefaultStateName {
		return errors.New("cannot delete default state")
	}

	delete(b.states, name)
	return os.RemoveAll(filepath.Join(b.stateWorkspaceDir(), name))
}

func (b *Local) StateMgr(name string) (statemgr.Full, error) {
	if s, ok := b.states[name]; ok {
		return s, nil
	}

	if err := b.createState(name); err != nil {
		return nil, err
	}

	statePath, stateOutPath, backupPath := b.StatePaths(name)
	log.Printf("[TRACE] backend/local: state manager for workspace %q will:\n - read initial snapshot from %s\n - write new snapshots to %s\n - create any backup at %s", name, statePath, stateOutPath, backupPath)

	s := statemgr.NewFilesystemBetweenPaths(statePath, stateOutPath)
	if backupPath != "" {
		s.SetBackupPath(backupPath)
	}

	if b.states == nil {
		b.states = map[string]statemgr.Full{}
	}
	b.states[name] = s
	return s, nil
}

// this only ensures that the named directory exists
func (b *Local) createState(name string) error {
	if name == backend.DefaultStateName {
		return nil
	}

	stateDir := filepath.Join(b.stateWorkspaceDir(), name)
	s, err := os.Stat(stateDir)
	if err == nil && s.IsDir() {
		// no need to check for os.IsNotExist, since that is covered by os.MkdirAll
		// which will catch the other possible errors as well.
		return nil
	}

	err = os.MkdirAll(stateDir, 0755)
	if err != nil {
		return err
	}

	return nil
}

// stateWorkspaceDir returns the directory where state environments are stored.
func (b *Local) stateWorkspaceDir() string {
	if b.StateWorkspaceDir != "" {
		return b.StateWorkspaceDir
	}

	return DefaultWorkspaceDir
}

// StatePaths returns the StatePath, StateOutPath, and StateBackupPath as
// configured from the CLI.
func (b *Local) StatePaths(name string) (stateIn, stateOut, backupOut string) {
	statePath := b.OverrideStatePath
	stateOutPath := b.OverrideStateOutPath
	backupPath := b.OverrideStateBackupPath

	isDefault := name == backend.DefaultStateName || name == ""

	baseDir := ""
	if !isDefault {
		baseDir = filepath.Join(b.stateWorkspaceDir(), name)
	}

	if statePath == "" {
		if isDefault {
			statePath = b.StatePath // s.StatePath applies only to the default workspace, since StateWorkspaceDir is used otherwise
		}
		if statePath == "" {
			statePath = filepath.Join(baseDir, DefaultStateFilename)
		}
	}
	if stateOutPath == "" {
		stateOutPath = statePath
	}
	if backupPath == "" {
		backupPath = b.StateBackupPath
	}
	switch backupPath {
	case "-":
		backupPath = ""
	case "":
		backupPath = stateOutPath + DefaultBackupExtension
	}

	return statePath, stateOutPath, backupPath
}

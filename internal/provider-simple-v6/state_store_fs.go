// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package simple

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

const fsStoreName = "simple6_fs"
const defaultStatesDir = "terraform.tfstate.d"

// FsStore allows storing state in the local filesystem.
//
// This state storage implementation differs from the old "local" backend in core,
// by storing all states in the custom, or default, states directory. In the "local"
// backend the default state was a special case and was handled differently to custom states.
type FsStore struct {
	// Configured values
	statesDir string
	chunkSize int64

	states map[string]*statemgr.Filesystem
}

var _ providers.StateStoreChunkSizeSetter = &FsStore{}

func stateStoreFsGetSchema() providers.Schema {
	return providers.Schema{
		Body: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				// Named workspace_dir to match what's present in the local backend
				"workspace_dir": {
					Type:        cty.String,
					Optional:    true,
					Description: "The directory where state files will be created. When unset the value will default to terraform.tfstate.d",
				},
			},
		},
	}
}

func (f *FsStore) ValidateStateStoreConfig(req providers.ValidateStateStoreConfigRequest) providers.ValidateStateStoreConfigResponse {
	var resp providers.ValidateStateStoreConfigResponse

	attrs := req.Config.AsValueMap()
	if v, ok := attrs["workspace_dir"]; ok {
		if !v.IsKnown() {
			resp.Diagnostics = resp.Diagnostics.Append(errors.New("the attribute \"workspace_dir\" cannot be an unknown value"))
			return resp
		}
	}

	return resp
}

func (f *FsStore) ConfigureStateStore(req providers.ConfigureStateStoreRequest) providers.ConfigureStateStoreResponse {
	resp := providers.ConfigureStateStoreResponse{}

	configVal := req.Config
	if v := configVal.GetAttr("workspace_dir"); !v.IsNull() {
		f.statesDir = v.AsString()
	} else {
		f.statesDir = defaultStatesDir
	}

	if f.states == nil {
		f.states = make(map[string]*statemgr.Filesystem)
	}

	// We need to select return a suggested chunk size; use the value suggested by Core
	resp.Capabilities.ChunkSize = req.Capabilities.ChunkSize
	f.chunkSize = req.Capabilities.ChunkSize

	return resp
}

func (f *FsStore) LockState(req providers.LockStateRequest) providers.LockStateResponse {
	resp := providers.LockStateResponse{}
	resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		"Locking not implemented",
		fmt.Sprintf("Could not lock state %q; state locking isn't implemented", req.StateId),
	))
	return resp
}

func (f *FsStore) UnlockState(req providers.UnlockStateRequest) providers.UnlockStateResponse {
	resp := providers.UnlockStateResponse{}
	resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
		tfdiags.Warning,
		"Unlocking not implemented",
		fmt.Sprintf("Could not unlock state %q; state locking isn't implemented", req.StateId),
	))
	return resp
}

func (f *FsStore) GetStates(req providers.GetStatesRequest) providers.GetStatesResponse {
	resp := providers.GetStatesResponse{}

	entries, err := os.ReadDir(f.statesDir)
	// no error if there's no envs configured
	if os.IsNotExist(err) {
		return resp
	}
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}

	var envs []string
	for _, entry := range entries {
		if entry.IsDir() {
			envs = append(envs, filepath.Base(entry.Name()))
		}
	}

	sort.Strings(envs)
	resp.States = envs
	return resp
}

func (f *FsStore) DeleteState(req providers.DeleteStateRequest) providers.DeleteStateResponse {
	resp := providers.DeleteStateResponse{}

	if req.StateId == "" {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("empty state name"))
		return resp
	}

	if req.StateId == backend.DefaultStateName {
		resp.Diagnostics = resp.Diagnostics.Append(errors.New("cannot delete default state"))
		return resp
	}

	delete(f.states, req.StateId)
	err := os.RemoveAll(filepath.Join(f.statesDir, req.StateId))
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error deleting state %q: %w", req.StateId, err))
		return resp
	}

	return resp
}

func (f *FsStore) getStatePath(stateId string) string {
	return path.Join(f.statesDir, stateId, "terraform.tfstate")
}

func (f *FsStore) getStateDir(stateId string) string {
	return path.Join(f.statesDir, stateId)
}

func (f *FsStore) ReadStateBytes(req providers.ReadStateBytesRequest) providers.ReadStateBytesResponse {
	log.Printf("[DEBUG] ReadStateBytes: reading data from the %q state", req.StateId)
	resp := providers.ReadStateBytesResponse{}

	// E.g. terraform.tfstate.d/foobar/terraform.tfstate
	path := f.getStatePath(req.StateId)
	file, err := os.Open(path)

	fileExists := true
	if err != nil {
		if _, ok := err.(*os.PathError); !ok {
			// Error other than the file not existing
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error opening state file %q: %w", path, err))
			return resp
		}
		fileExists = false
	}
	defer file.Close()

	buf := bytes.Buffer{}
	var processedBytes int

	if fileExists {
		for {
			b := make([]byte, f.chunkSize)
			n, err := file.Read(b)
			if err == io.EOF {
				break
			}
			if err != nil {
				resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error reading from state file %q: %w", path, err))
				return resp
			}
			buf.Write(b[0:n])
			processedBytes += n
		}
	}
	log.Printf("[DEBUG] ReadStateBytes: read %d bytes of data from state file %q", processedBytes, path)

	if processedBytes == 0 {
		// Does not exist, so return no bytes
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.Sourceless(
			tfdiags.Warning,
			"State doesn't exist",
			fmt.Sprintf("The %q state does not exist", req.StateId),
		))
	}

	resp.Bytes = buf.Bytes()
	return resp
}

func (f *FsStore) WriteStateBytes(req providers.WriteStateBytesRequest) providers.WriteStateBytesResponse {
	log.Printf("[DEBUG] WriteStateBytes: writing data to the %q state", req.StateId)
	resp := providers.WriteStateBytesResponse{}

	// E.g. terraform.tfstate.d/foobar/terraform.tfstate
	path := f.getStatePath(req.StateId)

	// Create or open state file
	dir := f.getStateDir(req.StateId)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error creating state file directory %q: %w", dir, err))
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error opening state file %q: %w", path, err))
	}

	buf := bytes.NewBuffer(req.Bytes)
	var processedBytes int
	if f.chunkSize == 0 {
		panic("WriteStateBytes: chunk size zero. This is an error in Terraform and should be reported")
	}
	for {
		data := buf.Next(int(f.chunkSize))
		if len(data) == 0 {
			break
		}
		n, err := file.Write(data)
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("error writing to state file %q: %w", path, err))
			return resp
		}

		processedBytes += n
	}
	log.Printf("[DEBUG] WriteStateBytes: wrote %d bytes of data to state file %q", processedBytes, path)

	if processedBytes == 0 {
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("missing state data: write action wrote %d bytes of data to file %q.", processedBytes, path))
	}

	return resp
}

func (f *FsStore) SetStateStoreChunkSize(typeName string, size int) {
	if typeName != fsStoreName {
		// If we hit this code it suggests someone's refactoring the PSS implementations used for testing
		panic(fmt.Sprintf("calling code tried to set the state store size on %s state store but the request reached the %s store implementation.",
			typeName,
			fsStoreName,
		))
	}

	f.chunkSize = int64(size)
}

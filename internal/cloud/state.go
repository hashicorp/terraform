// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	tfe "github.com/hashicorp/go-tfe"
	uuid "github.com/hashicorp/go-uuid"

	"github.com/hashicorp/terraform/internal/backend/local"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

// State implements the State interfaces in the state package to handle
// reading and writing the remote state to TFC. This State on its own does no
// local caching so every persist will go to the remote storage and local
// writes will go to memory.
type State struct {
	mu sync.Mutex

	// We track two pieces of meta data in addition to the state itself:
	//
	// lineage - the state's unique ID
	// serial  - the monotonic counter of "versions" of the state
	//
	// Both of these (along with state) have a sister field
	// that represents the values read in from an existing source.
	// All three of these values are used to determine if the new
	// state has changed from an existing state we read in.
	lineage, readLineage string
	serial, readSerial   uint64
	state, readState     *states.State
	disableLocks         bool
	tfeClient            *tfe.Client
	organization         string
	workspace            *tfe.Workspace
	stateUploadErr       bool
	forcePush            bool
	lockInfo             *statemgr.LockInfo
}

var ErrStateVersionUnauthorizedUpgradeState = errors.New(strings.TrimSpace(`
You are not authorized to read the full state version containing outputs.
State versions created by terraform v1.3.0 and newer do not require this level
of authorization and therefore this error can usually be fixed by upgrading the
remote state version.
`))

var _ statemgr.Full = (*State)(nil)
var _ statemgr.Migrator = (*State)(nil)
var _ local.IntermediateStateConditionalPersister = (*State)(nil)

// statemgr.Reader impl.
func (s *State) State() *states.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

// StateForMigration is part of our implementation of statemgr.Migrator.
func (s *State) StateForMigration() *statefile.File {
	s.mu.Lock()
	defer s.mu.Unlock()

	return statefile.New(s.state.DeepCopy(), s.lineage, s.serial)
}

// WriteStateForMigration is part of our implementation of statemgr.Migrator.
func (s *State) WriteStateForMigration(f *statefile.File, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !force {
		checkFile := statefile.New(s.state, s.lineage, s.serial)
		if err := statemgr.CheckValidImport(f, checkFile); err != nil {
			return err
		}
	}

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = f.State.DeepCopy()
	s.lineage = f.Lineage
	s.serial = f.Serial
	s.forcePush = force

	return nil
}

// DisableLocks turns the Lock and Unlock methods into no-ops. This is intended
// to be called during initialization of a state manager and should not be
// called after any of the statemgr.Full interface methods have been called.
func (s *State) DisableLocks() {
	s.disableLocks = true
}

// StateSnapshotMeta returns the metadata from the most recently persisted
// or refreshed persistent state snapshot.
//
// This is an implementation of statemgr.PersistentMeta.
func (s *State) StateSnapshotMeta() statemgr.SnapshotMeta {
	return statemgr.SnapshotMeta{
		Lineage: s.lineage,
		Serial:  s.serial,
	}
}

// statemgr.Writer impl.
func (s *State) WriteState(state *states.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = state.DeepCopy()
	s.forcePush = false

	return nil
}

// PersistState uploads a snapshot of the latest state as a StateVersion to Terraform Cloud
func (s *State) PersistState(schemas *terraform.Schemas) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[DEBUG] cloud/state: state read serial is: %d; serial is: %d", s.readSerial, s.serial)
	log.Printf("[DEBUG] cloud/state: state read lineage is: %s; lineage is: %s", s.readLineage, s.lineage)

	if s.readState != nil {
		lineageUnchanged := s.readLineage != "" && s.lineage == s.readLineage
		serialUnchanged := s.readSerial != 0 && s.serial == s.readSerial
		stateUnchanged := statefile.StatesMarshalEqual(s.state, s.readState)
		if stateUnchanged && lineageUnchanged && serialUnchanged {
			// If the state, lineage or serial haven't changed at all then we have nothing to do.
			return nil
		}
		s.serial++
	} else {
		// We might be writing a new state altogether, but before we do that
		// we'll check to make sure there isn't already a snapshot present
		// that we ought to be updating.
		err := s.refreshState()
		if err != nil {
			return fmt.Errorf("failed checking for existing remote state: %s", err)
		}
		log.Printf("[DEBUG] cloud/state: after refresh, state read serial is: %d; serial is: %d", s.readSerial, s.serial)
		log.Printf("[DEBUG] cloud/state: after refresh, state read lineage is: %s; lineage is: %s", s.readLineage, s.lineage)

		if s.lineage == "" { // indicates that no state snapshot is present yet
			lineage, err := uuid.GenerateUUID()
			if err != nil {
				return fmt.Errorf("failed to generate initial lineage: %v", err)
			}
			s.lineage = lineage
			s.serial++
		}
	}

	f := statefile.New(s.state, s.lineage, s.serial)

	var buf bytes.Buffer
	err := statefile.Write(f, &buf)
	if err != nil {
		return err
	}

	var jsonState []byte
	if schemas != nil {
		jsonState, err = jsonstate.Marshal(f, schemas)
		if err != nil {
			return err
		}
	}

	stateFile, err := statefile.Read(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return fmt.Errorf("failed to read state: %w", err)
	}

	ov, err := jsonstate.MarshalOutputs(stateFile.State.RootModule().OutputValues)
	if err != nil {
		return fmt.Errorf("failed to translate outputs: %w", err)
	}
	jsonStateOutputs, err := json.Marshal(ov)
	if err != nil {
		return fmt.Errorf("failed to marshal outputs to json: %w", err)
	}

	err = s.uploadState(s.lineage, s.serial, s.forcePush, buf.Bytes(), jsonState, jsonStateOutputs)
	if err != nil {
		s.stateUploadErr = true
		return fmt.Errorf("error uploading state: %w", err)
	}
	// After we've successfully persisted, what we just wrote is our new
	// reference state until someone calls RefreshState again.
	// We've potentially overwritten (via force) the state, lineage
	// and / or serial (and serial was incremented) so we copy over all
	// three fields so everything matches the new state and a subsequent
	// operation would correctly detect no changes to the lineage, serial or state.
	s.readState = s.state.DeepCopy()
	s.readLineage = s.lineage
	s.readSerial = s.serial
	return nil
}

// ShouldPersistIntermediateState implements local.IntermediateStateConditionalPersister
func (*State) ShouldPersistIntermediateState(info *local.IntermediateStatePersistInfo) bool {
	// We currently don't create intermediate snapshots for Terraform Cloud or
	// Terraform Enterprise at all, to avoid extra storage costs for Terraform
	// Enterprise customers.
	return false
}

func (s *State) uploadState(lineage string, serial uint64, isForcePush bool, state, jsonState, jsonStateOutputs []byte) error {
	ctx := context.Background()

	options := tfe.StateVersionCreateOptions{
		Lineage:          tfe.String(lineage),
		Serial:           tfe.Int64(int64(serial)),
		MD5:              tfe.String(fmt.Sprintf("%x", md5.Sum(state))),
		State:            tfe.String(base64.StdEncoding.EncodeToString(state)),
		Force:            tfe.Bool(isForcePush),
		JSONState:        tfe.String(base64.StdEncoding.EncodeToString(jsonState)),
		JSONStateOutputs: tfe.String(base64.StdEncoding.EncodeToString(jsonStateOutputs)),
	}

	// If we have a run ID, make sure to add it to the options
	// so the state will be properly associated with the run.
	runID := os.Getenv("TFE_RUN_ID")
	if runID != "" {
		options.Run = &tfe.Run{ID: runID}
	}
	// Create the new state.
	_, err := s.tfeClient.StateVersions.Create(ctx, s.workspace.ID, options)
	return err
}

// Lock calls the Client's Lock method if it's implemented.
func (s *State) Lock(info *statemgr.LockInfo) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disableLocks {
		return "", nil
	}
	ctx := context.Background()

	lockErr := &statemgr.LockError{Info: s.lockInfo}

	// Lock the workspace.
	_, err := s.tfeClient.Workspaces.Lock(ctx, s.workspace.ID, tfe.WorkspaceLockOptions{
		Reason: tfe.String("Locked by Terraform"),
	})
	if err != nil {
		if err == tfe.ErrWorkspaceLocked {
			lockErr.Info = info
			err = fmt.Errorf("%s (lock ID: \"%s/%s\")", err, s.organization, s.workspace.Name)
		}
		lockErr.Err = err
		return "", lockErr
	}

	s.lockInfo = info

	return s.lockInfo.ID, nil
}

// statemgr.Refresher impl.
func (s *State) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refreshState()
}

// refreshState is the main implementation of RefreshState, but split out so
// that we can make internal calls to it from methods that are already holding
// the s.mu lock.
func (s *State) refreshState() error {
	payload, err := s.getStatePayload()
	if err != nil {
		return err
	}

	// no remote state is OK
	if payload == nil {
		s.readState = nil
		s.lineage = ""
		s.serial = 0
		return nil
	}

	stateFile, err := statefile.Read(bytes.NewReader(payload.Data))
	if err != nil {
		return err
	}

	s.lineage = stateFile.Lineage
	s.serial = stateFile.Serial
	s.state = stateFile.State

	// Properties from the remote must be separate so we can
	// track changes as lineage, serial and/or state are mutated
	s.readLineage = stateFile.Lineage
	s.readSerial = stateFile.Serial
	s.readState = s.state.DeepCopy()
	return nil
}

func (s *State) getStatePayload() (*remote.Payload, error) {
	ctx := context.Background()

	sv, err := s.tfeClient.StateVersions.ReadCurrent(ctx, s.workspace.ID)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			// If no state exists, then return nil.
			return nil, nil
		}
		return nil, fmt.Errorf("error retrieving state: %v", err)
	}

	state, err := s.tfeClient.StateVersions.Download(ctx, sv.DownloadURL)
	if err != nil {
		return nil, fmt.Errorf("error downloading state: %v", err)
	}

	// If the state is empty, then return nil.
	if len(state) == 0 {
		return nil, nil
	}

	// Get the MD5 checksum of the state.
	sum := md5.Sum(state)

	return &remote.Payload{
		Data: state,
		MD5:  sum[:],
	}, nil
}

// Unlock calls the Client's Unlock method if it's implemented.
func (s *State) Unlock(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.disableLocks {
		return nil
	}

	ctx := context.Background()

	// We first check if there was an error while uploading the latest
	// state. If so, we will not unlock the workspace to prevent any
	// changes from being applied until the correct state is uploaded.
	if s.stateUploadErr {
		return nil
	}

	lockErr := &statemgr.LockError{Info: s.lockInfo}

	// With lock info this should be treated as a normal unlock.
	if s.lockInfo != nil {
		// Verify the expected lock ID.
		if s.lockInfo.ID != id {
			lockErr.Err = fmt.Errorf("lock ID does not match existing lock")
			return lockErr
		}

		// Unlock the workspace.
		_, err := s.tfeClient.Workspaces.Unlock(ctx, s.workspace.ID)
		if err != nil {
			lockErr.Err = err
			return lockErr
		}

		return nil
	}

	// Verify the optional force-unlock lock ID.
	if s.organization+"/"+s.workspace.Name != id {
		lockErr.Err = fmt.Errorf(
			"lock ID %q does not match existing lock ID \"%s/%s\"",
			id,
			s.organization,
			s.workspace.Name,
		)
		return lockErr
	}

	// Force unlock the workspace.
	_, err := s.tfeClient.Workspaces.ForceUnlock(ctx, s.workspace.ID)
	if err != nil {
		lockErr.Err = err
		return lockErr
	}

	return nil
}

// Delete the remote state.
func (s *State) Delete(force bool) error {

	var err error

	isSafeDeleteSupported := s.workspace.Permissions.CanForceDelete != nil
	if force || !isSafeDeleteSupported {
		err = s.tfeClient.Workspaces.Delete(context.Background(), s.organization, s.workspace.Name)
	} else {
		err = s.tfeClient.Workspaces.SafeDelete(context.Background(), s.organization, s.workspace.Name)
	}

	if err != nil && err != tfe.ErrResourceNotFound {
		return fmt.Errorf("error deleting workspace %s: %v", s.workspace.Name, err)
	}

	return nil
}

// GetRootOutputValues fetches output values from Terraform Cloud
func (s *State) GetRootOutputValues() (map[string]*states.OutputValue, error) {
	ctx := context.Background()

	so, err := s.tfeClient.StateVersionOutputs.ReadCurrent(ctx, s.workspace.ID)

	if err != nil {
		return nil, fmt.Errorf("could not read state version outputs: %w", err)
	}

	result := make(map[string]*states.OutputValue)

	for _, output := range so.Items {
		if output.DetailedType == nil {
			// If there is no detailed type information available, this state was probably created
			// with a version of terraform < 1.3.0. In this case, we'll eject completely from this
			// function and fall back to the old behavior of reading the entire state file, which
			// requires a higher level of authorization.
			log.Printf("[DEBUG] falling back to reading full state")

			if err := s.RefreshState(); err != nil {
				return nil, fmt.Errorf("failed to load state: %w", err)
			}

			state := s.State()
			if state == nil {
				// We know that there is supposed to be state (and this is not simply a new workspace
				// without state) because the fallback is only invoked when outputs are present but
				// detailed types are not available.
				return nil, ErrStateVersionUnauthorizedUpgradeState
			}

			return state.RootModule().OutputValues, nil
		}

		if output.Sensitive {
			// Since this is a sensitive value, the output must be requested explicitly in order to
			// read its value, which is assumed to be present by callers
			sensitiveOutput, err := s.tfeClient.StateVersionOutputs.Read(ctx, output.ID)
			if err != nil {
				return nil, fmt.Errorf("could not read state version output %s: %w", output.ID, err)
			}
			output.Value = sensitiveOutput.Value
		}

		cval, err := tfeOutputToCtyValue(*output)
		if err != nil {
			return nil, fmt.Errorf("could not decode output %s (ID %s)", output.Name, output.ID)
		}

		result[output.Name] = &states.OutputValue{
			Value:     cval,
			Sensitive: output.Sensitive,
		}
	}

	return result, nil
}

// tfeOutputToCtyValue decodes a combination of TFE output value and detailed-type to create a
// cty value that is suitable for use in terraform.
func tfeOutputToCtyValue(output tfe.StateVersionOutput) (cty.Value, error) {
	var result cty.Value
	bufType, err := json.Marshal(output.DetailedType)
	if err != nil {
		return result, fmt.Errorf("could not marshal output %s type: %w", output.ID, err)
	}

	var ctype cty.Type
	err = ctype.UnmarshalJSON(bufType)
	if err != nil {
		return result, fmt.Errorf("could not interpret output %s type: %w", output.ID, err)
	}

	result, err = gocty.ToCtyValue(output.Value, ctype)
	if err != nil {
		return result, fmt.Errorf("could not interpret value %v as type %s for output %s: %w", result, ctype.FriendlyName(), output.ID, err)
	}

	return result, nil
}

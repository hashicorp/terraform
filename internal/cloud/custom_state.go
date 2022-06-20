package cloud

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	tfe "github.com/hashicorp/go-tfe"
	uuid "github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform/internal/command/jsonstate"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terraform"
)

// CustomState implements the State interfaces in the state package to handle
// reading and writing the remote state to TFC. This State on its own does no
// local caching so every persist will go to the remote storage and local
// writes will go to memory.
type CustomState struct {
	mu sync.Mutex

	// Client Client

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
	schemas              *terraform.Schemas
	tfeClient            *tfe.Client
	organization         string
	workspace            *tfe.Workspace
	stateUploadErr       bool
	forcePush            bool
	lockInfo             *statemgr.LockInfo
}

var _ statemgr.Full = (*CustomState)(nil)
var _ statemgr.Migrator = (*CustomState)(nil)

// statemgr.Reader impl.
func (s *CustomState) State() *states.State {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.state.DeepCopy()
}

// StateForMigration is part of our implementation of statemgr.Migrator.
func (s *CustomState) StateForMigration() *statefile.File {
	s.mu.Lock()
	defer s.mu.Unlock()

	return statefile.New(s.state.DeepCopy(), s.lineage, s.serial)
}

// WriteStateForMigration is part of our implementation of statemgr.Migrator.
func (s *CustomState) WriteStateForMigration(f *statefile.File, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !force {
		checkFile := statefile.New(s.state, s.lineage, s.serial)
		if err := statemgr.CheckValidImport(f, checkFile); err != nil {
			return err
		}
	}

	// The remote backend needs to pass the `force` flag through to its client.
	// For backends that support such operations, inform the client
	// that a force push has been requested
	if force {
		s.EnableForcePush()
	}

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = f.State.DeepCopy()
	s.lineage = f.Lineage
	s.serial = f.Serial

	return nil
}

// DisableLocks turns the Lock and Unlock methods into no-ops. This is intended
// to be called during initialization of a state manager and should not be
// called after any of the statemgr.Full interface methods have been called.
func (s *CustomState) DisableLocks() {
	s.disableLocks = true
}

// StateSnapshotMeta returns the metadata from the most recently persisted
// or refreshed persistent state snapshot.
//
// This is an implementation of statemgr.PersistentMeta.
func (s *CustomState) StateSnapshotMeta() statemgr.SnapshotMeta {
	return statemgr.SnapshotMeta{
		Lineage: s.lineage,
		Serial:  s.serial,
	}
}

func (s *CustomState) WriteState(state *states.State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// We create a deep copy of the state here, because the caller also has
	// a reference to the given object and can potentially go on to mutate
	// it after we return, but we want the snapshot at this point in time.
	s.state = state.DeepCopy()

	return nil
}

// statemgr.Writer impl.
// func (s *CustomState) WriteState(state *states.State, schemas *terraform.Schemas) error {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()

// 	// We create a deep copy of the state here, because the caller also has
// 	// a reference to the given object and can potentially go on to mutate
// 	// it after we return, but we want the snapshot at this point in time.
// 	s.state = state.DeepCopy()
// 	s.schemas = schemas

// 	return nil
// }

// statemgr.Persister impl.
func (s *CustomState) PersistState() error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
		if s.lineage == "" { // indicates that no state snapshot is present yet
			lineage, err := uuid.GenerateUUID()
			if err != nil {
				return fmt.Errorf("failed to generate initial lineage: %v", err)
			}
			s.lineage = lineage
			s.serial = 0
		}
	}

	f := statefile.New(s.state, s.lineage, s.serial)

	var buf bytes.Buffer
	err := statefile.Write(f, &buf)
	if err != nil {
		return err
	}

	ctx := context.Background()

	options := tfe.StateVersionCreateOptions{
		Lineage: tfe.String(s.lineage),
		Serial:  tfe.Int64(int64(s.serial)),
		MD5:     tfe.String(fmt.Sprintf("%x", md5.Sum(buf.Bytes()))),
		State:   tfe.String(base64.StdEncoding.EncodeToString(buf.Bytes())),
		Force:   tfe.Bool(s.forcePush),
	}

	if s.schemas != nil {
		jsonState, err := jsonstate.Marshal(f, s.schemas)
		if err != nil {
			return err
		}
		fmt.Printf("jsonState: %+v", string(jsonState))
		options.ExtState = jsonState
	}

	// If we have a run ID, make sure to add it to the options
	// so the state will be properly associated with the run.
	runID := os.Getenv("TFE_RUN_ID")
	if runID != "" {
		options.Run = &tfe.Run{ID: runID}
	}

	// Create the new state.
	_, err = s.tfeClient.StateVersions.Create(ctx, s.workspace.ID, options)
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

// func getRemoteClient(tfeClient *tfe.Client, org string, ws *tfe.Workspace) *remoteClient {
// 	return &remoteClient{
// 		client:       tfeClient,
// 		organization: org,
// 		workspace:    ws,

// 		// This is optionally set during Terraform Enterprise runs.
// 		runID: os.Getenv("TFE_RUN_ID"),
// 	}

// }

// Lock calls the Client's Lock method if it's implemented.
func (s *CustomState) Lock(info *statemgr.LockInfo) (string, error) {
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
func (s *CustomState) RefreshState() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.refreshState()
}

// refreshState is the main implementation of RefreshState, but split out so
// that we can make internal calls to it from methods that are already holding
// the s.mu lock.
func (s *CustomState) refreshState() error {
	ctx := context.Background()

	sv, err := s.tfeClient.StateVersions.ReadCurrent(ctx, s.workspace.ID)
	if err != nil {
		if err == tfe.ErrResourceNotFound {
			// If no state exists, then return nil.
			s.readState = nil
			s.lineage = ""
			s.serial = 0
			return nil
		}
		return fmt.Errorf("error retrieving state: %v", err)
	}

	state, err := s.tfeClient.StateVersions.Download(ctx, sv.DownloadURL)

	if err != nil {
		return fmt.Errorf("error downloading state: %v", err)
	}

	// If the state is empty, then return nil.
	if len(state) == 0 {
		s.readState = nil
		s.lineage = ""
		s.serial = 0
		return nil
	}

	// Get the MD5 checksum of the state.
	sum := md5.Sum(state)

	payload := &remote.Payload{
		Data: state,
		MD5:  sum[:],
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

// Unlock calls the Client's Unlock method if it's implemented.
func (s *CustomState) Unlock(id string) error {
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
func (s *CustomState) Delete() error {
	err := s.tfeClient.Workspaces.Delete(context.Background(), s.organization, s.workspace.Name)
	if err != nil && err != tfe.ErrResourceNotFound {
		return fmt.Errorf("error deleting workspace %s: %v", s.workspace.Name, err)
	}

	return nil
}

// EnableForcePush to allow the remote client to overwrite state
// by implementing remote.ClientForcePusher
func (s *CustomState) EnableForcePush() {
	s.forcePush = true
}

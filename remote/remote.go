package remote

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform/terraform"
)

const (
	// LocalDirectory is the directory created in the working
	// dir to hold the remote state file.
	LocalDirectory = ".terraform"

	// HiddenStateFile is the name of the state file in the
	// LocalDirectory
	HiddenStateFile = "terraform.tfstate"

	// BackupHiddenStateFile is the path we backup the state
	// file to before modifications are made
	BackupHiddenStateFile = "terraform.tfstate.backup"
)

// StateChangeResult is used to communicate to a caller
// what actions have been taken when updating a state file
type StateChangeResult int

const (
	// StateChangeNoop indicates nothing has happened,
	// but that does not indicate an error. Everything is
	// just up to date. (Push/Pull)
	StateChangeNoop StateChangeResult = iota

	// StateChangeInit indicates that there is no local or
	// remote state, and that the state was initialized
	StateChangeInit

	// StateChangeUpdateLocal indicates the local state
	// was updated. (Pull)
	StateChangeUpdateLocal

	// StateChangeUpdateRemote indicates the remote state
	// was updated. (Push)
	StateChangeUpdateRemote

	// StateChangeLocalNewer means the pull was a no-op
	// because the local state is newer than that of the
	// server. This means a Push should take place. (Pull)
	StateChangeLocalNewer

	// StateChangeRemoteNewer means the push was a no-op
	// because the remote state is newer than that of the
	// local state. This means a Pull should take place.
	// (Push)
	StateChangeRemoteNewer

	// StateChangeConflict means that the push or pull
	// was a no-op because there is a conflict. This means
	// there are multiple state definitions at the same
	// serial number with different contents. This requires
	// an operator to intervene and resolve the conflict.
	// Shame on the user for doing concurrent apply.
	// (Push/Pull)
	StateChangeConflict
)

func (sc StateChangeResult) String() string {
	switch sc {
	case StateChangeNoop:
		return "Local and remote state in sync"
	case StateChangeInit:
		return "Local state initialized"
	case StateChangeUpdateLocal:
		return "Local state updated"
	case StateChangeUpdateRemote:
		return "Remote state updated"
	case StateChangeLocalNewer:
		return "Local state is newer than remote state, push required"
	case StateChangeRemoteNewer:
		return "Remote state is newer than local state, pull required"
	case StateChangeConflict:
		return "Local and remote state conflict, manual resolution required"
	default:
		return fmt.Sprintf("Unknown state change type: %d", sc)
	}
}

// SuccessfulPull is used to clasify the StateChangeResult for
// a pull operation. This is different by operation, but can be used
// to determine a proper exit code.
func (sc StateChangeResult) SuccessfulPull() bool {
	switch sc {
	case StateChangeNoop:
		return true
	case StateChangeInit:
		return true
	case StateChangeUpdateLocal:
		return true
	case StateChangeLocalNewer:
		return false
	case StateChangeConflict:
		return false
	default:
		return false
	}
}

// SuccessfulPush is used to clasify the StateChangeResult for
// a push operation. This is different by operation, but can be used
// to determine a proper exit code
func (sc StateChangeResult) SuccessfulPush() bool {
	switch sc {
	case StateChangeNoop:
		return true
	case StateChangeUpdateRemote:
		return true
	case StateChangeRemoteNewer:
		return false
	case StateChangeConflict:
		return false
	default:
		return false
	}
}

// EnsureDirectory is used to make sure the local storage
// directory exists
func EnsureDirectory() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get current directory: %v", err)
	}
	path := filepath.Join(cwd, LocalDirectory)
	if err := os.Mkdir(path, 0770); err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("Failed to make directory '%s': %v", path, err)
	}
	return nil
}

// HiddenStatePath is used to return the path to the hidden state file,
// should there be one.
// TODO: Rename to LocalStatePath
func HiddenStatePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed to get current directory: %v", err)
	}
	path := filepath.Join(cwd, LocalDirectory, HiddenStateFile)
	return path, nil
}

// HaveLocalState is used to check if we have a local state file
func HaveLocalState() (bool, error) {
	path, err := HiddenStatePath()
	if err != nil {
		return false, err
	}
	return ExistsFile(path)
}

// ExistsFile is used to check if a given file exists
func ExistsFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ValidConfig does a purely logical validation of the remote config
func ValidConfig(conf *terraform.RemoteState) error {
	// Default the type to Atlas
	if conf.Type == "" {
		conf.Type = "atlas"
	}
	_, err := NewClientByState(conf)
	if err != nil {
		return err
	}
	return nil
}

// ReadLocalState is used to read and parse the local state file
func ReadLocalState() (*terraform.State, []byte, error) {
	path, err := HiddenStatePath()
	if err != nil {
		return nil, nil, err
	}

	// Open the existing file
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("Failed to open state file '%s': %s", path, err)
	}

	// Decode the state
	state, err := terraform.ReadState(bytes.NewReader(raw))
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to read state file '%s': %v", path, err)
	}
	return state, raw, nil
}

// RefreshState is used to read the remote state given
// the configuration for the remote endpoint, and update
// the local state if necessary.
func RefreshState(conf *terraform.RemoteState) (StateChangeResult, error) {
	if conf == nil {
		return StateChangeNoop, fmt.Errorf("Missing remote server configuration")
	}

	// Read the state from the server
	client, err := NewClientByState(conf)
	if err != nil {
		return StateChangeNoop,
			fmt.Errorf("Failed to create remote client: %v", err)
	}
	payload, err := client.GetState()
	if err != nil {
		return StateChangeNoop,
			fmt.Errorf("Failed to read remote state: %v", err)
	}

	// Parse the remote state
	var remoteState *terraform.State
	if payload != nil {
		remoteState, err = terraform.ReadState(bytes.NewReader(payload.State))
		if err != nil {
			return StateChangeNoop,
				fmt.Errorf("Failed to parse remote state: %v", err)
		}

		// Ensure we understand the remote version!
		if remoteState.Version > terraform.StateVersion {
			return StateChangeNoop, fmt.Errorf(
				`Remote state is version %d, this version of Terraform only understands up to %d`, remoteState.Version, terraform.StateVersion)
		}
	}

	// Decode the state
	localState, raw, err := ReadLocalState()
	if err != nil {
		return StateChangeNoop, err
	}

	// We need to handle the matrix of cases in reconciling
	// the local and remote state. Primarily the concern is
	// around the Serial number which should grow monotonically.
	// Additionally, we use the MD5 to detect a conflict for
	// a given Serial.
	switch {
	case remoteState == nil && localState == nil:
		// Initialize a blank state
		out, _ := blankState(conf)
		if err := Persist(bytes.NewReader(out)); err != nil {
			return StateChangeNoop,
				fmt.Errorf("Failed to persist state: %v", err)
		}
		return StateChangeInit, nil

	case remoteState == nil && localState != nil:
		// User should probably do a push, nothing to do
		return StateChangeLocalNewer, nil

	case remoteState != nil && localState == nil:
		goto PERSIST

	case remoteState.Serial < localState.Serial:
		// User should probably do a push, nothing to do
		return StateChangeLocalNewer, nil

	case remoteState.Serial > localState.Serial:
		goto PERSIST

	case remoteState.Serial == localState.Serial:
		// Check for a hash collision on the local/remote state
		localMD5 := md5.Sum(raw)
		if bytes.Equal(localMD5[:md5.Size], payload.MD5) {
			// Hash collision, everything is up-to-date
			return StateChangeNoop, nil
		} else {
			// This is very bad. This means we have 2 state files
			// with the same Serial but a different hash. Most probably
			// explaination is two parallel apply operations. This
			// requires a manual reconciliation.
			return StateChangeConflict, nil
		}
	default:
		// We should not reach this point
		panic("Unhandled remote update case")
	}

PERSIST:
	// Update the local state from the remote state
	if err := Persist(bytes.NewReader(payload.State)); err != nil {
		return StateChangeNoop,
			fmt.Errorf("Failed to persist state: %v", err)
	}
	return StateChangeUpdateLocal, nil
}

// PushState is used to read the local state and
// update the remote state if necessary. The state push
// can be 'forced' to override any conflict detection
// on the server-side.
func PushState(conf *terraform.RemoteState, force bool) (StateChangeResult, error) {
	// Read the local state
	_, raw, err := ReadLocalState()
	if err != nil {
		return StateChangeNoop, err
	}

	// Check if there is no local state
	if raw == nil {
		return StateChangeNoop, fmt.Errorf("No local state to push")
	}

	// Push the state to the server
	client, err := NewClientByState(conf)
	if err != nil {
		return StateChangeNoop,
			fmt.Errorf("Failed to create remote client: %v", err)
	}
	err = client.PutState(raw, force)

	// Handle the various edge cases
	switch err {
	case nil:
		return StateChangeUpdateRemote, nil
	case ErrServerNewer:
		return StateChangeRemoteNewer, nil
	case ErrConflict:
		return StateChangeConflict, nil
	default:
		return StateChangeNoop, err
	}
}

// DeleteState is used to delete the remote state given
// the configuration for the remote endpoint.
func DeleteState(conf *terraform.RemoteState) error {
	if conf == nil {
		return fmt.Errorf("Missing remote server configuration")
	}

	// Setup the client
	client, err := NewClientByState(conf)
	if err != nil {
		return fmt.Errorf("Failed to create remote client: %v", err)
	}

	// Destroy the state
	err = client.DeleteState()
	if err != nil {
		return fmt.Errorf("Failed to delete remote state: %v", err)
	}
	return nil
}

// blankState is used to return a serialized form of a blank state
// with only the remote info.
func blankState(conf *terraform.RemoteState) ([]byte, error) {
	blank := terraform.NewState()
	blank.Remote = conf
	buf := bytes.NewBuffer(nil)
	err := terraform.WriteState(blank, buf)
	return buf.Bytes(), err
}

// PersistState is used to persist out the given terraform state
// in our local state cache location.
func PersistState(s *terraform.State) error {
	buf := bytes.NewBuffer(nil)
	if err := terraform.WriteState(s, buf); err != nil {
		return fmt.Errorf("Failed to encode state: %v", err)
	}
	if err := Persist(buf); err != nil {
		return err
	}
	return nil
}

// Persist is used to write out the state given by a reader (likely
// being streamed from a remote server) to the local storage.
func Persist(r io.Reader) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get current directory: %v", err)
	}
	statePath := filepath.Join(cwd, LocalDirectory, HiddenStateFile)
	backupPath := filepath.Join(cwd, LocalDirectory, BackupHiddenStateFile)

	// Backup the old file if it exists
	if err := CopyFile(statePath, backupPath); err != nil {
		return fmt.Errorf("Failed to backup state file '%s' to '%s': %v", statePath, backupPath, err)
	}

	// Open the state path
	fh, err := os.Create(statePath)
	if err != nil {
		return fmt.Errorf("Failed to open state file '%s': %v", statePath, err)
	}

	// Copy the new state
	_, err = io.Copy(fh, r)
	fh.Close()
	if err != nil {
		os.Remove(statePath)
		return fmt.Errorf("Failed to persist state file: %v", err)
	}
	return nil
}

// CopyFile is used to copy from a source file if it exists to a destination.
// This is used to create a backup of the state file.
func CopyFile(src, dst string) error {
	srcFH, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer srcFH.Close()

	dstFH, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFH.Close()

	_, err = io.Copy(dstFH, srcFH)
	return err
}

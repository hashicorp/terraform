package remote

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
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

	// DefaultServer is used when no server is provided. We use
	// the hosted cloud URL.
	DefaultServer = "http://www.hashicorp.com/"
)

// EnsureDirectory is used to make sure the local storage
// directory exists
func EnsureDirectory() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("Failed to get current directory: %v", err)
	}
	path := filepath.Join(cwd, LocalDirectory)
	if err := os.Mkdir(path, 0770); err != nil {
		return fmt.Errorf("Failed to make directory '%s': %v", path, err)
	}
	return nil
}

// HiddenStatePath is used to return the path to the hidden state file,
// should there be one.
func HiddenStatePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Failed to get current directory: %v", err)
	}
	path := filepath.Join(cwd, LocalDirectory, HiddenStateFile)
	return path, nil
}

// validConfig does a purely logical validation of the remote config
func validConfig(conf *terraform.RemoteState) error {
	// Verify the remote server configuration is sane
	if (conf.Server != "" || conf.AuthToken != "") && conf.Name == "" {
		return fmt.Errorf("Name must be provided for remote state storage")
	}
	if conf.Server != "" {
		if _, err := url.Parse(conf.Server); err != nil {
			return fmt.Errorf("Remote Server URL invalid: %v", err)
		}
	} else {
		// Fill in the default server
		conf.Server = DefaultServer
	}
	return nil
}

// ValidateConfig is used to take a remote state configuration,
// ensure the local directory exists and that the remote state
// does not conflict with an existing state file.
func ValidateConfig(conf *terraform.RemoteState) error {
	// Logical validation first
	if err := validConfig(conf); err != nil {
		return err
	}

	// Ensure the hidden directory
	if err := EnsureDirectory(); err != nil {
		return fmt.Errorf(
			"Remote state setup failed: %s", err)
	}

	// Get the path to the state file
	path, err := HiddenStatePath()
	if err != nil {
		return err
	}

	// Open the existing file
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("Failed to open state file '%s': %s", path, err)
		}
		return nil
	}
	defer f.Close()

	// Decode the state
	state, err := terraform.ReadState(f)
	if err != nil {
		return fmt.Errorf("Failed to read state file '%s': %v", path, err)
	}

	// If the hidden state file has no remote info, something
	// is definitely wrong...
	if state.Remote == nil {
		return fmt.Errorf(`State file '%s' missing remote storage information.
This is likely a bug, please report it.`)
	}

	// Check if there is a conflict
	if !state.Remote.Equals(conf) {
		return fmt.Errorf(
			"Conflicting definitions for remote storage in existing state file '%s'", path)
	}
	return nil
}

// ReadState is used to read the remote state given
// the configuration for the remote endpoint. We return
// a boolean indicating if the remote state exists, along
// with the state, and possible error.
func ReadState(conf *terraform.RemoteState) (io.Reader, error) {
	// TODO: Read actually from a server

	// Return the blank state, which is done if the server
	// returns a "not found" or equivalent
	return blankState(conf)
}

// blankState is used to return a serialized form of a blank state
// with only the remote info.
func blankState(conf *terraform.RemoteState) (io.Reader, error) {
	blank := terraform.NewState()
	blank.Remote = conf
	buf := bytes.NewBuffer(nil)
	err := terraform.WriteState(blank, buf)
	return buf, err
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
	if err := copyFile(statePath, backupPath); err != nil {
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

// copyFile is used to copy from a source file if it exists to a destination.
// This is used to create a backup of the state file.
func copyFile(src, dst string) error {
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

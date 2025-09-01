// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/version"
)

// BackendStateFile describes the overall structure of the file format used
// to track a working directory's active backend.
//
// The main interesting part of this is the [BackendStateFile.Backend] field,
// but [BackendStateFile.Version] is also important to make sure that the
// current Terraform CLI version will be able to understand the file.
type BackendStateFile struct {
	// Don't access this directly. It's here only for use during serialization
	// and deserialization of backend state file contents.
	Version int `json:"version"`

	// TFVersion is the version of Terraform that wrote this state. This is
	// really just for debugging purposes; we don't currently vary behavior
	// based on this field.
	TFVersion string `json:"terraform_version,omitempty"`

	// Backend tracks the configuration for the backend in use with
	// this state. This is used to track any changes in the `backend`
	// block's configuration.
	// Note: this also used to tracking changes in the `cloud` block
	Backend *BackendConfigState `json:"backend,omitempty"`

	// StateStore tracks the configuration for a state store in use
	// with this state. This is used to track any changes in the `state_store`
	// block's configuration or associated data about the provider facilitating
	// state storage
	StateStore *StateStoreConfigState `json:"state_store,omitempty"`

	// This is here just so we can sniff for the unlikely-but-possible
	// situation that someone is trying to use modern Terraform with a
	// directory that was most recently used with Terraform v0.8, before
	// there was any concept of backends. Don't access this field.
	Remote *struct{} `json:"remote,omitempty"`
}

// NewBackendStateFile returns a new [BackendStateFile] object that initially
// has no backend configured.
//
// Callers should then mutate [BackendStateFile.Backend] in the result to
// specify the explicit backend in use, if any.
func NewBackendStateFile() *BackendStateFile {
	return &BackendStateFile{
		// NOTE: We don't populate Version or TFVersion here because we
		// always clobber those when encoding a state file in
		// [EncodeBackendStateFile].
	}
}

// ParseBackendStateFile tries to decode the given byte slice as the backend
// state file format.
//
// Returns an error if the content is not valid syntax, or if the file is
// of an unsupported format version.
//
// This does not immediately decode the embedded backend config, and so
// it's possible that a subsequent call to [BackendConfigState.Config] will
// return further errors even if this call succeeds.
func ParseBackendStateFile(src []byte) (*BackendStateFile, error) {
	// To avoid any weird collisions with as-yet-unknown future versions of
	// the format, we'll do a first pass of decoding just the "version"
	// property, and then decode the rest only if we find the version number
	// that we're expecting.
	type VersionSniff struct {
		Version   int    `json:"version"`
		TFVersion string `json:"terraform_version,omitempty"`
	}
	var versionSniff VersionSniff
	err := json.Unmarshal(src, &versionSniff)
	if err != nil {
		return nil, fmt.Errorf("invalid syntax: %w", err)
	}
	if versionSniff.Version == 0 {
		// This could either mean that it's explicitly "version": 0 or that
		// the version property is missing. We'll assume the latter here
		// because state snapshot version 0 was an encoding/gob binary format
		// rather than a JSON format and so it would be very weird for
		// that to show up in a JSON file.
		return nil, fmt.Errorf("invalid syntax: no format version number")
	}
	if versionSniff.Version != 3 {
		return nil, fmt.Errorf("unsupported backend state version %d; you may need to use Terraform CLI v%s to work in this directory", versionSniff.Version, versionSniff.TFVersion)
	}

	// If we get here then we can be sure that this file at least _thinks_
	// it's format version 3.
	var stateFile BackendStateFile
	err = json.Unmarshal(src, &stateFile)
	if err != nil {
		return nil, fmt.Errorf("invalid syntax: %w", err)
	}
	if stateFile.Backend == nil && stateFile.Remote != nil {
		// It's very unlikely to get here, but one way it could happen is
		// if this working directory was most recently used with Terraform v0.8
		// or earlier, which didn't yet include the concept of backends.
		// This error message assumes that's the case.
		return nil, fmt.Errorf("this working directory uses legacy remote state and so must first be upgraded using Terraform v0.9")
	}
	if stateFile.Backend != nil && stateFile.StateStore != nil {
		return nil, fmt.Errorf("encountered a malformed backend state file that contains state for both a 'backend' and a 'state_store' block")
	}

	return &stateFile, nil
}

func EncodeBackendStateFile(f *BackendStateFile) ([]byte, error) {
	f.Version = 3 // we only support version 3
	f.TFVersion = version.SemVer.String()

	switch {
	case f.Backend != nil && f.StateStore != nil:
		return nil, fmt.Errorf("attempted to encode a malformed backend state file; it contains state for both a 'backend' and a 'state_store' block. This is a bug in Terraform and should be reported.")
	case f.Backend == nil && f.StateStore == nil:
		// This is valid - if the user has a backend state file and an implied local backend in use
		// the backend state file exists but has no Backend data.
	case f.Backend != nil:
		// Not implementing anything here - risk of breaking changes
	case f.StateStore != nil:
		err := f.StateStore.Validate()
		if err != nil {
			return nil, err
		}
	default:
		panic("error when determining whether backend state file was valid. This is a bug in Terraform and should be reported.")
	}

	return json.MarshalIndent(f, "", "  ")
}

func (f *BackendStateFile) DeepCopy() *BackendStateFile {
	if f == nil {
		return nil
	}
	ret := &BackendStateFile{
		Version:    f.Version,
		TFVersion:  f.TFVersion,
		Backend:    f.Backend.DeepCopy(),
		StateStore: f.StateStore.DeepCopy(),
	}
	if f.Remote != nil {
		// This shouldn't ever be present in an object held by a caller since
		// we'd return an error about it during load, but we'll set it anyway
		// just to minimize surprise.
		ret.Remote = &struct{}{}
	}
	return ret
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding/json"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
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
	// this state. This is used to track any changes in the backend
	// configuration.
	Backend *BackendState `json:"backend,omitempty"`

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
// it's possible that a subsequent call to [BackendState.Config] will
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

	return &stateFile, nil
}

func EncodeBackendStateFile(f *BackendStateFile) ([]byte, error) {
	f.Version = 3 // we only support version 3
	f.TFVersion = version.SemVer.String()
	return json.MarshalIndent(f, "", "  ")
}

func (f *BackendStateFile) DeepCopy() *BackendStateFile {
	if f == nil {
		return nil
	}
	ret := &BackendStateFile{
		Version:   f.Version,
		TFVersion: f.TFVersion,
		Backend:   f.Backend.DeepCopy(),
	}
	if f.Remote != nil {
		// This shouldn't ever be present in an object held by a caller since
		// we'd return an error about it during load, but we'll set it anyway
		// just to minimize surprise.
		ret.Remote = &struct{}{}
	}
	return ret
}

// BackendState describes the physical storage format for the backend state
// in a working directory, and provides the lowest-level API for decoding it.
type BackendState struct {
	Type      string          `json:"type"`   // Backend type
	ConfigRaw json.RawMessage `json:"config"` // Backend raw config
	Hash      uint64          `json:"hash"`   // Hash of portion of configuration from config files
}

// Empty returns true if there is no active backend.
//
// In practice this typically means that the working directory is using the
// implied local backend, but that decision is made by the caller.
func (s *BackendState) Empty() bool {
	return s == nil || s.Type == ""
}

// Config decodes the type-specific configuration object using the provided
// schema and returns the result as a cty.Value.
//
// An error is returned if the stored configuration does not conform to the
// given schema, or is otherwise invalid.
func (s *BackendState) Config(schema *configschema.Block) (cty.Value, error) {
	ty := schema.ImpliedType()
	if s == nil {
		return cty.NullVal(ty), nil
	}
	return ctyjson.Unmarshal(s.ConfigRaw, ty)
}

// SetConfig replaces (in-place) the type-specific configuration object using
// the provided value and associated schema.
//
// An error is returned if the given value does not conform to the implied
// type of the schema.
func (s *BackendState) SetConfig(val cty.Value, schema *configschema.Block) error {
	ty := schema.ImpliedType()
	buf, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return err
	}
	s.ConfigRaw = buf
	return nil
}

// ForPlan produces an alternative representation of the reciever that is
// suitable for storing in a plan. The current workspace must additionally
// be provided, to be stored alongside the backend configuration.
//
// The backend configuration schema is required in order to properly
// encode the backend-specific configuration settings.
func (s *BackendState) ForPlan(schema *configschema.Block, workspaceName string) (*plans.Backend, error) {
	if s == nil {
		return nil, nil
	}

	configVal, err := s.Config(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to decode backend config: %w", err)
	}
	return plans.NewBackend(s.Type, configVal, schema, workspaceName)
}

func (s *BackendState) DeepCopy() *BackendState {
	if s == nil {
		return nil
	}
	ret := &BackendState{
		Type: s.Type,
		Hash: s.Hash,
	}

	if s.ConfigRaw != nil {
		ret.ConfigRaw = make([]byte, len(s.ConfigRaw))
		copy(ret.ConfigRaw, s.ConfigRaw)
	}
	return ret
}

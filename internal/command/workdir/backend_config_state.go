// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
)

var _ ConfigState = &BackendConfigState{}
var _ DeepCopier[BackendConfigState] = &BackendConfigState{}
var _ PlanDataProvider[plans.Backend] = &BackendConfigState{}

// BackendConfigState describes the physical storage format for the backend state
// in a working directory, and provides the lowest-level API for decoding it.
type BackendConfigState struct {
	Type      string          `json:"type"`   // Backend type
	ConfigRaw json.RawMessage `json:"config"` // Backend raw config
	Hash      uint64          `json:"hash"`   // Hash of portion of configuration from config files
}

// Empty returns true if there is no active backend.
//
// In practice this typically means that the working directory is using the
// implied local backend, but that decision is made by the caller.
func (s *BackendConfigState) Empty() bool {
	return s == nil || s.Type == ""
}

// Config decodes the type-specific configuration object using the provided
// schema and returns the result as a cty.Value.
//
// An error is returned if the stored configuration does not conform to the
// given schema, or is otherwise invalid.
func (s *BackendConfigState) Config(schema *configschema.Block) (cty.Value, error) {
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
func (s *BackendConfigState) SetConfig(val cty.Value, schema *configschema.Block) error {
	if s == nil {
		return errors.New("SetConfig called on nil BackendConfigState receiver")
	}
	ty := schema.ImpliedType()
	buf, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return err
	}
	s.ConfigRaw = buf
	return nil
}

// PlanData produces an alternative representation of the receiver that is
// suitable for storing in a plan. The current workspace must additionally
// be provided, to be stored alongside the backend configuration.
//
// The backend configuration schema is required in order to properly
// encode the backend-specific configuration settings.
func (s *BackendConfigState) PlanData(schema *configschema.Block, workspaceName string) (*plans.Backend, error) {
	if s == nil {
		return nil, nil
	}

	configVal, err := s.Config(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to decode backend config: %w", err)
	}
	return plans.NewBackend(s.Type, configVal, schema, workspaceName)
}

func (s *BackendConfigState) DeepCopy() *BackendConfigState {
	if s == nil {
		return nil
	}
	ret := &BackendConfigState{
		Type: s.Type,
		Hash: s.Hash,
	}

	if s.ConfigRaw != nil {
		ret.ConfigRaw = make([]byte, len(s.ConfigRaw))
		copy(ret.ConfigRaw, s.ConfigRaw)
	}
	return ret
}

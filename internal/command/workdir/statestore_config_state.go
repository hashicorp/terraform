// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package workdir

import (
	"encoding"
	"encoding/json"
	"fmt"

	version "github.com/hashicorp/go-version"
	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

var _ ConfigState[StateStoreConfigState] = &StateStoreConfigState{}

// StateStoreConfigState describes the physical storage format for the state store
type StateStoreConfigState struct {
	Type      string          `json:"type"`     // State store type name
	Provider  *Provider       `json:"provider"` // Details about the state-storage provider
	ConfigRaw json.RawMessage `json:"config"`   // state_store block raw config, barring provider details
	Hash      uint64          `json:"hash"`     // Hash of portion of configuration from config files
}

// Provider is used in the StateStoreConfigState struct to describe the provider that's used for pluggable
// state storage. The data inside should mirror an entry in the dependency lock file.
// This is NOT state of a `provider` configuration block, or an entry in `required_providers`.
type Provider struct {
	Version *version.Version `json:"version"` // The specific provider version used for the state store. Should be set using a getproviders.Version, etc.
	Source  *Source          `json:"source"`  // The FQN/fully-qualified name of the provider.
}

type Source struct {
	Provider tfaddr.Provider `json:"-"` // For un/marshaling purposes this needs to be exported, but it's omitted from JSON
}

var _ encoding.TextMarshaler = &Source{}
var _ encoding.TextUnmarshaler = &Source{}

// MarshalText defers to the internal terraform-registry-address Provider's String method
func (p *Source) MarshalText() ([]byte, error) {
	fqn := p.Provider.String()
	return []byte(fqn), nil
}

// UnmarshalText uses existing methods for parsing `source` values from `required_providers` config to
// convert a source string into a terraform-registry-address Provider
func (p *Source) UnmarshalText(text []byte) error {
	input := string(text)
	provider, diag := addrs.ParseProviderSourceString(input)
	if diag != nil {
		return fmt.Errorf("error unmarshaling provider source from backend state file: %s", diag.ErrWithWarnings())
	}
	p.Provider = provider
	return nil
}

// Empty returns true if there is no active state store.
func (s *StateStoreConfigState) Empty() bool {
	return s == nil || s.Type == ""
}

// Config decodes the type-specific configuration object using the provided
// schema and returns the result as a cty.Value.
//
// An error is returned if the stored configuration does not conform to the
// given schema, or is otherwise invalid.
func (s *StateStoreConfigState) Config(schema *configschema.Block) (cty.Value, error) {
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
func (s *StateStoreConfigState) SetConfig(val cty.Value, schema *configschema.Block) error {
	ty := schema.ImpliedType()
	buf, err := ctyjson.Marshal(val, ty)
	if err != nil {
		return err
	}
	s.ConfigRaw = buf
	return nil
}

// ForPlan produces an alternative representation of the receiver that is
// suitable for storing in a plan. The current workspace must additionally
// be provided, to be stored alongside the state store configuration.
//
// The state_store configuration schema is required in order to properly
// encode the state store-specific configuration settings.
func (s *StateStoreConfigState) ForPlan(schema *configschema.Block, workspaceName string) (*plans.Backend, error) {
	if s == nil {
		return nil, nil
	}
	// TODO
	// What should a pluggable state store look like in a plan?
	return nil, nil
}

func (s *StateStoreConfigState) DeepCopy() *StateStoreConfigState {
	if s == nil {
		return nil
	}
	provider := &Provider{
		Version: s.Provider.Version,
		Source:  s.Provider.Source,
	}
	ret := &StateStoreConfigState{
		Type:     s.Type,
		Provider: provider,
		Hash:     s.Hash,
	}

	if s.ConfigRaw != nil {
		ret.ConfigRaw = make([]byte, len(s.ConfigRaw))
		copy(ret.ConfigRaw, s.ConfigRaw)
	}
	return ret
}

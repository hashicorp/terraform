// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type QueryStart struct {
	Address      string                     `json:"address"`
	ResourceType string                     `json:"resource_type"`
	InputConfig  map[string]json.RawMessage `json:"input_config,omitempty"`
}

type QueryResult struct {
	Address        string                     `json:"address"`
	DisplayName    string                     `json:"display_name"`
	Identity       map[string]json.RawMessage `json:"identity"`
	ResourceType   string                     `json:"resource_type"`
	ResourceObject map[string]json.RawMessage `json:"resource_object,omitempty"`
	Config         string                     `json:"config,omitempty"`
	ImportConfig   string                     `json:"import_config,omitempty"`
}

func NewQueryStart(addr addrs.AbsResourceInstance, input_config cty.Value) QueryStart {
	return QueryStart{
		Address:      addr.String(),
		ResourceType: addr.Resource.Resource.Type,
		InputConfig:  marshalValues(input_config),
	}
}

func NewQueryResult(listAddr addrs.AbsResourceInstance, value cty.Value, generated *genconfig.Resource) QueryResult {
	var config, importConfig string
	if generated != nil {
		config = generated.String()
		importConfig = string(generated.Import)
	}
	result := QueryResult{
		Address:        listAddr.String(),
		DisplayName:    value.GetAttr("display_name").AsString(),
		Identity:       marshalValues(value.GetAttr("identity")),
		ResourceType:   listAddr.Resource.Resource.Type,
		ResourceObject: marshalValues(value.GetAttr("state")),
		Config:         config,
		ImportConfig:   importConfig,
	}
	return result
}

func marshalValues(value cty.Value) map[string]json.RawMessage {
	if value == cty.NilVal || value.IsNull() {
		return nil
	}

	ret := make(map[string]json.RawMessage)
	it := value.ElementIterator()
	for it.Next() {
		k, v := it.Element()
		vJSON, _ := ctyjson.Marshal(v, v.Type())
		ret[k.AsString()] = json.RawMessage(vJSON)
	}
	return ret
}

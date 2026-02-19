// Copyright IBM Corp. 2014, 2026
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
	Address         string                     `json:"address"`
	DisplayName     string                     `json:"display_name"`
	Identity        map[string]json.RawMessage `json:"identity"`
	IdentityVersion int64                      `json:"identity_version"`
	ResourceType    string                     `json:"resource_type"`
	ResourceObject  map[string]json.RawMessage `json:"resource_object,omitempty"`
	Config          string                     `json:"config,omitempty"`
	ImportConfig    string                     `json:"import_config,omitempty"`
}

type QueryComplete struct {
	Address      string `json:"address"`
	ResourceType string `json:"resource_type"`
	Total        int    `json:"total"`
}

func NewQueryStart(addr addrs.AbsResourceInstance, inputConfig cty.Value) QueryStart {
	return QueryStart{
		Address:      addr.String(),
		ResourceType: addr.Resource.Resource.Type,
		InputConfig:  marshalValues(inputConfig),
	}
}

func NewQueryResult(listAddr addrs.AbsResourceInstance, value cty.Value, identityVersion int64, generated *genconfig.ResourceImport) QueryResult {
	result := QueryResult{
		Address:         listAddr.String(),
		DisplayName:     value.GetAttr("display_name").AsString(),
		Identity:        marshalValues(value.GetAttr("identity")),
		IdentityVersion: identityVersion,
		ResourceType:    listAddr.Resource.Resource.Type,
		ResourceObject:  marshalValues(value.GetAttr("state")),
	}

	if generated != nil {
		result.Config = generated.Resource.String()
		result.ImportConfig = string(generated.ImportBody)
	}
	return result
}

func NewQueryComplete(addr addrs.AbsResourceInstance, total int) QueryComplete {
	return QueryComplete{
		Address:      addr.String(),
		ResourceType: addr.Resource.Resource.Type,
		Total:        total,
	}
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

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"encoding/json"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"
)

type QueryResult struct {
	Address        string                     `json:"address"`
	DisplayName    string                     `json:"display_name"`
	Identity       map[string]json.RawMessage `json:"identity"`
	ResourceType   string                     `json:"resource_type"`
	ResourceObject map[string]json.RawMessage `json:"resource_object,omitempty"`
	Config         string                     `json:"config,omitempty"`
}

func NewQueryResult(addr addrs.AbsResourceInstance, value cty.Value) QueryResult {
	return QueryResult{
		Address:        addr.String(),
		DisplayName:    value.GetAttr("display_name").AsString(),
		Identity:       marshalValues(value.GetAttr("identity")),
		ResourceType:   addr.Resource.Resource.Type,
		ResourceObject: marshalValues(value.GetAttr("state")),
		// Config
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

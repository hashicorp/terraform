// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonlist

import (
	"encoding/json"
	"fmt"

	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

type Query struct {
	Address string `json:"address,omitempty"`

	Results []QueryResult `json:"results"`
}

type QueryResult struct {
	DisplayName string                     `json:"display_name"`
	Identity    map[string]json.RawMessage `json:"identity"`
	Resource    map[string]json.RawMessage `json:"resource,omitempty"`

	// TODO
	// Address string `json:"address,omitempty"`
	// Config string `json:"config,omitempty"`
}

func MarshalForRenderer(
	p *plans.Plan,
	schemas *terraform.Schemas,
) ([]Query, error) {
	return MarshalQueryInstances(p.Changes.Queries, schemas)
}

func MarshalQueryInstances(resources []*plans.QueryInstanceSrc, schemas *terraform.Schemas) ([]Query, error) {
	var ret []Query

	for _, rc := range resources {
		r, err := marshalQueryInstance(rc, schemas)
		if err != nil {
			return nil, err
		}
		ret = append(ret, r)
	}

	return ret, nil
}

func marshalQueryInstance(rc *plans.QueryInstanceSrc, schemas *terraform.Schemas) (Query, error) {
	var ret Query
	addr := rc.Addr
	ret.Address = addr.String()

	schema := schemas.ResourceTypeConfig(
		rc.ProviderAddr.Provider,
		addr.Resource.Resource.Mode,
		addr.Resource.Resource.Type,
	)
	if schema.Body == nil {
		return ret, fmt.Errorf("no schema found for %s (in provider %s)", ret.Address, rc.ProviderAddr.Provider)
	}

	query, err := rc.Decode(schema)
	if err != nil {
		return ret, err
	}

	data := query.Results.GetAttr("data")
	for it := data.ElementIterator(); it.Next(); {
		_, value := it.Element()

		result := QueryResult{
			DisplayName: value.GetAttr("display_name").AsString(),
			Identity:    marshalValues(value.GetAttr("identity")),
			Resource:    marshalValues(value.GetAttr("state")),
		}

		ret.Results = append(ret.Results, result)
	}

	return ret, nil
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

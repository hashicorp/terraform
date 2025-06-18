// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonlist

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
)

type QueryResult struct {
	Address string `json:"address,omitempty"`

	Identity json.RawMessage `json:"identity,omitempty"`

	Resource json.RawMessage `json:"resource,omitempty"`

	DisplayName string `json:"display_name,omitempty"`
}

func MarshalForRenderer(
	p *plans.Plan,
	schemas *terraform.Schemas,
) ([]QueryResult, error) {
	return MarshalQueryInstances(p.Changes.Queries, schemas)
}

func MarshalQueryInstances(resources []*plans.QueryInstanceSrc, schemas *terraform.Schemas) ([]QueryResult, error) {
	var ret []QueryResult

	for _, rc := range resources {
		r, err := marshalQueryInstance(rc, schemas)
		if err != nil {
			return nil, err
		}
		ret = append(ret, r...)
	}

	return ret, nil
}

func marshalQueryInstance(rc *plans.QueryInstanceSrc, schemas *terraform.Schemas) ([]QueryResult, error) {
	var ret []QueryResult
	addr := rc.Addr

	schema := schemas.ResourceTypeConfig(
		rc.ProviderAddr.Provider,
		addr.Resource.Resource.Mode,
		addr.Resource.Resource.Type,
	)
	if schema.Body == nil {
		return ret, fmt.Errorf("no schema found for %s (in provider %s)", addr.String(), rc.ProviderAddr.Provider)
	}

	query, err := rc.Decode(schema)
	if err != nil {
		return ret, err
	}

	data := query.Results.GetAttr("data")
	for it := data.ElementIterator(); it.Next(); {
		var r QueryResult
		r.Address = addr.String()

		_, value := it.Element()

		r.DisplayName = value.GetAttr("display_name").AsString()
		// identity
		// resource object
	}

	return ret, nil
}

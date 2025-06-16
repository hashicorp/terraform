// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package jsonlist

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func MarshalQueryInstances(resources []*plans.QueryInstanceSrc, schemas *terraform.Schemas) ([]string, error) {
	var ret []string

	for _, rc := range resources {
		r, err := marshalQueryInstance(rc, schemas)
		if err != nil {
			return nil, err
		}
		ret = append(ret, r)
	}

	return ret, nil
}

func marshalQueryInstance(rc *plans.QueryInstanceSrc, schemas *terraform.Schemas) (string, error) {
	var r string
	addr := rc.Addr

	schema := schemas.ResourceTypeConfig(
		rc.ProviderAddr.Provider,
		addr.Resource.Resource.Mode,
		addr.Resource.Resource.Type,
	)
	if schema.Body == nil {
		return r, fmt.Errorf("no schema found for %s (in provider %s)", addr.String(), rc.ProviderAddr.Provider)
	}

	query, err := rc.Decode(schema)
	if err != nil {
		return r, err
	}

	data := query.Results.GetAttr("data")
	for it := data.ElementIterator(); it.Next(); {
		_, value := it.Element()

		name := value.GetAttr("display_name").AsString()
		identity := value.GetAttr("identity")

		fmt.Printf("%s.%s\t%s\t%s\n", addr.Resource.Resource.Type, addr.Resource.Resource.Name, tfdiags.ObjectToString(identity), name)
	}

	return r, nil
}

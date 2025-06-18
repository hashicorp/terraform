// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func NewQueryResults(change *plans.QueryInstanceSrc) []*QueryResult {
	var ret []*QueryResult
	addr := newResourceAddr(change.Addr)

	for _, _ = range change.Results {
		r := &QueryResult{
			Addr:         addr,
			ResourceType: change.Addr.Resource.Resource.Type,
			// DisplayName: result.DisplayName,
		}
		ret = append(ret, r)
	}

	return ret
}

type QueryResult struct {
	Addr            ResourceAddr `json:"addr"`
	ResourceType    string       `json:"resource_type"`
	DisplayName     string       `json:"display_name"`
	Identity        ResourceAddr `json:"identity"`
	ResourceObject  ResourceAddr `json:"resource_object,omitempty"`
	GeneratedConfig string       `json:"generated_config,omitempty"`
}

func (r *QueryResult) String() string {
	return fmt.Sprintf("%s: New result", r.Addr.Addr)
}

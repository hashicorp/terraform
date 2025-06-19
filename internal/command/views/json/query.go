// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package json

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/plans"
)

func NewQueryResults(change *plans.QueryInstanceSrc) *QueryResult {
	addr := newResourceAddr(change.Addr)

	return &QueryResult{
		Addr:         addr,
		ResourceType: change.Addr.Resource.Resource.Type,
	}

}

type QueryResult struct {
	Addr         ResourceAddr `json:"addr"`
	ResourceType string       `json:"resource_type"`
}

func (r *QueryResult) String() string {
	return fmt.Sprintf("%s: Quering resources...", r.Addr.Addr)
}

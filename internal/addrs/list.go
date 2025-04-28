// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"

	"github.com/hashicorp/terraform/internal/collections"
)

// List is an address for a list block within a query configuration
type List struct {
	collections.UniqueKeyer[List]
	referenceable
	Type string
	Name string
}

func (r List) String() string {
	return fmt.Sprintf("%s.%s", r.Type, r.Name)
}

type ListResource struct {
	List     List
	Resource Resource
}

func (l List) UniqueKey() UniqueKey {
	return l // A List is its own UniqueKey
}

func (r List) uniqueKeySigil() {}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package schema

import "github.com/hashicorp/terraform/internal/legacy/helper/hashcode"

func HashString(v interface{}) int {
	return hashcode.String(v.(string))
}

func NewSet(f SchemaSetFunc, items []interface{}) *Set {
	s := &Set{F: f}
	for _, i := range items {
		s.Add(i)
	}

	return s
}

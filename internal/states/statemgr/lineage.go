// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statemgr

import (
	"fmt"

	uuid "github.com/hashicorp/go-uuid"
)

// NewLineage generates a new lineage identifier string. A lineage identifier
// is an opaque string that is intended to be unique in space and time, chosen
// when state is recorded at a location for the first time and then preserved
// afterwards to allow Terraform to recognize when one state snapshot is a
// predecessor or successor of another.
func NewLineage() string {
	lineage, err := uuid.GenerateUUID()
	if err != nil {
		panic(fmt.Errorf("Failed to generate lineage: %v", err))
	}
	return lineage
}

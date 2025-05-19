// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package statefile

import (
	version "github.com/hashicorp/go-version"

	"github.com/hashicorp/terraform/internal/states"
	tfversion "github.com/hashicorp/terraform/version"
)

// File is the in-memory representation of a state file. It includes the state
// itself along with various metadata used to track changing state files for
// the same configuration over time.
type File struct {
	// TerraformVersion is the version of Terraform that wrote this state file.
	TerraformVersion *version.Version

	// Serial is incremented on any operation that modifies
	// the State file. It is used to detect potentially conflicting
	// updates.
	Serial uint64

	// Lineage is set when a new, blank state file is created and then
	// never updated. This allows us to determine whether the serials
	// of two states can be meaningfully compared.
	// Apart from the guarantee that collisions between two lineages
	// are very unlikely, this value is opaque and external callers
	// should only compare lineage strings byte-for-byte for equality.
	Lineage string

	// State is the actual state represented by this file.
	State *states.State
}

func New(state *states.State, lineage string, serial uint64) *File {
	// To make life easier on callers, we'll accept a nil state here and just
	// allocate an empty one, which is required for this file to be successfully
	// written out.
	if state == nil {
		state = states.NewState()
	}

	return &File{
		TerraformVersion: tfversion.SemVer,
		State:            state,
		Lineage:          lineage,
		Serial:           serial,
	}
}

// DeepCopy is a convenience method to create a new File object whose state
// is a deep copy of the receiver's, as implemented by states.State.DeepCopy.
func (f *File) DeepCopy() *File {
	if f == nil {
		return nil
	}
	return &File{
		TerraformVersion: f.TerraformVersion,
		Serial:           f.Serial,
		Lineage:          f.Lineage,
		State:            f.State.DeepCopy(),
	}
}

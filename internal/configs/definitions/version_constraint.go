// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
)

// VersionConstraint represents a version constraint on some resource
// (e.g. Terraform Core, a provider, a module, ...) that carries with it
// a source range so that a helpful diagnostic can be printed in the event
// that a particular constraint does not match.
type VersionConstraint struct {
	Required  version.Constraints
	DeclRange hcl.Range
}

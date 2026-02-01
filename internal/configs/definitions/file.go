// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package definitions

import (
	"github.com/hashicorp/terraform/internal/experiments"
)

// File describes the contents of a single configuration file.
//
// Individual files are not usually used alone, but rather combined together
// with other files (conventionally, those in the same directory) to produce
// a *Module, using NewModule.
//
// At the level of an individual file we represent directly the structural
// elements present in the file, without any attempt to detect conflicting
// declarations. A File object can therefore be used for some basic static
// analysis of individual elements, but must be built into a Module to detect
// duplicate declarations.
type File struct {
	CoreVersionConstraints []VersionConstraint

	ActiveExperiments experiments.Set

	Backends          []*Backend
	StateStores       []*StateStore
	CloudConfigs      []*CloudConfig
	ProviderConfigs   []*Provider
	ProviderMetas     []*ProviderMeta
	RequiredProviders []*RequiredProviders

	Variables []*Variable
	Locals    []*Local
	Outputs   []*Output

	ModuleCalls []*ModuleCall

	ManagedResources   []*Resource
	DataResources      []*Resource
	EphemeralResources []*Resource

	Moved   []*Moved
	Removed []*Removed
	Import  []*Import

	Checks  []*Check
	Actions []*Action
}

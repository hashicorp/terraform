// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	tfaddr "github.com/hashicorp/terraform-registry-address"
)

// A ModulePackage represents a physical location where Terraform can retrieve
// a module package, which is an archive, repository, or other similar
// container which delivers the source code for one or more Terraform modules.
//
// A ModulePackage is a string in go-getter's address syntax. By convention,
// we use ModulePackage-typed values only for the result of successfully
// running the go-getter "detectors", which produces an address string which
// includes an explicit installation method prefix along with an address
// string in the format expected by that installation method.
//
// Note that although the "detector" phase of go-getter does do some simple
// normalization in certain cases, it isn't generally possible to compare
// two ModulePackage values to decide if they refer to the same package. Two
// equal ModulePackage values represent the same package, but there might be
// other non-equal ModulePackage values that also refer to that package, and
// there is no reliable way to determine that.
//
// Don't convert a user-provided string directly to ModulePackage. Instead,
// use ParseModuleSource with a remote module address and then access the
// ModulePackage value from the result, making sure to also handle the
// selected subdirectory if any. You should convert directly to ModulePackage
// only for a string that is hard-coded into the program (e.g. in a unit test)
// where you've ensured that it's already in the expected syntax.
type ModulePackage string

func (p ModulePackage) String() string {
	return string(p)
}

// A ModuleRegistryPackage is an extra indirection over a ModulePackage where
// we use a module registry to translate a more symbolic address (and
// associated version constraint given out of band) into a physical source
// location.
//
// ModuleRegistryPackage is distinct from ModulePackage because they have
// disjoint use-cases: registry package addresses are only used to query a
// registry in order to find a real module package address. These being
// distinct is intended to help future maintainers more easily follow the
// series of steps in the module installer, with the help of the type checker.
type ModuleRegistryPackage = tfaddr.ModulePackage

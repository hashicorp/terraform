// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package version

import "runtime/debug"

// See the docs for InterestingDependencies to understand what "interesting" is
// intended to mean here. We should keep this set relatively small to avoid
// bloating the logs too much.
var interestingDependencies = map[string]struct{}{
	"github.com/hashicorp/hcl/v2":            {},
	"github.com/zclconf/go-cty":              {},
	"github.com/hashicorp/go-tfe":            {},
	"github.com/hashicorp/terraform-svchost": {},
}

// InterestingDependencies returns the compiled-in module version info for
// a small number of dependencies that Terraform uses broadly and which we
// tend to upgrade relatively often as part of improvements to Terraform.
//
// The set of dependencies this reports might change over time if our
// opinions change about what's "interesting". This is here only to create
// a small number of extra annotations in a debug log to help us more easily
// cross-reference bug reports with dependency changelogs.
func InterestingDependencies() []*debug.Module {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		// Weird to not be built in module mode, but not a big deal.
		return nil
	}

	ret := make([]*debug.Module, 0, len(interestingDependencies))

	for _, mod := range info.Deps {
		if _, ok := interestingDependencies[mod.Path]; !ok {
			continue
		}
		if mod.Replace != nil {
			mod = mod.Replace
		}
		ret = append(ret, mod)
	}

	return ret
}

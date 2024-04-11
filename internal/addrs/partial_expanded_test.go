// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"fmt"
	"testing"
)

func TestPartialExpandedResourceIsTargetedBy(t *testing.T) {

	tcs := []struct {
		per    string
		target string
		want   bool
	}{
		{
			"test.a",
			"test.a",
			true,
		},
		{
			"test.a",
			"test.a[0]",
			true,
		},
		{
			"test.a[\"*\"]",
			"test.a",
			true,
		},
		{
			"test.a[\"*\"]",
			"test.a[0]",
			true,
		},
		{
			"test.a[\"*\"]",
			"test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[1].test.a",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod.test.a[\"*\"]",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod.test.a[\"*\"]",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod.test.a[\"*\"]",
			"module.mod.test.a[\"key\"]",
			true,
		},
		{
			"module.mod.test.a[\"*\"]",
			"module.mod[0].test.a",
			false,
		},
		{
			"module.mod[1].test.a[\"*\"]",
			"module.mod[\"key\"].test.a[0]",
			false,
		},
		{
			"module.mod[\"*\"].test.a",
			"module.mod.test.a",
			true,
		},
		{
			"module.mod[\"*\"].test.a",
			"module.mod.test.a[0]",
			true,
		},
		{
			"module.mod[\"*\"].test.a",
			"module.mod[0].test.a",
			true,
		},
		{
			"module.mod[\"*\"].test.a",
			"module.mod[\"key\"].test.a",
			true,
		},
	}

	for _, tc := range tcs {
		t.Run(fmt.Sprintf("PartialResource(%q).IsTargetedBy(%q)", tc.per, tc.target), func(t *testing.T) {
			per := mustParseAbsResourceInstanceStr(tc.per).PartialResource()
			target := mustParseTarget(tc.target)

			got := per.IsTargetedBy(target)
			if got != tc.want {
				t.Errorf("PartialResource(%q).IsTargetedBy(%q): got %v; want %v", tc.per, tc.target, got, tc.want)
			}
		})
	}

}

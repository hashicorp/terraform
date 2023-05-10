// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package globalref

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestAnalyzerContributingResources(t *testing.T) {
	azr := testAnalyzer(t, "contributing-resources")

	tests := map[string]struct {
		StartRefs func() []Reference
		WantAddrs []string
	}{
		"root output 'network'": {
			func() []Reference {
				return azr.ReferencesFromOutputValue(
					addrs.OutputValue{Name: "network"}.Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				`data.test_thing.environment`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc`,
			},
		},
		"root output 'c10s_url'": {
			func() []Reference {
				return azr.ReferencesFromOutputValue(
					addrs.OutputValue{Name: "c10s_url"}.Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				`data.test_thing.environment`,
				`module.compute.test_thing.load_balancer`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc`,

				// NOTE: module.compute.test_thing.controller isn't here
				// because we can see statically that the output value refers
				// only to the "string" attribute of
				// module.compute.test_thing.load_balancer , and so we
				// don't consider references inside the "list" blocks.
			},
		},
		"module.compute.test_thing.load_balancer": {
			func() []Reference {
				return azr.ReferencesFromResourceInstance(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "load_balancer",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("compute", addrs.NoKey)),
				)
			},
			[]string{
				`data.test_thing.environment`,
				`module.compute.test_thing.controller`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc`,
			},
		},
		"data.test_thing.environment": {
			func() []Reference {
				return azr.ReferencesFromResourceInstance(
					addrs.Resource{
						Mode: addrs.DataResourceMode,
						Type: "test_thing",
						Name: "environment",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				// Nothing! This one only refers to an input variable.
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			startRefs := test.StartRefs()
			addrs := azr.ContributingResources(startRefs...)

			want := test.WantAddrs
			got := make([]string, len(addrs))
			for i, addr := range addrs {
				got[i] = addr.String()
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong addresses\n%s", diff)
			}
		})
	}
}

func TestAnalyzerContributingResourceAttrs(t *testing.T) {
	azr := testAnalyzer(t, "contributing-resources")

	tests := map[string]struct {
		StartRefs func() []Reference
		WantAttrs []string
	}{
		"root output 'network'": {
			func() []Reference {
				return azr.ReferencesFromOutputValue(
					addrs.OutputValue{Name: "network"}.Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				`data.test_thing.environment.any.base_cidr_block`,
				`data.test_thing.environment.any.subnet_count`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc.string`,
			},
		},
		"root output 'c10s_url'": {
			func() []Reference {
				return azr.ReferencesFromOutputValue(
					addrs.OutputValue{Name: "c10s_url"}.Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				`data.test_thing.environment.any.base_cidr_block`,
				`data.test_thing.environment.any.subnet_count`,
				`module.compute.test_thing.load_balancer.string`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc.string`,
			},
		},
		"module.compute.test_thing.load_balancer": {
			func() []Reference {
				return azr.ReferencesFromResourceInstance(
					addrs.Resource{
						Mode: addrs.ManagedResourceMode,
						Type: "test_thing",
						Name: "load_balancer",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance.Child("compute", addrs.NoKey)),
				)
			},
			[]string{
				`data.test_thing.environment.any.base_cidr_block`,
				`data.test_thing.environment.any.subnet_count`,
				`module.compute.test_thing.controller`,
				`module.network.test_thing.subnet`,
				`module.network.test_thing.vpc.string`,
			},
		},
		"data.test_thing.environment": {
			func() []Reference {
				return azr.ReferencesFromResourceInstance(
					addrs.Resource{
						Mode: addrs.DataResourceMode,
						Type: "test_thing",
						Name: "environment",
					}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
				)
			},
			[]string{
				// Nothing! This one only refers to an input variable.
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			startRefs := test.StartRefs()
			refs := azr.ContributingResourceReferences(startRefs...)

			want := test.WantAttrs
			got := make([]string, len(refs))
			for i, ref := range refs {
				resAttr, ok := ref.ResourceAttr()
				if !ok {
					t.Errorf("%s is not a resource attr reference", resAttr.DebugString())
					continue
				}
				got[i] = resAttr.DebugString()
			}

			sort.Strings(got)

			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("wrong addresses\n%s", diff)
			}
		})
	}
}

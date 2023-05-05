// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package jsonconfig

import (
	"testing"
)

func TestFindSourceProviderConfig(t *testing.T) {
	tests := []struct {
		StartKey    string
		FullName    string
		ProviderMap map[string]providerConfig
		Want        string
	}{
		{
			StartKey:    "null",
			FullName:    "hashicorp/null",
			ProviderMap: map[string]providerConfig{},
			Want:        "",
		},
		{
			StartKey: "null",
			FullName: "hashicorp/null",
			ProviderMap: map[string]providerConfig{
				"null": {
					Name:          "null",
					FullName:      "hashicorp/null",
					ModuleAddress: "",
				},
			},
			Want: "null",
		},
		{
			StartKey: "null2",
			FullName: "hashicorp/null",
			ProviderMap: map[string]providerConfig{
				"null": {
					Name:          "null",
					FullName:      "hashicorp/null",
					ModuleAddress: "",
				},
			},
			Want: "",
		},
		{
			StartKey: "null",
			FullName: "hashicorp2/null",
			ProviderMap: map[string]providerConfig{
				"null": {
					Name:          "null",
					FullName:      "hashicorp/null",
					ModuleAddress: "",
				},
			},
			Want: "",
		},
		{
			StartKey: "module.a:null",
			FullName: "hashicorp/null",
			ProviderMap: map[string]providerConfig{
				"null": {
					Name:          "null",
					FullName:      "hashicorp/null",
					ModuleAddress: "",
				},
				"module.a:null": {
					Name:          "module.a:null",
					FullName:      "hashicorp/null",
					ModuleAddress: "module.a",
					parentKey:     "null",
				},
			},
			Want: "null",
		},
		{
			StartKey: "module.a:null",
			FullName: "hashicorp2/null",
			ProviderMap: map[string]providerConfig{
				"null": {
					Name:          "null",
					FullName:      "hashicorp/null",
					ModuleAddress: "",
				},
				"module.a:null": {
					Name:          "module.a:null",
					FullName:      "hashicorp2/null",
					ModuleAddress: "module.a",
					parentKey:     "null",
				},
			},
			Want: "module.a:null",
		},
	}

	for _, test := range tests {
		got := findSourceProviderKey(test.StartKey, test.FullName, test.ProviderMap)
		if got != test.Want {
			t.Errorf("wrong result:\nGot: %#v\nWant: %#v\n", got, test.Want)
		}
	}
}

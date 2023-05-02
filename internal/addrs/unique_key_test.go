// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
	"testing"
)

// TestUniqueKeyer aims to ensure that all of the types that have unique keys
// will continue to meet the UniqueKeyer contract under future changes.
//
// If you add a new implementation of UniqueKey, consider adding a test case
// for it here.
func TestUniqueKeyer(t *testing.T) {
	tests := []UniqueKeyer{
		CountAttr{Name: "index"},
		ForEachAttr{Name: "key"},
		TerraformAttr{Name: "workspace"},
		PathAttr{Name: "module"},
		InputVariable{Name: "foo"},
		ModuleCall{Name: "foo"},
		ModuleCallInstance{
			Call: ModuleCall{Name: "foo"},
			Key:  StringKey("a"),
		},
		ModuleCallOutput{
			Call: ModuleCall{Name: "foo"},
			Name: "bar",
		},
		ModuleCallInstanceOutput{
			Call: ModuleCallInstance{
				Call: ModuleCall{Name: "foo"},
				Key:  StringKey("a"),
			},
			Name: "bar",
		},
		Resource{
			Mode: ManagedResourceMode,
			Type: "foo",
			Name: "bar",
		},
		ResourceInstance{
			Resource: Resource{
				Mode: ManagedResourceMode,
				Type: "foo",
				Name: "bar",
			},
			Key: IntKey(1),
		},
		RootModuleInstance,
		RootModuleInstance.Child("foo", NoKey),
		RootModuleInstance.ResourceInstance(
			DataResourceMode,
			"boop",
			"beep",
			NoKey,
		),
		Self,
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s", test), func(t *testing.T) {
			a := test.UniqueKey()
			b := test.UniqueKey()

			// The following comparison will panic if the unique key is not
			// of a comparable type.
			if a != b {
				t.Fatalf("the two unique keys are not equal\na: %#v\b: %#v", a, b)
			}
		})
	}
}

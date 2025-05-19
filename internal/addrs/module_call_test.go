// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"
)

func TestAbsModuleCallOutput(t *testing.T) {
	testCases := map[string]struct {
		input    AbsModuleCall
		expected string
	}{
		"simple": {
			input: AbsModuleCall{
				Module: ModuleInstance{},
				Call: ModuleCall{
					Name: "hello",
				},
			},
			expected: "module.hello.foo",
		},
		"nested": {
			input: AbsModuleCall{
				Module: ModuleInstance{
					ModuleInstanceStep{
						Name:        "child",
						InstanceKey: NoKey,
					},
				},
				Call: ModuleCall{
					Name: "hello",
				},
			},
			expected: "module.child.module.hello.foo",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			output := tc.input.Output("foo")
			if output.String() != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, output.String())
			}
		})
	}
}

func TestAbsModuleCallOutput_ConfigOutputValue(t *testing.T) {
	testCases := map[string]struct {
		input    AbsModuleCall
		expected string
	}{
		"simple": {
			input: AbsModuleCall{
				Module: ModuleInstance{},
				Call: ModuleCall{
					Name: "hello",
				},
			},
			expected: "module.hello.output.foo",
		},
		"nested": {
			input: AbsModuleCall{
				Module: ModuleInstance{
					ModuleInstanceStep{
						Name:        "child",
						InstanceKey: NoKey,
					},
				},
				Call: ModuleCall{
					Name: "hello",
				},
			},
			expected: "module.child.module.hello.output.foo",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			output := tc.input.Output("foo").ConfigOutputValue()
			if output.String() != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, output.String())
			}
		})
	}
}

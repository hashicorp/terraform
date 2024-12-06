// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"
)

func TestModuleCallOutput_ConfigOutputValue(t *testing.T) {
	for name, tc := range map[string]struct {
		input   ModuleCallOutput
		expeced ConfigOutputValue
	}{
		"simple": {
			input: ModuleCallOutput{
				Call: ModuleCall{
					Name: "child_module",
				},
				Name: "output_name",
			},
			expeced: ConfigOutputValue{
				Module: []string{"child_module"},
				OutputValue: OutputValue{
					Name: "output_name",
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			result := tc.input.ConfigOutputValue()

			if !result.Module.Equal(tc.expeced.Module) {
				t.Fatalf("different module, expected %#v, got %#v", tc.expeced.Module, result.Module)
			}
			if !result.OutputValue.Equal(tc.expeced.OutputValue) {
				t.Fatalf("different output, expected %#v, got %#v", tc.expeced.OutputValue, result.OutputValue)
			}
		})
	}
}

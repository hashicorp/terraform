// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package addrs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/go-test/deep"
)

func TestAbsOutputValueInstanceEqual_true(t *testing.T) {
	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	ovs := []AbsOutputValue{
		foo.OutputValue("a"),
		foobar.OutputValue("b"),
	}
	for _, r := range ovs {
		t.Run(r.String(), func(t *testing.T) {
			if !r.Equal(r) {
				t.Fatalf("expected %#v to be equal to itself", r)
			}
		})
	}
}

func TestAbsOutputValueInstanceEqual_false(t *testing.T) {
	foo, diags := ParseModuleInstanceStr("module.foo")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}
	foobar, diags := ParseModuleInstanceStr("module.foo[1].module.bar")
	if len(diags) > 0 {
		t.Fatalf("unexpected diags: %s", diags.Err())
	}

	testCases := []struct {
		left  AbsOutputValue
		right AbsOutputValue
	}{
		{
			foo.OutputValue("a"),
			foo.OutputValue("b"),
		},
		{
			foo.OutputValue("a"),
			foobar.OutputValue("a"),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s = %s", tc.left, tc.right), func(t *testing.T) {
			if tc.left.Equal(tc.right) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.left, tc.right)
			}

			if tc.right.Equal(tc.left) {
				t.Fatalf("expected %#v not to be equal to %#v", tc.right, tc.left)
			}
		})
	}
}

func TestParseAbsOutputValueStr(t *testing.T) {
	tests := map[string]struct {
		want    AbsOutputValue
		wantErr string
	}{
		"module.foo": {
			wantErr: "An output name is required",
		},
		"module.foo.output": {
			wantErr: "An output name is required",
		},
		"module.foo.boop.beep": {
			wantErr: "Output address must start with \"output.\"",
		},
		"module.foo.output[0]": {
			wantErr: "An output name is required",
		},
		"output": {
			wantErr: "An output name is required",
		},
		"output[0]": {
			wantErr: "An output name is required",
		},
		"output.boop": {
			want: AbsOutputValue{
				Module: RootModuleInstance,
				OutputValue: OutputValue{
					Name: "boop",
				},
			},
		},
		"module.foo.output.beep": {
			want: AbsOutputValue{
				Module: mustParseModuleInstanceStr("module.foo"),
				OutputValue: OutputValue{
					Name: "beep",
				},
			},
		},
	}

	for input, tc := range tests {
		t.Run(input, func(t *testing.T) {
			got, diags := ParseAbsOutputValueStr(input)
			for _, problem := range deep.Equal(got, tc.want) {
				t.Errorf(problem)
			}
			if len(diags) > 0 {
				gotErr := diags.Err().Error()
				if tc.wantErr == "" {
					t.Errorf("got error, expected success: %s", gotErr)
				} else if !strings.Contains(gotErr, tc.wantErr) {
					t.Errorf("unexpected error\n got: %s\nwant: %s", gotErr, tc.wantErr)
				}
			} else {
				if tc.wantErr != "" {
					t.Errorf("got success, expected error: %s", tc.wantErr)
				}
			}
		})
	}
}

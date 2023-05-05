// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"flag"
	"reflect"
	"testing"
)

func TestFlagStringKV_impl(t *testing.T) {
	var _ flag.Value = new(FlagStringKV)
}

func TestFlagStringKV(t *testing.T) {
	cases := []struct {
		Input  string
		Output map[string]string
		Error  bool
	}{
		{
			"key=value",
			map[string]string{"key": "value"},
			false,
		},

		{
			"key=",
			map[string]string{"key": ""},
			false,
		},

		{
			"key=foo=bar",
			map[string]string{"key": "foo=bar"},
			false,
		},

		{
			"map.key=foo",
			map[string]string{"map.key": "foo"},
			false,
		},

		{
			"key",
			nil,
			true,
		},

		{
			"key=/path",
			map[string]string{"key": "/path"},
			false,
		},
	}

	for _, tc := range cases {
		f := new(FlagStringKV)
		err := f.Set(tc.Input)
		if err != nil != tc.Error {
			t.Fatalf("bad error. Input: %#v\n\nError: %s", tc.Input, err)
		}

		actual := map[string]string(*f)
		if !reflect.DeepEqual(actual, tc.Output) {
			t.Fatalf("bad: %#v", actual)
		}
	}
}

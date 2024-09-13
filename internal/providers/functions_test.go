// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"

	// set the correct global logger for tests
	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestFunctionCache(t *testing.T) {
	testAddr := addrs.NewDefaultProvider("test")

	type testCall struct {
		provider addrs.Provider
		name     string
		args     []cty.Value
		result   cty.Value
	}

	tests := []struct {
		first, second testCall
		expectErr     bool
	}{
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.False,
			},
			// result changed from true => false
			expectErr: true,
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.UnknownVal(cty.Bool),
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.False,
			},
			// result changed from unknown => false
			expectErr: true,
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
			second: testCall{
				provider: addrs.NewDefaultProvider("fake"),
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.False,
			},
			// OK because provider changed
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
			second: testCall{
				provider: testAddr,
				name:     "func",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.False,
			},
			// OK because function name changed
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok")},
				result:   cty.True,
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok"), cty.StringVal("ok")},
				result:   cty.False,
			},
			// OK because args changed
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args: []cty.Value{cty.ObjectVal(map[string]cty.Value{
					"attr": cty.NumberIntVal(1),
				})},
				result: cty.True,
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args: []cty.Value{cty.ObjectVal(map[string]cty.Value{
					"attr": cty.NumberIntVal(2),
				})},
				result: cty.False,
			},
			// OK because args changed
		},
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args: []cty.Value{cty.UnknownVal(cty.Object(map[string]cty.Type{
					"attr": cty.Number,
				}))},
				result: cty.UnknownVal(cty.Bool),
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args: []cty.Value{cty.ObjectVal(map[string]cty.Value{
					"attr": cty.NumberIntVal(2),
				})},
				result: cty.False,
			},
			// OK because args changed from unknown to known
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			results := NewFunctionResultsTable(nil)
			err := results.checkPrior(test.first.provider, test.first.name, test.first.args, test.first.result)
			if err != nil {
				t.Fatal("error on first call!", err)
			}

			err = results.checkPrior(test.second.provider, test.second.name, test.second.args, test.second.result)

			if err != nil && !test.expectErr {
				t.Fatal(err)
			}

			// reload the data to ensure we validate identically
			newResults := NewFunctionResultsTable(results.GetHashes())

			originalErr := err != nil
			reloadedErr := newResults.checkPrior(test.second.provider, test.second.name, test.second.args, test.second.result) != nil

			if originalErr != reloadedErr {
				t.Fatalf("original check returned err:%t, reloaded check returned err:%t", originalErr, reloadedErr)
			}
		})
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package lang

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/lang/marks"
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
			expectErr: false,
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
		{
			first: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok").Mark(marks.Sensitive)},
				result:   cty.StringVal("a"),
			},
			second: testCall{
				provider: testAddr,
				name:     "fun",
				args:     []cty.Value{cty.StringVal("ok").Mark(marks.Ephemeral)},
				result:   cty.StringVal("b"),
			},
			// The marks differ but the underlying arguments do not, so this is
			// still the same call and the differing result is inconsistent.
			expectErr: true,
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			results := NewFunctionResultsTable(nil)
			err := results.CheckPriorProvider(test.first.provider, test.first.name, test.first.args, test.first.result)
			if err != nil {
				t.Fatal("error on first call!", err)
			}

			err = results.CheckPriorProvider(test.second.provider, test.second.name, test.second.args, test.second.result)

			if err != nil && !test.expectErr {
				t.Fatal(err)
			}
			if err == nil && test.expectErr {
				t.Fatal("expected error")
			}

			// reload the data to ensure we validate identically
			newResults := NewFunctionResultsTable(results.GetHashes())

			originalErr := err != nil
			reloadedErr := newResults.CheckPriorProvider(test.second.provider, test.second.name, test.second.args, test.second.result) != nil

			if originalErr != reloadedErr {
				t.Fatalf("original check returned err:%t, reloaded check returned err:%t", originalErr, reloadedErr)
			}
		})
	}
}

// A value carrying two or more marks must hash consistently. cty renders a
// single mark deterministically, but falls back to ValueMarks.GoString for two
// or more, which iterates a Go map without sorting. Since Go randomizes map
// iteration order, hashing the marked value directly gave a different result
// from one call to the next and reported spurious inconsistencies.
//
// Every combination of mark types is covered, because the defect is in how the
// set of marks is rendered and so is indifferent to which marks are present.
// Each case is repeated often enough that an unstable hash is certain to be
// caught.
func TestFunctionCacheMarkedValues(t *testing.T) {
	deprecated := marks.NewDeprecation("deprecated", "test")
	alsoDeprecated := marks.NewDeprecation("deprecated by something else", "test")

	tests := map[string]cty.ValueMarks{
		"sensitive and ephemeral":   cty.NewValueMarks(marks.Sensitive, marks.Ephemeral),
		"sensitive and type":        cty.NewValueMarks(marks.Sensitive, marks.TypeType),
		"sensitive and deprecated":  cty.NewValueMarks(marks.Sensitive, deprecated),
		"ephemeral and deprecated":  cty.NewValueMarks(marks.Ephemeral, deprecated),
		"two distinct deprecations": cty.NewValueMarks(deprecated, alsoDeprecated),
		"three marks": cty.NewValueMarks(
			marks.Sensitive, marks.Ephemeral, deprecated,
		),
	}

	for name, valMarks := range tests {
		t.Run(name, func(t *testing.T) {
			args := []cty.Value{cty.StringVal("template").WithMarks(valMarks)}
			result := cty.StringVal("rendered").WithMarks(valMarks)

			results := NewFunctionResultsTable(nil)
			for i := 0; i < 100; i++ {
				if err := results.CheckPrior("fun", args, result); err != nil {
					t.Fatalf("call %d returned an unexpected error: %s", i, err)
				}
			}

			// Results reloaded from a plan must agree with the in-memory table.
			reloaded := NewFunctionResultsTable(results.GetHashes())
			for i := 0; i < 100; i++ {
				if err := reloaded.CheckPrior("fun", args, result); err != nil {
					t.Fatalf("reloaded call %d returned an unexpected error: %s", i, err)
				}
			}
		})
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package addrs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/tfdiags"
)

// TestParseAddressInstanceKeys checks that each kind of address accepts or
// rejects wildcard and unknown instance keys, which are parsed by the same
// shared functions for every address type.
func TestParseAddressInstanceKeys(t *testing.T) {
	unknownKey := func(t *testing.T, addr string, step int) hcl.Traversal {
		traversal := mustParseTraversalPattern(t, addr)
		traversal[step] = hcl.TraverseIndex{Key: cty.UnknownVal(cty.String)}
		return traversal
	}

	for name, tc := range map[string]struct {
		traversal func(t *testing.T) hcl.Traversal
		parse     func(hcl.Traversal) (string, tfdiags.Diagnostics)
		want      string
		wantErr   string
	}{
		"target pattern": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, "module.a[*].test_instance.foo[*]")
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				target, diags := ParseTarget(traversal)
				return target.String(), diags
			},
			want: "module.a[*].test_instance.foo[*]",
		},
		"wildcard module in concrete target": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, "module.a[*].test_instance.foo")
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseAbsTargetable(traversal)
				return "", diags
			},
			wantErr: "Invalid address: The module instance key cannot be a wildcard in this address.",
		},
		"unknown module key in concrete target": {
			traversal: func(t *testing.T) hcl.Traversal {
				return unknownKey(t, "module.a[0].test_instance.foo", 2)
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseAbsTargetable(traversal)
				return "", diags
			},
			wantErr: "Invalid address: The module instance key must be known in this address.",
		},
		"invalid module key": {
			traversal: func(t *testing.T) hcl.Traversal {
				traversal := mustParseTraversalPattern(t, "module.a[0]")
				traversal[2] = hcl.TraverseIndex{Key: cty.True}
				return traversal
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseModuleInstance(traversal)
				return "", diags
			},
			wantErr: "Invalid address: Invalid module instance key: either a string or an integer is required.",
		},
		"wildcard resource in move endpoint": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, "test_instance.foo[*]")
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseMoveEndpoint(traversal)
				return "", diags
			},
			wantErr: "Invalid address: The resource instance key cannot be a wildcard in this address.",
		},
		"wildcard module in move endpoint": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, "module.a[*]")
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseMoveEndpoint(traversal)
				return "", diags
			},
			wantErr: "Invalid address: The module instance key cannot be a wildcard in this address.",
		},
		"wildcard action instance": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, "action.test_action.foo[*]")
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				_, diags := ParseAbsActionInstance(traversal)
				return "", diags
			},
			wantErr: "Invalid address: The action instance key cannot be a wildcard in this address.",
		},
		"partial-expanded resource with wildcards": {
			traversal: func(t *testing.T) hcl.Traversal {
				return mustParseTraversalPattern(t, `module.a[*].module.b["x"].test_instance.foo[1]`)
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				per, _, diags := ParsePartialExpandedResource(traversal)
				return per.String(), diags
			},
			want: "module.a[*].module.b[*].test_instance.foo[*]",
		},
		"partial-expanded resource with an unknown key": {
			traversal: func(t *testing.T) hcl.Traversal {
				return unknownKey(t, `module.a[0].module.b["x"].test_instance.foo`, 5)
			},
			parse: func(traversal hcl.Traversal) (string, tfdiags.Diagnostics) {
				per, _, diags := ParsePartialExpandedResource(traversal)
				return per.String(), diags
			},
			want: "module.a[0].module.b[*].test_instance.foo[*]",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, diags := tc.parse(tc.traversal(t))
			if tc.wantErr != "" {
				if !diags.HasErrors() {
					t.Fatalf("unexpected success\nwant error: %s", tc.wantErr)
				}
				if gotErr := diags.Err().Error(); gotErr != tc.wantErr {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", gotErr, tc.wantErr)
				}
				return
			}
			if diags.HasErrors() {
				t.Fatalf("unexpected error: %s", diags.Err())
			}
			if got != tc.want {
				t.Errorf("wrong result\ngot:  %s\nwant: %s", got, tc.want)
			}
		})
	}
}

func mustParseTraversalPattern(t *testing.T, s string) hcl.Traversal {
	t.Helper()
	traversal, diags := hclsyntax.ParseTraversalPartial([]byte(s), "", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("invalid traversal %q: %s", s, diags.Error())
	}
	return traversal
}

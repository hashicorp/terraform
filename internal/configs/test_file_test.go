package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestTestRun_Validate(t *testing.T) {
	tcs := map[string]struct {
		expectedFailures []string
		diagnostic       string
	}{
		"empty": {},
		"supports_expected": {
			expectedFailures: []string{
				"check.expected_check",
				"var.expected_var",
				"output.expected_output",
				"test_resource.resource",
				"resource.test_resource.resource",
				"data.test_resource.resource",
			},
		},
		"count": {
			expectedFailures: []string{
				"count.index",
			},
			diagnostic: "You cannot expect failures from count.index. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.",
		},
		"foreach": {
			expectedFailures: []string{
				"each.key",
			},
			diagnostic: "You cannot expect failures from each.key. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.",
		},
		"local": {
			expectedFailures: []string{
				"local.value",
			},
			diagnostic: "You cannot expect failures from local.value. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.",
		},
		"module": {
			expectedFailures: []string{
				"module.my_module",
			},
			diagnostic: "You cannot expect failures from module.my_module. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.",
		},
		"path": {
			expectedFailures: []string{
				"path.walk",
			},
			diagnostic: "You cannot expect failures from path.walk. You can only expect failures from checkable objects such as input variables, output values, check blocks, managed resources and data sources.",
		},
	}
	for name, tc := range tcs {
		t.Run(name, func(t *testing.T) {
			run := &TestRun{}
			for _, addr := range tc.expectedFailures {
				run.ExpectFailures = append(run.ExpectFailures, parseTraversal(t, addr))
			}

			diags := run.Validate()

			if len(diags) > 1 {
				t.Fatalf("too many diags: %d", len(diags))
			}

			if len(tc.diagnostic) == 0 {
				if len(diags) != 0 {
					t.Fatalf("expected no diags but got: %s", diags[0].Description().Detail)
				}

				return
			}

			if diff := cmp.Diff(tc.diagnostic, diags[0].Description().Detail); len(diff) > 0 {
				t.Fatalf("unexpected diff:\n%s", diff)
			}
		})
	}
}

func parseTraversal(t *testing.T, addr string) hcl.Traversal {
	t.Helper()

	traversal, diags := hclsyntax.ParseTraversalAbs([]byte(addr), "", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatalf("invalid address: %s", diags.Error())
	}
	return traversal
}

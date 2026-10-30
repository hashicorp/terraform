// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"

	"github.com/hashicorp/terraform/internal/addrs"
)

func TestTestRunOptions_targetPatterns(t *testing.T) {
	want := []string{
		"test_resource.a",
		"module.child[*].test_resource.b",
		"module.child.module.grandchild[1]",
	}

	sources := map[string]func() (hcl.Body, hcl.Diagnostics){
		"native": func() (hcl.Body, hcl.Diagnostics) {
			f, diags := hclsyntax.ParseConfig([]byte(`
run "test" {
  plan_options {
    target = [
      test_resource.a,
      module.child[*].test_resource.b,
      module.child.module.grandchild[1],
    ]
  }
}
`), "main.tftest.hcl", hcl.InitialPos)
			return f.Body, diags
		},
		"json": func() (hcl.Body, hcl.Diagnostics) {
			f, diags := json.Parse([]byte(`{
  "run": {
    "test": {
      "plan_options": {
        "target": [
          "test_resource.a",
          "module.child[*].test_resource.b",
          "module.child.module.grandchild[1]"
        ]
      }
    }
  }
}`), "main.tftest.json")
			return f.Body, diags
		},
	}

	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			body, diags := source()
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			file, diags := loadTestFile(body, false)
			if diags.HasErrors() {
				t.Fatal(diags.Error())
			}

			var got []string
			for _, traversal := range file.Runs[0].Options.Target {
				target, diags := addrs.ParseTarget(traversal)
				if diags.HasErrors() {
					t.Fatal(diags.Err())
				}
				got = append(got, target.String())
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Fatalf("wrong targets\n%s", diff)
			}
		})
	}
}

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

			diags := run.Validate(nil)

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

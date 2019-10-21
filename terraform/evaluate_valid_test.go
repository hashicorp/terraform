package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/lang"
)

func TestStaticValidateReferences(t *testing.T) {
	tests := []struct {
		Ref     string
		WantErr string
	}{
		{
			"aws_instance.no_count",
			``,
		},
		{
			"aws_instance.count",
			``,
		},
		{
			"aws_instance.count[0]",
			``,
		},
		{
			"aws_instance.nonexist",
			`Reference to undeclared resource: A managed resource "aws_instance" "nonexist" has not been declared in the root module.`,
		},
		{
			"aws_instance.no_count[0]",
			`Unexpected resource instance key: Because aws_instance.no_count does not have "count" or "for_each" set, references to it must not include an index key. Remove the bracketed index to refer to the single instance of this resource.`,
		},
		{
			"aws_instance.count.foo",
			// In this case we return two errors that are somewhat redundant with
			// one another, but we'll accept that because they both report the
			// problem from different perspectives and so give the user more
			// opportunity to understand what's going on here.
			`2 problems:

- Missing resource instance key: Because aws_instance.count has "count" set, its attributes must be accessed on specific instances.

For example, to correlate with indices of a referring resource, use:
    aws_instance.count[count.index]
- Unsupported attribute: This object has no argument, nested block, or exported attribute named "foo".`,
		},
	}

	cfg := testModule(t, "static-validate-refs")
	evaluator := &Evaluator{
		Config: cfg,
		Schemas: &Schemas{
			Providers: map[string]*ProviderSchema{
				"aws": {
					ResourceTypes: map[string]*configschema.Block{
						"aws_instance": {},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Ref, func(t *testing.T) {
			traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(test.Ref), "", hcl.Pos{Line: 1, Column: 1})
			if hclDiags.HasErrors() {
				t.Fatal(hclDiags.Error())
			}

			refs, diags := lang.References([]hcl.Traversal{traversal})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			data := &evaluationStateData{
				Evaluator: evaluator,
			}

			diags = data.StaticValidateReferences(refs, nil)
			if diags.HasErrors() {
				if test.WantErr == "" {
					t.Fatalf("Unexpected diagnostics: %s", diags.Err())
				}

				gotErr := diags.Err().Error()
				if gotErr != test.WantErr {
					t.Fatalf("Wrong diagnostics\ngot:  %s\nwant: %s", gotErr, test.WantErr)
				}
				return
			}

			if test.WantErr != "" {
				t.Fatalf("Expected diagnostics, but got none\nwant: %s", test.WantErr)
			}
		})
	}
}

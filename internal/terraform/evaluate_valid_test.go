// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/langrefs"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestStaticValidateReferences(t *testing.T) {
	tests := []struct {
		Ref     string
		Src     addrs.Referenceable
		WantErr string
	}{
		{
			Ref:     "aws_instance.no_count",
			WantErr: ``,
		},
		{
			Ref:     "aws_instance.count",
			WantErr: ``,
		},
		{
			Ref:     "aws_instance.count[0]",
			WantErr: ``,
		},
		{
			Ref:     "aws_instance.nonexist",
			WantErr: `Reference to undeclared resource: A managed resource "aws_instance" "nonexist" has not been declared in the root module.`,
		},
		{
			Ref: "beep.boop",
			WantErr: `Reference to undeclared resource: A managed resource "beep" "boop" has not been declared in the root module.

Did you mean the data resource data.beep.boop?`,
		},
		{
			Ref:     "aws_instance.no_count[0]",
			WantErr: `Unexpected resource instance key: Because aws_instance.no_count does not have "count" or "for_each" set, references to it must not include an index key. Remove the bracketed index to refer to the single instance of this resource.`,
		},
		{
			Ref: "aws_instance.count.foo",
			// In this case we return two errors that are somewhat redundant with
			// one another, but we'll accept that because they both report the
			// problem from different perspectives and so give the user more
			// opportunity to understand what's going on here.
			WantErr: `2 problems:

- Missing resource instance key: Because aws_instance.count has "count" set, its attributes must be accessed on specific instances.

For example, to correlate with indices of a referring resource, use:
    aws_instance.count[count.index]
- Unsupported attribute: This object has no argument, nested block, or exported attribute named "foo".`,
		},
		{
			Ref:     "boop_instance.yep",
			WantErr: ``,
		},
		{
			Ref:     "boop_whatever.nope",
			WantErr: `Invalid resource type: A managed resource type "boop_whatever" is not supported by provider "registry.terraform.io/foobar/beep".`,
		},
		{
			Ref:     "data.boop_data.boop_nested",
			WantErr: `Reference to scoped resource: The referenced data resource "boop_data" "boop_nested" is not available from this context.`,
		},
		{
			Ref:     "ephemeral.beep.boop",
			WantErr: ``,
		},
		{
			Ref:     "ephemeral.beep.nonexistant",
			WantErr: `Reference to undeclared resource: An ephemeral resource "beep" "nonexistant" has not been declared in the root module.`,
		},
		{
			Ref:     "data.boop_data.boop_nested",
			WantErr: ``,
			Src:     addrs.Check{Name: "foo"},
		},
		{
			Ref: "run.zero",
			// This one resembles a reference to a previous run in a .tftest.hcl
			// file, but when inside a .tf file it must be understood as a
			// reference to a resource of type "run", just in case such a
			// resource type exists in some provider somewhere.
			WantErr: `Reference to undeclared resource: A managed resource "run" "zero" has not been declared in the root module.`,
		},
	}

	cfg := testModule(t, "static-validate-refs")
	evaluator := &Evaluator{
		Config: cfg,
		Plugins: schemaOnlyProvidersForTesting(map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("aws"): {
				ResourceTypes: map[string]providers.Schema{
					"aws_instance": {
						Block: &configschema.Block{},
					},
				},
			},
			addrs.MustParseProviderSourceString("foobar/beep"): {
				ResourceTypes: map[string]providers.Schema{
					// intentional mismatch between resource type prefix and provider type
					"boop_instance": {
						Block: &configschema.Block{},
					},
				},
				DataSources: map[string]providers.Schema{
					"boop_data": {
						Block: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {
									Type:     cty.String,
									Optional: true,
								},
							},
						},
					},
				},
				EphemeralResourceTypes: map[string]providers.Schema{
					"beep": {
						Block: &configschema.Block{},
					},
				},
			},
		}),
	}

	for _, test := range tests {
		t.Run(test.Ref, func(t *testing.T) {
			traversal, hclDiags := hclsyntax.ParseTraversalAbs([]byte(test.Ref), "", hcl.Pos{Line: 1, Column: 1})
			if hclDiags.HasErrors() {
				t.Fatal(hclDiags.Error())
			}

			refs, diags := langrefs.References(addrs.ParseRef, []hcl.Traversal{traversal})
			if diags.HasErrors() {
				t.Fatal(diags.Err())
			}

			diags = evaluator.StaticValidateReferences(refs, addrs.RootModule, nil, test.Src)
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

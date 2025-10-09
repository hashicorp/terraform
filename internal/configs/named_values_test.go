// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestVariableInvalidDefault(t *testing.T) {
	src := `
		variable foo {
			type = map(object({
				foo = bool
			}))

			default = {
				"thingy" = {
					foo = "string where bool is expected"
				}
			}
		}
	`

	hclF, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	_, diags = parseConfigFile(hclF.Body, nil, false, false)
	if !diags.HasErrors() {
		t.Fatal("unexpected success; want error")
	}

	for _, diag := range diags {
		if diag.Severity != hcl.DiagError {
			continue
		}
		if diag.Summary != "Invalid default value for variable" {
			t.Errorf("unexpected diagnostic summary: %q", diag.Summary)
			continue
		}
		if got, want := diag.Detail, `This default value is not compatible with the variable's type constraint: ["thingy"].foo: a bool is required.`; got != want {
			t.Errorf("wrong diagnostic detault\ngot:  %s\nwant: %s", got, want)
		}
	}
}

func TestOutputDeprecation(t *testing.T) {
	src := `
		output "foo" {
			value = "bar"
			deprecated = "This output is deprecated"
		}
	`

	hclF, diags := hclsyntax.ParseConfig([]byte(src), "test.tf", hcl.InitialPos)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	b, diags := parseConfigFile(hclF.Body, nil, false, false)
	if diags.HasErrors() {
		t.Fatalf("unexpected error: %q", diags)
	}

	if !b.Outputs[0].DeprecatedSet {
		t.Fatalf("expected output to be deprecated")
	}

	if b.Outputs[0].Deprecated != "This output is deprecated" {
		t.Fatalf("expected output to have deprecation message")
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackconfig

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestDecodeInputVariableBlock_constNotSupported(t *testing.T) {
	// const = true is not supported in the stacks component language.
	// This test documents that using const produces an "Unsupported argument"
	// error from the HCL schema validation.
	src := []byte(`variable "example" {
  type  = string
  const = true
}`)
	file, diags := hclsyntax.ParseConfig(src, "test.tfcomponent.hcl", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("unexpected parse error: %s", diags.Error())
	}

	content, diags := file.Body.Content(&hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "variable", LabelNames: []string{"name"}},
		},
	})
	if diags.HasErrors() {
		t.Fatalf("unexpected schema error: %s", diags.Error())
	}

	if len(content.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(content.Blocks))
	}

	_, decodeDiags := decodeInputVariableBlock(content.Blocks[0])
	if len(decodeDiags) != 1 {
		t.Fatalf("expected exactly 1 diagnostic, got %d:\n%s", len(decodeDiags), decodeDiags.NonFatalErr())
	}

	diag := decodeDiags[0]
	if got, want := diag.Description().Summary, "Unsupported argument"; got != want {
		t.Errorf("wrong summary\ngot:  %s\nwant: %s", got, want)
	}
	if got, want := diag.Description().Detail, `An argument named "const" is not expected here.`; got != want {
		t.Errorf("wrong detail\ngot:  %s\nwant: %s", got, want)
	}
}

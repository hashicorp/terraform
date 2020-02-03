package format

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/hcl/v2"
	"github.com/mitchellh/colorstring"

	"github.com/hashicorp/terraform/tfdiags"
)

func TestDiagnosticWarningsCompact(t *testing.T) {
	var diags tfdiags.Diagnostics
	diags = diags.Append(tfdiags.SimpleWarning("foo"))
	diags = diags.Append(tfdiags.SimpleWarning("foo"))
	diags = diags.Append(tfdiags.SimpleWarning("bar"))
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source foo",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 2, Column: 1, Byte: 5},
			End:      hcl.Pos{Line: 2, Column: 1, Byte: 5},
		},
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source foo",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source.tf",
			Start:    hcl.Pos{Line: 3, Column: 1, Byte: 7},
			End:      hcl.Pos{Line: 3, Column: 1, Byte: 7},
		},
	})
	diags = diags.Append(&hcl.Diagnostic{
		Severity: hcl.DiagWarning,
		Summary:  "source bar",
		Detail:   "...",
		Subject: &hcl.Range{
			Filename: "source2.tf",
			Start:    hcl.Pos{Line: 1, Column: 1, Byte: 1},
			End:      hcl.Pos{Line: 1, Column: 1, Byte: 1},
		},
	})

	// ConsolidateWarnings groups together the ones
	// that have source location information and that
	// have the same summary text.
	diags = diags.ConsolidateWarnings(1)

	// A zero-value Colorize just passes all the formatting
	// codes back to us, so we can test them literally.
	got := DiagnosticWarningsCompact(diags, &colorstring.Colorize{})
	want := `[bold][yellow]Warnings:[reset]

- foo
- foo
- bar
- source foo
  on source.tf line 2 (and 1 more)
- source bar
  on source2.tf line 1
`
	if got != want {
		t.Errorf(
			"wrong result\ngot:\n%s\n\nwant:\n%s\n\ndiff:\n%s",
			got, want, cmp.Diff(want, got),
		)
	}
}

package arguments

import (
	"testing"

	"github.com/apparentlymart/go-shquot/shquot"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseTest(t *testing.T) {
	tests := []struct {
		Input     []string
		Want      Test
		WantError string
	}{
		{
			nil,
			Test{
				Output: TestOutput{
					JUnitXMLFile: "",
				},
			},
			``,
		},
		{
			[]string{"-invalid"},
			Test{
				Output: TestOutput{
					JUnitXMLFile: "",
				},
			},
			`flag provided but not defined: -invalid`,
		},
		{
			[]string{"-junit-xml=result.xml"},
			Test{
				Output: TestOutput{
					JUnitXMLFile: "result.xml",
				},
			},
			``,
		},
		{
			[]string{"baz"},
			Test{
				Output: TestOutput{
					JUnitXMLFile: "",
				},
			},
			`Invalid command arguments`,
		},
	}

	baseCmdline := []string{"terraform", "test"}
	for _, test := range tests {
		name := shquot.POSIXShell(append(baseCmdline, test.Input...))
		t.Run(name, func(t *testing.T) {
			t.Log(name)
			got, diags := ParseTest(test.Input)

			if test.WantError != "" {
				if len(diags) != 1 {
					t.Fatalf("got %d diagnostics; want exactly 1\n%s", len(diags), diags.Err().Error())
				}
				if diags[0].Severity() != tfdiags.Error {
					t.Fatalf("got a warning; want an error\n%s", diags.Err().Error())
				}
				if desc := diags[0].Description(); desc.Summary != test.WantError {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", desc.Summary, test.WantError)
				}
			} else {
				if len(diags) != 0 {
					t.Fatalf("got %d diagnostics; want none\n%s", len(diags), diags.Err().Error())
				}
			}

			if diff := cmp.Diff(test.Want, got); diff != "" {
				t.Errorf("wrong result\n%s", diff)
			}
		})
	}
}

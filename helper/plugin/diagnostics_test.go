package plugin

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/plugin/proto"
)

func TestDiagnostics(t *testing.T) {
	diags := diagsFromWarnsErrs(
		[]string{
			"warning 1",
			"warning 2",
		},
		[]error{
			errors.New("error 1"),
			errors.New("error 2"),
		},
	)

	expected := []*proto.Diagnostic{
		{
			Severity: proto.Diagnostic_WARNING,
			Summary:  "warning 1",
		},
		{
			Severity: proto.Diagnostic_WARNING,
			Summary:  "warning 2",
		},
		{
			Severity: proto.Diagnostic_ERROR,
			Summary:  "error 1",
		},
		{
			Severity: proto.Diagnostic_ERROR,
			Summary:  "error 2",
		},
	}

	if !cmp.Equal(expected, diags) {
		t.Fatal(cmp.Diff(expected, diags))
	}
}

package plugin

import (
	"github.com/hashicorp/terraform/plugin/proto"
)

// diagsFromWarnsErrs converts the warnings and errors return by the lagacy
// provider to diagnostics.
func diagsFromWarnsErrs(warns []string, errs []error) (diags []*proto.Diagnostic) {
	for _, w := range warns {
		diags = appendDiag(diags, w)
	}

	for _, e := range errs {
		diags = appendDiag(diags, e)
	}

	return diags
}

// appendDiag appends a new diagnostic from a warning string or an error. This
// panics if d is not a string or error.
func appendDiag(diags []*proto.Diagnostic, d interface{}) []*proto.Diagnostic {
	switch d := d.(type) {
	case error:
		diags = append(diags, &proto.Diagnostic{
			Severity: proto.Diagnostic_ERROR,
			Summary:  d.Error(),
		})
	case string:
		diags = append(diags, &proto.Diagnostic{
			Severity: proto.Diagnostic_WARNING,
			Summary:  d,
		})
	case *proto.Diagnostic:
		diags = append(diags, d)
	case []*proto.Diagnostic:
		diags = append(diags, d...)
	}
	return diags
}

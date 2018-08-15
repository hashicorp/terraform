package convert

import (
	"github.com/hashicorp/terraform/plugin/proto"
	"github.com/hashicorp/terraform/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

// WarnsAndErrorsToProto converts the warnings and errors return by the legacy
// provider to protobuf diagnostics.
func WarnsAndErrsToProto(warns []string, errs []error) (diags []*proto.Diagnostic) {
	for _, w := range warns {
		diags = AppendProtoDiag(diags, w)
	}

	for _, e := range errs {
		diags = AppendProtoDiag(diags, e)
	}

	return diags
}

// AppendProtoDiag appends a new diagnostic from a warning string or an error.
// This panics if d is not a string or error.
func AppendProtoDiag(diags []*proto.Diagnostic, d interface{}) []*proto.Diagnostic {
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

// ProtoToDiagnostics converts a list of proto.Diagnostics to a tf.Diagnostics.
func ProtoToDiagnostics(ds []*proto.Diagnostic) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics
	for _, d := range ds {
		var severity tfdiags.Severity

		switch d.Severity {
		case proto.Diagnostic_ERROR:
			severity = tfdiags.Error
		case proto.Diagnostic_WARNING:
			severity = tfdiags.Warning
		}

		var newDiag tfdiags.Diagnostic

		// if there's an attribute path, we need to create a AttributeValue diagnostic
		if d.Attribute != nil {
			path := AttributePathToPath(d.Attribute)
			newDiag = tfdiags.AttributeValue(severity, d.Summary, d.Detail, path)
		} else {
			newDiag = tfdiags.Sourceless(severity, d.Summary, d.Detail)
		}

		diags = diags.Append(newDiag)
	}

	return diags
}

// AttributePathToPath takes the proto encoded path and converts it to a cty.Path
func AttributePathToPath(ap *proto.AttributePath) cty.Path {
	var p cty.Path
	for _, step := range ap.Steps {
		switch selector := step.Selector.(type) {
		case *proto.AttributePath_Step_AttributeName:
			p = p.GetAttr(selector.AttributeName)
		case *proto.AttributePath_Step_ElementKeyString:
			p = p.Index(cty.StringVal(selector.ElementKeyString))
		case *proto.AttributePath_Step_ElementKeyInt:
			p = p.Index(cty.NumberIntVal(selector.ElementKeyInt))
		}
	}
	return p
}

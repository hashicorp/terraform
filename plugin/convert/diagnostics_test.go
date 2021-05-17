package convert

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform/internal/tfdiags"
	proto "github.com/hashicorp/terraform/internal/tfplugin5"
	"github.com/zclconf/go-cty/cty"
)

var ignoreUnexported = cmpopts.IgnoreUnexported(
	proto.Diagnostic{},
	proto.Schema_Block{},
	proto.Schema_NestedBlock{},
	proto.Schema_Attribute{},
)

func TestProtoDiagnostics(t *testing.T) {
	diags := WarnsAndErrsToProto(
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

	if !cmp.Equal(expected, diags, ignoreUnexported) {
		t.Fatal(cmp.Diff(expected, diags, ignoreUnexported))
	}
}

func TestDiagnostics(t *testing.T) {
	type diagFlat struct {
		Severity tfdiags.Severity
		Attr     []interface{}
		Summary  string
		Detail   string
	}

	tests := map[string]struct {
		Cons func([]*proto.Diagnostic) []*proto.Diagnostic
		Want []diagFlat
	}{
		"nil": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return diags
			},
			nil,
		},
		"error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "simple error",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "simple error",
				},
			},
		},
		"detailed error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "simple error",
					Detail:   "detailed error",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "simple error",
					Detail:   "detailed error",
				},
			},
		},
		"warning": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "simple warning",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "simple warning",
				},
			},
		},
		"detailed warning": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				return append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "simple warning",
					Detail:   "detailed warning",
				})
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "simple warning",
					Detail:   "detailed warning",
				},
			},
		},
		"multi error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "first error",
				}, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "second error",
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "first error",
				},
				{
					Severity: tfdiags.Error,
					Summary:  "second error",
				},
			},
		},
		"warning and error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_WARNING,
					Summary:  "warning",
				}, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "error",
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Warning,
					Summary:  "warning",
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error",
				},
			},
		},
		"attr error": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags, &proto.Diagnostic{
					Severity: proto.Diagnostic_ERROR,
					Summary:  "error",
					Detail:   "error detail",
					Attribute: &proto.AttributePath{
						Steps: []*proto.AttributePath_Step{
							{
								Selector: &proto.AttributePath_Step_AttributeName{
									AttributeName: "attribute_name",
								},
							},
						},
					},
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "error",
					Detail:   "error detail",
					Attr:     []interface{}{"attribute_name"},
				},
			},
		},
		"multi attr": {
			func(diags []*proto.Diagnostic) []*proto.Diagnostic {
				diags = append(diags,
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 1",
						Detail:   "error 1 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 2",
						Detail:   "error 2 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_WARNING,
						Summary:  "warning",
						Detail:   "warning detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_ElementKeyInt{
										ElementKeyInt: 1,
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
					&proto.Diagnostic{
						Severity: proto.Diagnostic_ERROR,
						Summary:  "error 3",
						Detail:   "error 3 detail",
						Attribute: &proto.AttributePath{
							Steps: []*proto.AttributePath_Step{
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "attr",
									},
								},
								{
									Selector: &proto.AttributePath_Step_ElementKeyString{
										ElementKeyString: "idx",
									},
								},
								{
									Selector: &proto.AttributePath_Step_AttributeName{
										AttributeName: "sub",
									},
								},
							},
						},
					},
				)

				return diags
			},
			[]diagFlat{
				{
					Severity: tfdiags.Error,
					Summary:  "error 1",
					Detail:   "error 1 detail",
					Attr:     []interface{}{"attr"},
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error 2",
					Detail:   "error 2 detail",
					Attr:     []interface{}{"attr", "sub"},
				},
				{
					Severity: tfdiags.Warning,
					Summary:  "warning",
					Detail:   "warning detail",
					Attr:     []interface{}{"attr", 1, "sub"},
				},
				{
					Severity: tfdiags.Error,
					Summary:  "error 3",
					Detail:   "error 3 detail",
					Attr:     []interface{}{"attr", "idx", "sub"},
				},
			},
		},
	}

	flattenTFDiags := func(ds tfdiags.Diagnostics) []diagFlat {
		var flat []diagFlat
		for _, item := range ds {
			desc := item.Description()

			var attr []interface{}

			for _, a := range tfdiags.GetAttribute(item) {
				switch step := a.(type) {
				case cty.GetAttrStep:
					attr = append(attr, step.Name)
				case cty.IndexStep:
					switch step.Key.Type() {
					case cty.Number:
						i, _ := step.Key.AsBigFloat().Int64()
						attr = append(attr, int(i))
					case cty.String:
						attr = append(attr, step.Key.AsString())
					}
				}
			}

			flat = append(flat, diagFlat{
				Severity: item.Severity(),
				Attr:     attr,
				Summary:  desc.Summary,
				Detail:   desc.Detail,
			})
		}
		return flat
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// we take the
			tfDiags := ProtoToDiagnostics(tc.Cons(nil))

			flat := flattenTFDiags(tfDiags)

			if !cmp.Equal(flat, tc.Want, typeComparer, valueComparer, equateEmpty) {
				t.Fatal(cmp.Diff(flat, tc.Want, typeComparer, valueComparer, equateEmpty))
			}
		})
	}
}

// Test that a diagnostic with a present but empty attribute results in a
// whole body diagnostic. We verify this by inspecting the resulting Subject
// from the diagnostic when considered in the context of a config body.
func TestProtoDiagnostics_emptyAttributePath(t *testing.T) {
	protoDiags := []*proto.Diagnostic{
		{
			Severity: proto.Diagnostic_ERROR,
			Summary:  "error 1",
			Detail:   "error 1 detail",
			Attribute: &proto.AttributePath{
				Steps: []*proto.AttributePath_Step{
					// this slice is intentionally left empty
				},
			},
		},
	}
	tfDiags := ProtoToDiagnostics(protoDiags)

	testConfig := `provider "test" {
  foo = "bar"
}`
	f, parseDiags := hclsyntax.ParseConfig([]byte(testConfig), "test.tf", hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		t.Fatal(parseDiags)
	}
	diags := tfDiags.InConfigBody(f.Body, "")

	if len(tfDiags) != 1 {
		t.Fatalf("expected 1 diag, got %d", len(tfDiags))
	}
	got := diags[0].Source().Subject
	want := &tfdiags.SourceRange{
		Filename: "test.tf",
		Start:    tfdiags.SourcePos{Line: 1, Column: 1},
		End:      tfdiags.SourcePos{Line: 1, Column: 1},
	}

	if !cmp.Equal(got, want, typeComparer, valueComparer) {
		t.Fatal(cmp.Diff(got, want, typeComparer, valueComparer))
	}
}

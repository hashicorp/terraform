// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/msgpack"
	"google.golang.org/protobuf/testing/protocmp"

	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/policy/proto"
)

func TestEncodeResourceAttributes(t *testing.T) {
	t.Run("nil value", func(t *testing.T) {
		got, err := EncodeResourceAttributes(cty.NilVal)
		if err != nil {
			t.Fatal(err)
		}
		if got != nil {
			t.Fatalf("expected no attributes, got %v", got)
		}
	})

	t.Run("value with unknowns and sensitive values", func(t *testing.T) {
		val := cty.ObjectVal(map[string]cty.Value{
			"id":     cty.UnknownVal(cty.String),
			"name":   cty.StringVal("a"),
			"secret": cty.StringVal("s").Mark(marks.Sensitive),
			"list":   cty.ListVal([]cty.Value{cty.StringVal("x"), cty.StringVal("y").Mark(marks.Sensitive)}),
		})
		got, err := EncodeResourceAttributes(val)
		if err != nil {
			t.Fatal(err)
		}

		raw, err := msgpack.Unmarshal(got.Raw, cty.DynamicPseudoType)
		if err != nil {
			t.Fatal(err)
		}
		want, _ := val.UnmarkDeep()
		if !raw.RawEquals(want) {
			t.Fatalf("wrong raw value\ngot:  %#v\nwant: %#v", raw, want)
		}

		wantPaths := []*proto.AttributePath{
			{Steps: []*proto.AttributePath_Step{
				{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "list"}},
				{Selector: &proto.AttributePath_Step_ElementKeyInt{ElementKeyInt: 1}},
			}},
			{Steps: []*proto.AttributePath_Step{
				{Selector: &proto.AttributePath_Step_AttributeName{AttributeName: "secret"}},
			}},
		}
		if diff := cmp.Diff(wantPaths, got.RedactedPaths, protocmp.Transform(), protocmp.SortRepeated(func(a, b *proto.AttributePath) bool {
			return a.String() < b.String()
		})); diff != "" {
			t.Fatalf("wrong redacted paths (-want +got):\n%s", diff)
		}
	})

	t.Run("value that can't be serialized", func(t *testing.T) {
		type thing struct{}
		capsule := cty.CapsuleVal(cty.Capsule("thing", reflect.TypeOf(thing{})), &thing{})
		if _, err := EncodeResourceAttributes(cty.ObjectVal(map[string]cty.Value{"c": capsule})); err == nil {
			t.Fatal("expected an error")
		}
	})
}

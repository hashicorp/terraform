package configs

import (
	"fmt"
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hcltest"
	"github.com/zclconf/go-cty/cty"
)

var (
	ignoreUnexported = cmpopts.IgnoreUnexported(version.Constraint{})
	comparer         = cmp.Comparer(func(x, y RequiredProvider) bool {
		if x.Name != y.Name {
			return false
		}
		if x.Source != y.Source {
			return false
		}
		if x.Requirement.Required.String() != y.Requirement.Required.String() {
			return false
		}
		return true
	})
)

func TestDecodeRequiredProvidersBlock_legacy(t *testing.T) {
	block := &hcl.Block{
		Type: "required_providers",
		Body: hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"default": {
					Name: "default",
					Expr: hcltest.MockExprLiteral(cty.StringVal("1.0.0")),
				},
			},
		}),
	}

	want := &RequiredProvider{
		Name:        "default",
		Requirement: testVC("1.0.0"),
	}

	got, diags := decodeRequiredProvidersBlock(block)
	if diags.HasErrors() {
		t.Fatalf("unexpected error")
	}
	if len(got) != 1 {
		t.Fatalf("wrong number of results, got %d, wanted 1", len(got))
	}
	if !cmp.Equal(got[0], want, ignoreUnexported, comparer) {
		t.Fatalf("wrong result:\n %s", cmp.Diff(got[0], want, ignoreUnexported, comparer))
	}
}

func TestDecodeRequiredProvidersBlock_provider_source(t *testing.T) {
	block := &hcl.Block{
		Type: "required_providers",
		Body: hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"my_test": {
					Name: "my_test",
					Expr: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
						"source":  cty.StringVal("mycloud/test"),
						"version": cty.StringVal("2.0.0"),
					})),
				},
			},
		}),
	}

	want := &RequiredProvider{
		Name:        "my_test",
		Source:      "mycloud/test",
		Requirement: testVC("2.0.0"),
	}
	got, diags := decodeRequiredProvidersBlock(block)
	if diags.HasErrors() {
		t.Fatalf("unexpected error")
	}
	if len(got) != 1 {
		t.Fatalf("wrong number of results, got %d, wanted 1", len(got))
	}
	if !cmp.Equal(got[0], want, ignoreUnexported, comparer) {
		t.Fatalf("wrong result:\n %s", cmp.Diff(got[0], want, ignoreUnexported, comparer))
	}
}

func TestDecodeRequiredProvidersBlock_mixed(t *testing.T) {
	block := &hcl.Block{
		Type: "required_providers",
		Body: hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"legacy": {
					Name: "legacy",
					Expr: hcltest.MockExprLiteral(cty.StringVal("1.0.0")),
				},
				"my_test": {
					Name: "my_test",
					Expr: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
						"source":  cty.StringVal("mycloud/test"),
						"version": cty.StringVal("2.0.0"),
					})),
				},
			},
		}),
	}

	want := []*RequiredProvider{
		{
			Name:        "legacy",
			Requirement: testVC("1.0.0"),
		},
		{
			Name:        "my_test",
			Source:      "mycloud/test",
			Requirement: testVC("2.0.0"),
		},
	}

	got, diags := decodeRequiredProvidersBlock(block)

	sort.SliceStable(got, func(i, j int) bool {
		return got[i].Name < got[j].Name
	})

	if diags.HasErrors() {
		t.Fatalf("unexpected error")
	}
	if len(got) != 2 {
		t.Fatalf("wrong number of results, got %d, wanted 2", len(got))
	}
	for i, rp := range want {
		if !cmp.Equal(got[i], rp, ignoreUnexported, comparer) {
			t.Fatalf("wrong result:\n %s", cmp.Diff(got[0], rp, ignoreUnexported, comparer))
		}
	}
}

func TestDecodeRequiredProvidersBlock_version_error(t *testing.T) {
	block := &hcl.Block{
		Type: "required_providers",
		Body: hcltest.MockBody(&hcl.BodyContent{
			Attributes: hcl.Attributes{
				"my_test": {
					Name: "my_test",
					Expr: hcltest.MockExprLiteral(cty.ObjectVal(map[string]cty.Value{
						"source":  cty.StringVal("mycloud/test"),
						"version": cty.StringVal("invalid"),
					})),
				},
			},
		}),
	}

	want := []*RequiredProvider{
		{
			Name:   "my_test",
			Source: "mycloud/test",
		},
	}

	got, diags := decodeRequiredProvidersBlock(block)
	if !diags.HasErrors() {
		t.Fatalf("expected error, got success")
	} else {
		fmt.Printf(diags[0].Summary)
	}
	if len(got) != 1 {
		t.Fatalf("wrong number of results, got %d, wanted 1", len(got))
	}
	for i, rp := range want {
		if !cmp.Equal(got[i], rp, ignoreUnexported, comparer) {
			t.Fatalf("wrong result:\n %s", cmp.Diff(got[0], rp, ignoreUnexported, comparer))
		}
	}
}

func testVC(ver string) VersionConstraint {
	constraint, _ := version.NewConstraint(ver)
	return VersionConstraint{
		Required:  constraint,
		DeclRange: hcl.Range{},
	}
}

// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"sort"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/lang/marks"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/schemarepo"
	"github.com/hashicorp/terraform/internal/states"
)

func TestPolicyStaticProviderConfig(t *testing.T) {
	schemas := &schemarepo.Schemas{Providers: map[addrs.Provider]providers.ProviderSchema{
		addrs.NewDefaultProvider("test"): *relationshipsTestProvider().GetProviderSchemaResponse,
	}}
	region := func(v cty.Value) cty.Value {
		return cty.ObjectVal(map[string]cty.Value{"region": v})
	}
	tests := map[string]struct {
		files   map[string]string
		addr    string
		outside bool
		// want is the decoded configuration; cty.NilVal if it can't be
		// known statically.
		want cty.Value
	}{
		"literal": {
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "a"
					region = "r1"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"].a`,
			want: region(cty.StringVal("r1")),
		},
		"no provider block": {
			files: map[string]string{"main.tf": `
				provider "test" {
					alias  = "a"
					region = "r1"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
			want: region(cty.NullVal(cty.String)),
		},
		"JSON literal": {
			files: map[string]string{"main.tf.json": `{"provider": {"test": [{"alias": "j", "region": "r1"}]}}`},
			addr:  `provider["registry.terraform.io/hashicorp/test"].j`,
			want:  region(cty.StringVal("r1")),
		},
		"JSON template": {
			files: map[string]string{"main.tf.json": `{
				"variable": {"r": {"default": "r1"}},
				"provider": {"test": [{"alias": "j", "region": "${var.r}"}]}
			}`},
			addr: `provider["registry.terraform.io/hashicorp/test"].j`,
		},
		"variable": {
			files: map[string]string{"main.tf": `
				variable "r" {
					default = "r1"
				}
				provider "test" {
					region = var.r
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"function call": {
			files: map[string]string{"main.tf": `
				provider "test" {
					region = lower("R1")
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"unsupported argument": {
			files: map[string]string{"main.tf": `
				provider "test" {
					zone = "z"
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/test"]`,
		},
		"module": {
			files: map[string]string{
				"main.tf": `
					module "child" {
						source = "./child"
					}
				`,
				"child/main.tf": `
					provider "test" {
						region = "r1"
					}
				`,
			},
			addr: `module.child.provider["registry.terraform.io/hashicorp/test"]`,
			want: region(cty.StringVal("r1")),
		},
		"module not in the configuration": {
			files: map[string]string{"main.tf": ``},
			addr:  `module.gone.provider["registry.terraform.io/hashicorp/test"]`,
		},
		"values from outside the configuration": {
			files:   map[string]string{"main.tf": ``},
			addr:    `provider["registry.terraform.io/hashicorp/test"]`,
			outside: true,
		},
		"no schema": {
			files: map[string]string{"main.tf": `
				terraform {
					required_providers {
						other = {
							source = "hashicorp/other"
						}
					}
				}
			`},
			addr: `provider["registry.terraform.io/hashicorp/other"]`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := testModuleInline(t, test.files)
			got := policyStaticProviderConfig(cfg, schemas, mustProviderConfig(test.addr), test.outside)
			if test.want == cty.NilVal {
				if got != cty.NilVal {
					t.Fatalf("expected no static configuration, got %#v", got)
				}
				return
			}
			if !got.RawEquals(test.want) {
				t.Fatalf("wrong configuration\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TestPolicyStateValue(t *testing.T) {
	schema := providers.Schema{Body: &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id":     {Type: cty.String, Computed: true},
			"secret": {Type: cty.String, Optional: true, Sensitive: true},
			"other":  {Type: cty.String, Optional: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"nested": {
				Nesting: configschema.NestingList,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"key": {Type: cty.String, Optional: true, Sensitive: true},
					},
				},
			},
		},
	}}
	obj := &states.ResourceInstanceObjectSrc{
		AttrsJSON: []byte(`{"id":"i","secret":"s","other":"o","nested":[{"key":"k"}]}`),
		// A path that was marked sensitive in configuration.
		AttrSensitivePaths: []cty.Path{cty.GetAttrPath("other")},
		Status:             states.ObjectReady,
	}

	// The policy client redacts the paths the schema declares sensitive.
	val, err := policyStateValue(obj, schema)
	if err != nil {
		t.Fatal(err)
	}
	_, pvms := val.UnmarkDeepWithPaths()
	paths, _ := marks.PathsWithMark(pvms, marks.Sensitive)
	if len(paths) != 1 || !paths[0].Equals(cty.GetAttrPath("other")) {
		t.Fatalf("wrong sensitive paths: %#v", paths)
	}

	t.Run("schema version mismatch", func(t *testing.T) {
		old := *obj
		old.SchemaVersion = 1
		if _, err := policyStateValue(&old, schema); err == nil {
			t.Fatal("expected an error for a state object with another schema version")
		}
	})

	t.Run("invalid value", func(t *testing.T) {
		bad := *obj
		bad.AttrsJSON = []byte(`{"unknown_attr":true}`)
		if _, err := policyStateValue(&bad, schema); err == nil {
			t.Fatal("expected an error for a state object that doesn't conform to the schema")
		}
	})
}

func TestPolicyBagRead(t *testing.T) {
	val := cty.ObjectVal(map[string]cty.Value{
		"s":     cty.StringVal("a"),
		"empty": cty.StringVal(""),
		"null":  cty.NullVal(cty.String),
		"n":     cty.NumberIntVal(10),
		"b":     cty.False,
		"list":  cty.ListVal([]cty.Value{cty.StringVal("x"), cty.StringVal(""), cty.StringVal("y")}),
		"nested": cty.ListVal([]cty.Value{
			cty.ObjectVal(map[string]cty.Value{"id": cty.StringVal("p")}),
			cty.ObjectVal(map[string]cty.Value{"id": cty.UnknownVal(cty.String)}),
		}),
		"unknown": cty.UnknownVal(cty.String),
		"map":     cty.MapVal(map[string]cty.Value{"k": cty.StringVal("v")}),
	})
	tests := map[string]struct {
		path        []string
		wantKnown   []string
		wantUnknown bool
	}{
		"string":        {path: []string{"s"}, wantKnown: []string{"string:a"}},
		"empty string":  {path: []string{"empty"}},
		"null":          {path: []string{"null"}},
		"number":        {path: []string{"n"}, wantKnown: []string{"number:10"}},
		"bool":          {path: []string{"b"}, wantKnown: []string{"bool:false"}},
		"list":          {path: []string{"list"}, wantKnown: []string{"string:x", "string:y"}},
		"nested":        {path: []string{"nested", "id"}, wantKnown: []string{"string:p"}, wantUnknown: true},
		"unknown":       {path: []string{"unknown"}, wantUnknown: true},
		"doesn't fit":   {path: []string{"map", "k"}, wantUnknown: true},
		"no such attr":  {path: []string{"nope"}, wantUnknown: true},
		"not primitive": {path: []string{"nested"}, wantUnknown: true},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bag := policyBag{known: make(map[string]bool)}
			bag.read(val, test.path)
			var got []string
			for v := range bag.known {
				got = append(got, v)
			}
			sort.Strings(got)
			if diff := cmp.Diff(test.wantKnown, got); diff != "" {
				t.Errorf("wrong known values (-want +got):\n%s", diff)
			}
			if bag.unknown != test.wantUnknown {
				t.Errorf("wrong unknown %t", bag.unknown)
			}
		})
	}
}

func TestPolicyProviderSame(t *testing.T) {
	region := func(r string) cty.Value { return cty.ObjectVal(map[string]cty.Value{"region": cty.StringVal(r)}) }
	tests := map[string]struct {
		a, b        PolicyProvider
		same, known bool
	}{
		"same address":         {a: PolicyProvider{Addr: "p"}, b: PolicyProvider{Addr: "p"}, same: true},
		"equal values":         {a: PolicyProvider{Addr: "p", Config: region("a")}, b: PolicyProvider{Addr: "q", Config: region("a")}, same: true},
		"different values":     {a: PolicyProvider{Addr: "p", Config: region("a")}, b: PolicyProvider{Addr: "q", Config: region("b")}, known: true},
		"a value is not known": {a: PolicyProvider{Addr: "p", Config: region("a")}, b: PolicyProvider{Addr: "q"}},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := test.a.same(&test.b); got != test.same {
				t.Errorf("same = %t, want %t", got, test.same)
			}
			if got := test.a.differs(&test.b); got != test.known {
				t.Errorf("differs = %t, want %t", got, test.known)
			}
		})
	}
}

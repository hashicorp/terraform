// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/genconfig"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/zclconf/go-cty/cty"
)

// BenchmarkJSONHookPostListQuery measures JSON output for 10,000 rows,
// excluding configuration generation and logger setup.
func BenchmarkJSONHookPostListQuery(b *testing.B) {
	for _, tc := range []struct {
		name       string
		attributes int
		config     bool
	}{
		{"config_small", 8, true},
		{"config_large", 32, true},
		{"no_config_small", 8, false},
		{"no_config_large", 32, false},
	} {
		b.Run(tc.name, func(b *testing.B) {
			id, results := benchmarkListQueryResults(b, 10000, tc.attributes, tc.config)
			f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
			if err != nil {
				b.Fatal(err)
			}
			b.Cleanup(func() { f.Close() })
			hook := newJSONHook(NewJSONView(NewView(&terminal.Streams{
				Stdout: &terminal.OutputStream{File: f},
				Stderr: &terminal.OutputStream{File: f},
			})))
			b.ReportAllocs()
			for b.Loop() {
				if _, err := hook.PostListQuery(id, results, 1); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func benchmarkListQueryResults(b *testing.B, count, attributes int, config bool) (terraform.HookResourceIdentity, plans.QueryResults) {
	b.Helper()
	addr := addrs.Resource{Mode: addrs.ListResourceMode, Type: "test_resource", Name: "inventory"}.
		Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	schema := &configschema.Block{Attributes: map[string]*configschema.Attribute{
		"secret": {Type: cty.String, Optional: true, Sensitive: true},
		"tags":   {Type: cty.Map(cty.String), Optional: true},
	}}
	for j := 0; j < attributes; j++ {
		schema.Attributes[fmt.Sprintf("attribute_%02d", j)] = &configschema.Attribute{Type: cty.String, Optional: true}
	}
	rows := make([]cty.Value, count)
	elements := make([]genconfig.ResourceListElement, count)
	for i := range rows {
		identity := cty.ObjectVal(map[string]cty.Value{
			"id":      cty.StringVal(fmt.Sprintf("id-%08d", i)),
			"region":  cty.StringVal("region-one"),
			"account": cty.StringVal("123456789012"),
			"type":    cty.StringVal("test_resource"),
		})
		values := map[string]cty.Value{
			"secret": cty.StringVal("fixture-secret"),
			"tags":   cty.MapVal(map[string]cty.Value{"Name": cty.StringVal(fmt.Sprintf("resource-%d", i))}),
		}
		for j := 0; j < attributes; j++ {
			values[fmt.Sprintf("attribute_%02d", j)] = cty.StringVal(fmt.Sprintf("row-%08d-%s", i, strings.Repeat("x", 80)))
		}
		state := cty.ObjectVal(values)
		rows[i] = cty.ObjectVal(map[string]cty.Value{
			"display_name": cty.StringVal(fmt.Sprintf("Resource %d", i)),
			"identity":     identity,
			"state":        state,
		})
		elements[i] = genconfig.ResourceListElement{Config: state, Identity: identity}
	}
	results := plans.QueryResults{Value: cty.ObjectVal(map[string]cty.Value{"data": cty.ListVal(rows)})}
	if config {
		identitySchema := &configschema.Object{Nesting: configschema.NestingSingle, Attributes: map[string]*configschema.Attribute{}}
		for _, name := range []string{"id", "region", "account", "type"} {
			identitySchema.Attributes[name] = &configschema.Attribute{Type: cty.String, Required: true}
		}
		generated, diags := genconfig.GenerateListResourceContents(addr, schema, identitySchema,
			addrs.LocalProviderConfig{LocalName: "test"}, elements)
		if diags.HasErrors() {
			b.Fatal(diags.Err())
		}
		if len(generated.Imports) != count {
			b.Fatalf("generated %d imports, want %d", len(generated.Imports), count)
		}
		results.Generated = generated
	}
	return terraform.HookResourceIdentity{Addr: addr}, results
}

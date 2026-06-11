// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package arguments

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestParseProvidersSchema_valid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *ProvidersSchema
	}{
		"json": {
			[]string{"-json"},
			&ProvidersSchema{
				JSON: true,
				Vars: &Vars{},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersSchema(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseProvidersSchema_invalid(t *testing.T) {
	testCases := map[string]struct {
		args      []string
		want      *ProvidersSchema
		wantDiags tfdiags.Diagnostics
	}{
		"missing json": {
			nil,
			&ProvidersSchema{
				Vars: &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"The -json flag is required",
					"The `terraform providers schema` command requires the `-json` flag.",
				),
			},
		},
		"too many positional arguments": {
			[]string{"-json", "extra"},
			&ProvidersSchema{
				JSON: true,
				Vars: &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Too many command line arguments",
					"Expected no positional arguments.",
				),
			},
		},
		"unknown flag and missing json": {
			[]string{"-wat"},
			&ProvidersSchema{
				Vars: &Vars{},
			},
			tfdiags.Diagnostics{
				tfdiags.Sourceless(
					tfdiags.Error,
					"Failed to parse command-line flags",
					"flag provided but not defined: -wat",
				),
				tfdiags.Sourceless(
					tfdiags.Error,
					"The -json flag is required",
					"The `terraform providers schema` command requires the `-json` flag.",
				),
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Vars{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, gotDiags := ParseProvidersSchema(tc.args)
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Fatalf("unexpected result\n%s", diff)
			}
			tfdiags.AssertDiagnosticsMatch(t, gotDiags, tc.wantDiags)
		})
	}
}

func TestParseProvidersSchema_provider(t *testing.T) {
	t.Run("normalizes a bare name to its FQN", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-provider", "aws"})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.ProviderSet {
			t.Fatalf("expected ProviderSet to be true")
		}
		if got.Provider.String() != "registry.terraform.io/hashicorp/aws" {
			t.Fatalf("expected normalized FQN, got %q", got.Provider.String())
		}
	})

	t.Run("accepts a full source string", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-provider=registry.terraform.io/hashicorp/test"})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.ProviderSet || got.Provider.String() != "registry.terraform.io/hashicorp/test" {
			t.Fatalf("expected the full source FQN, got set=%t %q", got.ProviderSet, got.Provider.String())
		}
	})

	t.Run("empty -provider is treated as omitted", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-provider="})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.ProviderSet {
			t.Fatalf("expected an empty -provider to be omitted")
		}
	})

	t.Run("a lone empty value after a real one is not a repeat", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-provider=aws", "-provider="})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.ProviderSet || got.Provider.String() != "registry.terraform.io/hashicorp/aws" {
			t.Fatalf("expected aws to be selected, got set=%t %q", got.ProviderSet, got.Provider.String())
		}
	})

	t.Run("repeated non-empty -provider is an error", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-provider=aws", "-provider=google"})
		if !diags.HasErrors() {
			t.Fatalf("expected a duplicate-flag diagnostic")
		}
	})

	t.Run("invalid provider syntax surfaces the parser diagnostic", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-provider=aws.us_east_1"})
		if !diags.HasErrors() {
			t.Fatalf("expected a parser diagnostic for an alias-like value")
		}
	})
}

func TestParseProvidersSchema_kind(t *testing.T) {
	t.Run("valid kind is recorded", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-kind=data-source"})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.KindSet || got.Kind != KindDataSource {
			t.Fatalf("expected KindDataSource, got set=%t %q", got.KindSet, got.Kind)
		}
	})

	t.Run("empty -kind is treated as omitted", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-kind="})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.KindSet {
			t.Fatalf("expected an empty -kind to be omitted")
		}
	})

	t.Run("repeated non-empty -kind is an error", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-kind=resource", "-kind=data-source"})
		if !diags.HasErrors() {
			t.Fatalf("expected a duplicate-flag diagnostic")
		}
	})

	t.Run("invalid kind lists the valid labels", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-kind=resources"})
		if !diags.HasErrors() {
			t.Fatalf("expected an invalid-kind diagnostic")
		}
		var found bool
		for _, d := range diags {
			desc := d.Description()
			if desc.Summary == "Invalid -kind value" {
				found = true
				for _, label := range ProviderSchemaKinds() {
					if !strings.Contains(desc.Detail, label) {
						t.Errorf("invalid-kind detail should list %q: %s", label, desc.Detail)
					}
				}
			}
		}
		if !found {
			t.Fatalf("expected an \"Invalid -kind value\" diagnostic, got %v", diags)
		}
	})
}

func TestParseProvidersSchema_type(t *testing.T) {
	t.Run("value is stored verbatim", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-type=aws_instance"})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.TypeSet || got.Type != "aws_instance" {
			t.Fatalf("expected type=aws_instance, got set=%t %q", got.TypeSet, got.Type)
		}
	})

	t.Run("empty -type is treated as omitted", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-type="})
		if len(diags) > 0 {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got.TypeSet {
			t.Fatalf("expected an empty -type to be omitted")
		}
	})

	t.Run("repeated non-empty -type is an error", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-type=a", "-type=b"})
		if !diags.HasErrors() {
			t.Fatalf("expected a duplicate-flag diagnostic")
		}
	})

	t.Run("-kind=provider with a non-empty -type is rejected", func(t *testing.T) {
		_, diags := ParseProvidersSchema([]string{"-json", "-kind=provider", "-type=anything"})
		if !diags.HasErrors() {
			t.Fatalf("expected a -kind=provider/-type diagnostic")
		}
		var found bool
		for _, d := range diags {
			if d.Description().Summary == "Invalid combination of -kind and -type" {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected the combination diagnostic, got %v", diags)
		}
	})

	t.Run("-kind=provider with an empty -type is allowed", func(t *testing.T) {
		got, diags := ParseProvidersSchema([]string{"-json", "-kind=provider", "-type="})
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if !got.KindSet || got.Kind != KindProvider {
			t.Fatalf("expected kind=provider, got set=%t %q", got.KindSet, got.Kind)
		}
		if got.TypeSet {
			t.Fatalf("expected an empty -type to be omitted")
		}
	})
}

func TestParseProviderSelector(t *testing.T) {
	// Bare names and shorthand normalize to the default registry/namespace FQN;
	// a full source string round-trips unchanged.
	valid := map[string]string{
		"aws":                                 "registry.terraform.io/hashicorp/aws",
		"hashicorp/aws":                       "registry.terraform.io/hashicorp/aws",
		"registry.terraform.io/hashicorp/aws": "registry.terraform.io/hashicorp/aws",
		"example.com/foo/bar":                 "example.com/foo/bar",
	}
	for raw, want := range valid {
		t.Run("valid/"+raw, func(t *testing.T) {
			got, diags := parseProviderSelector(raw)
			if diags.HasErrors() {
				t.Fatalf("unexpected diags for %q: %s", raw, diags.Err())
			}
			if got.String() != want {
				t.Fatalf("parseProviderSelector(%q) = %q, want %q", raw, got.String(), want)
			}
		})
	}

	// Aliases (DNS-label-invalid type), malformed sources, and too many path
	// parts are all rejected by the parser's own diagnostic.
	invalid := []string{
		"aws.us_east_1",
		"foo/bar/baz/qux",
		"hashicorp/",
		"/aws",
	}
	for _, raw := range invalid {
		t.Run("invalid/"+raw, func(t *testing.T) {
			if _, diags := parseProviderSelector(raw); !diags.HasErrors() {
				t.Fatalf("expected %q to be rejected", raw)
			}
		})
	}
}

func TestParseProvidersSchemaKind(t *testing.T) {
	valid := []string{
		"provider",
		"resource",
		"data-source",
		"ephemeral-resource",
		"list-resource",
		"function",
		"resource-identity",
		"action",
		"state-store",
	}
	for _, label := range valid {
		t.Run("valid/"+label, func(t *testing.T) {
			k, ok := ParseProviderSchemaKind(label)
			if !ok {
				t.Fatalf("expected %q to be a valid kind", label)
			}
			if string(k) != label {
				t.Fatalf("expected kind %q, got %q", label, k)
			}
		})
	}

	// Plurals, shorthand, alternate spellings, and casing are all rejected.
	invalid := []string{
		"",
		"resources",
		"Resource",
		"data_source",
		"datasource",
		"ephemeral",
		"func",
		"identity",
		"actions",
		"state_store",
		"statestore",
		"bogus",
	}
	for _, label := range invalid {
		t.Run("invalid/"+label, func(t *testing.T) {
			if _, ok := ParseProviderSchemaKind(label); ok {
				t.Fatalf("expected %q to be rejected", label)
			}
		})
	}
}

func TestProvidersSchemaKindIsMapBacked(t *testing.T) {
	if KindProvider.IsMapBacked() {
		t.Errorf("provider kind should not be map-backed")
	}
	mapBacked := []Kind{
		KindResource,
		KindDataSource,
		KindEphemeralResource,
		KindListResource,
		KindFunction,
		KindResourceIdentity,
		KindAction,
		KindStateStore,
	}
	for _, k := range mapBacked {
		if !k.IsMapBacked() {
			t.Errorf("kind %q should be map-backed", k)
		}
	}
}

func TestParseProvidersSchema_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"var": {
			args: []string{"-json", "-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"var-file": {
			args: []string{"-json", "-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"both": {
			args: []string{
				"-json",
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseProvidersSchema(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected vars: %#v", vars)
			}
		})
	}
}

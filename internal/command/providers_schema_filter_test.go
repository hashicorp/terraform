// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"sort"
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/jsonprovider"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
)

// filterTestSchemas builds a small single-provider *terraform.Schemas suitable
// for exercising the pure filter function.
func filterTestSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("test"): {
				ResourceTypes: map[string]providers.Schema{
					"test_instance": {
						Body: &configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"id": {Type: cty.String, Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

// emptyBody is a tiny reusable schema body.
func emptyBody() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"id": {Type: cty.String, Computed: true},
		},
	}
}

// identityBody is a tiny reusable identity schema.
func identityBody() *configschema.Object {
	return &configschema.Object{
		Nesting: configschema.NestingSingle,
		Attributes: map[string]*configschema.Attribute{
			"id": {Type: cty.String, Required: true},
		},
	}
}

// listResourceBody is a list-resource body with the required nested "config"
// block, so the schema can also be marshaled without panicking.
func listResourceBody() *configschema.Block {
	return &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {Type: cty.DynamicPseudoType, Computed: true},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Nesting: configschema.NestingSingle,
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"filter": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}
}

// multiProviderSchemas builds a synthetic multi-provider *terraform.Schemas that
// covers every schema category and a few deliberate cross-category collisions:
//
//   - aws_instance exists as a managed resource (with an identity), and as a
//     data source, so a wildcard -type=aws_instance fans out across
//     resource_schemas, resource_identity_schemas, and data_source_schemas.
//   - aws_s3_bucket is a managed resource without an identity.
//
// The "random" provider is intentionally minimal to exercise multi-provider
// pruning and wildcard-kind selection across providers.
func multiProviderSchemas() *terraform.Schemas {
	return &terraform.Schemas{
		Providers: map[addrs.Provider]providers.ProviderSchema{
			addrs.NewDefaultProvider("aws"): {
				Provider: providers.Schema{Body: emptyBody()},
				ResourceTypes: map[string]providers.Schema{
					"aws_instance":  {Body: emptyBody(), Identity: identityBody()},
					"aws_s3_bucket": {Body: emptyBody()},
				},
				DataSources: map[string]providers.Schema{
					"aws_instance": {Body: emptyBody()},
					"aws_ami":      {Body: emptyBody()},
				},
				EphemeralResourceTypes: map[string]providers.Schema{
					"aws_secret": {Body: emptyBody()},
				},
				ListResourceTypes: map[string]providers.Schema{
					"aws_instances": {Body: listResourceBody()},
				},
				Functions: map[string]providers.FunctionDecl{
					"arn_parse": {ReturnType: cty.String},
				},
				Actions: map[string]providers.ActionSchema{
					"aws_reboot": {ConfigSchema: emptyBody()},
				},
				StateStores: map[string]providers.Schema{
					"aws_s3_state": {Body: emptyBody()},
				},
			},
			addrs.NewDefaultProvider("random"): {
				Provider: providers.Schema{Body: emptyBody()},
				ResourceTypes: map[string]providers.Schema{
					"random_pet": {Body: emptyBody()},
				},
			},
		},
	}
}

// providerKeys returns the sorted FQNs of the providers in a filtered result.
func providerKeys(s *terraform.Schemas) []string {
	keys := make([]string, 0, len(s.Providers))
	for addr := range s.Providers {
		keys = append(keys, addr.String())
	}
	sort.Strings(keys)
	return keys
}

// sortedKeys returns the sorted keys of a string-keyed map.
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestFilterProviderSchemas_plumbing verifies the pure filter function threads
// the resource-emission directive and filters echo through, and that the
// no-selector case is an exact pass-through of the loaded schemas.
func TestFilterProviderSchemas_plumbing(t *testing.T) {
	schemas := filterTestSchemas()

	t.Run("no selectors is a pass-through", func(t *testing.T) {
		filtered, emit, filters, diags := filterProviderSchemas(schemas, selectors{})
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if filtered != schemas {
			t.Errorf("expected the loaded schemas to be returned unchanged")
		}
		if emit != jsonprovider.EmitAll {
			t.Errorf("expected EmitAll, got %v", emit)
		}
		if filters != nil {
			t.Errorf("expected nil filters echo, got %+v", filters)
		}
	})

	t.Run("directive and filters echo are threaded through", func(t *testing.T) {
		sel := selectors{
			kind:    arguments.KindResourceIdentity,
			kindSet: true,
		}
		_, emit, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if emit != jsonprovider.ResourceIdentityOnly {
			t.Errorf("expected ResourceIdentityOnly, got %v", emit)
		}
		if filters == nil {
			t.Fatalf("expected a non-nil filters echo")
		}
		if filters.Kind != string(arguments.KindResourceIdentity) {
			t.Errorf("expected filters.kind=%q, got %q", arguments.KindResourceIdentity, filters.Kind)
		}
	})
}

// TestFilterProviderSchemas_provider exercises the -provider selector: an exact
// provider match keeps the whole provider schema, no-match is an actionable
// error that lists the loaded providers, and the originals are never mutated.
func TestFilterProviderSchemas_provider(t *testing.T) {
	t.Run("match keeps only the selected provider with all categories", func(t *testing.T) {
		schemas := multiProviderSchemas()
		sel := selectors{provider: addrs.NewDefaultProvider("aws"), providerSet: true}

		filtered, emit, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if emit != jsonprovider.EmitAll {
			t.Errorf("expected EmitAll for a provider-only selector, got %v", emit)
		}
		if got := providerKeys(filtered); !equalStrings(got, []string{"registry.terraform.io/hashicorp/aws"}) {
			t.Fatalf("expected only the aws provider, got %v", got)
		}
		aws := filtered.Providers[addrs.NewDefaultProvider("aws")]
		// All categories of the selected provider are retained.
		if aws.Provider.Body == nil {
			t.Errorf("expected provider config to be retained")
		}
		if len(aws.ResourceTypes) != 2 || len(aws.DataSources) != 2 || len(aws.Functions) != 1 ||
			len(aws.Actions) != 1 || len(aws.StateStores) != 1 || len(aws.EphemeralResourceTypes) != 1 ||
			len(aws.ListResourceTypes) != 1 {
			t.Errorf("expected all aws categories to be retained, got %#v", aws)
		}
		if filters == nil || filters.Provider != "registry.terraform.io/hashicorp/aws" {
			t.Errorf("expected filters.provider to be the normalized FQN, got %+v", filters)
		}

		// The originals must be untouched.
		if len(schemas.Providers) != 2 {
			t.Errorf("original schemas were mutated: %d providers remain", len(schemas.Providers))
		}
	})

	t.Run("no-match is an error listing the loaded providers, sorted", func(t *testing.T) {
		schemas := multiProviderSchemas()
		sel := selectors{provider: addrs.NewDefaultProvider("google"), providerSet: true}

		_, _, filters, diags := filterProviderSchemas(schemas, sel)
		if !diags.HasErrors() {
			t.Fatalf("expected a no-match diagnostic")
		}
		// The filters echo is still produced (normalized FQN) even on no-match.
		if filters == nil || filters.Provider != "registry.terraform.io/hashicorp/google" {
			t.Errorf("expected filters.provider echo, got %+v", filters)
		}

		desc := diags[0].Description()
		if !strings.Contains(desc.Summary, "registry.terraform.io/hashicorp/google") {
			t.Errorf("summary should name the missing provider: %q", desc.Summary)
		}
		// Loaded providers are listed, sorted (aws before random).
		wantAWS := strings.Index(desc.Detail, "registry.terraform.io/hashicorp/aws")
		wantRandom := strings.Index(desc.Detail, "registry.terraform.io/hashicorp/random")
		if wantAWS < 0 || wantRandom < 0 {
			t.Errorf("detail should list the loaded providers: %q", desc.Detail)
		}
		if wantAWS > wantRandom {
			t.Errorf("loaded providers should be listed sorted: %q", desc.Detail)
		}
	})
}

// nonEmptyCategories returns the sorted set of schema categories that survive in
// a pruned provider schema (treating the provider config as the "provider"
// category when its body is present).
func nonEmptyCategories(ps providers.ProviderSchema) []string {
	var cats []string
	if ps.Provider.Body != nil {
		cats = append(cats, "provider")
	}
	if len(ps.ResourceTypes) > 0 {
		cats = append(cats, "resource_types")
	}
	if len(ps.DataSources) > 0 {
		cats = append(cats, "data_sources")
	}
	if len(ps.EphemeralResourceTypes) > 0 {
		cats = append(cats, "ephemeral")
	}
	if len(ps.ListResourceTypes) > 0 {
		cats = append(cats, "list")
	}
	if len(ps.Functions) > 0 {
		cats = append(cats, "functions")
	}
	if len(ps.Actions) > 0 {
		cats = append(cats, "actions")
	}
	if len(ps.StateStores) > 0 {
		cats = append(cats, "state_stores")
	}
	sort.Strings(cats)
	return cats
}

// categoryEntryKeys returns the sorted keys of the named map-backed category.
func categoryEntryKeys(ps providers.ProviderSchema, category string) []string {
	switch category {
	case "resource_types":
		return sortedKeys(ps.ResourceTypes)
	case "data_sources":
		return sortedKeys(ps.DataSources)
	case "ephemeral":
		return sortedKeys(ps.EphemeralResourceTypes)
	case "list":
		return sortedKeys(ps.ListResourceTypes)
	case "functions":
		return sortedKeys(ps.Functions)
	case "actions":
		return sortedKeys(ps.Actions)
	case "state_stores":
		return sortedKeys(ps.StateStores)
	}
	return nil
}

// TestFilterProviderSchemas_kind exercises -kind for every canonical category:
// each prunes to exactly one category, sets the correct emit directive, drops
// providers that expose nothing in that category, and echoes the canonical
// label. A valid kind selecting nothing is an empty success.
func TestFilterProviderSchemas_kind(t *testing.T) {
	const awsFQN = "registry.terraform.io/hashicorp/aws"
	const randomFQN = "registry.terraform.io/hashicorp/random"

	cases := map[string]struct {
		kind          arguments.Kind
		wantEmit      jsonprovider.ResourceEmit
		wantProviders []string
		awsCategory   string
		awsKeys       []string // nil for the non-map-backed provider category
	}{
		"provider": {
			arguments.KindProvider, jsonprovider.EmitAll,
			[]string{awsFQN, randomFQN}, "provider", nil,
		},
		"resource": {
			arguments.KindResource, jsonprovider.ResourceBlockOnly,
			[]string{awsFQN, randomFQN}, "resource_types",
			[]string{"aws_instance", "aws_s3_bucket"},
		},
		"data-source": {
			arguments.KindDataSource, jsonprovider.EmitAll,
			[]string{awsFQN}, "data_sources",
			[]string{"aws_ami", "aws_instance"},
		},
		"ephemeral-resource": {
			arguments.KindEphemeralResource, jsonprovider.EmitAll,
			[]string{awsFQN}, "ephemeral", []string{"aws_secret"},
		},
		"list-resource": {
			arguments.KindListResource, jsonprovider.EmitAll,
			[]string{awsFQN}, "list", []string{"aws_instances"},
		},
		"function": {
			arguments.KindFunction, jsonprovider.EmitAll,
			[]string{awsFQN}, "functions", []string{"arn_parse"},
		},
		"resource-identity": {
			arguments.KindResourceIdentity, jsonprovider.ResourceIdentityOnly,
			[]string{awsFQN}, "resource_types", []string{"aws_instance"},
		},
		"action": {
			arguments.KindAction, jsonprovider.EmitAll,
			[]string{awsFQN}, "actions", []string{"aws_reboot"},
		},
		"state-store": {
			arguments.KindStateStore, jsonprovider.EmitAll,
			[]string{awsFQN}, "state_stores", []string{"aws_s3_state"},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			schemas := multiProviderSchemas()
			filtered, emit, filters, diags := filterProviderSchemas(schemas, selectors{kind: tc.kind, kindSet: true})
			if diags.HasErrors() {
				t.Fatalf("unexpected diags: %s", diags.Err())
			}
			if emit != tc.wantEmit {
				t.Errorf("emit = %v, want %v", emit, tc.wantEmit)
			}
			if filters == nil || filters.Kind != string(tc.kind) {
				t.Errorf("expected filters.kind=%q, got %+v", tc.kind, filters)
			}
			if got := providerKeys(filtered); !equalStrings(got, tc.wantProviders) {
				t.Fatalf("providers = %v, want %v", got, tc.wantProviders)
			}
			aws := filtered.Providers[addrs.NewDefaultProvider("aws")]
			if got := nonEmptyCategories(aws); !equalStrings(got, []string{tc.awsCategory}) {
				t.Fatalf("aws categories = %v, want only %q", got, tc.awsCategory)
			}
			if tc.awsKeys != nil {
				if got := categoryEntryKeys(aws, tc.awsCategory); !equalStrings(got, tc.awsKeys) {
					t.Errorf("aws %s keys = %v, want %v", tc.awsCategory, got, tc.awsKeys)
				}
			}
		})
	}

	t.Run("valid kind no-match is empty success", func(t *testing.T) {
		schemas := multiProviderSchemas()
		// The random provider exposes no data sources, so isolating it with
		// -kind=data-source selects nothing without raising an error.
		sel := selectors{
			provider:    addrs.NewDefaultProvider("random"),
			providerSet: true,
			kind:        arguments.KindDataSource,
			kindSet:     true,
		}
		filtered, _, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("expected empty success, got diags: %s", diags.Err())
		}
		if len(filtered.Providers) != 0 {
			t.Fatalf("expected no providers in the result, got %v", providerKeys(filtered))
		}
		if filters == nil || filters.Kind != string(arguments.KindDataSource) {
			t.Errorf("expected the filters echo to still be present, got %+v", filters)
		}
	})
}

// TestFilterProviderSchemas_type exercises -type alone, in composition with
// -provider and -kind, and its exact/case-sensitive matching and wildcard
// fan-out semantics.
func TestFilterProviderSchemas_type(t *testing.T) {
	const awsFQN = "registry.terraform.io/hashicorp/aws"
	awsAddr := addrs.NewDefaultProvider("aws")

	t.Run("wildcard kind fans out across every category sharing the key", func(t *testing.T) {
		schemas := multiProviderSchemas()
		// aws_instance is a managed resource (with identity) and a data source.
		sel := selectors{typ: "aws_instance", typeSet: true}
		filtered, emit, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if emit != jsonprovider.EmitAll {
			t.Errorf("expected EmitAll for a wildcard-kind -type, got %v", emit)
		}
		if filters == nil || filters.Type != "aws_instance" {
			t.Errorf("expected filters.type=aws_instance, got %+v", filters)
		}
		// random has no aws_instance, so it is dropped.
		if got := providerKeys(filtered); !equalStrings(got, []string{awsFQN}) {
			t.Fatalf("providers = %v, want only aws", got)
		}
		aws := filtered.Providers[awsAddr]
		if got := nonEmptyCategories(aws); !equalStrings(got, []string{"data_sources", "resource_types"}) {
			t.Fatalf("aws categories = %v, want data_sources+resource_types", got)
		}
		if !equalStrings(sortedKeys(aws.ResourceTypes), []string{"aws_instance"}) {
			t.Errorf("resource_types = %v, want [aws_instance]", sortedKeys(aws.ResourceTypes))
		}
		if !equalStrings(sortedKeys(aws.DataSources), []string{"aws_instance"}) {
			t.Errorf("data_sources = %v, want [aws_instance]", sortedKeys(aws.DataSources))
		}
		// The retained resource keeps its identity so EmitAll renders the
		// resource_identity_schemas category too.
		if aws.ResourceTypes["aws_instance"].Identity == nil {
			t.Errorf("expected the retained aws_instance resource to keep its identity")
		}
	})

	t.Run("matching is exact and case-sensitive", func(t *testing.T) {
		for _, typ := range []string{"AWS_INSTANCE", "aws_inst", "instance"} {
			schemas := multiProviderSchemas()
			filtered, _, _, diags := filterProviderSchemas(schemas, selectors{typ: typ, typeSet: true})
			if diags.HasErrors() {
				t.Fatalf("unexpected diags for %q: %s", typ, diags.Err())
			}
			if len(filtered.Providers) != 0 {
				t.Errorf("type %q should match nothing, got %v", typ, providerKeys(filtered))
			}
		}
	})

	t.Run("provider and type compose", func(t *testing.T) {
		schemas := multiProviderSchemas()
		sel := selectors{provider: awsAddr, providerSet: true, typ: "aws_ami", typeSet: true}
		filtered, _, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if filters == nil || filters.Provider != awsFQN || filters.Type != "aws_ami" {
			t.Errorf("expected provider+type echo, got %+v", filters)
		}
		aws := filtered.Providers[awsAddr]
		// aws_ami is only a data source.
		if got := nonEmptyCategories(aws); !equalStrings(got, []string{"data_sources"}) {
			t.Fatalf("aws categories = %v, want only data_sources", got)
		}
		if !equalStrings(sortedKeys(aws.DataSources), []string{"aws_ami"}) {
			t.Errorf("data_sources = %v, want [aws_ami]", sortedKeys(aws.DataSources))
		}
	})

	t.Run("kind and type narrow to one category", func(t *testing.T) {
		tests := map[string]struct {
			kind     arguments.Kind
			typ      string
			wantEmit jsonprovider.ResourceEmit
			wantCat  string
		}{
			"resource":          {arguments.KindResource, "aws_instance", jsonprovider.ResourceBlockOnly, "resource_types"},
			"data-source":       {arguments.KindDataSource, "aws_instance", jsonprovider.EmitAll, "data_sources"},
			"resource-identity": {arguments.KindResourceIdentity, "aws_instance", jsonprovider.ResourceIdentityOnly, "resource_types"},
		}
		for name, tc := range tests {
			t.Run(name, func(t *testing.T) {
				schemas := multiProviderSchemas()
				sel := selectors{kind: tc.kind, kindSet: true, typ: tc.typ, typeSet: true}
				filtered, emit, _, diags := filterProviderSchemas(schemas, sel)
				if diags.HasErrors() {
					t.Fatalf("unexpected diags: %s", diags.Err())
				}
				if emit != tc.wantEmit {
					t.Errorf("emit = %v, want %v", emit, tc.wantEmit)
				}
				if got := providerKeys(filtered); !equalStrings(got, []string{awsFQN}) {
					t.Fatalf("providers = %v, want only aws", got)
				}
				aws := filtered.Providers[awsAddr]
				if got := nonEmptyCategories(aws); !equalStrings(got, []string{tc.wantCat}) {
					t.Fatalf("aws categories = %v, want only %q", got, tc.wantCat)
				}
			})
		}
	})

	t.Run("resource-identity type without an identity is empty success", func(t *testing.T) {
		schemas := multiProviderSchemas()
		// aws_s3_bucket is a managed resource without an identity schema.
		sel := selectors{kind: arguments.KindResourceIdentity, kindSet: true, typ: "aws_s3_bucket", typeSet: true}
		filtered, _, _, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if len(filtered.Providers) != 0 {
			t.Fatalf("expected empty result, got %v", providerKeys(filtered))
		}
	})

	t.Run("full composition of provider, kind, and type", func(t *testing.T) {
		schemas := multiProviderSchemas()
		sel := selectors{
			provider:    awsAddr,
			providerSet: true,
			kind:        arguments.KindResource,
			kindSet:     true,
			typ:         "aws_instance",
			typeSet:     true,
		}
		filtered, emit, filters, diags := filterProviderSchemas(schemas, sel)
		if diags.HasErrors() {
			t.Fatalf("unexpected diags: %s", diags.Err())
		}
		if emit != jsonprovider.ResourceBlockOnly {
			t.Errorf("emit = %v, want ResourceBlockOnly", emit)
		}
		if filters == nil || filters.Provider != awsFQN || filters.Kind != "resource" || filters.Type != "aws_instance" {
			t.Fatalf("expected all three filters echoed, got %+v", filters)
		}
		aws := filtered.Providers[awsAddr]
		if got := nonEmptyCategories(aws); !equalStrings(got, []string{"resource_types"}) {
			t.Fatalf("aws categories = %v, want only resource_types", got)
		}
		if !equalStrings(sortedKeys(aws.ResourceTypes), []string{"aws_instance"}) {
			t.Errorf("resource_types = %v, want [aws_instance]", sortedKeys(aws.ResourceTypes))
		}
	})

	t.Run("valid type no-match is empty success", func(t *testing.T) {
		schemas := multiProviderSchemas()
		filtered, _, filters, diags := filterProviderSchemas(schemas, selectors{typ: "does_not_exist", typeSet: true})
		if diags.HasErrors() {
			t.Fatalf("expected empty success, got diags: %s", diags.Err())
		}
		if len(filtered.Providers) != 0 {
			t.Fatalf("expected no providers, got %v", providerKeys(filtered))
		}
		if filters == nil || filters.Type != "does_not_exist" {
			t.Errorf("expected filters.type echo, got %+v", filters)
		}
	})
}

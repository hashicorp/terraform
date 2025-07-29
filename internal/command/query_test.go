// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"path"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/zclconf/go-cty/cty"
)

func TestQuery(t *testing.T) {
	tests := []struct {
		name        string
		directory   string
		expectedOut string
		expectedErr []string
		initCode    int
	}{
		{
			name:      "basic query",
			directory: "basic",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:      "query referencing local variable",
			directory: "with-locals",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

`,
		},
		{
			name:        "config with no query block",
			directory:   "no-list-block",
			expectedOut: "",
			expectedErr: []string{`
Error: No resources to query

The configuration does not contain any resources that can be queried.
`},
		},
		{
			name:        "missing query file",
			directory:   "missing-query-file",
			expectedOut: "",
			expectedErr: []string{`
Error: No resources to query

The configuration does not contain any resources that can be queried.
`},
		},
		{
			name:        "missing configuration",
			directory:   "missing-configuration",
			expectedOut: "",
			expectedErr: []string{`
Error: No configuration files

Query requires a query configuration to be present. Create a Terraform query
configuration file (.tfquery.hcl file) and try again.
`},
		},
		{
			name:        "invalid query syntax",
			directory:   "invalid-syntax",
			expectedOut: "",
			initCode:    1,
			expectedErr: []string{`
Error: Unsupported block type

  on query.tfquery.hcl line 11:
  11: resource "test_instance" "example" {

Blocks of type "resource" are not expected here.
`},
		},
		{
			name:      "empty result",
			directory: "empty-result",
			expectedOut: `list.test_instance.example   id=test-instance-1   Test Instance 1
list.test_instance.example   id=test-instance-2   Test Instance 2

Warning: list block(s) [list.test_instance.example2] have 0 results.`,
			initCode: 0,
		},
	}

	for _, ts := range tests[len(tests)-1:] {
		t.Run(ts.name, func(t *testing.T) {
			td := t.TempDir()
			testCopyDir(t, testFixturePath(path.Join("query", ts.directory)), td)
			t.Chdir(td)
			providerSource, close := newMockProviderSource(t, map[string][]string{
				"hashicorp/test": {"1.0.0"},
			})
			defer close()

			p := queryFixtureProvider()
			view, done := testView(t)
			meta := Meta{
				testingOverrides:          metaOverridesForProvider(p),
				View:                      view,
				AllowExperimentalFeatures: true,
				ProviderSource:            providerSource,
			}

			init := &InitCommand{Meta: meta}
			code := init.Run(nil)
			output := done(t)
			if code != ts.initCode {
				t.Fatalf("expected status code %d but got %d: %s", ts.initCode, code, output.All())
			}

			view, done = testView(t)
			meta.View = view

			c := &QueryCommand{Meta: meta}
			args := []string{"-no-color"}
			code = c.Run(args)
			output = done(t)
			actual := strings.TrimSpace(output.All())
			if len(ts.expectedErr) == 0 {
				if code != 0 {
					t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
				}

				// Check that we have query output
				expected := strings.TrimSpace(ts.expectedOut)
				if diff := cmp.Diff(expected, actual); diff != "" {
					t.Errorf("expected query output to contain \n%q, \ngot: \n%q, \ndiff: %s", expected, actual, diff)
				}

			} else {
				for _, expected := range ts.expectedErr {
					expected := strings.TrimSpace(expected)
					if diff := cmp.Diff(expected, actual); diff != "" {
						t.Errorf("expected error message to contain '%s', \ngot: %s, \ndiff: %s", expected, actual, diff)
					}
				}
			}
		})
	}
}

func queryFixtureProvider() *testing_provider.MockProvider {
	p := testProvider()
	instanceListSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {
				Type:     cty.DynamicPseudoType,
				Computed: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"ami": {
							Type:     cty.String,
							Required: true,
						},
					},
				},
				Nesting: configschema.NestingSingle,
			},
		},
	}
	databaseListSchema := &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"data": {
				Type:     cty.DynamicPseudoType,
				Computed: true,
			},
		},
		BlockTypes: map[string]*configschema.NestedBlock{
			"config": {
				Block: configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"engine": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
				Nesting: configschema.NestingSingle,
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"ami": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
			"test_database": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
						"engine": {
							Type:     cty.String,
							Optional: true,
						},
					},
				},
			},
		},
		ListResourceTypes: map[string]providers.Schema{
			"test_instance": {Body: instanceListSchema},
			"test_database": {Body: databaseListSchema},
		},
	}

	// Mock the ListResources method for query operations
	p.ListResourceFn = func(request providers.ListResourceRequest) providers.ListResourceResponse {
		// Check the config to determine what kind of response to return
		wholeConfigMap := request.Config.AsValueMap()

		configMap := wholeConfigMap["config"]

		// For empty results test case
		ami, ok := configMap.AsValueMap()["ami"]
		if ok && ami.AsString() == "ami-nonexistent" {
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data":   cty.ListValEmpty(cty.DynamicPseudoType),
					"config": configMap,
				}),
			}
		}

		switch request.TypeName {
		case "test_instance":
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-instance-1"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":  cty.StringVal("test-instance-1"),
								"ami": cty.StringVal("ami-12345"),
							}),
							"display_name": cty.StringVal("Test Instance 1"),
						}),
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-instance-2"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":  cty.StringVal("test-instance-2"),
								"ami": cty.StringVal("ami-67890"),
							}),
							"display_name": cty.StringVal("Test Instance 2"),
						}),
					}),
					"config": configMap,
				}),
			}
		case "test_database":
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data": cty.ListVal([]cty.Value{
						cty.ObjectVal(map[string]cty.Value{
							"identity": cty.ObjectVal(map[string]cty.Value{
								"id": cty.StringVal("test-db-1"),
							}),
							"state": cty.ObjectVal(map[string]cty.Value{
								"id":     cty.StringVal("test-db-1"),
								"engine": cty.StringVal("mysql"),
							}),
							"display_name": cty.StringVal("Test Database 1"),
						}),
					}),
					"config": configMap,
				}),
			}
		default:
			return providers.ListResourceResponse{
				Result: cty.ObjectVal(map[string]cty.Value{
					"data":   cty.ListVal([]cty.Value{}),
					"config": configMap,
				}),
			}
		}
	}

	return p
}

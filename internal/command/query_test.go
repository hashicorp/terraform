// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"encoding/json"
	"fmt"
	"path"
	"slices"
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

Warning: list block(s) [list.test_instance.example2] returned 0 results.`,
			initCode: 0,
		},
	}

	for _, ts := range tests {
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
			if len(ts.expectedErr) == 0 {
				if code != 0 {
					t.Fatalf("bad: %d\n\n%s", code, output.Stderr())
				}
				actual := strings.TrimSpace(output.Stdout())

				// Check that we have query output
				expected := strings.TrimSpace(ts.expectedOut)
				if diff := cmp.Diff(expected, actual); diff != "" {
					t.Errorf("expected query output to contain \n%q, \ngot: \n%q, \ndiff: %s", expected, actual, diff)
				}

			} else {
				actual := strings.TrimSpace(output.Stderr())
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
				Identity: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
					},
					Nesting: configschema.NestingSingle,
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
				Identity: &configschema.Object{
					Attributes: map[string]*configschema.Attribute{
						"id": {
							Type:     cty.String,
							Computed: true,
						},
					},
					Nesting: configschema.NestingSingle,
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

func TestQuery_JSON(t *testing.T) {
	tmp := t.TempDir()
	tests := []struct {
		name        string
		directory   string
		expectedRes []map[string]any
		initCode    int
		opts        []string
	}{
		{
			name:      "basic query",
			directory: "basic",
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"input_config":  map[string]any{"ami": "ami-12345"},
					},
					"type": "list_start",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"resource_type": "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"resource_type": "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
		{
			name:      "basic query - generate config",
			directory: "basic",
			opts:      []string{fmt.Sprintf("-generate-config-out=%s/new", tmp)},
			expectedRes: []map[string]any{
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Starting query...",
					"list_start": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"input_config":  map[string]any{"ami": "ami-12345"},
					},
					"type": "list_start",
				},
				{"@level": "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 1",
						"identity": map[string]any{
							"id": "test-instance-1",
						},
						"resource_type": "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-12345",
							"id":  "test-instance-1",
						},
						"config":        "resource \"test_instance\" \"example_0\" {\n  provider = test\n  ami      = \"ami-12345\"\n  id       = \"test-instance-1\"\n}",
						"import_config": "import {\n  to       = test_instance.example_0\n  provider = test\n  identity = {\n    id = \"test-instance-1\"\n  }\n}",
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: Result found",
					"list_resource_found": map[string]any{
						"address":      "list.test_instance.example",
						"display_name": "Test Instance 2",
						"identity": map[string]any{
							"id": "test-instance-2",
						},
						"resource_type": "test_instance",
						"resource_object": map[string]any{
							"ami": "ami-67890",
							"id":  "test-instance-2",
						},
						"config":        "resource \"test_instance\" \"example_1\" {\n  provider = test\n  ami      = \"ami-67890\"\n  id       = \"test-instance-2\"\n}",
						"import_config": "import {\n  to       = test_instance.example_1\n  provider = test\n  identity = {\n    id = \"test-instance-2\"\n  }\n}",
					},
					"type": "list_resource_found",
				},
				{
					"@level":   "info",
					"@message": "list.test_instance.example: List complete",
					"list_complete": map[string]any{
						"address":       "list.test_instance.example",
						"resource_type": "test_instance",
						"total":         float64(2),
					},
					"type": "list_complete",
				},
			},
		},
	}

	for _, ts := range tests {
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
			args := []string{"-no-color", "-json"}
			code = c.Run(append(args, ts.opts...))
			output = done(t)
			if code != 0 {
				t.Fatalf("bad: %d\n\n%s", code, output.All())
			}
			// convert output to JSON array
			actual := strings.TrimSpace(output.Stdout())
			conc := fmt.Sprintf("[%s]", strings.Join(strings.Split(actual, "\n"), ","))
			actualRes := make([]map[string]any, 0)
			err := json.NewDecoder(strings.NewReader(conc)).Decode(&actualRes)
			if err != nil {
				t.Fatalf("failed to unmarshal: %s", err)
			}

			// remove unnecessary fields
			for _, item := range actualRes {
				delete(item, "@module")
				delete(item, "@timestamp")
				delete(item, "ui")
			}

			// Check that the output matches the expected results
			actualRes = slices.Delete(actualRes, 0, 1)
			if diff := cmp.Diff(ts.expectedRes, actualRes); diff != "" {
				t.Errorf("expected query output to contain \n%q, \ngot: \n%q, \ndiff: %s", ts.expectedRes, actualRes, diff)
			}
		})
	}
}

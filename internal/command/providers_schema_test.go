// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package command

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

func TestProvidersSchema_error(t *testing.T) {
	ui := new(cli.MockUi)
	c := &ProvidersSchemaCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	if code := c.Run(nil); code != 1 {
		fmt.Println(ui.OutputWriter.String())
		t.Fatalf("expected error: \n%s", ui.OutputWriter.String())
	}
}

func TestProvidersSchema_output(t *testing.T) {
	fixtureDir := "testdata/providers-schema"
	testDirs, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			td := t.TempDir()
			inputDir := filepath.Join(fixtureDir, entry.Name())
			testCopyDir(t, inputDir, td)
			t.Chdir(td)

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.2.3"},
			})
			defer close()

			p := providersSchemaFixtureProvider()
			ui := new(cli.MockUi)
			view, done := testView(t)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				View:             view,
				ProviderSource:   providerSource,
			}

			// `terrafrom init`
			ic := &InitCommand{
				Meta: m,
			}
			if code := ic.Run([]string{}); code != 0 {
				t.Fatalf("init failed\n%s", done(t).Stderr())
			}

			// `terraform provider schemas` command
			pc := &ProvidersSchemaCommand{Meta: m}
			if code := pc.Run([]string{"-json"}); code != 0 {
				t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
			}
			var got, want providerSchemas

			gotString := ui.OutputWriter.String()
			json.Unmarshal([]byte(gotString), &got)

			wantFile, err := os.Open("output.json")
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			defer wantFile.Close()
			byteValue, err := io.ReadAll(wantFile)
			if err != nil {
				t.Fatalf("err: %s", err)
			}
			json.Unmarshal([]byte(byteValue), &want)

			if !cmp.Equal(got, want) {
				t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
			}
		})
	}
}

func TestProvidersSchema_output_withStateStore(t *testing.T) {
	// State with a 'baz' provider not in the config
	originalState := states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "baz_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				AttrsJSON: []byte(`{"id":"bar"}`),
				Status:    states.ObjectReady,
			},
			addrs.AbsProviderConfig{
				Provider: addrs.NewDefaultProvider("baz"),
				Module:   addrs.RootModule,
			},
		)
	})

	// Create a temporary working directory that includes config using
	// a state store in the `test` provider
	td := t.TempDir()
	testCopyDir(t, testFixturePath("provider-schemas-state-store"), td)
	t.Chdir(td)

	// Get bytes describing the state
	var stateBuf bytes.Buffer
	if err := statefile.Write(statefile.New(originalState, "", 1), &stateBuf); err != nil {
		t.Fatalf("error during test setup: %s", err)
	}

	// Create a mock that contains a persisted "default" state that uses the bytes from above.
	mockProvider := mockPluggableStateStorageProvider()
	mockProvider.MockStates = map[string]interface{}{
		"default": stateBuf.Bytes(),
	}
	mockProviderAddressTest := addrs.NewDefaultProvider("test")

	// Mock for the provider in the state
	mockProviderAddressBaz := addrs.NewDefaultProvider("baz")

	ui := new(cli.MockUi)
	c := &ProvidersSchemaCommand{
		Meta: Meta{
			Ui:                        ui,
			AllowExperimentalFeatures: true,
			testingOverrides: &testingOverrides{
				Providers: map[addrs.Provider]providers.Factory{
					mockProviderAddressTest: providers.FactoryFixed(mockProvider),
					mockProviderAddressBaz:  providers.FactoryFixed(mockProvider),
				},
			},
		},
	}

	args := []string{"-json"}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Does the output mention the 2 providers, and the name of the state store?
	wantOutput := []string{
		mockProviderAddressBaz.String(),  // provider from state
		mockProviderAddressTest.String(), // provider from config
		"test_store",                     // the name of the state store implemented in the provider
	}

	output := ui.OutputWriter.String()
	for _, want := range wantOutput {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %s:\n%s", want, output)
		}
	}

	// Does the output match the full expected schema?
	var got, want providerSchemas

	gotString := ui.OutputWriter.String()
	err := json.Unmarshal([]byte(gotString), &got)
	if err != nil {
		t.Fatal(err)
	}

	wantFile, err := os.Open("output.json")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer wantFile.Close()
	byteValue, err := io.ReadAll(wantFile)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	err = json.Unmarshal([]byte(byteValue), &want)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(got, want) {
		t.Fatalf("wrong result:\n %v\n", cmp.Diff(got, want))
	}
}

type providerSchemas struct {
	FormatVersion string                    `json:"format_version"`
	Schemas       map[string]providerSchema `json:"provider_schemas"`
}

type providerSchema struct {
	Provider          interface{}            `json:"provider,omitempty"`
	ResourceSchemas   map[string]interface{} `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]interface{} `json:"data_source_schemas,omitempty"`
	StateStoreSchemas map[string]interface{} `json:"state_store_schemas,omitempty"`
}

// testProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/providers-schema.
func providersSchemaFixtureProvider() *testing_provider.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = providersSchemaFixtureSchema()
	return p
}

// providersSchemaFixtureSchema returns a schema suitable for processing the
// configuration in testdata/providers-schema.ß
func providersSchemaFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Body: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Body: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id":  {Type: cty.String, Optional: true, Computed: true},
						"ami": {Type: cty.String, Optional: true},
						"volumes": {
							NestedType: &configschema.Object{
								Nesting: configschema.NestingList,
								Attributes: map[string]*configschema.Attribute{
									"size":        {Type: cty.String, Required: true},
									"mount_point": {Type: cty.String, Required: true},
								},
							},
							Optional: true,
						},
					},
				},
			},
		},
	}
}

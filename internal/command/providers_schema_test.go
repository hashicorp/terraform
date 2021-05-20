package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"
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
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			td := tempDir(t)
			inputDir := filepath.Join(fixtureDir, entry.Name())
			testCopyDir(t, inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			providerSource, close := newMockProviderSource(t, map[string][]string{
				"test": {"1.2.3"},
			})
			defer close()

			p := providersSchemaFixtureProvider()
			ui := new(cli.MockUi)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
				ProviderSource:   providerSource,
			}

			// `terrafrom init`
			ic := &InitCommand{
				Meta: m,
			}
			if code := ic.Run([]string{}); code != 0 {
				t.Fatalf("init failed\n%s", ui.ErrorWriter)
			}

			// flush the init output from the mock ui
			ui.OutputWriter.Reset()

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
			byteValue, err := ioutil.ReadAll(wantFile)
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

type providerSchemas struct {
	FormatVersion string                    `json:"format_version"`
	Schemas       map[string]providerSchema `json:"provider_schemas"`
}

type providerSchema struct {
	Provider          interface{}            `json:"provider,omitempty"`
	ResourceSchemas   map[string]interface{} `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]interface{} `json:"data_source_schemas,omitempty"`
}

// testProvider returns a mock provider that is configured for basic
// operation with the configuration in testdata/providers-schema.
func providersSchemaFixtureProvider() *terraform.MockProvider {
	p := testProvider()
	p.GetProviderSchemaResponse = providersSchemaFixtureSchema()
	return p
}

// providersSchemaFixtureSchema returns a schema suitable for processing the
// configuration in testdata/providers-schema.ß
func providersSchemaFixtureSchema() *providers.GetProviderSchemaResponse {
	return &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"region": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
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

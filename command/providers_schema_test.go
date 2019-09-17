package command

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/mitchellh/cli"

	"github.com/hashicorp/terraform/helper/copy"
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
	// there's only one test at this time. This can be refactored to have
	// multiple test cases in individual directories as needed.
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
			copy.CopyDir(inputDir, td)
			defer os.RemoveAll(td)
			defer testChdir(t, td)()

			p := showFixtureProvider()
			ui := new(cli.MockUi)
			m := Meta{
				testingOverrides: metaOverridesForProvider(p),
				Ui:               ui,
			}

			// `terrafrom init`
			ic := &InitCommand{
				Meta: m,
				providerInstaller: &mockProviderInstaller{
					Providers: map[string][]string{
						"test": []string{"1.2.3"},
					},
					Dir: m.pluginDir(),
				},
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

package command

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/providers"
)

func TestMetadataSchema_error(t *testing.T) {
	t.Parallel()
	ui := testUiWrapped(t)
	c := &MetadataSchemaCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	// This test will always error because it's missing the -json flag
	if code := c.Run(nil); code != 1 {
		t.Fatalf("expected error, got:\n%s", ui.OutputWriter.String())
	}
}

func TestMetadataSchemas_output(t *testing.T) {
	td := t.TempDir()
	testCopyDir(t, testFixturePath("metadata-schema"), td)
	t.Chdir(td)
	p := simpleMockProvider()
	pf := &testingOverrides{
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"):                        providers.FactoryFixed(p),
			addrs.MustParseProviderSourceString("austinvalle/bufo"): providers.FactoryFixed(p),
			addrs.NewDefaultProvider("example"):                     providers.FactoryFixed(p),
		},
	}

	t.Run("full output", func(t *testing.T) {
		ui := testUiWrapped(t)
		c := &MetadataSchemaCommand{
			Meta: Meta{
				Ui:               ui,
				testingOverrides: pf,
			},
		}

		if code := c.Run([]string{"-json"}); code != 0 {
			t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
		}
		output := ui.OutputWriter.String()
		var got metadataSchemas
		json.Unmarshal([]byte(output), &got)
		if len(got.Schemas) != 3 {
			t.Fatalf("wrong number of provider schemas returned. Got %d, expected 3", len(got.Schemas))
		}
	})

	t.Run("filtered by name", func(t *testing.T) {
		ui := testUiWrapped(t)
		c := &MetadataSchemaCommand{
			Meta: Meta{
				Ui:               ui,
				testingOverrides: pf,
			},
		}
		if code := c.Run([]string{"-json", "-name=test_action"}); code != 0 {
			t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
		}
		output := ui.OutputWriter.String()
		var got metadataSchemas
		json.Unmarshal([]byte(output), &got)

		// every test provider has one test_action
		if len(got.Schemas) != 3 {
			t.Fatalf("wrong number of provider schemas returned. Got %d, expected 3", len(got.Schemas))
		}
		// every provider has the same schema
		for _, s := range got.Schemas {
			if len(s.DataSourceSchemas) > 0 {
				t.Errorf("unexpected ds schemas found in results")
			}
			if len(s.ResourceSchemas) > 0 {
				t.Errorf("unexpected rs schemas found in results")
			}
			if len(s.ActionSchemas) != 1 {
				t.Errorf("wrong number of action schemas. Got %d, expected 1", len(s.ActionSchemas))
			}
			if len(s.ListSchemas) != 0 {
				t.Errorf("unexpected list resource schemas found in results")
			}
			if len(s.Functions) > 0 {
				t.Errorf("unexpected functions found in results")
			}
		}
	})

	t.Run("filtered by provider", func(t *testing.T) {
		ui := testUiWrapped(t)
		c := &MetadataSchemaCommand{
			Meta: Meta{
				Ui:               ui,
				testingOverrides: pf,
			},
		}
		if code := c.Run([]string{"-json", "-provider=austinvalle/bufo"}); code != 0 {
			t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
		}
		output := ui.OutputWriter.String()
		var got providerSchemas
		json.Unmarshal([]byte(output), &got)
		if len(got.Schemas) > 1 {
			t.Fatalf("too many provider schemas returned")
		}
		// todo:  check that the return is populated
	})

	t.Run("pruned output", func(t *testing.T) {
		ui := testUiWrapped(t)
		c := &MetadataSchemaCommand{
			Meta: Meta{
				Ui:               ui,
				testingOverrides: pf,
			},
		}
		if code := c.Run([]string{"-json", "-prune=true"}); code != 0 {
			t.Fatalf("wrong exit status %d; want 0\nstderr: %s", code, ui.ErrorWriter.String())
		}
		output := ui.OutputWriter.String()
		var got metadataSchemas
		json.Unmarshal([]byte(output), &got)

		// since all providers have the same schema, the results are a bit weird and not what you'd expect to see in reality
		if len(got.Schemas) != 3 {
			t.Fatalf("wrong number of provider schemas returned. Got %d, expected 3", len(got.Schemas))
		}
		// still should only be one resource schema returned (from each)
		for _, s := range got.Schemas {
			if len(s.DataSourceSchemas) > 0 {
				t.Errorf("unexpected ds schemas found in results")
			}
			if len(s.ActionSchemas) > 0 {
				t.Errorf("unexpected action schemas found in results")
			}
			if len(s.ResourceSchemas) != 1 {
				t.Errorf("wrong number of resource schemas. Got %d, expected 1", len(s.ResourceSchemas))
			}
			if len(s.ListSchemas) != 0 {
				t.Errorf("unexpected list resource schemas found in results")
			}
			if len(s.Functions) > 0 {
				t.Errorf("unexpected functions found in results")
			}
		}
	})
}

// if this command gets implemented, this will be refactored to be a subset of the overall return
type metadataSchemas struct {
	FormatVersion string                            `json:"format_version"`
	Schemas       map[string]metadataProviderSchema `json:"provider_schemas"`
}

type metadataProviderSchema struct {
	Provider          any            `json:"provider,omitempty"`
	ResourceSchemas   map[string]any `json:"resource_schemas,omitempty"`
	DataSourceSchemas map[string]any `json:"data_source_schemas,omitempty"`
	StateStoreSchemas map[string]any `json:"state_store_schemas,omitempty"`
	ActionSchemas     map[string]any `json:"action_schemas,omitempty"`
	ListSchemas       map[string]any `json:"list_schemas,omitempty"`
	Functions         map[string]any `json:"functions,omitempty"`
}

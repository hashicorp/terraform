package command

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/copy"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestImport(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-implicit"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	configured := false
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		configured = true

		cfg := req.Config
		if !cfg.Type().HasAttribute("foo") {
			return providers.ConfigureProviderResponse{
				Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("configuration has no foo argument")),
			}
		}
		if got, want := cfg.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			return providers.ConfigureProviderResponse{
				Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("foo argument is %#v, but want %#v", got, want)),
			}
		}

		return providers.ConfigureProviderResponse{}
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

// "remote" state provided by the "local" backend
func TestImport_remoteState(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("import-provider-remote-state"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := "imported.tfstate"

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": []string{"1.2.3"},
	})
	defer close()

	// init our backend
	ui := cli.NewMockUi()
	view, _ := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	ic := &InitCommand{
		Meta: m,
	}

	// (Using log here rather than t.Log so that these messages interleave with other trace logs)
	log.Print("[TRACE] TestImport_remoteState running: terraform init")
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	p := testProvider()
	ui = new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	configured := false
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		var diags tfdiags.Diagnostics
		configured = true
		if got, want := req.Config.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			diags = diags.Append(fmt.Errorf("wrong \"foo\" value %#v; want %#v", got, want))
		}
		return providers.ConfigureProviderResponse{
			Diagnostics: diags,
		}
	}

	args := []string{
		"test_instance.foo",
		"bar",
	}
	log.Printf("[TRACE] TestImport_remoteState running: terraform import %s %s", args[0], args[1])
	if code := c.Run(args); code != 0 {
		fmt.Println(ui.OutputWriter)
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// verify that the local state was unlocked after import
	if _, err := os.Stat(filepath.Join(td, fmt.Sprintf(".%s.lock.info", statePath))); !os.IsNotExist(err) {
		t.Fatal("state left locked after import")
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

// early failure on import should not leave stale lock
func TestImport_initializationErrorShouldUnlock(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("import-provider-remote-state"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := "imported.tfstate"

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": []string{"1.2.3"},
	})
	defer close()

	// init our backend
	ui := cli.NewMockUi()
	view, _ := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	ic := &InitCommand{
		Meta: m,
	}

	// (Using log here rather than t.Log so that these messages interleave with other trace logs)
	log.Print("[TRACE] TestImport_initializationErrorShouldUnlock running: terraform init")
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	// overwrite the config with one including a resource from an invalid provider
	copy.CopyFile(filepath.Join(testFixturePath("import-provider-invalid"), "main.tf"), filepath.Join(td, "main.tf"))

	p := testProvider()
	ui = new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"unknown_instance.baz",
		"bar",
	}
	log.Printf("[TRACE] TestImport_initializationErrorShouldUnlock running: terraform import %s %s", args[0], args[1])

	// this should fail
	if code := c.Run(args); code != 1 {
		fmt.Println(ui.OutputWriter)
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// specifically, it should fail due to a missing provider
	msg := strings.ReplaceAll(ui.ErrorWriter.String(), "\n", " ")
	if want := `provider registry.terraform.io/hashicorp/unknown: required by this configuration but no version is selected`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}

	// verify that the local state was unlocked after initialization error
	if _, err := os.Stat(filepath.Join(td, fmt.Sprintf(".%s.lock.info", statePath))); !os.IsNotExist(err) {
		t.Fatal("state left locked after import")
	}
}

func TestImport_providerConfigWithVar(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	configured := false
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		var diags tfdiags.Diagnostics
		configured = true
		if got, want := req.Config.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			diags = diags.Append(fmt.Errorf("wrong \"foo\" value %#v; want %#v", got, want))
		}
		return providers.ConfigureProviderResponse{
			Diagnostics: diags,
		}
	}

	args := []string{
		"-state", statePath,
		"-var", "foo=bar",
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigWithDataSource(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-datasource"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
		DataSources: map[string]providers.Schema{
			"test_data": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 1 {
		t.Fatalf("bad, wanted error: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestImport_providerConfigWithVarDefault(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var-default"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	configured := false
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		var diags tfdiags.Diagnostics
		configured = true
		if got, want := req.Config.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			diags = diags.Append(fmt.Errorf("wrong \"foo\" value %#v; want %#v", got, want))
		}
		return providers.ConfigureProviderResponse{
			Diagnostics: diags,
		}
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_providerConfigWithVarFile(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var-file"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		Provider: providers.Schema{
			Block: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"foo": {Type: cty.String, Optional: true},
				},
			},
		},
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	configured := false
	p.ConfigureProviderFn = func(req providers.ConfigureProviderRequest) providers.ConfigureProviderResponse {
		var diags tfdiags.Diagnostics
		configured = true
		if got, want := req.Config.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			diags = diags.Append(fmt.Errorf("wrong \"foo\" value %#v; want %#v", got, want))
		}
		return providers.ConfigureProviderResponse{
			Diagnostics: diags,
		}
	}

	args := []string{
		"-state", statePath,
		"-var-file", "blah.tfvars",
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	// Verify that we were called
	if !configured {
		t.Fatal("Configure should be called")
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_allowMissingResourceConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = &providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State: cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"id": {Type: cty.String, Optional: true, Computed: true},
					},
				},
			},
		},
	}

	args := []string{
		"-state", statePath,
		"-allow-missing-config",
		"test_instance.foo",
		"bar",
	}

	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !p.ImportResourceStateCalled {
		t.Fatal("ImportResourceState should be called")
	}

	testStateOutput(t, statePath, testImportStr)
}

func TestImport_emptyConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("empty"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `No Terraform configuration files`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_missingResourceConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"test_instance.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `resource address "test_instance.foo" does not exist`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_missingModuleConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"module.baz.test_instance.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `module.baz is not defined in the configuration`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImportModuleVarFile(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("import-module-var-file"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": []string{"1.2.3"},
	})
	defer close()

	// init to install the module
	ui := new(cli.MockUi)
	view, _ := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	ic := &InitCommand{
		Meta: m,
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	// import
	ui = new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}
	args := []string{
		"-state", statePath,
		"module.child.test_instance.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 0 {
		t.Fatalf("import failed; expected success")
	}
}

// This test covers an edge case where a module with a complex input variable
// of nested objects has an invalid default which is overridden by the calling
// context, and is used in locals. If we don't evaluate module call variables
// for the import walk, this results in an error.
//
// The specific example has a variable "foo" which is a nested object:
//
//   foo = { bar = { baz = true } }
//
// This is used as foo = var.foo in the call to the child module, which then
// uses the traversal foo.bar.baz in a local. A default value in the child
// module of {} causes this local evaluation to error, breaking import.
func TestImportModuleInputVariableEvaluation(t *testing.T) {
	td := tempDir(t)
	testCopyDir(t, testFixturePath("import-module-input-variable"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := testTempFile(t)

	p := testProvider()
	p.GetProviderSchemaResponse = &providers.GetProviderSchemaResponse{
		ResourceTypes: map[string]providers.Schema{
			"test_instance": {
				Block: &configschema.Block{
					Attributes: map[string]*configschema.Attribute{
						"foo": {Type: cty.String, Optional: true},
					},
				},
			},
		},
	}

	providerSource, close := newMockProviderSource(t, map[string][]string{
		"test": {"1.2.3"},
	})
	defer close()

	// init to install the module
	ui := new(cli.MockUi)
	view, _ := testView(t)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
		View:             view,
		ProviderSource:   providerSource,
	}

	ic := &InitCommand{
		Meta: m,
	}
	if code := ic.Run([]string{}); code != 0 {
		t.Fatalf("init failed\n%s", ui.ErrorWriter)
	}

	// import
	ui = new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}
	args := []string{
		"-state", statePath,
		"module.child.test_instance.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 0 {
		t.Fatalf("import failed; expected success")
	}
}

func TestImport_dataResource(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"data.test_data_source.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `A managed resource address is required`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_invalidResourceAddr(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"bananas",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `Error: Invalid address`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_targetIsModule(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	view, _ := testView(t)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
			View:             view,
		},
	}

	args := []string{
		"-state", statePath,
		"module.foo",
		"bar",
	}
	code := c.Run(args)
	if code != 1 {
		t.Fatalf("import succeeded; expected failure")
	}

	msg := ui.ErrorWriter.String()
	if want := `Error: Invalid address`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

const testImportStr = `
test_instance.foo:
  ID = yay
  provider = provider["registry.terraform.io/hashicorp/test"]
`

package command

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/hashicorp/terraform/plugin"
	"github.com/hashicorp/terraform/plugin/discovery"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/tfdiags"
)

func TestImport(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-implicit"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
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
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	configured := false
	p.ConfigureNewFn = func(req providers.ConfigureRequest) providers.ConfigureResponse {
		configured = true

		cfg := req.Config
		if !cfg.Type().HasAttribute("foo") {
			return providers.ConfigureResponse{
				Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("configuration has no foo argument")),
			}
		}
		if got, want := cfg.GetAttr("foo"), cty.StringVal("bar"); !want.RawEquals(got) {
			return providers.ConfigureResponse{
				Diagnostics: tfdiags.Diagnostics{}.Append(fmt.Errorf("foo argument is %#v, but want %#v", got, want)),
			}
		}

		return providers.ConfigureResponse{}
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
	copy.CopyDir(testFixturePath("import-provider-remote-state"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	statePath := "imported.tfstate"

	// init our backend
	ui := new(cli.MockUi)
	m := Meta{
		testingOverrides: metaOverridesForProvider(testProvider()),
		Ui:               ui,
	}

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
		t.Fatalf("bad: \n%s", ui.ErrorWriter)
	}

	p := testProvider()
	ui = new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
	}

	args := []string{
		"test_instance.foo",
		"bar",
	}

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

func TestImport_providerConfigWithVar(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
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

func TestImport_providerConfigWithVarDefault(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-var-default"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
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
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	configured := false
	p.ConfigureFn = func(c *terraform.ResourceConfig) error {
		configured = true

		if v, ok := c.Get("foo"); !ok || v.(string) != "bar" {
			return fmt.Errorf("bad value: %#v", v)
		}

		return nil
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

func TestImport_customProvider(t *testing.T) {
	defer testChdir(t, testFixturePath("import-provider-aliased"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
			},
		},
	}

	args := []string{
		"-provider", "test.alias",
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

	testStateOutput(t, statePath, testImportCustomProviderStr)
}

func TestImport_allowMissingResourceConfig(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	p.ImportResourceStateFn = nil
	p.ImportResourceStateResponse = providers.ImportResourceStateResponse{
		ImportedResources: []providers.ImportedResource{
			{
				TypeName: "test_instance",
				State:    cty.ObjectVal(map[string]cty.Value{
					"id": cty.StringVal("yay"),
				}),
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
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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

func TestImport_dataResource(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	if want := `resource address must refer to a managed resource`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_invalidResourceAddr(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	if want := `invalid resource address "bananas"`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

func TestImport_targetIsModule(t *testing.T) {
	defer testChdir(t, testFixturePath("import-missing-resource-config"))()

	statePath := testTempFile(t)

	p := testProvider()
	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
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
	if want := `resource address must include a full resource spec`; !strings.Contains(msg, want) {
		t.Errorf("incorrect message\nwant substring: %s\ngot:\n%s", want, msg)
	}
}

// make sure we search the full plugin path during import
func TestImport_pluginDir(t *testing.T) {
	td := tempDir(t)
	copy.CopyDir(testFixturePath("import-provider"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	// make a fake provider in a custom plugin directory
	if err := os.Mkdir("plugins", 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile("plugins/terraform-provider-test_v1.1.1_x4", []byte("invalid binary"), 0755); err != nil {
		t.Fatal(err)
	}

	ui := new(cli.MockUi)
	c := &ImportCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	// store our custom plugin path, which would normally happen during init
	if err := c.storePluginPath([]string{"./plugins"}); err != nil {
		t.Fatal(err)
	}

	// Now we need to go through some plugin init.
	// This discovers our fake plugin and writes the lock file.
	initCmd := &InitCommand{
		Meta: Meta{
			pluginPath: []string{"./plugins"},
			Ui:         new(cli.MockUi),
		},
		providerInstaller: &discovery.ProviderInstaller{
			PluginProtocolVersion: plugin.Handshake.ProtocolVersion,
		},
	}
	if err := initCmd.getProviders(".", nil, false); err != nil {
		t.Fatal(err)
	}

	args := []string{
		"test_instance.foo",
		"bar",
	}
	if code := c.Run(args); code == 0 {
		t.Fatalf("expected error, got: %s", ui.OutputWriter)
	}

	outMsg := ui.OutputWriter.String()
	// if we were missing a plugin, the output will have some explanation
	// about requirements. If discovery starts verifying binary compatibility,
	// we will need to write a dummy provider above.
	if strings.Contains(outMsg, "requirements") {
		t.Fatal("unexpected output:", outMsg)
	}

	// We wanted a plugin execution error, rather than a requirement error.
	// Looking for "exec" in the error should suffice for now.
	errMsg := ui.ErrorWriter.String()
	if !strings.Contains(errMsg, "exec") {
		t.Fatal("unexpected error:", errMsg)
	}
}

const testImportStr = `
test_instance.foo:
  ID = yay
  provider = provider.test
`

const testImportCustomProviderStr = `
test_instance.foo:
  ID = yay
  provider = provider.test.alias
`

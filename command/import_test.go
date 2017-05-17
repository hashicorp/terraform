package command

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/cli"
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
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

	p.ImportStateFn = nil
	p.ImportStateReturn = []*terraform.InstanceState{
		&terraform.InstanceState{
			ID: "yay",
			Ephemeral: terraform.EphemeralState{
				Type: "test_instance",
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

	if !p.ImportStateCalled {
		t.Fatal("ImportState should be called")
	}

	testStateOutput(t, statePath, testImportCustomProviderStr)
}

const testImportStr = `
test_instance.foo:
  ID = yay
  provider = test
`

const testImportCustomProviderStr = `
test_instance.foo:
  ID = yay
  provider = test.alias
`

package command

import (
	"os"
	"strings"
	"testing"

	"github.com/mitchellh/cli"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/helper/copy"
)

func setupTest(fixturepath string, args ...string) (*cli.MockUi, int) {
	ui := new(cli.MockUi)
	p := testProvider()
	p.GetSchemaReturn = &terraform.ProviderSchema{
		ResourceTypes: map[string]*configschema.Block{
			"test_instance": {
				Attributes: map[string]*configschema.Attribute{
					"ami": {Type: cty.String, Optional: true},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"network_interface": {
						Nesting: configschema.NestingList,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"device_index": {Type: cty.String, Optional: true},
								"description":  {Type: cty.String, Optional: true},
							},
						},
					},
				},
			},
		},
	}
	c := &ValidateCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(p),
			Ui:               ui,
		},
	}

	args = append(args, testFixturePath(fixturepath))

	code := c.Run(args)
	return ui, code
}

func TestValidateCommand(t *testing.T) {
	if ui, code := setupTest("validate-valid"); code != 0 {
		t.Fatalf("unexpected non-successful exit code %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestValidateCommandWithTfvarsFile(t *testing.T) {
	// Create a temporary working directory that is empty because this test
	// requires scanning the current working directory by validate command.
	td := tempDir(t)
	copy.CopyDir(testFixturePath("validate-valid/with-tfvars-file"), td)
	defer os.RemoveAll(td)
	defer testChdir(t, td)()

	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args := []string{}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestValidateFailingCommand(t *testing.T) {
	if ui, code := setupTest("validate-invalid"); code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

func TestValidateFailingCommandMissingQuote(t *testing.T) {
	// FIXME: Re-enable once we've updated core for new data structures
	t.Skip("test temporarily disabled until deep validate supports new config structures")

	ui, code := setupTest("validate-invalid/missing_quote")

	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "IDENT test") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestValidateFailingCommandMissingVariable(t *testing.T) {
	// FIXME: Re-enable once we've updated core for new data structures
	t.Skip("test temporarily disabled until deep validate supports new config structures")

	ui, code := setupTest("validate-invalid/missing_var")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "config: unknown variable referenced: 'description'; define it with a 'variable' block") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestSameProviderMutipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_providers")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	wantError := "Error: Duplicate provider configuration"
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestSameModuleMultipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_modules")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	wantError := "Error: Duplicate module call"
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestSameResourceMultipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_resources")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	wantError := `Error: Duplicate resource "aws_instance" configuration`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestOutputWithoutValueShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/outputs")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	wantError := `The attribute "value" is required, but no definition was found.`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
	wantError = `An attribute named "values" is not expected here. Did you mean "value"?`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestModuleWithIncorrectNameShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/incorrectmodulename")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantError := `Error: Invalid module instance name`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
	wantError = `Error: Variables not allowed`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestWronglyUsedInterpolationShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/interpolation")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	wantError := `Error: Variables not allowed`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
	wantError = `A static variable reference is required.`
	if !strings.Contains(ui.ErrorWriter.String(), wantError) {
		t.Fatalf("Missing error string %q\n\n'%s'", wantError, ui.ErrorWriter.String())
	}
}

func TestMissingDefinedVar(t *testing.T) {
	ui, code := setupTest("validate-invalid/missing_defined_var")
	// This is allowed because validate tests only that variables are referenced
	// correctly, not that they all have defined values.
	if code != 0 {
		t.Fatalf("Should have passed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

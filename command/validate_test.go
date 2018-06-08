package command

import (
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/helper/copy"
	"github.com/mitchellh/cli"
)

func setupTest(fixturepath string, args ...string) (*cli.MockUi, int) {
	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			testingOverrides: metaOverridesForProvider(testProvider()),
			Ui:               ui,
		},
	}

	args = append(args, testFixturePath(fixturepath))

	code := c.Run(args)
	return ui, code
}

func TestValidateCommand(t *testing.T) {
	if ui, code := setupTest("validate-valid"); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
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
	ui, code := setupTest("validate-invalid/missing_quote")

	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "IDENT test") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestValidateFailingCommandMissingVariable(t *testing.T) {
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
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "provider.aws: multiple configurations present; only one configuration is allowed per provider") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestSameModuleMultipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_modules")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "module \"multi_module\": module repeated multiple times") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestSameResourceMultipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_resources")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "aws_instance.web: resource repeated multiple times") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestOutputWithoutValueShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/outputs")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "output \"myvalue\": missing required 'value' argument") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestModuleWithIncorrectNameShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/incorrectmodulename")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "module name must be a letter or underscore followed by only letters, numbers, dashes, and underscores") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
	if !strings.Contains(ui.ErrorWriter.String(), "module source cannot contain interpolations") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestWronglyUsedInterpolationShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/interpolation")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "depends on value cannot contain interpolations") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
	if !strings.Contains(ui.ErrorWriter.String(), "variable \"vairable_with_interpolation\": default may not contain interpolations") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestMissingDefinedVar(t *testing.T) {
	ui, code := setupTest("validate-invalid/missing_defined_var")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "Required variable not set:") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestMissingDefinedVarConfigOnly(t *testing.T) {
	ui, code := setupTest("validate-invalid/missing_defined_var", "-check-variables=false")
	if code != 0 {
		t.Fatalf("Should have passed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
}

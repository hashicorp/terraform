package command

import (
	"strings"
	"testing"

	"github.com/mitchellh/cli"
)

func setupTest(fixturepath string) (*cli.MockUi, int) {
	ui := new(cli.MockUi)
	c := &ValidateCommand{
		Meta: Meta{
			Ui: ui,
		},
	}

	args := []string{
		testFixturePath(fixturepath),
	}

	code := c.Run(args)
	return ui, code
}
func TestValidateCommand(t *testing.T) {
	if ui, code := setupTest("validate-valid"); code != 0 {
		t.Fatalf("bad: %d\n\n%s", code, ui.ErrorWriter.String())
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
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "config: unknown variable referenced: 'description'. define it with 'variable' blocks") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestSameProviderMutipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_providers")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "provider.aws: declared multiple times, you can only declare a provider once") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestSameModuleMultipleTimesShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/multiple_modules")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "multi_module: module repeated multiple times") {
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
	if !strings.HasSuffix(strings.TrimSpace(ui.ErrorWriter.String()), "output is missing required 'value' key") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

func TestModuleWithIncorrectNameShouldFail(t *testing.T) {
	ui, code := setupTest("validate-invalid/incorrectmodulename")
	if code != 1 {
		t.Fatalf("Should have failed: %d\n\n%s", code, ui.ErrorWriter.String())
	}

	if !strings.Contains(ui.ErrorWriter.String(), "module name can only contain letters, numbers, dashes, and underscores") {
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
	if !strings.Contains(ui.ErrorWriter.String(), "Variable 'vairable_with_interpolation': cannot contain interpolations") {
		t.Fatalf("Should have failed: %d\n\n'%s'", code, ui.ErrorWriter.String())
	}
}

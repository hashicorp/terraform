package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
)

func TestContext2Apply_variableCustomValidationsRoot(t *testing.T) {
	// This test is for custom validation rules associated with root module
	// variables, and specifically that we handle the situation where their
	// values are unknown during validation, skipping the validation check
	// altogether. (Root module variables are never known during validation.)
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "test" {
  type = string
	default = "four"

  validation {
	condition     = length(var.test) > 5
	error_message = "Value is only ${length(var.test)}."
  }
}
`,
	})

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
		Variables: InputValues{
			"test": &InputValue{
				Value:      cty.UnknownVal(cty.String),
				SourceType: ValueFromCLIArg,
			},
		},
	})

	_, diags := ctx.Apply()
	if diags.HasErrors() {
		t.Fatalf("unexpected error\ngot: %s", diags.Err().Error())
	}
}

func TestContext2Apply_variableCustomValidationsRootError(t *testing.T) {
	// This test is for custom validation rules associated with root module
	// variables, and specifically that we handle the situation where their
	// values are unknown during validation, skipping the validation check
	// altogether. (Root module variables are never known during validation.)
	m := testModuleInline(t, map[string]string{
		"main.tf": `
variable "test" {
  type = string
	default = "four"

  validation {
		condition     = length(var.test) > 5
		error_message = "not a sentence"
  }
}
`,
	})

	p := testProvider("test")
	ctx := testContext2(t, &ContextOpts{
		Config: m,
		Providers: map[addrs.Provider]providers.Factory{
			addrs.NewDefaultProvider("test"): testProviderFuncFixed(p),
		},
	})

	_, diags := ctx.Apply()
	want := `Invalid validation error message: Validation error message must be at least one full English sentence starting with an uppercase letter and ending with a period or question mark.`
	if !diags.HasErrors() {
		t.Fatalf("expected an error that contained %q", want)
	}
	if got := diags.Err().Error(); !strings.Contains(got, want) {
		t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
	}
}

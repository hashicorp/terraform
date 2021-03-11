package terraform

import (
	"strings"
	"testing"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/providers"
)

func Test_looksLikeSentences(t *testing.T) {
	tests := map[string]struct {
		args string
		want bool
	}{
		"empty sentence": {
			args: "",
			want: false,
		},
		"valid sentence": {
			args: "A valid sentence.",
			want: true,
		},
		"valid sentence with an accent": {
			args: `A Valid sentence with an accent "é".`,
			want: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if got := looksLikeSentences(tt.args); got != tt.want {
				t.Errorf("looksLikeSentences() = %v, want %v", got, tt.want)
			}
		})
	}
}

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

func TestContext2Apply_variableCustomValidationsNilStringError(t *testing.T) {
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
		error_message = ""
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
	want := `Invalid validation error message: An empty string is not a valid nor useful error message.`
	if !diags.HasErrors() {
		t.Fatalf("expected an error that contained %q", want)
	}
	if got := diags.Err().Error(); !strings.Contains(got, want) {
		t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
	}
}

func TestContext2Apply_variableCustomValidationsNonStringError(t *testing.T) {
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
		error_message = [4, 3]
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
	want := `Invalid validation error message: Invalid validation error message result value: string required.`
	if !diags.HasErrors() {
		t.Fatalf("expected an error that contained %q", want)
	}
	if got := diags.Err().Error(); !strings.Contains(got, want) {
		t.Fatalf("expected error to contain %q\nerror was:\n%s", want, got)
	}
}

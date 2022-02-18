package terraform

import (
	"fmt"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestPrepareFinalInputVariableValue(t *testing.T) {
	// This is just a concise way to define a bunch of *configs.Variable
	// objects to use in our tests below. We're only going to decode this
	// config, not fully evaluate it.
	cfgSrc := `
		variable "nullable_required" {
		}
		variable "nullable_optional_default_string" {
			default = "hello"
		}
		variable "nullable_optional_default_null" {
			default = null
		}
		variable "constrained_string_nullable_required" {
			type = string
		}
		variable "constrained_string_nullable_optional_default_string" {
			type    = string
			default = "hello"
		}
		variable "constrained_string_nullable_optional_default_bool" {
			type    = string
			default = true
		}
		variable "constrained_string_nullable_optional_default_null" {
			type    = string
			default = null
		}
		variable "required" {
			nullable = false
		}
		variable "optional_default_string" {
			nullable = false
			default  = "hello"
		}
		variable "constrained_string_required" {
			nullable = false
			type     = string
		}
		variable "constrained_string_optional_default_string" {
			nullable = false
			type     = string
			default  = "hello"
		}
		variable "constrained_string_optional_default_bool" {
			nullable = false
			type     = string
			default  = true
		}
		variable "constrained_string_sensitive_required" {
			sensitive = true
			nullable  = false
			type      = string
		}
	`
	cfg := testModuleInline(t, map[string]string{
		"main.tf": cfgSrc,
	})
	variableConfigs := cfg.Module.Variables

	// Because we loaded our pseudo-module from a temporary file, the
	// declaration source ranges will have unpredictable filenames. We'll
	// fix that here just to make things easier below.
	for _, vc := range variableConfigs {
		vc.DeclRange.Filename = "main.tf"
	}

	tests := []struct {
		varName string
		given   cty.Value
		want    cty.Value
		wantErr string
	}{
		// nullable_required
		{
			"nullable_required",
			cty.NilVal,
			cty.UnknownVal(cty.DynamicPseudoType),
			`Required variable not set: The variable "nullable_required" is required, but is not set.`,
		},
		{
			"nullable_required",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.DynamicPseudoType),
			``, // "required" for a nullable variable means only that it must be set, even if it's set to null
		},
		{
			"nullable_required",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"nullable_required",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// nullable_optional_default_string
		{
			"nullable_optional_default_string",
			cty.NilVal,
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"nullable_optional_default_string",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.DynamicPseudoType), // nullable variables can be really set to null, masking the default
			``,
		},
		{
			"nullable_optional_default_string",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"nullable_optional_default_string",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// nullable_optional_default_null
		{
			"nullable_optional_default_null",
			cty.NilVal,
			cty.NullVal(cty.DynamicPseudoType), // the declared default value
			``,
		},
		{
			"nullable_optional_default_null",
			cty.NullVal(cty.String),
			cty.NullVal(cty.String), // nullable variables can be really set to null, masking the default
			``,
		},
		{
			"nullable_optional_default_null",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"nullable_optional_default_null",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_nullable_required
		{
			"constrained_string_nullable_required",
			cty.NilVal,
			cty.UnknownVal(cty.String),
			`Required variable not set: The variable "constrained_string_nullable_required" is required, but is not set.`,
		},
		{
			"constrained_string_nullable_required",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.String), // the null value still gets converted to match the type constraint
			``,                      // "required" for a nullable variable means only that it must be set, even if it's set to null
		},
		{
			"constrained_string_nullable_required",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_nullable_required",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_nullable_optional_default_string
		{
			"constrained_string_nullable_optional_default_string",
			cty.NilVal,
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"constrained_string_nullable_optional_default_string",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.String), // nullable variables can be really set to null, masking the default
			``,
		},
		{
			"constrained_string_nullable_optional_default_string",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_nullable_optional_default_string",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_nullable_optional_default_bool
		{
			"constrained_string_nullable_optional_default_bool",
			cty.NilVal,
			cty.StringVal("true"), // the declared default value, automatically converted to match type constraint
			``,
		},
		{
			"constrained_string_nullable_optional_default_bool",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.String), // nullable variables can be really set to null, masking the default
			``,
		},
		{
			"constrained_string_nullable_optional_default_bool",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_nullable_optional_default_bool",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_nullable_optional_default_null
		{
			"constrained_string_nullable_optional_default_null",
			cty.NilVal,
			cty.NullVal(cty.String),
			``,
		},
		{
			"constrained_string_nullable_optional_default_null",
			cty.NullVal(cty.DynamicPseudoType),
			cty.NullVal(cty.String),
			``,
		},
		{
			"constrained_string_nullable_optional_default_null",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_nullable_optional_default_null",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// required
		{
			"required",
			cty.NilVal,
			cty.UnknownVal(cty.DynamicPseudoType),
			`Required variable not set: The variable "required" is required, but is not set.`,
		},
		{
			"required",
			cty.NullVal(cty.DynamicPseudoType),
			cty.UnknownVal(cty.DynamicPseudoType),
			`Required variable not set: Unsuitable value for var.required set from outside of the configuration: required variable may not be set to null.`,
		},
		{
			"required",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"required",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// optional_default_string
		{
			"optional_default_string",
			cty.NilVal,
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"optional_default_string",
			cty.NullVal(cty.DynamicPseudoType),
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"optional_default_string",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"optional_default_string",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_required
		{
			"constrained_string_required",
			cty.NilVal,
			cty.UnknownVal(cty.String),
			`Required variable not set: The variable "constrained_string_required" is required, but is not set.`,
		},
		{
			"constrained_string_required",
			cty.NullVal(cty.DynamicPseudoType),
			cty.UnknownVal(cty.String),
			`Required variable not set: Unsuitable value for var.constrained_string_required set from outside of the configuration: required variable may not be set to null.`,
		},
		{
			"constrained_string_required",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_required",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_optional_default_string
		{
			"constrained_string_optional_default_string",
			cty.NilVal,
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"constrained_string_optional_default_string",
			cty.NullVal(cty.DynamicPseudoType),
			cty.StringVal("hello"), // the declared default value
			``,
		},
		{
			"constrained_string_optional_default_string",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_optional_default_string",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// constrained_string_optional_default_bool
		{
			"constrained_string_optional_default_bool",
			cty.NilVal,
			cty.StringVal("true"), // the declared default value, automatically converted to match type constraint
			``,
		},
		{
			"constrained_string_optional_default_bool",
			cty.NullVal(cty.DynamicPseudoType),
			cty.StringVal("true"), // the declared default value, automatically converted to match type constraint
			``,
		},
		{
			"constrained_string_optional_default_bool",
			cty.StringVal("ahoy"),
			cty.StringVal("ahoy"),
			``,
		},
		{
			"constrained_string_optional_default_bool",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},

		// sensitive
		{
			"constrained_string_sensitive_required",
			cty.UnknownVal(cty.String),
			cty.UnknownVal(cty.String),
			``,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%s %#v", test.varName, test.given), func(t *testing.T) {
			varAddr := addrs.InputVariable{Name: test.varName}.Absolute(addrs.RootModuleInstance)
			varCfg := variableConfigs[test.varName]
			if varCfg == nil {
				t.Fatalf("invalid variable name %q", test.varName)
			}

			t.Logf(
				"test case\nvariable:    %s\nconstraint:  %#v\ndefault:     %#v\nnullable:    %#v\ngiven value: %#v",
				varAddr,
				varCfg.Type,
				varCfg.Default,
				varCfg.Nullable,
				test.given,
			)

			rawVal := &InputValue{
				Value:      test.given,
				SourceType: ValueFromCaller,
			}

			got, diags := prepareFinalInputVariableValue(
				varAddr, rawVal, varCfg,
			)

			if test.wantErr != "" {
				if !diags.HasErrors() {
					t.Errorf("unexpected success\nwant error: %s", test.wantErr)
				} else if got, want := diags.Err().Error(), test.wantErr; got != want {
					t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
				}
			} else {
				if diags.HasErrors() {
					t.Errorf("unexpected error\ngot: %s", diags.Err().Error())
				}
			}

			// NOTE: should still have returned some reasonable value even if there was an error
			if !test.want.RawEquals(got) {
				t.Fatalf("wrong result\ngot:  %#v\nwant: %#v", got, test.want)
			}
		})
	}

	t.Run("SourceType error message variants", func(t *testing.T) {
		tests := []struct {
			SourceType  ValueSourceType
			SourceRange tfdiags.SourceRange
			WantTypeErr string
			WantNullErr string
		}{
			{
				ValueFromUnknown,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set from outside of the configuration: string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set from outside of the configuration: required variable may not be set to null.`,
			},
			{
				ValueFromConfig,
				tfdiags.SourceRange{
					Filename: "example.tf",
					Start:    tfdiags.SourcePos(hcl.InitialPos),
					End:      tfdiags.SourcePos(hcl.InitialPos),
				},
				`Invalid value for input variable: The given value is not suitable for var.constrained_string_required declared at main.tf:32,3-41: string required.`,
				`Required variable not set: The given value is not suitable for var.constrained_string_required defined at main.tf:32,3-41: required variable may not be set to null.`,
			},
			{
				ValueFromAutoFile,
				tfdiags.SourceRange{
					Filename: "example.auto.tfvars",
					Start:    tfdiags.SourcePos(hcl.InitialPos),
					End:      tfdiags.SourcePos(hcl.InitialPos),
				},
				`Invalid value for input variable: The given value is not suitable for var.constrained_string_required declared at main.tf:32,3-41: string required.`,
				`Required variable not set: The given value is not suitable for var.constrained_string_required defined at main.tf:32,3-41: required variable may not be set to null.`,
			},
			{
				ValueFromNamedFile,
				tfdiags.SourceRange{
					Filename: "example.tfvars",
					Start:    tfdiags.SourcePos(hcl.InitialPos),
					End:      tfdiags.SourcePos(hcl.InitialPos),
				},
				`Invalid value for input variable: The given value is not suitable for var.constrained_string_required declared at main.tf:32,3-41: string required.`,
				`Required variable not set: The given value is not suitable for var.constrained_string_required defined at main.tf:32,3-41: required variable may not be set to null.`,
			},
			{
				ValueFromCLIArg,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set using -var="constrained_string_required=...": string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set using -var="constrained_string_required=...": required variable may not be set to null.`,
			},
			{
				ValueFromEnvVar,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set using the TF_VAR_constrained_string_required environment variable: string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set using the TF_VAR_constrained_string_required environment variable: required variable may not be set to null.`,
			},
			{
				ValueFromInput,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set using an interactive prompt: string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set using an interactive prompt: required variable may not be set to null.`,
			},
			{
				// NOTE: This isn't actually a realistic case for this particular
				// function, because if we have a value coming from a plan then
				// we must be in the apply step, and we shouldn't be able to
				// get past the plan step if we have invalid variable values,
				// and during planning we'll always have other source types.
				ValueFromPlan,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set from outside of the configuration: string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set from outside of the configuration: required variable may not be set to null.`,
			},
			{
				ValueFromCaller,
				tfdiags.SourceRange{},
				`Invalid value for input variable: Unsuitable value for var.constrained_string_required set from outside of the configuration: string required.`,
				`Required variable not set: Unsuitable value for var.constrained_string_required set from outside of the configuration: required variable may not be set to null.`,
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%s %s", test.SourceType, test.SourceRange.StartString()), func(t *testing.T) {
				varAddr := addrs.InputVariable{Name: "constrained_string_required"}.Absolute(addrs.RootModuleInstance)
				varCfg := variableConfigs[varAddr.Variable.Name]
				t.Run("type error", func(t *testing.T) {
					rawVal := &InputValue{
						Value:       cty.EmptyObjectVal,
						SourceType:  test.SourceType,
						SourceRange: test.SourceRange,
					}

					_, diags := prepareFinalInputVariableValue(
						varAddr, rawVal, varCfg,
					)
					if !diags.HasErrors() {
						t.Fatalf("unexpected success; want error")
					}

					if got, want := diags.Err().Error(), test.WantTypeErr; got != want {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
					}
				})
				t.Run("null error", func(t *testing.T) {
					rawVal := &InputValue{
						Value:       cty.NullVal(cty.DynamicPseudoType),
						SourceType:  test.SourceType,
						SourceRange: test.SourceRange,
					}

					_, diags := prepareFinalInputVariableValue(
						varAddr, rawVal, varCfg,
					)
					if !diags.HasErrors() {
						t.Fatalf("unexpected success; want error")
					}

					if got, want := diags.Err().Error(), test.WantNullErr; got != want {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
					}
				})
			})
		}
	})

	t.Run("SensitiveVariable error message variants, with source variants", func(t *testing.T) {
		tests := []struct {
			SourceType  ValueSourceType
			SourceRange tfdiags.SourceRange
			WantTypeErr string
			HideSubject bool
		}{
			{
				ValueFromUnknown,
				tfdiags.SourceRange{},
				"Invalid value for input variable: Unsuitable value for var.constrained_string_sensitive_required set from outside of the configuration: string required.",
				false,
			},
			{
				ValueFromConfig,
				tfdiags.SourceRange{
					Filename: "example.tf",
					Start:    tfdiags.SourcePos(hcl.InitialPos),
					End:      tfdiags.SourcePos(hcl.InitialPos),
				},
				`Invalid value for input variable: The given value is not suitable for var.constrained_string_sensitive_required declared at main.tf:46,3-51: string required.

var.constrained_string_sensitive_required is marked as sensitive. Invalid value defined at example.tf:1,1-1.`,
				true,
			},
		}

		for _, test := range tests {
			t.Run(fmt.Sprintf("%s %s", test.SourceType, test.SourceRange.StartString()), func(t *testing.T) {
				varAddr := addrs.InputVariable{Name: "constrained_string_sensitive_required"}.Absolute(addrs.RootModuleInstance)
				varCfg := variableConfigs[varAddr.Variable.Name]
				t.Run("type error", func(t *testing.T) {
					rawVal := &InputValue{
						Value:       cty.EmptyObjectVal,
						SourceType:  test.SourceType,
						SourceRange: test.SourceRange,
					}

					_, diags := prepareFinalInputVariableValue(
						varAddr, rawVal, varCfg,
					)
					if !diags.HasErrors() {
						t.Fatalf("unexpected success; want error")
					}

					if got, want := diags.Err().Error(), test.WantTypeErr; got != want {
						t.Errorf("wrong error\ngot:  %s\nwant: %s", got, want)
					}

					if test.HideSubject != (diags[0].Source().Subject == nil) {
						t.Errorf("Subject (code context) should have been masked\ngot:  %v", diags[0].Source().Subject)
					}
				})
			})
		}
	})
}

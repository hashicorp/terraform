package cloud

import (
	"context"
	"reflect"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/initwd"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/zclconf/go-cty/cty"
)

func TestRemoteStoredVariableValue(t *testing.T) {
	tests := map[string]struct {
		Def       *tfe.Variable
		Want      cty.Value
		WantError string
	}{
		"string literal": {
			&tfe.Variable{
				Key:       "test",
				Value:     "foo",
				HCL:       false,
				Sensitive: false,
			},
			cty.StringVal("foo"),
			``,
		},
		"string HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `"foo"`,
				HCL:       true,
				Sensitive: false,
			},
			cty.StringVal("foo"),
			``,
		},
		"list HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `[]`,
				HCL:       true,
				Sensitive: false,
			},
			cty.EmptyTupleVal,
			``,
		},
		"null HCL": {
			&tfe.Variable{
				Key:       "test",
				Value:     `null`,
				HCL:       true,
				Sensitive: false,
			},
			cty.NullVal(cty.DynamicPseudoType),
			``,
		},
		"literal sensitive": {
			&tfe.Variable{
				Key:       "test",
				HCL:       false,
				Sensitive: true,
			},
			cty.UnknownVal(cty.String),
			``,
		},
		"HCL sensitive": {
			&tfe.Variable{
				Key:       "test",
				HCL:       true,
				Sensitive: true,
			},
			cty.DynamicVal,
			``,
		},
		"HCL computation": {
			// This (stored expressions containing computation) is not a case
			// we intentionally supported, but it became possible for remote
			// operations in Terraform 0.12 (due to Terraform Cloud/Enterprise
			// just writing the HCL verbatim into generated `.tfvars` files).
			// We support it here for consistency, and we continue to support
			// it in both places for backward-compatibility. In practice,
			// there's little reason to do computation in a stored variable
			// value because references are not supported.
			&tfe.Variable{
				Key:       "test",
				Value:     `[for v in ["a"] : v]`,
				HCL:       true,
				Sensitive: false,
			},
			cty.TupleVal([]cty.Value{cty.StringVal("a")}),
			``,
		},
		"HCL syntax error": {
			&tfe.Variable{
				Key:       "test",
				Value:     `[`,
				HCL:       true,
				Sensitive: false,
			},
			cty.DynamicVal,
			`Invalid expression for var.test: The value of variable "test" is marked in the remote workspace as being specified in HCL syntax, but the given value is not valid HCL. Stored variable values must be valid literal expressions and may not contain references to other variables or calls to functions.`,
		},
		"HCL with references": {
			&tfe.Variable{
				Key:       "test",
				Value:     `foo.bar`,
				HCL:       true,
				Sensitive: false,
			},
			cty.DynamicVal,
			`Invalid expression for var.test: The value of variable "test" is marked in the remote workspace as being specified in HCL syntax, but the given value is not valid HCL. Stored variable values must be valid literal expressions and may not contain references to other variables or calls to functions.`,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			v := &remoteStoredVariableValue{
				definition: test.Def,
			}
			// This ParseVariableValue implementation ignores the parsing mode,
			// so we'll just always parse literal here. (The parsing mode is
			// selected by the remote server, not by our local configuration.)
			gotIV, diags := v.ParseVariableValue(configs.VariableParseLiteral)
			if test.WantError != "" {
				if !diags.HasErrors() {
					t.Fatalf("missing expected error\ngot:  <no error>\nwant: %s", test.WantError)
				}
				errStr := diags.Err().Error()
				if errStr != test.WantError {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", errStr, test.WantError)
				}
			} else {
				if diags.HasErrors() {
					t.Fatalf("unexpected error\ngot:  %s\nwant: <no error>", diags.Err().Error())
				}
				got := gotIV.Value
				if !test.Want.RawEquals(got) {
					t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, test.Want)
				}
			}
		})
	}
}

func TestRemoteContextWithVars(t *testing.T) {
	catTerraform := tfe.CategoryTerraform
	catEnv := tfe.CategoryEnv

	tests := map[string]struct {
		Opts      *tfe.VariableCreateOptions
		WantError string
	}{
		"Terraform variable": {
			&tfe.VariableCreateOptions{
				Category: &catTerraform,
			},
			`Value for undeclared variable: A variable named "key" was assigned a value, but the root module does not declare a variable of that name. To use this value, add a "variable" block to the configuration.`,
		},
		"environment variable": {
			&tfe.VariableCreateOptions{
				Category: &catEnv,
			},
			``,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			configDir := "./testdata/empty"

			b, bCleanup := testBackendWithName(t)
			defer bCleanup()

			_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
			defer configCleanup()

			workspaceID, err := b.getRemoteWorkspaceID(context.Background(), testBackendSingleWorkspaceName)
			if err != nil {
				t.Fatal(err)
			}

			streams, _ := terminal.StreamsForTesting(t)
			view := views.NewStateLocker(arguments.ViewHuman, views.NewView(streams))

			op := &backend.Operation{
				ConfigDir:    configDir,
				ConfigLoader: configLoader,
				StateLocker:  clistate.NewLocker(0, view),
				Workspace:    testBackendSingleWorkspaceName,
			}

			v := test.Opts
			if v.Key == nil {
				key := "key"
				v.Key = &key
			}
			b.client.Variables.Create(context.TODO(), workspaceID, *v)

			_, _, diags := b.LocalRun(op)

			if test.WantError != "" {
				if !diags.HasErrors() {
					t.Fatalf("missing expected error\ngot:  <no error>\nwant: %s", test.WantError)
				}
				errStr := diags.Err().Error()
				if errStr != test.WantError {
					t.Fatalf("wrong error\ngot:  %s\nwant: %s", errStr, test.WantError)
				}
				// When Context() returns an error, it should unlock the state,
				// so re-locking it is expected to succeed.
				stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
				if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
					t.Fatalf("unexpected error locking state: %s", err.Error())
				}
			} else {
				if diags.HasErrors() {
					t.Fatalf("unexpected error\ngot:  %s\nwant: <no error>", diags.Err().Error())
				}
				// When Context() succeeds, this should fail w/ "workspace already locked"
				stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
				if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err == nil {
					t.Fatal("unexpected success locking state after Context")
				}
			}
		})
	}
}

func TestRemoteVariablesDoNotOverride(t *testing.T) {
	catTerraform := tfe.CategoryTerraform

	varName1 := "key1"
	varName2 := "key2"
	varName3 := "key3"

	varValue1 := "value1"
	varValue2 := "value2"
	varValue3 := "value3"

	tests := map[string]struct {
		localVariables    map[string]backend.UnparsedVariableValue
		remoteVariables   []*tfe.VariableCreateOptions
		expectedVariables terraform.InputValues
	}{
		"no local variables": {
			map[string]backend.UnparsedVariableValue{},
			[]*tfe.VariableCreateOptions{
				{
					Key:      &varName1,
					Value:    &varValue1,
					Category: &catTerraform,
				},
				{
					Key:      &varName2,
					Value:    &varValue2,
					Category: &catTerraform,
				},
				{
					Key:      &varName3,
					Value:    &varValue3,
					Category: &catTerraform,
				},
			},
			terraform.InputValues{
				varName1: &terraform.InputValue{
					Value:      cty.StringVal(varValue1),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName2: &terraform.InputValue{
					Value:      cty.StringVal(varValue2),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName3: &terraform.InputValue{
					Value:      cty.StringVal(varValue3),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
			},
		},
		"single conflicting local variable": {
			map[string]backend.UnparsedVariableValue{
				varName3: testUnparsedVariableValue{source: terraform.ValueFromNamedFile, value: cty.StringVal(varValue3)},
			},
			[]*tfe.VariableCreateOptions{
				{
					Key:      &varName1,
					Value:    &varValue1,
					Category: &catTerraform,
				}, {
					Key:      &varName2,
					Value:    &varValue2,
					Category: &catTerraform,
				}, {
					Key:      &varName3,
					Value:    &varValue3,
					Category: &catTerraform,
				},
			},
			terraform.InputValues{
				varName1: &terraform.InputValue{
					Value:      cty.StringVal(varValue1),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName2: &terraform.InputValue{
					Value:      cty.StringVal(varValue2),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName3: &terraform.InputValue{
					Value:      cty.StringVal(varValue3),
					SourceType: terraform.ValueFromNamedFile,
					SourceRange: tfdiags.SourceRange{
						Filename: "fake.tfvars",
						Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
						End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					},
				},
			},
		},
		"no conflicting local variable": {
			map[string]backend.UnparsedVariableValue{
				varName3: testUnparsedVariableValue{source: terraform.ValueFromNamedFile, value: cty.StringVal(varValue3)},
			},
			[]*tfe.VariableCreateOptions{
				{
					Key:      &varName1,
					Value:    &varValue1,
					Category: &catTerraform,
				}, {
					Key:      &varName2,
					Value:    &varValue2,
					Category: &catTerraform,
				},
			},
			terraform.InputValues{
				varName1: &terraform.InputValue{
					Value:      cty.StringVal(varValue1),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName2: &terraform.InputValue{
					Value:      cty.StringVal(varValue2),
					SourceType: terraform.ValueFromInput,
					SourceRange: tfdiags.SourceRange{
						Filename: "",
						Start:    tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
						End:      tfdiags.SourcePos{Line: 0, Column: 0, Byte: 0},
					},
				},
				varName3: &terraform.InputValue{
					Value:      cty.StringVal(varValue3),
					SourceType: terraform.ValueFromNamedFile,
					SourceRange: tfdiags.SourceRange{
						Filename: "fake.tfvars",
						Start:    tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
						End:      tfdiags.SourcePos{Line: 1, Column: 1, Byte: 0},
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			configDir := "./testdata/variables"

			b, bCleanup := testBackendWithName(t)
			defer bCleanup()

			_, configLoader, configCleanup := initwd.MustLoadConfigForTests(t, configDir)
			defer configCleanup()

			workspaceID, err := b.getRemoteWorkspaceID(context.Background(), testBackendSingleWorkspaceName)
			if err != nil {
				t.Fatal(err)
			}

			streams, _ := terminal.StreamsForTesting(t)
			view := views.NewStateLocker(arguments.ViewHuman, views.NewView(streams))

			op := &backend.Operation{
				ConfigDir:    configDir,
				ConfigLoader: configLoader,
				StateLocker:  clistate.NewLocker(0, view),
				Workspace:    testBackendSingleWorkspaceName,
				Variables:    test.localVariables,
			}

			for _, v := range test.remoteVariables {
				b.client.Variables.Create(context.TODO(), workspaceID, *v)
			}

			lr, _, diags := b.LocalRun(op)

			if diags.HasErrors() {
				t.Fatalf("unexpected error\ngot:  %s\nwant: <no error>", diags.Err().Error())
			}
			// When Context() succeeds, this should fail w/ "workspace already locked"
			stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
			if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err == nil {
				t.Fatal("unexpected success locking state after Context")
			}

			actual := lr.PlanOpts.SetVariables
			expected := test.expectedVariables

			for expectedKey := range expected {
				actualValue := actual[expectedKey]
				expectedValue := expected[expectedKey]

				if !reflect.DeepEqual(*actualValue, *expectedValue) {
					t.Fatalf("unexpected variable '%s'\ngot:  %v\nwant: %v", expectedKey, actualValue, expectedValue)
				}
			}
		})
	}
}

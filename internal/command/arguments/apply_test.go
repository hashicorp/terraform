package arguments

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/plans"
)

func TestParseApply_basicValid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Apply
	}{
		"defaults": {
			nil,
			&Apply{
				AutoApprove:  false,
				InputEnabled: true,
				PlanPath:     "",
				ViewType:     ViewHuman,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.NormalMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
		"auto-approve, disabled input, and plan path": {
			[]string{"-auto-approve", "-input=false", "saved.tfplan"},
			&Apply{
				AutoApprove:  true,
				InputEnabled: false,
				PlanPath:     "saved.tfplan",
				ViewType:     ViewHuman,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.NormalMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
		"destroy mode": {
			[]string{"-destroy"},
			&Apply{
				AutoApprove:  false,
				InputEnabled: true,
				PlanPath:     "",
				ViewType:     ViewHuman,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.DestroyMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
		"JSON view disables input": {
			[]string{"-json", "-auto-approve"},
			&Apply{
				AutoApprove:  true,
				InputEnabled: false,
				PlanPath:     "",
				ViewType:     ViewJSON,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.NormalMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Operation{}, Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApply(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseApply_json(t *testing.T) {
	testCases := map[string]struct {
		args        []string
		wantSuccess bool
	}{
		"-json": {
			[]string{"-json"},
			false,
		},
		"-json -auto-approve": {
			[]string{"-json", "-auto-approve"},
			true,
		},
		"-json saved.tfplan": {
			[]string{"-json", "saved.tfplan"},
			true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApply(tc.args)

			if tc.wantSuccess {
				if len(diags) > 0 {
					t.Errorf("unexpected diags: %v", diags)
				}
			} else {
				if got, want := diags.Err().Error(), "Plan file or auto-approve required"; !strings.Contains(got, want) {
					t.Errorf("wrong diags\n got: %s\nwant: %s", got, want)
				}
			}

			if got.ViewType != ViewJSON {
				t.Errorf("unexpected view type. got: %#v, want: %#v", got.ViewType, ViewJSON)
			}
		})
	}
}

func TestParseApply_invalid(t *testing.T) {
	got, diags := ParseApply([]string{"-frob"})
	if len(diags) == 0 {
		t.Fatal("expected diags but got none")
	}
	if got, want := diags.Err().Error(), "flag provided but not defined"; !strings.Contains(got, want) {
		t.Fatalf("wrong diags\n got: %s\nwant: %s", got, want)
	}
	if got.ViewType != ViewHuman {
		t.Fatalf("wrong view type, got %#v, want %#v", got.ViewType, ViewHuman)
	}
}

func TestParseApply_tooManyArguments(t *testing.T) {
	got, diags := ParseApply([]string{"saved.tfplan", "please"})
	if len(diags) == 0 {
		t.Fatal("expected diags but got none")
	}
	if got, want := diags.Err().Error(), "Too many command line arguments"; !strings.Contains(got, want) {
		t.Fatalf("wrong diags\n got: %s\nwant: %s", got, want)
	}
	if got.ViewType != ViewHuman {
		t.Fatalf("wrong view type, got %#v, want %#v", got.ViewType, ViewHuman)
	}
}

func TestParseApply_targets(t *testing.T) {
	foobarbaz, _ := addrs.ParseTargetStr("foo_bar.baz")
	boop, _ := addrs.ParseTargetStr("module.boop")
	testCases := map[string]struct {
		args    []string
		want    []addrs.Targetable
		wantErr string
	}{
		"no targets by default": {
			args: nil,
			want: nil,
		},
		"one target": {
			args: []string{"-target=foo_bar.baz"},
			want: []addrs.Targetable{foobarbaz.Subject},
		},
		"two targets": {
			args: []string{"-target=foo_bar.baz", "-target", "module.boop"},
			want: []addrs.Targetable{foobarbaz.Subject, boop.Subject},
		},
		"invalid traversal": {
			args:    []string{"-target=foo."},
			want:    nil,
			wantErr: "Dot must be followed by attribute name",
		},
		"invalid target": {
			args:    []string{"-target=data[0].foo"},
			want:    nil,
			wantErr: "A data source name is required",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApply(tc.args)
			if len(diags) > 0 {
				if tc.wantErr == "" {
					t.Fatalf("unexpected diags: %v", diags)
				} else if got := diags.Err().Error(); !strings.Contains(got, tc.wantErr) {
					t.Fatalf("wrong diags\n got: %s\nwant: %s", got, tc.wantErr)
				}
			}
			if !cmp.Equal(got.Operation.Targets, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(got.Operation.Targets, tc.want))
			}
		})
	}
}

func TestParseApply_replace(t *testing.T) {
	foobarbaz, _ := addrs.ParseAbsResourceInstanceStr("foo_bar.baz")
	foobarbeep, _ := addrs.ParseAbsResourceInstanceStr("foo_bar.beep")
	testCases := map[string]struct {
		args    []string
		want    []addrs.AbsResourceInstance
		wantErr string
	}{
		"no addresses by default": {
			args: nil,
			want: nil,
		},
		"one address": {
			args: []string{"-replace=foo_bar.baz"},
			want: []addrs.AbsResourceInstance{foobarbaz},
		},
		"two addresses": {
			args: []string{"-replace=foo_bar.baz", "-replace", "foo_bar.beep"},
			want: []addrs.AbsResourceInstance{foobarbaz, foobarbeep},
		},
		"non-resource-instance address": {
			args:    []string{"-replace=module.boop"},
			want:    nil,
			wantErr: "A resource instance address is required here.",
		},
		"data resource address": {
			args:    []string{"-replace=data.foo.bar"},
			want:    nil,
			wantErr: "Only managed resources can be used",
		},
		"invalid traversal": {
			args:    []string{"-replace=foo."},
			want:    nil,
			wantErr: "Dot must be followed by attribute name",
		},
		"invalid address": {
			args:    []string{"-replace=data[0].foo"},
			want:    nil,
			wantErr: "A data source name is required",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApply(tc.args)
			if len(diags) > 0 {
				if tc.wantErr == "" {
					t.Fatalf("unexpected diags: %v", diags)
				} else if got := diags.Err().Error(); !strings.Contains(got, tc.wantErr) {
					t.Fatalf("wrong diags\n got: %s\nwant: %s", got, tc.wantErr)
				}
			}
			if !cmp.Equal(got.Operation.ForceReplace, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(got.Operation.Targets, tc.want))
			}
		})
	}
}

func TestParseApply_vars(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want []FlagNameValue
	}{
		"no var flags by default": {
			args: nil,
			want: nil,
		},
		"one var": {
			args: []string{"-var", "foo=bar"},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
			},
		},
		"one var-file": {
			args: []string{"-var-file", "cool.tfvars"},
			want: []FlagNameValue{
				{Name: "-var-file", Value: "cool.tfvars"},
			},
		},
		"ordering preserved": {
			args: []string{
				"-var", "foo=bar",
				"-var-file", "cool.tfvars",
				"-var", "boop=beep",
			},
			want: []FlagNameValue{
				{Name: "-var", Value: "foo=bar"},
				{Name: "-var-file", Value: "cool.tfvars"},
				{Name: "-var", Value: "boop=beep"},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApply(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if vars := got.Vars.All(); !cmp.Equal(vars, tc.want) {
				t.Fatalf("unexpected result\n%s", cmp.Diff(vars, tc.want))
			}
			if got, want := got.Vars.Empty(), len(tc.want) == 0; got != want {
				t.Fatalf("expected Empty() to return %t, but was %t", want, got)
			}
		})
	}
}

func TestParseApplyDestroy_basicValid(t *testing.T) {
	testCases := map[string]struct {
		args []string
		want *Apply
	}{
		"defaults": {
			nil,
			&Apply{
				AutoApprove:  false,
				InputEnabled: true,
				ViewType:     ViewHuman,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.DestroyMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
		"auto-approve and disabled input": {
			[]string{"-auto-approve", "-input=false"},
			&Apply{
				AutoApprove:  true,
				InputEnabled: false,
				ViewType:     ViewHuman,
				State:        &State{Lock: true},
				Vars:         &Vars{},
				Operation: &Operation{
					PlanMode:    plans.DestroyMode,
					Parallelism: 10,
					Refresh:     true,
				},
			},
		},
	}

	cmpOpts := cmpopts.IgnoreUnexported(Operation{}, Vars{}, State{})

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, diags := ParseApplyDestroy(tc.args)
			if len(diags) > 0 {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if diff := cmp.Diff(tc.want, got, cmpOpts); diff != "" {
				t.Errorf("unexpected result\n%s", diff)
			}
		})
	}
}

func TestParseApplyDestroy_invalid(t *testing.T) {
	t.Run("explicit destroy mode", func(t *testing.T) {
		got, diags := ParseApplyDestroy([]string{"-destroy"})
		if len(diags) == 0 {
			t.Fatal("expected diags but got none")
		}
		if got, want := diags.Err().Error(), "Invalid mode option:"; !strings.Contains(got, want) {
			t.Fatalf("wrong diags\n got: %s\nwant: %s", got, want)
		}
		if got.ViewType != ViewHuman {
			t.Fatalf("wrong view type, got %#v, want %#v", got.ViewType, ViewHuman)
		}
	})
}

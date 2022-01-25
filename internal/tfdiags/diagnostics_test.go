package tfdiags

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl/v2"
)

func TestBuild(t *testing.T) {
	type diagFlat struct {
		Severity Severity
		Summary  string
		Detail   string
		Subject  *SourceRange
		Context  *SourceRange
	}

	tests := map[string]struct {
		Cons func(Diagnostics) Diagnostics
		Want []diagFlat
	}{
		"nil": {
			func(diags Diagnostics) Diagnostics {
				return diags
			},
			nil,
		},
		"fmt.Errorf": {
			func(diags Diagnostics) Diagnostics {
				diags = diags.Append(fmt.Errorf("oh no bad"))
				return diags
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "oh no bad",
				},
			},
		},
		"errors.New": {
			func(diags Diagnostics) Diagnostics {
				diags = diags.Append(errors.New("oh no bad"))
				return diags
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "oh no bad",
				},
			},
		},
		"hcl.Diagnostic": {
			func(diags Diagnostics) Diagnostics {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Something bad happened",
					Detail:   "It was really, really bad.",
					Subject: &hcl.Range{
						Filename: "foo.tf",
						Start:    hcl.Pos{Line: 1, Column: 10, Byte: 9},
						End:      hcl.Pos{Line: 2, Column: 3, Byte: 25},
					},
					Context: &hcl.Range{
						Filename: "foo.tf",
						Start:    hcl.Pos{Line: 1, Column: 1, Byte: 0},
						End:      hcl.Pos{Line: 3, Column: 1, Byte: 30},
					},
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "Something bad happened",
					Detail:   "It was really, really bad.",
					Subject: &SourceRange{
						Filename: "foo.tf",
						Start:    SourcePos{Line: 1, Column: 10, Byte: 9},
						End:      SourcePos{Line: 2, Column: 3, Byte: 25},
					},
					Context: &SourceRange{
						Filename: "foo.tf",
						Start:    SourcePos{Line: 1, Column: 1, Byte: 0},
						End:      SourcePos{Line: 3, Column: 1, Byte: 30},
					},
				},
			},
		},
		"hcl.Diagnostics": {
			func(diags Diagnostics) Diagnostics {
				diags = diags.Append(hcl.Diagnostics{
					{
						Severity: hcl.DiagError,
						Summary:  "Something bad happened",
						Detail:   "It was really, really bad.",
					},
					{
						Severity: hcl.DiagWarning,
						Summary:  "Also, somebody sneezed",
						Detail:   "How rude!",
					},
				})
				return diags
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "Something bad happened",
					Detail:   "It was really, really bad.",
				},
				{
					Severity: Warning,
					Summary:  "Also, somebody sneezed",
					Detail:   "How rude!",
				},
			},
		},
		"multierror.Error": {
			func(diags Diagnostics) Diagnostics {
				err := multierror.Append(nil, errors.New("bad thing A"))
				err = multierror.Append(err, errors.New("bad thing B"))
				diags = diags.Append(err)
				return diags
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "bad thing A",
				},
				{
					Severity: Error,
					Summary:  "bad thing B",
				},
			},
		},
		"concat Diagnostics": {
			func(diags Diagnostics) Diagnostics {
				var moreDiags Diagnostics
				moreDiags = moreDiags.Append(errors.New("bad thing A"))
				moreDiags = moreDiags.Append(errors.New("bad thing B"))
				return diags.Append(moreDiags)
			},
			[]diagFlat{
				{
					Severity: Error,
					Summary:  "bad thing A",
				},
				{
					Severity: Error,
					Summary:  "bad thing B",
				},
			},
		},
		"single Diagnostic": {
			func(diags Diagnostics) Diagnostics {
				return diags.Append(SimpleWarning("Don't forget your toothbrush!"))
			},
			[]diagFlat{
				{
					Severity: Warning,
					Summary:  "Don't forget your toothbrush!",
				},
			},
		},
		"multiple appends": {
			func(diags Diagnostics) Diagnostics {
				diags = diags.Append(SimpleWarning("Don't forget your toothbrush!"))
				diags = diags.Append(fmt.Errorf("exploded"))
				return diags
			},
			[]diagFlat{
				{
					Severity: Warning,
					Summary:  "Don't forget your toothbrush!",
				},
				{
					Severity: Error,
					Summary:  "exploded",
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			gotDiags := test.Cons(nil)
			var got []diagFlat
			for _, item := range gotDiags {
				desc := item.Description()
				source := item.Source()
				got = append(got, diagFlat{
					Severity: item.Severity(),
					Summary:  desc.Summary,
					Detail:   desc.Detail,
					Subject:  source.Subject,
					Context:  source.Context,
				})
			}

			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(test.Want))
			}
		})
	}
}

func TestDiagnosticsErr(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var diags Diagnostics
		err := diags.Err()
		if err != nil {
			t.Errorf("got non-nil error %#v; want nil", err)
		}
	})
	t.Run("warning only", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(SimpleWarning("bad"))
		err := diags.Err()
		if err != nil {
			t.Errorf("got non-nil error %#v; want nil", err)
		}
	})
	t.Run("one error", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		err := diags.Err()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		if got, want := err.Error(), "didn't work"; got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("two errors", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(errors.New("didn't work either"))
		err := diags.Err()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("error and warning", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(SimpleWarning("didn't work either"))
		err := diags.Err()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		// Since this "as error" mode is just a fallback for
		// non-diagnostics-aware situations like tests, we don't actually
		// distinguish warnings and errors here since the point is to just
		// get the messages rendered. User-facing code should be printing
		// each diagnostic separately, so won't enter this codepath,
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestDiagnosticsErrWithWarnings(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var diags Diagnostics
		err := diags.ErrWithWarnings()
		if err != nil {
			t.Errorf("got non-nil error %#v; want nil", err)
		}
	})
	t.Run("warning only", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(SimpleWarning("bad"))
		err := diags.ErrWithWarnings()
		if err == nil {
			t.Errorf("got nil error; want NonFatalError")
			return
		}
		if _, ok := err.(NonFatalError); !ok {
			t.Errorf("got %T; want NonFatalError", err)
		}
	})
	t.Run("one error", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		err := diags.ErrWithWarnings()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		if got, want := err.Error(), "didn't work"; got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("two errors", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(errors.New("didn't work either"))
		err := diags.ErrWithWarnings()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("error and warning", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(SimpleWarning("didn't work either"))
		err := diags.ErrWithWarnings()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		// Since this "as error" mode is just a fallback for
		// non-diagnostics-aware situations like tests, we don't actually
		// distinguish warnings and errors here since the point is to just
		// get the messages rendered. User-facing code should be printing
		// each diagnostic separately, so won't enter this codepath,
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
	})
}

func TestDiagnosticsNonFatalErr(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var diags Diagnostics
		err := diags.NonFatalErr()
		if err != nil {
			t.Errorf("got non-nil error %#v; want nil", err)
		}
	})
	t.Run("warning only", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(SimpleWarning("bad"))
		err := diags.NonFatalErr()
		if err == nil {
			t.Errorf("got nil error; want NonFatalError")
			return
		}
		if _, ok := err.(NonFatalError); !ok {
			t.Errorf("got %T; want NonFatalError", err)
		}
	})
	t.Run("one error", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		err := diags.NonFatalErr()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		if got, want := err.Error(), "didn't work"; got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
		if _, ok := err.(NonFatalError); !ok {
			t.Errorf("got %T; want NonFatalError", err)
		}
	})
	t.Run("two errors", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(errors.New("didn't work either"))
		err := diags.NonFatalErr()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
		if _, ok := err.(NonFatalError); !ok {
			t.Errorf("got %T; want NonFatalError", err)
		}
	})
	t.Run("error and warning", func(t *testing.T) {
		var diags Diagnostics
		diags = diags.Append(errors.New("didn't work"))
		diags = diags.Append(SimpleWarning("didn't work either"))
		err := diags.NonFatalErr()
		if err == nil {
			t.Fatalf("got nil error %#v; want non-nil", err)
		}
		// Since this "as error" mode is just a fallback for
		// non-diagnostics-aware situations like tests, we don't actually
		// distinguish warnings and errors here since the point is to just
		// get the messages rendered. User-facing code should be printing
		// each diagnostic separately, so won't enter this codepath,
		want := strings.TrimSpace(`
2 problems:

- didn't work
- didn't work either
`)
		if got := err.Error(); got != want {
			t.Errorf("wrong error message\ngot:  %s\nwant: %s", got, want)
		}
		if _, ok := err.(NonFatalError); !ok {
			t.Errorf("got %T; want NonFatalError", err)
		}
	})
}

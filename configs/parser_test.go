package configs

import (
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"

	"github.com/hashicorp/hcl2/hcl"
	"github.com/spf13/afero"
)

// testParser returns a parser that reads files from the given map, which
// is from paths to file contents.
//
// Since this function uses only in-memory objects, it should never fail.
// If any errors are encountered in practice, this function will panic.
func testParser(files map[string]string) *Parser {
	fs := afero.Afero{Fs: afero.NewMemMapFs()}

	for filePath, contents := range files {
		dirPath := path.Dir(filePath)
		err := fs.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			panic(err)
		}
		err = fs.WriteFile(filePath, []byte(contents), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	return NewParser(fs)
}

// testModuleFromFile reads a single file, wraps it in a module, and returns
// it. This is a helper for use in unit tests.
func testModuleFromFile(filename string) (*Module, hcl.Diagnostics) {
	parser := NewParser(nil)
	f, diags := parser.LoadConfigFile(filename)
	mod, modDiags := NewModule([]*File{f}, nil)
	diags = append(diags, modDiags...)
	return mod, modDiags
}

// testModuleFromDir reads configuration from the given directory path as
// a module and returns it. This is a helper for use in unit tests.
func testModuleFromDir(path string) (*Module, hcl.Diagnostics) {
	parser := NewParser(nil)
	return parser.LoadConfigDir(path)
}

func assertNoDiagnostics(t *testing.T, diags hcl.Diagnostics) bool {
	t.Helper()
	return assertDiagnosticCount(t, diags, 0)
}

func assertDiagnosticCount(t *testing.T, diags hcl.Diagnostics, want int) bool {
	t.Helper()
	if len(diags) != 0 {
		t.Errorf("wrong number of diagnostics %d; want %d", len(diags), want)
		for _, diag := range diags {
			t.Logf("- %s", diag)
		}
		return true
	}
	return false
}

func assertDiagnosticSummary(t *testing.T, diags hcl.Diagnostics, want string) bool {
	t.Helper()

	for _, diag := range diags {
		if diag.Summary == want {
			return false
		}
	}

	t.Errorf("missing diagnostic summary %q", want)
	for _, diag := range diags {
		t.Logf("- %s", diag)
	}
	return true
}

func assertResultDeepEqual(t *testing.T, got, want interface{}) bool {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("wrong result\ngot: %swant: %s", spew.Sdump(got), spew.Sdump(want))
		return true
	}
	return false
}

func stringPtr(s string) *string {
	return &s
}

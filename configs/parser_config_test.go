package configs

import (
	"io/ioutil"
	"path/filepath"
	"testing"
)

// TestParseLoadConfigFileSuccess is a simple test that just verifies that
// a number of test configuration files (in test-fixtures/valid-files) can
// be parsed without raising any diagnostics.
//
// This test does not verify that reading these files produces the correct
// file element contents. More detailed assertions may be made on some subset
// of these configuration files in other tests.
func TestParserLoadConfigFileSuccess(t *testing.T) {
	files, err := ioutil.ReadDir("test-fixtures/valid-files")
	if err != nil {
		t.Fatal(err)
	}

	for _, info := range files {
		name := info.Name()
		t.Run(name, func(t *testing.T) {
			src, err := ioutil.ReadFile(filepath.Join("test-fixtures/valid-files", name))
			if err != nil {
				t.Fatal(err)
			}

			parser := testParser(map[string]string{
				name: string(src),
			})

			_, diags := parser.LoadConfigFile(name)
			if len(diags) != 0 {
				t.Errorf("unexpected diagnostics")
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
			}
		})
	}
}

package configupgrade

import (
	"bytes"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestUpgradeValid(t *testing.T) {
	// This test uses the contents of the test-fixtures/valid directory as
	// a table of tests. Every directory there must have both "input" and
	// "want" subdirectories, where "input" is the configuration to be
	// upgraded and "want" is the expected result.
	fixtureDir := "test-fixtures/valid"
	testDirs, err := ioutil.ReadDir(fixtureDir)
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range testDirs {
		if !entry.IsDir() {
			continue
		}
		t.Run(entry.Name(), func(t *testing.T) {
			inputDir := filepath.Join(fixtureDir, entry.Name(), "input")
			wantDir := filepath.Join(fixtureDir, entry.Name(), "want")

			inputSrc, err := LoadModule(inputDir)
			if err != nil {
				t.Fatal(err)
			}
			wantSrc, err := LoadModule(wantDir)
			if err != nil {
				t.Fatal(err)
			}

			gotSrc, diags := Upgrade(inputSrc)
			if diags.HasErrors() {
				t.Error(diags.Err())
			}

			// Upgrade uses a nil entry as a signal to delete a file, which
			// we can't test here because we aren't modifying an existing
			// dir in place, so we'll just ignore those and leave that mechanism
			// to be tested elsewhere.

			for name, got := range gotSrc {
				if gotSrc[name] == nil {
					delete(gotSrc, name)
					continue
				}
				want, wanted := wantSrc[name]
				if !wanted {
					t.Errorf("unexpected extra output file %q\n=== GOT ===\n%s", name, got)
					continue
				}

				if !bytes.Equal(got, want) {
					t.Errorf("wrong content in %q\n=== GOT ===\n%s\n=== WANT ===\n%s", name, got, want)
				}
			}

			for name, want := range wantSrc {
				if _, present := gotSrc[name]; !present {
					t.Errorf("missing output file %q\n=== WANT ===\n%s", name, want)
				}
			}
		})
	}
}

func TestUpgradeRenameJSON(t *testing.T) {
	inputDir := filepath.Join("test-fixtures/valid/rename-json/input")
	inputSrc, err := LoadModule(inputDir)
	if err != nil {
		t.Fatal(err)
	}

	gotSrc, diags := Upgrade(inputSrc)
	if diags.HasErrors() {
		t.Error(diags.Err())
	}

	// This test fixture is also fully covered by TestUpgradeValid, so
	// we're just testing that the file was renamed here.
	src, exists := gotSrc["misnamed-json.tf"]
	if src != nil {
		t.Errorf("misnamed-json.tf still has content")
	} else if !exists {
		t.Errorf("misnamed-json.tf not marked for deletion")
	}

	src, exists = gotSrc["misnamed-json.tf.json"]
	if src == nil || !exists {
		t.Errorf("misnamed-json.tf.json was not created")
	}
}

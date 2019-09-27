package projectconfigs

import (
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLoad(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got, diags := Load("testdata/empty")
		if diags.HasErrors() {
			t.Fatalf("Unexpected problems: %s", diags.Err().Error())
		}

		want := &Config{
			ProjectRoot: "testdata/empty",
			ConfigFile:  "testdata/empty/.terraform-project.hcl",
			Source:      []byte{},
			Context:     map[string]*ContextValue{},
			Locals:      map[string]*LocalValue{},
			Workspaces:  map[string]*Workspace{},
		}
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("unexpected result\n%s", diff)
		}
	})
}

func TestFindProjectRoot(t *testing.T) {
	tests := []struct {
		StartDir string
		Want     string
		WantErr  string
	}{
		{
			"testdata/subdirs",
			"testdata/subdirs",
			``,
		},
		{
			"testdata/subdirs/",
			"testdata/subdirs",
			``,
		},
		{
			"./testdata/./subdirs",
			"testdata/subdirs",
			``,
		},
		{
			"testdata/subdirs/sub",
			"testdata/subdirs",
			``,
		},
		{
			// NOTE: This test will fail if for some reason the Terraform
			// module directory is cloned beneath some other directory
			// that has a .terraform-project.hcl directory in it. To make
			// the test pass, move your Terraform work tree somewhere else.
			"testdata/nonexist",
			"",
			`start directory "testdata/nonexist" does not exist`,
		},
		{
			"testdata/subdirs/.terraform-project.hcl",
			"",
			`invalid start directory "testdata/subdirs/.terraform-project.hcl": not a directory`,
		},
		{
			"testdata",
			"",
			`no parent directory of testdata contains either a .terraform-project.hcl or a .terraform-project.hcl.json file`,
		},
	}

	for _, test := range tests {
		t.Run(test.StartDir, func(t *testing.T) {
			got, err := FindProjectRoot(test.StartDir)

			if err != nil {
				if test.WantErr == "" {
					t.Fatalf("unexpected error\ngot:  %s\nwant: <nil>", err)
				}
				if got, want := err.Error(), test.WantErr; got != want {
					t.Fatalf("unexpected error\ngot:  %s\nwant: %s", got, want)
				}
				return
			}
			if test.WantErr != "" {
				t.Fatalf("success, but expected error\ngot:  <nil>\nwant: %s", test.WantErr)
			}

			// FindProjectRoot returns an absolute path, but our expectations
			// are relative, so we'll adjust in order to match them.
			want, err := filepath.Abs(test.Want)
			if err != nil {
				t.Fatal(err)
			}

			if got != want {
				t.Fatalf("unexpected result\ngot:  %s\nwant: %s", got, want)
			}
		})
	}
}

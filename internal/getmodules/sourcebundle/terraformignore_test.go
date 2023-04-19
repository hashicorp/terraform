package sourcebundle

import (
	"testing"
)

func TestTerraformIgnore(t *testing.T) {
	// path to directory without .terraformignore
	p, err := parseIgnoreFile("testdata")
	if err != nil {
		t.Fatal(err)
	}
	if len(p) != 4 {
		t.Fatal("A directory without .terraformignore should get the default patterns")
	}

	// load the .terraformignore file's patterns
	ignoreRules, err := parseIgnoreFile("testdata/archive-dir")
	if err != nil {
		t.Fatal(err)
	}
	type file struct {
		// the actual path, should be file path format /dir/subdir/file.extension
		path string
		// should match
		match bool
	}
	paths := []file{
		{
			path:  ".terraform/",
			match: true,
		},
		{
			path:  "included.txt",
			match: false,
		},
		{
			path:  ".terraform/foo/bar",
			match: true,
		},
		{
			path:  ".terraform/foo/bar/more/directories/so/many",
			match: true,
		},
		{
			path:  ".terraform/foo/ignored-subdirectory/",
			match: true,
		},
		{
			path:  "baz.txt",
			match: true,
		},
		{
			path:  "parent/foo/baz.txt",
			match: true,
		},
		{
			path:  "parent/foo/bar.tf",
			match: true,
		},
		{
			path:  "parent/bar/bar.tf",
			match: false,
		},
		// baz.txt is ignored, but a file name including it should not be
		{
			path:  "something/with-baz.txt",
			match: false,
		},
		{
			path:  "something/baz.x",
			match: false,
		},
		// Getting into * patterns
		{
			path:  "foo/ignored-doc.md",
			match: true,
		},
		// Should match [a-z] group
		{
			path:  "bar/something-a.txt",
			match: true,
		},
		// ignore sub- terraform.d paths
		{
			path:  "some-module/terraform.d/x",
			match: true,
		},
		// but not the root one
		{
			path:  "terraform.d/",
			match: false,
		},
		{
			path:  "terraform.d/foo",
			match: false,
		},
		// We ignore the directory, but a file of the same name could exist
		{
			path:  "terraform.d",
			match: false,
		},
		// boop.text is ignored everywhere
		{
			path:  "baz/boop.txt",
			match: true,
		},
		// except at current directory
		{
			path:  "boop.txt",
			match: false,
		},
	}
	for i, p := range paths {
		match := matchIgnoreRule(p.path, ignoreRules)
		if match != p.match {
			t.Fatalf("%s at index %d should be %t", p.path, i, p.match)
		}
	}
}

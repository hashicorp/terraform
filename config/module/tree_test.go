package module

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/copy"
)

func TestTreeChild(t *testing.T) {
	var nilTree *Tree
	if nilTree.Child(nil) != nil {
		t.Fatal("child should be nil")
	}

	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "child"))
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Should be able to get the root child
	if c := tree.Child([]string{}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "root" {
		t.Fatalf("bad: %#v", c.Name())
	} else if !reflect.DeepEqual(c.Path(), []string(nil)) {
		t.Fatalf("bad: %#v", c.Path())
	}

	// Should be able to get the root child
	if c := tree.Child(nil); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "root" {
		t.Fatalf("bad: %#v", c.Name())
	} else if !reflect.DeepEqual(c.Path(), []string(nil)) {
		t.Fatalf("bad: %#v", c.Path())
	}

	// Should be able to get the foo child
	if c := tree.Child([]string{"foo"}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "foo" {
		t.Fatalf("bad: %#v", c.Name())
	} else if !reflect.DeepEqual(c.Path(), []string{"foo"}) {
		t.Fatalf("bad: %#v", c.Path())
	}

	// Should be able to get the nested child
	if c := tree.Child([]string{"foo", "bar"}); c == nil {
		t.Fatal("should not be nil")
	} else if c.Name() != "bar" {
		t.Fatalf("bad: %#v", c.Name())
	} else if !reflect.DeepEqual(c.Path(), []string{"foo", "bar"}) {
		t.Fatalf("bad: %#v", c.Path())
	}
}

func TestTreeLoad(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "basic"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage, GetModeNone); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	if err := tree.Load(storage, GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadStr)
	if actual != expected {
		t.Fatalf("bad: \n\n%s", actual)
	}
}

func TestTreeLoad_duplicate(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "dup"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err == nil {
		t.Fatalf("should error")
	}
}

func TestTreeLoad_copyable(t *testing.T) {
	dir := tempDir(t)
	storage := &getter.FolderStorage{StorageDir: dir}
	cfg := testConfig(t, "basic")
	tree := NewTree("", cfg)

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	if err := tree.Load(storage, GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Now we copy the directory, this COPIES symlink values, and
	// doesn't create symlinks themselves. That is important.
	dir2 := tempDir(t)
	os.RemoveAll(dir2)
	defer os.RemoveAll(dir2)
	if err := copy.CopyDir(dir, dir2); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Now copy the configuration
	cfgDir := tempDir(t)
	os.RemoveAll(cfgDir)
	defer os.RemoveAll(cfgDir)
	if err := copy.CopyDir(cfg.Dir, cfgDir); err != nil {
		t.Fatalf("err: %s", err)
	}

	{
		cfg, err := config.LoadDir(cfgDir)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		tree := NewTree("", cfg)
		storage := &getter.FolderStorage{StorageDir: dir2}

		// This should not error since we already got it!
		if err := tree.Load(storage, GetModeNone); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !tree.Loaded() {
			t.Fatal("should be loaded")
		}
	}
}

func TestTreeLoad_parentRef(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "basic-parent"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage, GetModeNone); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	if err := tree.Load(storage, GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadParentStr)
	if actual != expected {
		t.Fatalf("bad: \n\n%s", actual)
	}
}

func TestTreeLoad_subdir(t *testing.T) {
	fixtures := []string{
		"basic-subdir",
		"basic-tar-subdir",

		// Passing a subpath to go getter extracts only this subpath. The old
		// internal code would keep the entire directory structure, allowing a
		// top-level module to reference others through its parent directory.
		// TODO: this can be removed as a breaking change in a major release.
		"tar-subdir-to-parent",
	}

	for _, tc := range fixtures {
		t.Run(tc, func(t *testing.T) {
			storage := testStorage(t)
			tree := NewTree("", testConfig(t, tc))

			if tree.Loaded() {
				t.Fatal("should not be loaded")
			}

			// This should error because we haven't gotten things yet
			if err := tree.Load(storage, GetModeNone); err == nil {
				t.Fatal("should error")
			}

			if tree.Loaded() {
				t.Fatal("should not be loaded")
			}

			// This should get things
			if err := tree.Load(storage, GetModeGet); err != nil {
				t.Fatalf("err: %s", err)
			}

			if !tree.Loaded() {
				t.Fatal("should be loaded")
			}

			// This should no longer error
			if err := tree.Load(storage, GetModeNone); err != nil {
				t.Fatalf("err: %s", err)
			}

			actual := strings.TrimSpace(tree.String())
			expected := strings.TrimSpace(treeLoadSubdirStr)
			if actual != expected {
				t.Fatalf("bad: \n\n%s", actual)
			}
		})
	}
}

func TestTree_recordSubDir(t *testing.T) {
	td, err := ioutil.TempDir("", "tf-module")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)

	dir := filepath.Join(td, "0131bf0fef686e090b16bdbab4910ddf")

	subDir := "subDirName"

	tree := Tree{}

	// record and read the subdir path
	if err := tree.recordSubdir(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err := tree.getSubdir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if actual != subDir {
		t.Fatalf("expected subDir %q, got %q", subDir, actual)
	}

	// overwrite the path, and nmake sure we get the new one
	subDir = "newSubDir"
	if err := tree.recordSubdir(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err = tree.getSubdir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if actual != subDir {
		t.Fatalf("expected subDir %q, got %q", subDir, actual)
	}

	// create a fake entry
	if err := ioutil.WriteFile(subdirRecordsPath(dir), []byte("BAD DATA"), 0644); err != nil {
		t.Fatal(err)
	}

	// this should fail because there aare now 2 entries
	actual, err = tree.getSubdir(dir)
	if err == nil {
		t.Fatal("expected multiple subdir entries")
	}

	// writing the subdir entry should remove the incorrect value
	if err := tree.recordSubdir(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err = tree.getSubdir(dir)
	if err != nil {
		t.Fatal(err)
	}

	if actual != subDir {
		t.Fatalf("expected subDir %q, got %q", subDir, actual)
	}
}

func TestTreeModules(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))
	actual := tree.Modules()

	expected := []*Module{
		&Module{Name: "foo", Source: "./foo"},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("bad: %#v", actual)
	}
}

func TestTreeName(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))
	actual := tree.Name()

	if actual != RootName {
		t.Fatalf("bad: %#v", actual)
	}
}

// This is a table-driven test for tree validation. This is the preferred
// way to test Validate. Non table-driven tests exist historically but
// that style shouldn't be done anymore.
func TestTreeValidate_table(t *testing.T) {
	cases := []struct {
		Name    string
		Fixture string
		Err     string
	}{
		{
			"provider alias in child",
			"validate-alias-good",
			"",
		},

		{
			"undefined provider alias in child",
			"validate-alias-bad",
			"alias must be defined",
		},

		{
			"root module named root",
			"validate-module-root",
			"cannot contain module",
		},

		{
			"grandchild module named root",
			"validate-module-root-grandchild",
			"",
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d-%s", i, tc.Name), func(t *testing.T) {
			tree := NewTree("", testConfig(t, tc.Fixture))
			if err := tree.Load(testStorage(t), GetModeGet); err != nil {
				t.Fatalf("err: %s", err)
			}

			err := tree.Validate()
			if (err != nil) != (tc.Err != "") {
				t.Fatalf("err: %s", err)
			}
			if err == nil {
				return
			}
			if !strings.Contains(err.Error(), tc.Err) {
				t.Fatalf("err should contain %q: %s", tc.Err, err)
			}
		})
	}
}

func TestTreeValidate_badChild(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-child-bad"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildOutput(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-output"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildOutputToModule(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-output-to-module"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildVar(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-var"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badRoot(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-root-bad"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_good(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-child-good"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestTreeValidate_notLoaded(t *testing.T) {
	tree := NewTree("", testConfig(t, "basic"))

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_requiredChildVar(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-required-var"))

	if err := tree.Load(testStorage(t), GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	err := tree.Validate()
	if err == nil {
		t.Fatal("should error")
	}

	// ensure both variables are mentioned in the output
	errMsg := err.Error()
	for _, v := range []string{"feature", "memory"} {
		if !strings.Contains(errMsg, v) {
			t.Fatalf("no mention of missing variable %q", v)
		}
	}
}

func TestTreeValidate_unknownModule(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-module-unknown"))

	if err := tree.Load(testStorage(t), GetModeNone); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeProviders_basic(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "basic-parent-providers"))

	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	var a, b *Tree
	for _, child := range tree.Children() {
		if child.Name() == "a" {
			a = child
		}
	}

	rootProviders := tree.config.ProviderConfigsByFullName()
	topRaw := rootProviders["top.foo"]

	if a == nil {
		t.Fatal("could not find module 'a'")
	}

	for _, child := range a.Children() {
		if child.Name() == "c" {
			b = child
		}
	}

	if b == nil {
		t.Fatal("could not find module 'c'")
	}

	aProviders := a.config.ProviderConfigsByFullName()
	bottomRaw := aProviders["bottom.foo"]
	bProviders := b.config.ProviderConfigsByFullName()
	bBottom := bProviders["bottom"]

	// compare the configs
	// top.foo should have been copied to a.top
	aTop := aProviders["top"]
	if !reflect.DeepEqual(aTop.RawConfig.RawMap(), topRaw.RawConfig.RawMap()) {
		log.Fatalf("expected config %#v, got %#v",
			topRaw.RawConfig.RawMap(),
			aTop.RawConfig.RawMap(),
		)
	}

	if !reflect.DeepEqual(aTop.Path, []string{RootName}) {
		log.Fatalf(`expected scope for "top": {"root"}, got %#v`, aTop.Path)
	}

	if !reflect.DeepEqual(bBottom.RawConfig.RawMap(), bottomRaw.RawConfig.RawMap()) {
		t.Fatalf("expected config %#v, got %#v",
			bottomRaw.RawConfig.RawMap(),
			bBottom.RawConfig.RawMap(),
		)
	}
	if !reflect.DeepEqual(bBottom.Path, []string{RootName, "a"}) {
		t.Fatalf(`expected scope for "bottom": {"root", "a"}, got %#v`, bBottom.Path)
	}
}

func TestTreeProviders_implicit(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "implicit-parent-providers"))

	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	var child *Tree
	for _, c := range tree.Children() {
		if c.Name() == "child" {
			child = c
		}
	}

	if child == nil {
		t.Fatal("could not find module 'child'")
	}

	// child should have inherited foo
	providers := child.config.ProviderConfigsByFullName()
	foo := providers["foo"]

	if foo == nil {
		t.Fatal("could not find provider 'foo' in child module")
	}

	if !reflect.DeepEqual([]string{RootName}, foo.Path) {
		t.Fatalf(`expected foo scope of {"root"}, got %#v`, foo.Path)
	}

	expected := map[string]interface{}{
		"value": "from root",
	}

	if !reflect.DeepEqual(expected, foo.RawConfig.RawMap()) {
		t.Fatalf(`expected "foo" config %#v, got: %#v`, expected, foo.RawConfig.RawMap())
	}
}

func TestTreeProviders_implicitMultiLevel(t *testing.T) {
	storage := testStorage(t)
	tree := NewTree("", testConfig(t, "implicit-grandparent-providers"))

	if err := tree.Load(storage, GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	var child, grandchild *Tree
	for _, c := range tree.Children() {
		if c.Name() == "child" {
			child = c
		}
	}

	if child == nil {
		t.Fatal("could not find module 'child'")
	}

	for _, c := range child.Children() {
		if c.Name() == "grandchild" {
			grandchild = c
		}
	}
	if grandchild == nil {
		t.Fatal("could not find module 'grandchild'")
	}

	// child should have inherited foo
	providers := child.config.ProviderConfigsByFullName()
	foo := providers["foo"]

	if foo == nil {
		t.Fatal("could not find provider 'foo' in child module")
	}

	if !reflect.DeepEqual([]string{RootName}, foo.Path) {
		t.Fatalf(`expected foo scope of {"root"}, got %#v`, foo.Path)
	}

	expected := map[string]interface{}{
		"value": "from root",
	}

	if !reflect.DeepEqual(expected, foo.RawConfig.RawMap()) {
		t.Fatalf(`expected "foo" config %#v, got: %#v`, expected, foo.RawConfig.RawMap())
	}

	// grandchild should have inherited bar
	providers = grandchild.config.ProviderConfigsByFullName()
	bar := providers["bar"]

	if bar == nil {
		t.Fatal("could not find provider 'bar' in grandchild module")
	}

	if !reflect.DeepEqual([]string{RootName, "child"}, bar.Path) {
		t.Fatalf(`expected bar scope of {"root", "child"}, got %#v`, bar.Path)
	}

	expected = map[string]interface{}{
		"value": "from child",
	}

	if !reflect.DeepEqual(expected, bar.RawConfig.RawMap()) {
		t.Fatalf(`expected "bar" config %#v, got: %#v`, expected, bar.RawConfig.RawMap())
	}
}

const treeLoadStr = `
root
  foo (path: foo)
`

const treeLoadParentStr = `
root
  a (path: a)
    b (path: a, b)
`
const treeLoadSubdirStr = `
root
  foo (path: foo)
    bar (path: foo, bar)
`

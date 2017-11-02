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

	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/helper/copy"
)

func TestTreeChild(t *testing.T) {
	var nilTree *Tree
	if nilTree.Child(nil) != nil {
		t.Fatal("child should be nil")
	}

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	tree := NewTree("", testConfig(t, "child"))
	if err := tree.Load(storage); err != nil {
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
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "basic"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	if err := tree.Load(storage); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadStr)
	if actual != expected {
		t.Fatalf("bad: \n\n%s", actual)
	}
}

func TestTreeLoad_duplicate(t *testing.T) {
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "dup"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err == nil {
		t.Fatalf("should error")
	}
}

func TestTreeLoad_copyable(t *testing.T) {
	dir := tempDir(t)
	storage := &Storage{
		StorageDir: dir,
		Mode:       GetModeGet,
	}
	cfg := testConfig(t, "basic")
	tree := NewTree("", cfg)

	// This should get things
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
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
		storage := &Storage{
			StorageDir: dir2,
			Mode:       GetModeNone,
		}

		// This should not error since we already got it!
		if err := tree.Load(storage); err != nil {
			t.Fatalf("err: %s", err)
		}

		if !tree.Loaded() {
			t.Fatal("should be loaded")
		}
	}
}

func TestTreeLoad_parentRef(t *testing.T) {
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "basic-parent"))

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should error because we haven't gotten things yet
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err == nil {
		t.Fatal("should error")
	}

	if tree.Loaded() {
		t.Fatal("should not be loaded")
	}

	// This should get things
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// This should no longer error
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
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
		"tar-subdir-to-parent",
	}

	for _, tc := range fixtures {
		t.Run(tc, func(t *testing.T) {
			storage := testStorage(t, nil)
			tree := NewTree("", testConfig(t, tc))

			if tree.Loaded() {
				t.Fatal("should not be loaded")
			}

			// This should error because we haven't gotten things yet
			storage.Mode = GetModeNone
			if err := tree.Load(storage); err == nil {
				t.Fatal("should error")
			}

			if tree.Loaded() {
				t.Fatal("should not be loaded")
			}

			// This should get things
			storage.Mode = GetModeGet
			if err := tree.Load(storage); err != nil {
				t.Fatalf("err: %s", err)
			}

			if !tree.Loaded() {
				t.Fatal("should be loaded")
			}

			// This should no longer error
			storage.Mode = GetModeNone
			if err := tree.Load(storage); err != nil {
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

func TestTree_recordManifest(t *testing.T) {
	td, err := ioutil.TempDir("", "tf-module")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)

	storage := Storage{StorageDir: td}

	dir := filepath.Join(td, "0131bf0fef686e090b16bdbab4910ddf")

	subDir := "subDirName"

	// record and read the subdir path
	if err := storage.recordModuleRoot(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err := storage.getModuleRoot(dir)
	if err != nil {
		t.Fatal(err)
	}

	if actual != subDir {
		t.Fatalf("expected subDir %q, got %q", subDir, actual)
	}

	// overwrite the path, and nmake sure we get the new one
	subDir = "newSubDir"
	if err := storage.recordModuleRoot(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err = storage.getModuleRoot(dir)
	if err != nil {
		t.Fatal(err)
	}

	if actual != subDir {
		t.Fatalf("expected subDir %q, got %q", subDir, actual)
	}

	// create a fake entry
	if err := ioutil.WriteFile(filepath.Join(td, manifestName), []byte("BAD DATA"), 0644); err != nil {
		t.Fatal(err)
	}

	// this should fail because there aare now 2 entries
	actual, err = storage.getModuleRoot(dir)
	if err == nil {
		t.Fatal("expected multiple subdir entries")
	}

	// writing the subdir entry should remove the incorrect value
	if err := storage.recordModuleRoot(dir, subDir); err != nil {
		t.Fatal(err)
	}
	actual, err = storage.getModuleRoot(dir)
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
			storage := testStorage(t, nil)
			storage.Mode = GetModeGet
			if err := tree.Load(storage); err != nil {
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

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildOutput(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-output"))

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildOutputToModule(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-output-to-module"))

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badChildVar(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-bad-var"))

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_badRoot(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-root-bad"))

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeValidate_good(t *testing.T) {
	tree := NewTree("", testConfig(t, "validate-child-good"))

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
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

	storage := testStorage(t, nil)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
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

	storage := testStorage(t, nil)
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := tree.Validate(); err == nil {
		t.Fatal("should error")
	}
}

func TestTreeProviders_basic(t *testing.T) {
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "basic-parent-providers"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
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

func TestTreeLoad_conflictingSubmoduleNames(t *testing.T) {
	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "conficting-submodule-names"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("load failed: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	// Try to reload
	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("reload failed: %s", err)
	}

	// verify that the grand-children are correctly loaded
	for _, c := range tree.Children() {
		for _, gc := range c.Children() {
			if len(gc.config.Resources) != 1 {
				t.Fatalf("expected 1 resource in %s, got %d", gc.name, len(gc.config.Resources))
			}
			res := gc.config.Resources[0]
			switch gc.path[0] {
			case "a":
				if res.Name != "a-c" {
					t.Fatal("found wrong resource in a/c:", res.Name)
				}
			case "b":
				if res.Name != "b-c" {
					t.Fatal("found wrong resource in b/c:", res.Name)
				}
			}
		}
	}
}

// changing the source for a module but not the module "path"
func TestTreeLoad_changeIntermediateSource(t *testing.T) {
	// copy the config to our tempdir this time, since we're going to edit it
	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(td)

	if err := copyDir(td, filepath.Join(fixtureDir, "change-intermediate-source")); err != nil {
		t.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(td); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(wd)

	if err := os.MkdirAll(".terraform/modules", 0777); err != nil {
		t.Fatal(err)
	}
	storage := &Storage{StorageDir: ".terraform/modules"}
	cfg, err := config.LoadDir("./")
	if err != nil {
		t.Fatal(err)
	}
	tree := NewTree("", cfg)
	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("load failed: %s", err)
	}

	// now we change the source of our module, without changing its path
	if err := os.Rename("main.tf.disabled", "main.tf"); err != nil {
		t.Fatal(err)
	}

	// reload the tree
	cfg, err = config.LoadDir("./")
	if err != nil {
		t.Fatal(err)
	}
	tree = NewTree("", cfg)
	if err := tree.Load(storage); err != nil {
		t.Fatalf("load failed: %s", err)
	}

	// check for our resource in b
	for _, c := range tree.Children() {
		for _, gc := range c.Children() {
			if len(gc.config.Resources) != 1 {
				t.Fatalf("expected 1 resource in %s, got %d", gc.name, len(gc.config.Resources))
			}
			res := gc.config.Resources[0]
			expected := "c-b"
			if res.Name != expected {
				t.Fatalf("expexted resource %q, got %q", expected, res.Name)
			}
		}
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

const treeLoadRegistrySubdirStr = `
root
  foo (path: foo)
`

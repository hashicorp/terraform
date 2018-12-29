package module

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/test"
)

func TestGetModule(t *testing.T) {
	server := test.Registry()
	defer server.Close()
	disco := test.Disco(server)

	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)
	storage := NewStorage(td, disco)

	// this module exists in a test fixture, and is known by the test.Registry
	// relative to our cwd.
	err = storage.GetModule(filepath.Join(td, "foo"), "registry/local/sub")
	if err != nil {
		t.Fatal(err)
	}

	// list everything to make sure nothing else got unpacked in here
	ls, err := ioutil.ReadDir(td)
	if err != nil {
		t.Fatal(err)
	}

	var names []string
	for _, info := range ls {
		names = append(names, info.Name())
	}

	if !(len(names) == 1 && names[0] == "foo") {
		t.Fatalf("expected only directory 'foo', found entries %q", names)
	}

	_, err = os.Stat(filepath.Join(td, "foo", "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
}

// GitHub archives always contain the module source in a single subdirectory,
// so the registry will return a path with with a `//*` suffix. We need to make
// sure this doesn't intefere with our internal handling of `//` subdir.
func TestRegistryGitHubArchive(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	disco := test.Disco(server)
	storage := testStorage(t, disco)

	tree := NewTree("", testConfig(t, "registry-tar-subdir"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	// stop the registry server, and make sure that we don't need to call out again
	server.Close()
	tree = NewTree("", testConfig(t, "registry-tar-subdir"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadSubdirStr)
	if actual != expected {
		t.Fatalf("got: \n\n%s\nexpected: \n\n%s", actual, expected)
	}
}

// Test that the //subdir notation can be used with registry modules
func TestRegisryModuleSubdir(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	disco := test.Disco(server)
	storage := testStorage(t, disco)
	tree := NewTree("", testConfig(t, "registry-subdir"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(tree.String())
	expected := strings.TrimSpace(treeLoadRegistrySubdirStr)
	if actual != expected {
		t.Fatalf("got: \n\n%s\nexpected: \n\n%s", actual, expected)
	}
}

func TestAccRegistryDiscover(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("skipping ACC test")
	}

	// simply check that we get a valid github URL for this from the registry
	module, err := regsrc.ParseModuleSource("hashicorp/consul/aws")
	if err != nil {
		t.Fatal(err)
	}

	s := NewStorage("/tmp", nil)
	loc, err := s.registry.ModuleLocation(module, "")
	if err != nil {
		t.Fatal(err)
	}

	u, err := url.Parse(loc)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(u.Host, "github.com") {
		t.Fatalf("expected host 'github.com', got: %q", u.Host)
	}

	if !strings.Contains(u.String(), "consul") {
		t.Fatalf("url doesn't contain 'consul': %s", u.String())
	}
}

func TestAccRegistryLoad(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("skipping ACC test")
	}

	storage := testStorage(t, nil)
	tree := NewTree("", testConfig(t, "registry-load"))

	storage.Mode = GetModeGet
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	if !tree.Loaded() {
		t.Fatal("should be loaded")
	}

	storage.Mode = GetModeNone
	if err := tree.Load(storage); err != nil {
		t.Fatalf("err: %s", err)
	}

	// TODO expand this further by fetching some metadata from the registry
	actual := strings.TrimSpace(tree.String())
	if !strings.Contains(actual, "(path: vault)") {
		t.Fatal("missing vault module, got:\n", actual)
	}
}

package module

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
)

// Map of module names and location of test modules.
// Only one version for now, as we only lookup latest from the registry.
type testMod struct {
	location string
	version  string
}

// All the locationes from the mockRegistry start with a file:// scheme. If
// the the location string here doesn't have a scheme, the mockRegistry will
// find the absolute path and return a complete URL.
var testMods = map[string][]testMod{
	"registry/foo/bar": {{
		location: "file:///download/registry/foo/bar/0.2.3//*?archive=tar.gz",
		version:  "0.2.3",
	}},
	"registry/foo/baz": {{
		location: "file:///download/registry/foo/baz/1.10.0//*?archive=tar.gz",
		version:  "1.10.0",
	}},
	"registry/local/sub": {{
		location: "test-fixtures/registry-tar-subdir/foo.tgz//*?archive=tar.gz",
		version:  "0.1.2",
	}},
	"exists-in-registry/identifier/provider": {{
		location: "file:///registry/exists",
		version:  "0.2.0",
	}},
	"test-versions/name/provider": {
		{version: "2.2.0"},
		{version: "2.1.1"},
		{version: "1.2.2"},
		{version: "1.2.1"},
	},
}

func latestVersion(versions []string) string {
	var col version.Collection
	for _, v := range versions {
		ver, err := version.NewVersion(v)
		if err != nil {
			panic(err)
		}
		col = append(col, ver)
	}

	sort.Sort(col)
	return col[len(col)-1].String()
}

func mockRegHandler() http.Handler {
	mux := http.NewServeMux()

	download := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		// handle download request
		re := regexp.MustCompile(`^([-a-z]+/\w+/\w+).*/download$`)
		// download lookup
		matches := re.FindStringSubmatch(p)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		versions, ok := testMods[matches[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		mod := versions[0]

		location := mod.location
		if !strings.HasPrefix(location, "file:///") {
			// we can't use filepath.Abs because it will clean `//`
			wd, _ := os.Getwd()
			location = fmt.Sprintf("file://%s/%s", wd, location)
		}

		w.Header().Set("X-Terraform-Get", location)
		w.WriteHeader(http.StatusNoContent)
		// no body
		return
	}

	versions := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		re := regexp.MustCompile(`^([-a-z]+/\w+/\w+)/versions$`)
		matches := re.FindStringSubmatch(p)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		name := matches[1]
		versions, ok := testMods[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// only adding the single requested module for now
		// this is the minimal that any regisry is epected to support
		mpvs := &response.ModuleProviderVersions{
			Source: name,
		}

		for _, v := range versions {
			mv := &response.ModuleVersion{
				Version: v.version,
			}
			mpvs.Versions = append(mpvs.Versions, mv)
		}

		resp := response.ModuleVersions{
			Modules: []*response.ModuleProviderVersions{mpvs},
		}

		js, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}

	mux.Handle("/v1/modules/",
		http.StripPrefix("/v1/modules/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "/download") {
				download(w, r)
				return
			}

			if strings.HasSuffix(r.URL.Path, "/versions") {
				versions(w, r)
				return
			}

			http.NotFound(w, r)
		})),
	)

	mux.HandleFunc("/.well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"modules.v1":"http://localhost/v1/modules/"}`)
	})
	return mux
}

// Just enough like a registry to exercise our code.
// Returns the location of the latest version
func mockRegistry() *httptest.Server {
	server := httptest.NewServer(mockRegHandler())
	return server
}

// GitHub archives always contain the module source in a single subdirectory,
// so the registry will return a path with with a `//*` suffix. We need to make
// sure this doesn't intefere with our internal handling of `//` subdir.
func TestRegistryGitHubArchive(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	disco := testDisco(server)
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
	server := mockRegistry()
	defer server.Close()

	disco := testDisco(server)
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

	s := NewStorage("/tmp", nil, nil)
	loc, err := s.lookupModuleLocation(module, "")
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

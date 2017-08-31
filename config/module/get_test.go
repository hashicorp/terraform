package module

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
)

// map of module names and version for test module.
// only one version for now, as we only lookup latest from the registry
var testMods = map[string]string{
	"registry/foo/bar": "0.2.3",
	"registry/foo/baz": "1.10.0",
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

// Just enough like a registry to exercise our code.
// Returns the location of the latest version
func mockRegistry() *httptest.Server {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)

	mux.Handle("/v1/modules/",
		http.StripPrefix("/v1/modules/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := strings.TrimLeft(r.URL.Path, "/")
			// handle download request
			download := regexp.MustCompile(`^(\w+/\w+/\w+)/download$`)

			// download lookup
			matches := download.FindStringSubmatch(p)
			if len(matches) != 2 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			version, ok := testMods[matches[1]]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			location := fmt.Sprintf("%s/download/%s/%s", server.URL, matches[1], version)
			w.Header().Set(xTerraformGet, location)
			w.WriteHeader(http.StatusNoContent)
			// no body
			return
		})),
	)

	return server
}

func TestDetectRegistry(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	detector := registryDetector{
		api:    server.URL + "/v1/modules/",
		client: server.Client(),
	}

	for _, tc := range []struct {
		module   string
		location string
		found    bool
		err      bool
	}{
		{
			module:   "registry/foo/bar",
			location: "download/registry/foo/bar/0.2.3",
			found:    true,
		},
		{
			module:   "registry/foo/baz",
			location: "download/registry/foo/baz/1.10.0",
			found:    true,
		},
		// this should not be found, but not stop detection
		{
			module: "registry/foo/notfound",
			found:  false,
		},

		// a full url should not be detected
		{
			module: "http://example.com/registry/foo/notfound",
			found:  false,
		},

		// paths should not be detected
		{
			module: "./local/foo/notfound",
			found:  false,
		},
		{
			module: "/local/foo/notfound",
			found:  false,
		},

		// wrong number of parts can't be regisry IDs
		{
			module: "something/registry/foo/notfound",
			found:  false,
		},
	} {

		t.Run(tc.module, func(t *testing.T) {
			loc, ok, err := detector.Detect(tc.module, "")
			if (err == nil) == tc.err {
				t.Fatalf("expected error? %t; got error :%v", tc.err, err)
			}

			if ok != tc.found {
				t.Fatalf("expected OK == %t", tc.found)
			}

			loc = strings.TrimPrefix(loc, server.URL+"/")
			if strings.TrimPrefix(loc, server.URL) != tc.location {
				t.Fatalf("expected location: %q, got %q", tc.location, loc)
			}
		})

	}
}

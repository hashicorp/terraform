package module

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	getter "github.com/hashicorp/go-getter"
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
		source   string
		location string
		found    bool
		err      bool
	}{
		{
			source:   "registry/foo/bar",
			location: "download/registry/foo/bar/0.2.3",
			found:    true,
		},
		{
			source:   "registry/foo/baz",
			location: "download/registry/foo/baz/1.10.0",
			found:    true,
		},
		// this should not be found, but not stop detection
		{
			source: "registry/foo/notfound",
			found:  false,
		},

		// a full url should not be detected
		{
			source: "http://example.com/registry/foo/notfound",
			found:  false,
		},

		// paths should not be detected
		{
			source: "./local/foo/notfound",
			found:  false,
		},
		{
			source: "/local/foo/notfound",
			found:  false,
		},

		// wrong number of parts can't be regisry IDs
		{
			source: "something/registry/foo/notfound",
			found:  false,
		},
	} {

		t.Run(tc.source, func(t *testing.T) {
			loc, ok, err := detector.Detect(tc.source, "")
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

// check that the full set of detectors works as expected
func TestDetectors(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	regDetector := &registryDetector{
		api:    server.URL + "/v1/modules/",
		client: server.Client(),
	}

	detectors := []getter.Detector{
		new(getter.GitHubDetector),
		new(getter.BitBucketDetector),
		new(getter.S3Detector),
		new(localDetector),
		regDetector,
	}

	for _, tc := range []struct {
		source   string
		location string
		fixture  string
		err      bool
	}{
		{
			source:   "registry/foo/bar",
			location: "download/registry/foo/bar/0.2.3",
		},
		// this should not be found, but not stop detection
		{
			source: "registry/foo/notfound",
			err:    true,
		},
		// a full url should be unchanged
		{
			source: "http://example.com/registry/foo/notfound?" +
				"checksum=sha256:f19056b80a426d797ff9e470da069c171a6c6befa83e2da7f6c706207742acab",
			location: "http://example.com/registry/foo/notfound?" +
				"checksum=sha256:f19056b80a426d797ff9e470da069c171a6c6befa83e2da7f6c706207742acab",
		},

		// forced getters will return untouched
		{
			source:   "git::http://example.com/registry/foo/notfound?param=value",
			location: "git::http://example.com/registry/foo/notfound?param=value",
		},

		// local paths should be detected as such, even if they're match
		// registry modules.
		{
			source: "./registry/foo/bar",
			err:    true,
		},
		{
			source: "/registry/foo/bar",
			err:    true,
		},

		// wrong number of parts can't be regisry IDs
		{
			source: "something/registry/foo/notfound",
			err:    true,
		},

		// make sure a local module that looks like a registry id takes precedence
		{
			source:  "namespace/identifier/provider",
			fixture: "discover-subdirs",
			// this should be found locally
			location: "file://" + filepath.Join(wd, fixtureDir, "discover-subdirs/namespace/identifier/provider"),
		},
	} {

		t.Run(tc.source, func(t *testing.T) {
			dir := wd
			if tc.fixture != "" {
				dir = filepath.Join(wd, fixtureDir, tc.fixture)
				if err := os.Chdir(dir); err != nil {
					t.Fatal(err)
				}
				defer os.Chdir(wd)
			}

			loc, err := getter.Detect(tc.source, dir, detectors)
			if (err == nil) == tc.err {
				t.Fatalf("expected error? %t; got error :%v", tc.err, err)
			}

			loc = strings.TrimPrefix(loc, server.URL+"/")
			if strings.TrimPrefix(loc, server.URL) != tc.location {
				t.Fatalf("expected location: %q, got %q", tc.location, loc)
			}
		})

	}
}

package test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"sort"
	"strings"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/response"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
)

// Disco return a *disco.Disco mapping registry.terraform.io, localhost,
// localhost.localdomain, and example.com to the test server.
func Disco(s *httptest.Server) *disco.Disco {
	services := map[string]interface{}{
		// Note that both with and without trailing slashes are supported behaviours
		// TODO: add specific tests to enumerate both possibilities.
		"modules.v1":   fmt.Sprintf("%s/v1/modules", s.URL),
		"providers.v1": fmt.Sprintf("%s/v1/providers", s.URL),
	}
	d := disco.NewWithCredentialsSource(credsSrc)

	d.ForceHostServices(svchost.Hostname("registry.terraform.io"), services)
	d.ForceHostServices(svchost.Hostname("localhost"), services)
	d.ForceHostServices(svchost.Hostname("localhost.localdomain"), services)
	d.ForceHostServices(svchost.Hostname("example.com"), services)
	return d
}

// Map of module names and location of test modules.
// Only one version for now, as we only lookup latest from the registry.
type testMod struct {
	location string
	version  string
}

// Map of provider names and location of test providers.
// Only one version for now, as we only lookup latest from the registry.
type testProvider struct {
	version string
	os      string
	arch    string
	url     string
}

const (
	testCred = "test-auth-token"
)

var (
	regHost  = svchost.Hostname(regsrc.PublicRegistryHost.Normalized())
	credsSrc = auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
		regHost: {"token": testCred},
	})
)

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
		location: "testdata/registry-tar-subdir/foo.tgz//*?archive=tar.gz",
		version:  "0.1.2",
	}},
	"exists-in-registry/identifier/provider": {{
		location: "file:///registry/exists",
		version:  "0.2.0",
	}},
	"relative/foo/bar": {{ // There is an exception for the "relative/" prefix in the test registry server
		location: "/relative-path",
		version:  "0.2.0",
	}},
	"test-versions/name/provider": {
		{version: "2.2.0"},
		{version: "2.1.1"},
		{version: "1.2.2"},
		{version: "1.2.1"},
	},
	"private/name/provider": {
		{version: "1.0.0"},
	},
}

var testProviders = map[string][]testProvider{
	"-/foo": {
		{
			version: "0.2.3",
			url:     "https://releases.hashicorp.com/terraform-provider-foo/0.2.3/terraform-provider-foo.zip",
		},
		{version: "0.3.0"},
	},
	"-/bar": {
		{
			version: "0.1.1",
			url:     "https://releases.hashicorp.com/terraform-provider-bar/0.1.1/terraform-provider-bar.zip",
		},
		{version: "0.1.2"},
	},
}

func providerAlias(provider string) string {
	re := regexp.MustCompile("^-/")
	if re.MatchString(provider) {
		return re.ReplaceAllString(provider, "terraform-providers/")
	}
	return provider
}

func init() {
	// Add provider aliases
	for provider, info := range testProviders {
		alias := providerAlias(provider)
		testProviders[alias] = info
	}
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

	moduleDownload := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		// handle download request
		re := regexp.MustCompile(`^([-a-z]+/\w+/\w+).*/download$`)
		// download lookup
		matches := re.FindStringSubmatch(p)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// check for auth
		if strings.Contains(matches[0], "private/") {
			if !strings.Contains(r.Header.Get("Authorization"), testCred) {
				http.Error(w, "", http.StatusForbidden)
				return
			}
		}

		versions, ok := testMods[matches[1]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		mod := versions[0]

		location := mod.location
		if !strings.HasPrefix(matches[0], "relative/") && !strings.HasPrefix(location, "file:///") {
			// we can't use filepath.Abs because it will clean `//`
			wd, _ := os.Getwd()
			location = fmt.Sprintf("file://%s/%s", wd, location)
		}

		w.Header().Set("X-Terraform-Get", location)
		w.WriteHeader(http.StatusNoContent)
		// no body
		return
	}

	moduleVersions := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		re := regexp.MustCompile(`^([-a-z]+/\w+/\w+)/versions$`)
		matches := re.FindStringSubmatch(p)
		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// check for auth
		if strings.Contains(matches[1], "private/") {
			if !strings.Contains(r.Header.Get("Authorization"), testCred) {
				http.Error(w, "", http.StatusForbidden)
			}
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
				moduleDownload(w, r)
				return
			}

			if strings.HasSuffix(r.URL.Path, "/versions") {
				moduleVersions(w, r)
				return
			}

			http.NotFound(w, r)
		})),
	)

	providerDownload := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		v := strings.Split(string(p), "/")

		if len(v) != 6 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		name := fmt.Sprintf("%s/%s", v[0], v[1])

		providers, ok := testProviders[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// for this test / moment we will only return the one provider
		loc := response.TerraformProviderPlatformLocation{
			DownloadURL: providers[0].url,
		}

		js, err := json.Marshal(loc)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)

	}

	providerVersions := func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimLeft(r.URL.Path, "/")
		re := regexp.MustCompile(`^([-a-z]+/\w+)/versions$`)
		matches := re.FindStringSubmatch(p)

		if len(matches) != 2 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// check for auth
		if strings.Contains(matches[1], "private/") {
			if !strings.Contains(r.Header.Get("Authorization"), testCred) {
				http.Error(w, "", http.StatusForbidden)
			}
		}

		name := providerAlias(fmt.Sprintf("%s", matches[1]))
		versions, ok := testProviders[name]
		if !ok {
			http.NotFound(w, r)
			return
		}

		// only adding the single requested provider for now
		// this is the minimal that any registry is expected to support
		pvs := &response.TerraformProviderVersions{
			ID: name,
		}

		for _, v := range versions {
			pv := &response.TerraformProviderVersion{
				Version: v.version,
			}
			pvs.Versions = append(pvs.Versions, pv)
		}

		js, err := json.Marshal(pvs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}

	mux.Handle("/v1/providers/",
		http.StripPrefix("/v1/providers/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/download") {
				providerDownload(w, r)
				return
			}

			if strings.HasSuffix(r.URL.Path, "/versions") {
				providerVersions(w, r)
				return
			}

			http.NotFound(w, r)
		})),
	)

	mux.HandleFunc("/.well-known/terraform.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"modules.v1":"http://localhost/v1/modules/", "providers.v1":"http://localhost/v1/providers/"}`)
	})
	return mux
}

// Registry returns an httptest server that mocks out some registry functionality.
func Registry() *httptest.Server {
	return httptest.NewServer(mockRegHandler())
}

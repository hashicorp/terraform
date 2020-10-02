package providercache

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform-svchost/disco"
	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/getproviders"
)

func TestEnsureProviderVersions_local_source(t *testing.T) {
	// create filesystem source using the test provider cache dir
	source := getproviders.NewFilesystemMirrorSource("testdata/cachedir")

	// create a temporary workdir
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	// set up the installer using the temporary directory and filesystem source
	platform := getproviders.Platform{OS: "linux", Arch: "amd64"}
	dir := NewDirWithPlatform(tmpDirPath, platform)
	installer := NewInstaller(dir, source)

	tests := map[string]struct {
		provider string
		version  string
		wantHash getproviders.Hash // getproviders.NilHash if not expected to be installed
		err      string
	}{
		"install-unpacked": {
			provider: "null",
			version:  "2.0.0",
			wantHash: getproviders.HashScheme1.New("qjsREM4DqEWECD43FcPqddZ9oxCG+IaMTxvWPciS05g="),
		},
		"invalid-zip-file": {
			provider: "null",
			version:  "2.1.0",
			wantHash: getproviders.NilHash,
			err:      "zip: not a valid zip file",
		},
		"version-constraint-unmet": {
			provider: "null",
			version:  "2.2.0",
			wantHash: getproviders.NilHash,
			err:      "no available releases match the given constraints 2.2.0",
		},
		"missing-executable": {
			provider: "missing/executable",
			version:  "2.0.0",
			wantHash: getproviders.NilHash, // installation fails for a provider with no executable
			err:      "provider binary not found: could not find executable file starting with terraform-provider-executable",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			ctx := context.TODO()

			provider := addrs.MustParseProviderSourceString(test.provider)
			versionConstraint := getproviders.MustParseVersionConstraints(test.version)
			version := getproviders.MustParseVersion(test.version)
			reqs := getproviders.Requirements{
				provider: versionConstraint,
			}

			newLocks, err := installer.EnsureProviderVersions(ctx, depsfile.NewLocks(), reqs, InstallNewProvidersOnly)
			gotProviderlocks := newLocks.AllProviders()
			wantProviderLocks := map[addrs.Provider]*depsfile.ProviderLock{
				provider: depsfile.NewProviderLock(
					provider,
					version,
					getproviders.MustParseVersionConstraints("= 2.0.0"),
					[]getproviders.Hash{
						test.wantHash,
					},
				),
			}
			if test.wantHash == getproviders.NilHash {
				wantProviderLocks = map[addrs.Provider]*depsfile.ProviderLock{}
			}

			if diff := cmp.Diff(wantProviderLocks, gotProviderlocks, depsfile.ProviderLockComparer); diff != "" {
				t.Errorf("wrong selected\n%s", diff)
			}

			if test.err == "" && err == nil {
				return
			}

			switch err := err.(type) {
			case InstallerError:
				providerError, ok := err.ProviderErrors[provider]
				if !ok {
					t.Fatalf("did not get error for provider %s", provider)
				}

				if got := providerError.Error(); got != test.err {
					t.Fatalf("wrong result\ngot:  %s\nwant: %s\n", got, test.err)
				}
			default:
				t.Fatalf("wrong error type. Expected InstallerError, got %T", err)
			}
		})
	}
}

// This test only verifies protocol errors and does not try for successfull
// installation (at the time of writing, the test files aren't signed so the
// signature verification fails); that's left to the e2e tests.
func TestEnsureProviderVersions_protocol_errors(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	// create a temporary workdir
	tmpDirPath, err := ioutil.TempDir("", "terraform-test-providercache")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDirPath)

	version0 := getproviders.MustParseVersionConstraints("0.1.0") // supports protocol version 1.0
	version1 := getproviders.MustParseVersion("1.2.0")            // this is the expected result in tests with a match
	version2 := getproviders.MustParseVersionConstraints("2.0")   // supports protocol version 99

	// set up the installer using the temporary directory and mock source
	platform := getproviders.Platform{OS: "gameboy", Arch: "lr35902"}
	dir := NewDirWithPlatform(tmpDirPath, platform)
	installer := NewInstaller(dir, source)

	tests := map[string]struct {
		provider     addrs.Provider
		inputVersion getproviders.VersionConstraints
		wantVersion  getproviders.Version
	}{
		"too old": {
			addrs.MustParseProviderSourceString("example.com/awesomesauce/happycloud"),
			version0,
			version1,
		},
		"too new": {
			addrs.MustParseProviderSourceString("example.com/awesomesauce/happycloud"),
			version2,
			version1,
		},
		"unsupported": {
			addrs.MustParseProviderSourceString("example.com/weaksauce/unsupported-protocol"),
			version0,
			getproviders.UnspecifiedVersion,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			reqs := getproviders.Requirements{
				test.provider: test.inputVersion,
			}
			ctx := context.TODO()
			_, err := installer.EnsureProviderVersions(ctx, depsfile.NewLocks(), reqs, InstallNewProvidersOnly)

			switch err := err.(type) {
			case nil:
				t.Fatalf("expected error, got success")
			case InstallerError:
				providerError, ok := err.ProviderErrors[test.provider]
				if !ok {
					t.Fatalf("did not get error for provider %s", test.provider)
				}

				switch providerError := providerError.(type) {
				case getproviders.ErrProtocolNotSupported:
					if !providerError.Suggestion.Same(test.wantVersion) {
						t.Fatalf("wrong result\ngot:  %s\nwant: %s\n", providerError.Suggestion, test.wantVersion)
					}
				default:
					t.Fatalf("wrong error type. Expected ErrProtocolNotSupported, got %T", err)
				}
			default:
				t.Fatalf("wrong error type. Expected InstallerError, got %T", err)
			}
		})
	}
}

// testServices starts up a local HTTP server running a fake provider registry
// service and returns a service discovery object pre-configured to consider
// the host "example.com" to be served by the fake registry service.
//
// The returned discovery object also knows the hostname "not.example.com"
// which does not have a provider registry at all and "too-new.example.com"
// which has a "providers.v99" service that is inoperable but could be useful
// to test the error reporting for detecting an unsupported protocol version.
// It also knows fails.example.com but it refers to an endpoint that doesn't
// correctly speak HTTP, to simulate a protocol error.
//
// The second return value is a function to call at the end of a test function
// to shut down the test server. After you call that function, the discovery
// object becomes useless.
func testServices(t *testing.T) (services *disco.Disco, baseURL string, cleanup func()) {
	server := httptest.NewServer(http.HandlerFunc(fakeRegistryHandler))

	services = disco.New()
	services.ForceHostServices(svchost.Hostname("example.com"), map[string]interface{}{
		"providers.v1": server.URL + "/providers/v1/",
	})
	services.ForceHostServices(svchost.Hostname("not.example.com"), map[string]interface{}{})
	services.ForceHostServices(svchost.Hostname("too-new.example.com"), map[string]interface{}{
		// This service doesn't actually work; it's here only to be
		// detected as "too new" by the discovery logic.
		"providers.v99": server.URL + "/providers/v99/",
	})
	services.ForceHostServices(svchost.Hostname("fails.example.com"), map[string]interface{}{
		"providers.v1": server.URL + "/fails-immediately/",
	})

	// We'll also permit registry.terraform.io here just because it's our
	// default and has some unique features that are not allowed on any other
	// hostname. It behaves the same as example.com, which should be preferred
	// if you're not testing something specific to the default registry in order
	// to ensure that most things are hostname-agnostic.
	services.ForceHostServices(svchost.Hostname("registry.terraform.io"), map[string]interface{}{
		"providers.v1": server.URL + "/providers/v1/",
	})

	return services, server.URL, func() {
		server.Close()
	}
}

// testRegistrySource is a wrapper around testServices that uses the created
// discovery object to produce a Source instance that is ready to use with the
// fake registry services.
//
// As with testServices, the second return value is a function to call at the end
// of your test in order to shut down the test server.
func testRegistrySource(t *testing.T) (source *getproviders.RegistrySource, baseURL string, cleanup func()) {
	services, baseURL, close := testServices(t)
	source = getproviders.NewRegistrySource(services)
	return source, baseURL, close
}

func fakeRegistryHandler(resp http.ResponseWriter, req *http.Request) {
	path := req.URL.EscapedPath()
	if strings.HasPrefix(path, "/fails-immediately/") {
		// Here we take over the socket and just close it immediately, to
		// simulate one possible way a server might not be an HTTP server.
		hijacker, ok := resp.(http.Hijacker)
		if !ok {
			// Not hijackable, so we'll just fail normally.
			// If this happens, tests relying on this will fail.
			resp.WriteHeader(500)
			resp.Write([]byte(`cannot hijack`))
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			resp.WriteHeader(500)
			resp.Write([]byte(`hijack failed`))
			return
		}
		conn.Close()
		return
	}

	if strings.HasPrefix(path, "/pkg/") {
		switch path {
		case "/pkg/awesomesauce/happycloud_1.2.0.zip":
			resp.Write([]byte("some zip file"))
		case "/pkg/awesomesauce/happycloud_1.2.0_SHA256SUMS":
			resp.Write([]byte("000000000000000000000000000000000000000000000000000000000000f00d happycloud_1.2.0.zip\n"))
		case "/pkg/awesomesauce/happycloud_1.2.0_SHA256SUMS.sig":
			resp.Write([]byte("GPG signature"))
		default:
			resp.WriteHeader(404)
			resp.Write([]byte("unknown package file download"))
		}
		return
	}

	if !strings.HasPrefix(path, "/providers/v1/") {
		resp.WriteHeader(404)
		resp.Write([]byte(`not a provider registry endpoint`))
		return
	}

	pathParts := strings.Split(path, "/")[3:]
	if len(pathParts) < 2 {
		resp.WriteHeader(404)
		resp.Write([]byte(`unexpected number of path parts`))
		return
	}
	log.Printf("[TRACE] fake provider registry request for %#v", pathParts)
	if len(pathParts) == 2 {
		switch pathParts[0] + "/" + pathParts[1] {

		case "-/legacy":
			// NOTE: This legacy lookup endpoint is specific to
			// registry.terraform.io and not expected to work on any other
			// registry host.
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"namespace":"legacycorp"}`))

		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace or provider type for direct lookup`))
		}
	}

	if len(pathParts) < 3 {
		resp.WriteHeader(404)
		resp.Write([]byte(`unexpected number of path parts`))
		return
	}

	if pathParts[2] == "versions" {
		if len(pathParts) != 3 {
			resp.WriteHeader(404)
			resp.Write([]byte(`extraneous path parts`))
			return
		}

		switch pathParts[0] + "/" + pathParts[1] {
		case "awesomesauce/happycloud":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			// Note that these version numbers are intentionally misordered
			// so we can test that the client-side code places them in the
			// correct order (lowest precedence first).
			resp.Write([]byte(`{"versions":[{"version":"0.1.0","protocols":["1.0"]},{"version":"2.0.0","protocols":["99.0"]},{"version":"1.2.0","protocols":["5.0"]}, {"version":"1.0.0","protocols":["5.0"]}]}`))
		case "weaksauce/unsupported-protocol":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"versions":[{"version":"0.1.0","protocols":["0.1"]}]}`))
		case "weaksauce/no-versions":
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write([]byte(`{"versions":[]}`))
		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace or provider type`))
		}
		return
	}

	if len(pathParts) == 6 && pathParts[3] == "download" {
		switch pathParts[0] + "/" + pathParts[1] {
		case "awesomesauce/happycloud":
			if pathParts[4] == "nonexist" {
				resp.WriteHeader(404)
				resp.Write([]byte(`unsupported OS`))
				return
			}
			version := pathParts[2]
			body := map[string]interface{}{
				"protocols":             []string{"99.0"},
				"os":                    pathParts[4],
				"arch":                  pathParts[5],
				"filename":              "happycloud_" + version + ".zip",
				"shasum":                "000000000000000000000000000000000000000000000000000000000000f00d",
				"download_url":          "/pkg/awesomesauce/happycloud_" + version + ".zip",
				"shasums_url":           "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS",
				"shasums_signature_url": "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS.sig",
				"signing_keys": map[string]interface{}{
					"gpg_public_keys": []map[string]interface{}{
						{
							"ascii_armor": getproviders.HashicorpPublicKey,
						},
					},
				},
			}
			enc, err := json.Marshal(body)
			if err != nil {
				resp.WriteHeader(500)
				resp.Write([]byte("failed to encode body"))
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write(enc)
		case "weaksauce/unsupported-protocol":
			var protocols []string
			version := pathParts[2]
			switch version {
			case "0.1.0":
				protocols = []string{"1.0"}
			case "2.0.0":
				protocols = []string{"99.0"}
			default:
				protocols = []string{"5.0"}
			}

			body := map[string]interface{}{
				"protocols":             protocols,
				"os":                    pathParts[4],
				"arch":                  pathParts[5],
				"filename":              "happycloud_" + version + ".zip",
				"shasum":                "000000000000000000000000000000000000000000000000000000000000f00d",
				"download_url":          "/pkg/awesomesauce/happycloud_" + version + ".zip",
				"shasums_url":           "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS",
				"shasums_signature_url": "/pkg/awesomesauce/happycloud_" + version + "_SHA256SUMS.sig",
				"signing_keys": map[string]interface{}{
					"gpg_public_keys": []map[string]interface{}{
						{
							"ascii_armor": getproviders.HashicorpPublicKey,
						},
					},
				},
			}
			enc, err := json.Marshal(body)
			if err != nil {
				resp.WriteHeader(500)
				resp.Write([]byte("failed to encode body"))
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			resp.Write(enc)
		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace/provider/version/architecture`))
		}
		return
	}

	resp.WriteHeader(404)
	resp.Write([]byte(`unrecognized path scheme`))
}

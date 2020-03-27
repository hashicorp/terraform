package getproviders

import (
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"
	disco "github.com/hashicorp/terraform-svchost/disco"
)

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
func testRegistrySource(t *testing.T) (source *RegistrySource, baseURL string, cleanup func()) {
	services, baseURL, close := testServices(t)
	source = NewRegistrySource(services)
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
			resp.Write([]byte(`{"versions":[{"version":"1.2.0"}, {"version":"1.0.0"}]}`))
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
			resp.Header().Set("Content-Type", "application/json")
			resp.WriteHeader(200)
			// Note that these version numbers are intentionally misordered
			// so we can test that the client-side code places them in the
			// correct order (lowest precedence first).
			resp.Write([]byte(`{"protocols":["5.0"],"os":"` + pathParts[4] + `","arch":"` + pathParts[5] + `","filename":"happycloud_` + pathParts[2] + `.zip","download_url":"/pkg/happycloud_` + pathParts[2] + `.zip","shasum":"000000000000000000000000000000000000000000000000000000000000f00d"}`))
		default:
			resp.WriteHeader(404)
			resp.Write([]byte(`unknown namespace/provider/version/architecture`))
		}
		return
	}

	resp.WriteHeader(404)
	resp.Write([]byte(`unrecognized path scheme`))
}

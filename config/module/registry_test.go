package module

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/svchost/disco"
)

// Return a transport to use for this test server.
// This not only loads the tls.Config from the test server for proper cert
// validation, but also inserts a Dialer that resolves localhost and
// example.com to 127.0.0.1 with the correct port, since 127.0.0.1 on its own
// isn't a valid registry hostname.
// TODO: cert validation not working here, so we use don't verify for now.
func mockTransport(server *httptest.Server) *http.Transport {
	u, _ := url.Parse(server.URL)
	_, port, _ := net.SplitHostPort(u.Host)

	transport := cleanhttp.DefaultTransport()
	transport.TLSClientConfig = server.TLS
	transport.TLSClientConfig.InsecureSkipVerify = true
	transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, _ := net.SplitHostPort(addr)
		switch host {
		case "example.com", "localhost", "localhost.localdomain", "registry.terraform.io":
			addr = "127.0.0.1"
			if port != "" {
				addr += ":" + port
			}
		}
		return (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext(ctx, network, addr)
	}
	return transport
}

func TestMockDiscovery(t *testing.T) {
	server := mockTLSRegistry()
	defer server.Close()

	regDisco := disco.NewDisco()
	regDisco.Transport = mockTransport(server)

	regURL := regDisco.DiscoverServiceURL("example.com", serviceID)

	if regURL == nil {
		t.Fatal("no registry service discovered")
	}

	if regURL.Host != "example.com" {
		t.Fatal("expected registry host example.com, got:", regURL.Host)
	}
}

func TestLookupModuleVersions(t *testing.T) {
	server := mockTLSRegistry()
	defer server.Close()
	regDisco := disco.NewDisco()
	regDisco.Transport = mockTransport(server)

	// test with and without a hostname
	for _, src := range []string{
		"example.com/test-versions/name/provider",
		"test-versions/name/provider",
	} {
		modsrc, err := regsrc.ParseModuleSource(src)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := lookupModuleVersions(regDisco, modsrc)
		if err != nil {
			t.Fatal(err)
		}

		if len(resp.Modules) != 1 {
			t.Fatal("expected 1 module, got", len(resp.Modules))
		}

		mod := resp.Modules[0]
		name := "test-versions/name/provider"
		if mod.Source != name {
			t.Fatalf("expected module name %q, got %q", name, mod.Source)
		}

		if len(mod.Versions) != 4 {
			t.Fatal("expected 4 versions, got", len(mod.Versions))
		}

		for _, v := range mod.Versions {
			_, err := version.NewVersion(v.Version)
			if err != nil {
				t.Fatalf("invalid version %q: %s", v.Version, err)
			}
		}
	}
}

func TestACCLookupModuleVersions(t *testing.T) {
	server := mockTLSRegistry()
	defer server.Close()
	regDisco := disco.NewDisco()

	// test with and without a hostname
	for _, src := range []string{
		"terraform-aws-modules/vpc/aws",
		defaultRegistry + "/terraform-aws-modules/vpc/aws",
	} {
		modsrc, err := regsrc.ParseModuleSource(src)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := lookupModuleVersions(regDisco, modsrc)
		if err != nil {
			t.Fatal(err)
		}

		if len(resp.Modules) != 1 {
			t.Fatal("expected 1 module, got", len(resp.Modules))
		}

		mod := resp.Modules[0]
		name := "terraform-aws-modules/vpc/aws"
		if mod.Source != name {
			t.Fatalf("expected module name %q, got %q", name, mod.Source)
		}

		if len(mod.Versions) == 0 {
			t.Fatal("expected multiple versions, got 0")
		}

		for _, v := range mod.Versions {
			_, err := version.NewVersion(v.Version)
			if err != nil {
				t.Fatalf("invalid version %q: %s", v.Version, err)
			}
		}
	}
}

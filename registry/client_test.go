package registry

import (
	"fmt"
	"os"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/registry/test"
	"github.com/hashicorp/terraform/svchost/disco"
)

func TestLookupModuleVersions(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	// test with and without a hostname
	for _, src := range []string{
		"example.com/test-versions/name/provider",
		"test-versions/name/provider",
	} {
		modsrc, err := regsrc.ParseModuleSource(src)
		if err != nil {
			t.Fatal(err)
		}

		resp, err := client.ModuleVersions(modsrc)
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

func TestInvalidRegistry(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	src := "non-existent.localhost.localdomain/test-versions/name/provider"
	modsrc, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := client.ModuleVersions(modsrc); err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryAuth(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	src := "private/name/provider"
	mod, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	// both should fail without auth
	_, err = client.ModuleVersions(mod)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = client.ModuleLocation(mod, "1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}

	// Also test without a credentials source
	client.services.SetCredentialsSource(nil)

	_, err = client.ModuleVersions(mod)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ModuleLocation(mod, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLookupModuleLocationRelative(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	src := "relative/foo/bar"
	mod, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	got, err := client.ModuleLocation(mod, "0.2.0")
	if err != nil {
		t.Fatal(err)
	}

	want := server.URL + "/relative-path"
	if got != want {
		t.Errorf("wrong location %s; want %s", got, want)
	}
}

func TestAccLookupModuleVersions(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip()
	}
	regDisco := disco.New()

	// test with and without a hostname
	for _, src := range []string{
		"terraform-aws-modules/vpc/aws",
		regsrc.PublicRegistryHost.String() + "/terraform-aws-modules/vpc/aws",
	} {
		modsrc, err := regsrc.ParseModuleSource(src)
		if err != nil {
			t.Fatal(err)
		}

		s := NewClient(regDisco, nil)
		resp, err := s.ModuleVersions(modsrc)
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

// the error should reference the config source exatly, not the discovered path.
func TestLookupLookupModuleError(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	// this should not be found in teh registry
	src := "bad/local/path"
	mod, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	_, err = client.ModuleLocation(mod, "0.2.0")
	if err == nil {
		t.Fatal("expected error")
	}

	// check for the exact quoted string to ensure we didn't prepend a hostname.
	if !strings.Contains(err.Error(), `"bad/local/path"`) {
		t.Fatal("error should not include the hostname. got:", err)
	}
}

func TestLookupProviderVersions(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	tests := []struct {
		name string
	}{
		{"foo"},
		{"bar"},
	}
	for _, tt := range tests {
		provider := regsrc.NewTerraformProvider(tt.name, "", "")
		resp, err := client.TerraformProviderVersions(provider)
		if err != nil {
			t.Fatal(err)
		}

		name := fmt.Sprintf("terraform-providers/%s", tt.name)
		if resp.ID != name {
			t.Fatalf("expected provider name %q, got %q", name, resp.ID)
		}

		if len(resp.Versions) != 2 {
			t.Fatal("expected 2 versions, got", len(resp.Versions))
		}

		for _, v := range resp.Versions {
			_, err := version.NewVersion(v.Version)
			if err != nil {
				t.Fatalf("invalid version %q: %s", v, err)
			}
		}
	}
}

func TestLookupProviderLocation(t *testing.T) {
	server := test.Registry()
	defer server.Close()

	client := NewClient(test.Disco(server), nil)

	tests := []struct {
		Name    string
		Version string
		Err     bool
	}{
		{
			"foo",
			"0.2.3",
			false,
		},
		{
			"bar",
			"0.1.1",
			false,
		},
		{
			"baz",
			"0.0.0",
			true,
		},
	}
	for _, tt := range tests {
		// FIXME: the tests are set up to succeed - os/arch is not being validated at this time
		p := regsrc.NewTerraformProvider(tt.Name, "linux", "amd64")

		locationMetadata, err := client.TerraformProviderLocation(p, tt.Version)
		if tt.Err {
			if err == nil {
				t.Fatal("succeeded; want error")
			}
			return
		} else if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		downloadURL := fmt.Sprintf("https://releases.hashicorp.com/terraform-provider-%s/%s/terraform-provider-%s.zip", tt.Name, tt.Version, tt.Name)

		if locationMetadata.DownloadURL != downloadURL {
			t.Fatalf("incorrect download URL: expected %q, got %q", downloadURL, locationMetadata.DownloadURL)
		}
	}

}

package module

import (
	"os"
	"testing"

	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/registry/regsrc"
	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/svchost/auth"
	"github.com/hashicorp/terraform/svchost/disco"
)

func TestLookupModuleVersions(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	regDisco := testDisco(server)

	// test with and without a hostname
	for _, src := range []string{
		"example.com/test-versions/name/provider",
		"test-versions/name/provider",
	} {
		modsrc, err := regsrc.ParseModuleSource(src)
		if err != nil {
			t.Fatal(err)
		}

		s := &Storage{Services: regDisco}
		resp, err := s.lookupModuleVersions(modsrc)
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

func TestRegistryAuth(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	regDisco := testDisco(server)
	storage := testStorage(t, regDisco)

	src := "private/name/provider"
	mod, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	// both should fail without auth
	_, err = storage.lookupModuleVersions(mod)
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = storage.lookupModuleLocation(mod, "1.0.0")
	if err == nil {
		t.Fatal("expected error")
	}

	storage.Creds = auth.StaticCredentialsSource(map[svchost.Hostname]map[string]interface{}{
		svchost.Hostname(defaultRegistry): {"token": testCredentials},
	})

	_, err = storage.lookupModuleVersions(mod)
	if err != nil {
		t.Fatal(err)
	}
	_, err = storage.lookupModuleLocation(mod, "1.0.0")
	if err != nil {
		t.Fatal(err)
	}

}
func TestLookupModuleLocationRelative(t *testing.T) {
	server := mockRegistry()
	defer server.Close()

	regDisco := testDisco(server)
	storage := testStorage(t, regDisco)

	src := "relative/foo/bar"
	mod, err := regsrc.ParseModuleSource(src)
	if err != nil {
		t.Fatal(err)
	}

	got, err := storage.lookupModuleLocation(mod, "0.2.0")
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

		s := &Storage{
			Services: regDisco,
		}
		resp, err := s.lookupModuleVersions(modsrc)
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

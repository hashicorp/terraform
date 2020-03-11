package configs

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

// TestNewModule_provider_fqns exercises module.gatherProviderLocalNames()
func TestNewModule_provider_local_name(t *testing.T) {
	mod, diags := testModuleFromDir("testdata/providers-explicit-fqn")
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	p := addrs.NewProvider(addrs.DefaultRegistryHost, "foo", "test")
	if name, exists := mod.ProviderLocalNames[p]; !exists {
		t.Fatal("provider FQN foo/test not found")
	} else {
		if name != "foo-test" {
			t.Fatalf("provider localname mismatch: got %s, want foo-test", name)
		}
	}

	// ensure the reverse lookup (fqn to local name) works as well
	localName := mod.LocalNameForProvider(p)
	if localName != "foo-test" {
		t.Fatal("provider local name not found")
	}
}

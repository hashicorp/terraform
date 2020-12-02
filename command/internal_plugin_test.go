package command

import (
	"testing"
)

func TestInternalPlugin_InternalProviders(t *testing.T) {
	m := new(Meta)
	providers := m.internalProviders()
	// terraform is the only provider moved back to internal
	for _, name := range []string{"terraform"} {
		pf, ok := providers[name]
		if !ok {
			t.Errorf("Expected to find %s in InternalProviders", name)
		}

		provider, err := pf()
		if err != nil {
			t.Fatal(err)
		}

		if provider == nil {
			t.Fatal("provider factory returned a nil provider")
		}
	}
}

func TestInternalPlugin_InternalProvisioners(t *testing.T) {
	for _, name := range []string{"file", "local-exec", "remote-exec"} {
		if _, ok := InternalProvisioners[name]; !ok {
			t.Errorf("Expected to find %s in InternalProvisioners", name)
		}
	}
}

func TestInternalPlugin_BuildPluginCommandString(t *testing.T) {
	actual, err := BuildPluginCommandString("provisioner", "remote-exec")
	if err != nil {
		t.Fatalf(err.Error())
	}

	expected := "-TFSPACE-internal-plugin-TFSPACE-provisioner-TFSPACE-remote-exec"
	if actual[len(actual)-len(expected):] != expected {
		t.Errorf("Expected command to end with %s; got:\n%s\n", expected, actual)
	}
}

func TestInternalPlugin_StripArgFlags(t *testing.T) {
	actual := StripArgFlags([]string{"provisioner", "remote-exec", "-var-file=my_vars.tfvars", "-flag"})
	expected := []string{"provisioner", "remote-exec"}
	// Must be same length and order.
	if len(actual) != len(expected) || expected[0] != actual[0] || actual[1] != actual[1] {
		t.Fatalf("Expected args to be exactly '%s', got '%s'", expected, actual)
	}
}

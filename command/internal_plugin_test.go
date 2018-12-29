package command

import "testing"

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
	for _, name := range []string{"chef", "file", "local-exec", "remote-exec", "salt-masterless"} {
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

package command

import "testing"

// providers are all external for now
//func TestInternalPlugin_InternalProviders(t *testing.T) {
//    // Note this is a randomish sample and does not check for all plugins
//    for _, name := range []string{"atlas", "consul", "docker", "template"} {
//        if _, ok := InternalProviders[name]; !ok {
//            t.Errorf("Expected to find %s in InternalProviders", name)
//        }
//    }
//}

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

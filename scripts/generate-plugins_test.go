package main

import "testing"

func TestMakeProvisionerMap(t *testing.T) {
	p := makeProvisionerMap([]plugin{
		{
			Package:    "file",
			PluginName: "file",
			TypeName:   "Provisioner",
			Path:       "builtin/provisioners/file",
			ImportName: "fileprovisioner",
		},
		{
			Package:    "localexec",
			PluginName: "local-exec",
			TypeName:   "Provisioner",
			Path:       "builtin/provisioners/local-exec",
			ImportName: "localexecprovisioner",
		},
		{
			Package:    "remoteexec",
			PluginName: "remote-exec",
			TypeName:   "Provisioner",
			Path:       "builtin/provisioners/remote-exec",
			ImportName: "remoteexecprovisioner",
		},
	})

	expected := `	"file":   fileprovisioner.Provisioner,
	"local-exec":   localexecprovisioner.Provisioner,
	"remote-exec":   remoteexecprovisioner.Provisioner,
`

	if p != expected {
		t.Errorf("Provisioner output does not match expected format.\n -- Expected -- \n%s\n -- Found --\n%s\n", expected, p)
	}
}

func TestDeriveName(t *testing.T) {
	actual := deriveName("builtin/provisioners", "builtin/provisioners/magic/remote-exec")
	expected := "magic-remote-exec"
	if actual != expected {
		t.Errorf("Expected %s; found %s", expected, actual)
	}
}

func TestDeriveImport(t *testing.T) {
	actual := deriveImport("provider", "magic-aws")
	expected := "magicawsprovider"
	if actual != expected {
		t.Errorf("Expected %s; found %s", expected, actual)
	}
}

func contains(plugins []plugin, name string) bool {
	for _, plugin := range plugins {
		if plugin.PluginName == name {
			return true
		}
	}
	return false
}

func TestDiscoverTypesProviders(t *testing.T) {
	plugins, err := discoverTypesInPath("../builtin/providers", "terraform.ResourceProvider", "Provider")
	if err != nil {
		t.Fatalf(err.Error())
	}
	// We're just going to spot-check, not do this exhaustively
	if !contains(plugins, "aws") {
		t.Errorf("Expected to find aws provider")
	}
	if !contains(plugins, "docker") {
		t.Errorf("Expected to find docker provider")
	}
	if !contains(plugins, "dnsimple") {
		t.Errorf("Expected to find dnsimple provider")
	}
	if !contains(plugins, "triton") {
		t.Errorf("Expected to find triton provider")
	}
	if contains(plugins, "file") {
		t.Errorf("Found unexpected provider file")
	}
}

func TestDiscoverTypesProvisioners(t *testing.T) {
	plugins, err := discoverTypesInPath("../builtin/provisioners", "terraform.ResourceProvisioner", "Provisioner")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !contains(plugins, "remote-exec") {
		t.Errorf("Expected to find remote-exec provisioner")
	}
	if contains(plugins, "aws") {
		t.Errorf("Found unexpected provisioner aws")
	}
}

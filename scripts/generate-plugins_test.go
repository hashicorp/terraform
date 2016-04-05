package main

import "testing"

func TestMakeProvisionerMap(t *testing.T) {
	p := makeProvisionerMap([]plugin{
		{
			Package:    "file",
			PluginName: "file",
			TypeName:   "ResourceProvisioner",
			Path:       "builtin/provisioners/file",
			ImportName: "fileresourceprovisioner",
		},
		{
			Package:    "localexec",
			PluginName: "local-exec",
			TypeName:   "ResourceProvisioner",
			Path:       "builtin/provisioners/local-exec",
			ImportName: "localexecresourceprovisioner",
		},
		{
			Package:    "remoteexec",
			PluginName: "remote-exec",
			TypeName:   "ResourceProvisioner",
			Path:       "builtin/provisioners/remote-exec",
			ImportName: "remoteexecresourceprovisioner",
		},
	})

	expected := `	"file": func() terraform.ResourceProvisioner { return new(fileresourceprovisioner.ResourceProvisioner) },
	"local-exec": func() terraform.ResourceProvisioner { return new(localexecresourceprovisioner.ResourceProvisioner) },
	"remote-exec": func() terraform.ResourceProvisioner { return new(remoteexecresourceprovisioner.ResourceProvisioner) },
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
	plugins, err := discoverTypesInPath("../builtin/provisioners", "ResourceProvisioner", "")
	if err != nil {
		t.Fatalf(err.Error())
	}
	if !contains(plugins, "chef") {
		t.Errorf("Expected to find chef provisioner")
	}
	if !contains(plugins, "remote-exec") {
		t.Errorf("Expected to find remote-exec provisioner")
	}
	if contains(plugins, "aws") {
		t.Errorf("Found unexpected provisioner aws")
	}
}

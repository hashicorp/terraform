package module

import (
	"testing"

	"github.com/hashicorp/terraform/registry/response"
)

func TestNewestModuleVersion(t *testing.T) {
	mpv := &response.ModuleProviderVersions{
		Source: "registry/test/module",
		Versions: []*response.ModuleVersion{
			{Version: "0.0.4"},
			{Version: "0.3.1"},
			{Version: "2.0.1"},
			{Version: "1.2.0"},
		},
	}

	m, err := newestVersion(mpv.Versions, "")
	if err != nil {
		t.Fatal(err)
	}

	expected := "2.0.1"
	if m.Version != expected {
		t.Fatalf("expected version %q, got %q", expected, m.Version)
	}

	// now with a constraint
	m, err = newestVersion(mpv.Versions, "~>1.0")
	if err != nil {
		t.Fatal(err)
	}

	expected = "1.2.0"
	if m.Version != expected {
		t.Fatalf("expected version %q, got %q", expected, m.Version)
	}
}

func TestNewestInvalidModuleVersion(t *testing.T) {
	mpv := &response.ModuleProviderVersions{
		Source: "registry/test/module",
		Versions: []*response.ModuleVersion{
			{Version: "WTF"},
			{Version: "2.0.1"},
		},
	}

	m, err := newestVersion(mpv.Versions, "")
	if err != nil {
		t.Fatal(err)
	}

	expected := "2.0.1"
	if m.Version != expected {
		t.Fatalf("expected version %q, got %q", expected, m.Version)
	}
}

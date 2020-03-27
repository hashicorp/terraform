package getproviders

import (
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestLookupLegacyProvider(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	got, err := LookupLegacyProvider(
		addrs.NewLegacyProvider("legacy"),
		source,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := addrs.Provider{
		Hostname:  defaultRegistryHost,
		Namespace: "legacycorp",
		Type:      "legacy",
	}
	if got != want {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
}

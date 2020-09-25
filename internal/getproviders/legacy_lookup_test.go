package getproviders

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/addrs"
)

func TestLookupLegacyProvider(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	got, gotMoved, err := LookupLegacyProvider(
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
	if !gotMoved.IsZero() {
		t.Errorf("wrong moved result\ngot:  %#v\nwant: %#v", gotMoved, addrs.Provider{})
	}
}

func TestLookupLegacyProvider_moved(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	got, gotMoved, err := LookupLegacyProvider(
		addrs.NewLegacyProvider("moved"),
		source,
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	want := addrs.Provider{
		Hostname:  defaultRegistryHost,
		Namespace: "hashicorp",
		Type:      "moved",
	}
	wantMoved := addrs.Provider{
		Hostname:  defaultRegistryHost,
		Namespace: "acme",
		Type:      "moved",
	}
	if got != want {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", got, want)
	}
	if gotMoved != wantMoved {
		t.Errorf("wrong result\ngot:  %#v\nwant: %#v", gotMoved, wantMoved)
	}
}

func TestLookupLegacyProvider_invalidResponse(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	got, _, err := LookupLegacyProvider(
		addrs.NewLegacyProvider("invalid"),
		source,
	)
	if !got.IsZero() {
		t.Errorf("got non-zero addr\ngot:  %#v\nwant: %#v", got, nil)
	}
	wantErr := "Error parsing provider ID from Registry: Invalid provider source string"
	if gotErr := err.Error(); !strings.Contains(gotErr, wantErr) {
		t.Fatalf("unexpected error: got %q, want %q", gotErr, wantErr)
	}
}

func TestLookupLegacyProvider_unexpectedTypeChange(t *testing.T) {
	source, _, close := testRegistrySource(t)
	defer close()

	got, _, err := LookupLegacyProvider(
		addrs.NewLegacyProvider("changetype"),
		source,
	)
	if !got.IsZero() {
		t.Errorf("got non-zero addr\ngot:  %#v\nwant: %#v", got, nil)
	}
	wantErr := `Registry returned provider with type "newtype", expected "changetype"`
	if gotErr := err.Error(); gotErr != wantErr {
		t.Fatalf("unexpected error: got %q, want %q", gotErr, wantErr)
	}
}

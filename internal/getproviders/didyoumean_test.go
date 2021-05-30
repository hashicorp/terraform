package getproviders

import (
	"context"
	"testing"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/addrs"
)

func TestMissingProviderSuggestion(t *testing.T) {
	// Most of these test cases rely on specific "magic" provider addresses
	// that are implemented by the fake registry source returned by
	// testRegistrySource. Refer to that function for more details on how
	// they work.

	t.Run("happy path", func(t *testing.T) {
		ctx := context.Background()
		source, _, close := testRegistrySource(t)
		defer close()

		// testRegistrySource handles -/legacy as a valid legacy provider
		// lookup mapping to legacycorp/legacy.
		legacyAddr := addrs.NewDefaultProvider("legacy")
		got := MissingProviderSuggestion(
			ctx,
			addrs.NewDefaultProvider("legacy"),
			source,
			Requirements{
				legacyAddr: MustParseVersionConstraints(">= 1.0.0"),
			},
		)

		want := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "legacycorp",
			Type:      "legacy",
		}
		if got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("provider moved", func(t *testing.T) {
		ctx := context.Background()
		source, _, close := testRegistrySource(t)
		defer close()

		// testRegistrySource handles -/moved as a valid legacy provider
		// lookup mapping to hashicorp/moved but with an additional "redirect"
		// to acme/moved. This mimics how for some providers there is both
		// a copy under terraform-providers for v0.12 compatibility _and_ a
		// copy in some other namespace for v0.13 or later to use. Our naming
		// suggestions ignore the v0.12-compatible one and suggest the
		// other one.
		moved := addrs.NewDefaultProvider("moved")
		want := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "acme",
			Type:      "moved",
		}

		got := MissingProviderSuggestion(
			ctx,
			moved,
			source,
			Requirements{
				moved: MustParseVersionConstraints(">= 1.0.0"),
			},
		)

		if got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}

		// If a provider has moved, but there's provider requirements
		// for something of the same type, we'll return that one
		// and skip the legacy lookup process. In practice,
		// hopefully this is also "acme" but it's "zcme" here to
		// exercise the codepath
		want2 := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "zcme",
			Type:      "moved",
		}
		got2 := MissingProviderSuggestion(
			ctx,
			moved,
			source,
			Requirements{
				moved: MustParseVersionConstraints(">= 1.0.0"),
				want2: MustParseVersionConstraints(">= 1.0.0"),
			},
		)

		if got2 != want2 {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got2, want2)
		}
	})
	t.Run("invalid response", func(t *testing.T) {
		ctx := context.Background()
		source, _, close := testRegistrySource(t)
		defer close()

		// testRegistrySource handles -/invalid by returning an invalid
		// provider address, which MissingProviderSuggestion should reject
		// and behave as if there was no suggestion available.
		want := addrs.NewDefaultProvider("invalid")
		got := MissingProviderSuggestion(
			ctx,
			want,
			source,
			Requirements{
				want: MustParseVersionConstraints(">= 1.0.0"),
			},
		)
		if got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("another registry", func(t *testing.T) {
		ctx := context.Background()
		source, _, close := testRegistrySource(t)
		defer close()

		// Because this provider address isn't on registry.terraform.io,
		// MissingProviderSuggestion won't even attempt to make a suggestion
		// for it.
		want := addrs.Provider{
			Hostname:  svchost.Hostname("example.com"),
			Namespace: "whatever",
			Type:      "foo",
		}
		got := MissingProviderSuggestion(
			ctx,
			want,
			source,
			Requirements{
				want: MustParseVersionConstraints(">= 1.0.0"),
			},
		)
		if got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}
	})
	t.Run("another namespace", func(t *testing.T) {
		ctx := context.Background()
		source, _, close := testRegistrySource(t)
		defer close()

		// Because this provider address isn't in
		// registry.terraform.io/hashicorp/..., MissingProviderSuggestion
		// will provide the same addr since there's no alternative in Requirements
		want := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "whatever",
			Type:      "foo",
		}
		got := MissingProviderSuggestion(
			ctx,
			want,
			source,
			Requirements{
				want: MustParseVersionConstraints(">= 1.0.0"),
			},
		)
		if got != want {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got, want)
		}

		// If there is a provider required that has the same type,
		// but different namespace, we can suggest that
		foo := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "hashicorp",
			Type:      "foo",
		}
		realFoo := addrs.Provider{
			Hostname:  defaultRegistryHost,
			Namespace: "acme",
			Type:      "foo",
		}
		got2 := MissingProviderSuggestion(
			ctx,
			foo,
			source,
			Requirements{
				foo:     MustParseVersionConstraints(">= 1.0.0"),
				realFoo: MustParseVersionConstraints(">= 1.0.0"),
			},
		)
		if got2 != realFoo {
			t.Errorf("wrong result\ngot:  %s\nwant: %s", got2, realFoo)
		}
	})
}

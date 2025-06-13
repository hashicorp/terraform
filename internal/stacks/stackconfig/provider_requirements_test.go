package stackconfig

import (
	"reflect"
	"testing"

	tfaddr "github.com/hashicorp/terraform-registry-address"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func TestProviderRequirements_ProviderForLocalName(t *testing.T) {
	t.Run("built-in provider gets resolved when not listed in requirements", func(t *testing.T) {
		expectedProvider := tfaddr.Provider{
			Type:      "terraform",
			Namespace: "hashicorp",
			Hostname:  "registry.terraform.io",
		}

		t.Run("and requirements are empty", func(t *testing.T) {
			var pr *ProviderRequirements
			actualProvider, ok := pr.ProviderForLocalName("terraform")

			if !ok {
				t.Fatalf("expected built-in provider to be resolved, but got ok = false")
			}

			if !reflect.DeepEqual(expectedProvider, actualProvider) {
				t.Fatalf("expected built-in provider to be %v, but got %v", expectedProvider, actualProvider)
			}
		})

		t.Run("and requirements are NOT empty", func(t *testing.T) {
			pr := createTestProviderRequirements(t)
			actualProvider, ok := pr.ProviderForLocalName("terraform")

			if !ok {
				t.Fatalf("expected built-in provider to be resolved, but got ok = false")
			}

			if !reflect.DeepEqual(expectedProvider, actualProvider) {
				t.Fatalf("expected built-in provider to be %v, but got %v", expectedProvider, actualProvider)
			}
		})
	})
}

func createTestProviderRequirements(t *testing.T) *ProviderRequirements {
	return &ProviderRequirements{
		Requirements: map[string]ProviderRequirement{
			"dummy": {
				LocalName: "dummy_provider",
				Provider: addrs.Provider{
					Type:      "dummy",
					Namespace: "test",
					Hostname:  "testing.com",
				},
				VersionConstraints: nil,
				DeclRange:          tfdiags.SourceRange{},
			},
		},
	}
}

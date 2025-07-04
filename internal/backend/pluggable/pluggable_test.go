package pluggable

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/providers"
	testing_provider "github.com/hashicorp/terraform/internal/providers/testing"
)

func TestNewPluggable(t *testing.T) {
	cases := map[string]struct {
		provider providers.Interface
		typeName string

		wantError string
	}{
		"no error when inputs are provided": {
			provider: &testing_provider.MockProvider{},
			typeName: "foo_bar",
		},
		"no error when store name has underscores": {
			provider: &testing_provider.MockProvider{},
			// foo provider containing fizz_buzz store
			typeName: "foo_fizz_buzz",
		},
		"error when store type not provided": {
			provider:  &testing_provider.MockProvider{},
			typeName:  "",
			wantError: "Attempted to initialize pluggable state with an empty string identifier for the state store.",
		},
		"error when provider interface is nil ": {
			provider:  nil,
			typeName:  "foo_bar",
			wantError: "Attempted to initialize pluggable state with a nil provider interface.",
		},
	}

	for tn, tc := range cases {
		t.Run(tn, func(t *testing.T) {
			_, err := NewPluggable(tc.provider, tc.typeName)

			if err != nil {
				if tc.wantError == "" {
					t.Fatalf("unexpected error: %s", err)
				}
				if !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error %q but got %q", tc.wantError, err)
				}
				return
			}
			if err == nil && tc.wantError != "" {
				t.Fatalf("expected error %q but got none", tc.wantError)
			}
		})
	}
}

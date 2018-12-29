package regsrc

import (
	"testing"
)

func TestNewTerraformProviderNamespace(t *testing.T) {
	tests := []struct {
		name              string
		provider          string
		expectedNamespace string
		expectedName      string
	}{
		{
			name:              "default",
			provider:          "null",
			expectedNamespace: "-",
			expectedName:      "null",
		}, {
			name:              "explicit",
			provider:          "terraform-providers/null",
			expectedNamespace: "terraform-providers",
			expectedName:      "null",
		}, {
			name:              "community",
			provider:          "community-providers/null",
			expectedNamespace: "community-providers",
			expectedName:      "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := NewTerraformProvider(tt.provider, "", "")

			if actual == nil {
				t.Fatal("NewTerraformProvider() unexpectedly returned nil provider")
			}

			if v := actual.RawNamespace; v != tt.expectedNamespace {
				t.Fatalf("RawNamespace = %v, wanted %v", v, tt.expectedNamespace)
			}
			if v := actual.RawName; v != tt.expectedName {
				t.Fatalf("RawName = %v, wanted %v", v, tt.expectedName)
			}
		})
	}
}

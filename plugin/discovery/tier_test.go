package discovery

import "testing"

func TestProviderTier_String(t *testing.T) {
	var emptyTier ProviderTier

	tests := []struct {
		name     string
		tier     ProviderTier
		expected string
	}{
		{
			"default",
			emptyTier,
			"Community",
		},
		{
			"community",
			providerTierCommunity,
			"Community",
		},
		{
			"partner",
			providerTierPartner,
			"Partner",
		},
		{
			"official",
			providerTierOfficial,
			"Official",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.tier.String()
			if actual != tt.expected {
				t.Errorf("wanted %q, got %q", tt.expected, actual)
			}

		})
	}

}

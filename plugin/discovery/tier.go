package discovery

// ProviderTier is a type which holds an iota constant for passing the tier
// between functions.
type ProviderTier int

const (
	providerTierCommunity ProviderTier = iota
	providerTierPartner
	providerTierOfficial
)

// String returns the string representation of the ProviderTier enum.
func (t ProviderTier) String() string {
	return [...]string{
		"Community",
		"Partner",
		"Official",
	}[t]
}

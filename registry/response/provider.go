package response

import (
	"time"
)

// Provider is the response structure with the data for a single provider
// version. This is just the metadata. A full provider response will be
// ProviderDetail.
type Provider struct {
	ID string `json:"id"`

	//---------------------------------------------------------------
	// Metadata about the overall provider.

	Owner       string    `json:"owner"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Downloads   int       `json:"downloads"`
}

// ProviderDetail represents a Provider with full detail.
type ProviderDetail struct {
	Provider

	//---------------------------------------------------------------
	// The fields below are only set when requesting this specific
	// module. They are available to easily know all available versions
	// without multiple API calls.

	Versions []string `json:"versions"` // All versions
}

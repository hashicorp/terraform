package cloudflare

import (
	"time"
)

type FirewallConfiguration struct {
	Target string `json:"target,omitempty"`
	Value  string `json:"value,omitempty"`
}

// Firewall - Firewall for zone.
type Firewall struct {
	ID string `json:"id,omitempty"`

	Notes        string   `json:"notes,omitempty"`
	AllowedModes []string `json:"allowed_modes,omitempty"`
	Mode         string   `json:"mode,omitempty"`

	Configuration *FirewallConfiguration `json:"configuration,omitempty"`
	Scope         *ZoneOwner             `json:"scope,omitempty"`

	CreatedOn  time.Time `json:"created_on,omitempty"`
	ModifiedOn time.Time `json:"modified_on,omitempty"`
}

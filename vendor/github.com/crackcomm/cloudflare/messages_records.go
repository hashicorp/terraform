package cloudflare

import "time"

// Record - Cloudflare DNS Record.
type Record struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Name    string `json:"name,omitempty"`
	Content string `json:"content,omitempty"`

	Proxiable bool `json:"proxiable,omitempty"`
	Proxied   bool `json:"proxied,omitempty"`
	Locked    bool `json:"locked,omitempty"`

	TTL      int `json:"ttl,omitempty"`
	Priority int `json:"priority,omitempty"`

	CreatedOn  time.Time `json:"created_on,omitempty"`
	ModifiedOn time.Time `json:"modified_on,omitempty"`

	ZoneID   string `json:"zone_id,omitempty"`
	ZoneName string `json:"zone_name,omitempty"`
}

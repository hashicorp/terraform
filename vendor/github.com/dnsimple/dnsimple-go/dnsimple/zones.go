package dnsimple

import (
	"fmt"
)

// ZonesService handles communication with the zone related
// methods of the DNSimple API.
//
// See https://developer.dnsimple.com/v2/zones/
type ZonesService struct {
	client *Client
}

// Zone represents a Zone in DNSimple.
type Zone struct {
	ID        int    `json:"id,omitempty"`
	AccountID int    `json:"account_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Reverse   bool   `json:"reverse,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

// ZoneFile represents a Zone File in DNSimple.
type ZoneFile struct {
	Zone string `json:"zone,omitempty"`
}

// ZoneListOptions specifies the optional parameters you can provide
// to customize the ZonesService.ListZones method.
type ZoneListOptions struct {
	// Select domains where the name contains given string.
	NameLike string `url:"name_like,omitempty"`

	ListOptions
}

// ZoneResponse represents a response from an API method that returns a Zone struct.
type ZoneResponse struct {
	Response
	Data *Zone `json:"data"`
}

// ZonesResponse represents a response from an API method that returns a collection of Zone struct.
type ZonesResponse struct {
	Response
	Data []Zone `json:"data"`
}

// ZoneFileResponse represents a response from an API method that returns a ZoneFile struct.
type ZoneFileResponse struct {
	Response
	Data *ZoneFile `json:"data"`
}

// ListZones the zones for an account.
//
// See https://developer.dnsimple.com/v2/zones/#list
func (s *ZonesService) ListZones(accountID string, options *ZoneListOptions) (*ZonesResponse, error) {
	path := versioned(fmt.Sprintf("/%v/zones", accountID))
	zonesResponse := &ZonesResponse{}

	path, err := addURLQueryOptions(path, options)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.get(path, zonesResponse)
	if err != nil {
		return zonesResponse, err
	}

	zonesResponse.HttpResponse = resp
	return zonesResponse, nil
}

// GetZone fetches a zone.
//
// See https://developer.dnsimple.com/v2/zones/#get
func (s *ZonesService) GetZone(accountID string, zoneName string) (*ZoneResponse, error) {
	path := versioned(fmt.Sprintf("/%v/zones/%v", accountID, zoneName))
	zoneResponse := &ZoneResponse{}

	resp, err := s.client.get(path, zoneResponse)
	if err != nil {
		return nil, err
	}

	zoneResponse.HttpResponse = resp
	return zoneResponse, nil
}

// GetZoneFile fetches a zone file.
//
// See https://developer.dnsimple.com/v2/zones/#get-file
func (s *ZonesService) GetZoneFile(accountID string, zoneName string) (*ZoneFileResponse, error) {
	path := versioned(fmt.Sprintf("/%v/zones/%v/file", accountID, zoneName))
	zoneFileResponse := &ZoneFileResponse{}

	resp, err := s.client.get(path, zoneFileResponse)
	if err != nil {
		return nil, err
	}

	zoneFileResponse.HttpResponse = resp
	return zoneFileResponse, nil
}

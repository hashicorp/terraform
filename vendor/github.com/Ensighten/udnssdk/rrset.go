package udnssdk

import (
	"fmt"
)

// ZonesService handles communication with the Zone related blah blah
type RRSetsService struct {
	client *Client
}

type RRSet struct {
	OwnerName string   `json:"ownerName"`
	RRType    string   `json:"rrtype"`
	TTL       int      `json:"ttl"`
	RData     []string `json:"rdata"`
}

type RRSetListDTO struct {
	ZoneName                string  `json:"zoneName"`
	Rrsets                  []RRSet `json:"rrsets"`
	Queryinfoq              string  `json:"queryinfo/q"`
	Queryinfosort           string  `json:"queryinfo/reverse"`
	Queryinfolimit          string  `json:"queryinfo/limit"`
	ResultinfototalCount    string  `json:"resultinfo/totalCount"`
	Resultinfooffset        string  `json:"resultinfo/offset"`
	ResultinforeturnedCount string  `json:"resultinfo/returnedCount"`
}
type rrsetWrapper struct {
	RRSet RRSet `json:"rrset"`
}

// rrsetPath generates the resource path for given rrset that belongs to a zone.
func rrsetPath(zone string, rrtype interface{}, rrset interface{}) string {
	path := fmt.Sprintf("zones/%s/rrsets", zone)
	if rrtype != nil {

		path += fmt.Sprintf("/%v", rrtype)
		if rrset != nil {
			path += fmt.Sprintf("/%v", rrset)
		}
	}
	return path
}

// List the zone rrsets.
//
func (s *RRSetsService) GetRRSets(zone string, rrsetName, rrsetType string) ([]RRSet, *Response, error) {
	// TODO: Soooo... This function does not handle pagination of RRSets....
	//v := url.Values{}

	if rrsetType == "" {
		rrsetType = "ANY"
	}
	reqStr := rrsetPath(zone, rrsetType, rrsetName)
	var rrsld RRSetListDTO
	//wrappedRRSets := []RRSet{}

	res, err := s.client.get(reqStr, &rrsld)
	if err != nil {
		return []RRSet{}, res, err
	}

	rrsets := []RRSet{}
	for _, rrset := range rrsld.Rrsets {
		rrsets = append(rrsets, rrset)
	}

	return rrsets, res, nil
}

// CreateRRSet creates a zone rrset.
//
func (s *RRSetsService) CreateRRSet(zone string, rrsetAttributes RRSet) (*Response, error) {
	path := rrsetPath(zone, rrsetAttributes.RRType, rrsetAttributes.OwnerName)
	var retval interface{}
	res, err := s.client.post(path, rrsetAttributes, &retval)
	//log.Printf("CreateRRSet Retval: %+v", retval)
	if err != nil {
		return res, err
	}
	return res, err
}

// UpdateRRSet updates a zone rrset.
//
func (s *RRSetsService) UpdateRRSet(zone string, rrsetAttributes RRSet) (*Response, error) {
	path := rrsetPath(zone, rrsetAttributes.RRType, rrsetAttributes.OwnerName)
	var retval interface{}

	res, err := s.client.put(path, rrsetAttributes, &retval)
	//log.Printf("UpdateRRSet Retval: %+v", retval)

	if err != nil {
		return res, err
	}

	return res, nil
}

// DeleteRRSet deletes a zone rrset.
//
func (s *RRSetsService) DeleteRRSet(zone string, rrsetAttributes RRSet) (*Response, error) {
	path := rrsetPath(zone, rrsetAttributes.RRType, rrsetAttributes.OwnerName)

	return s.client.delete(path, nil)
}

// UpdateIP updates the IP of specific A rrset.
//
// This is not part of the standard API. However,
// this is useful for Dynamic DNS (DDNS or DynDNS).
/*
func (rrset *RRSet) UpdateIP(client *Client, IP string) error {
  newdata := []string{IP}
  newRRSet := RRSet{RData: newdata, OwnerName: rrset.OwnerName}
	_, _, err := client.Zones.UpdateRRSet(rrset.ZoneId, rrset.Id, newRRSet)
	return err
}
*/

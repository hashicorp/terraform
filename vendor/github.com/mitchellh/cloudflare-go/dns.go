package cloudflare

import (
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"
)

/*
Create a DNS record.

API reference:
  https://api.cloudflare.com/#dns-records-for-a-zone-create-dns-record
  POST /zones/:zone_identifier/dns_records
*/
func (api *API) CreateDNSRecord(zoneID string, rr DNSRecord) (DNSRecord, error) {
	uri := "/zones/" + zoneID + "/dns_records"
	res, err := api.makeRequest("POST", uri, rr)
	if err != nil {
		return DNSRecord{}, errors.Wrap(err, errMakeRequestError)
	}
	var r DNSRecordResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return DNSRecord{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

/*
Fetches DNS records for a zone.

API reference:
  https://api.cloudflare.com/#dns-records-for-a-zone-list-dns-records
  GET /zones/:zone_identifier/dns_records
*/
func (api *API) DNSRecords(zoneID string, rr DNSRecord) ([]DNSRecord, error) {
	// Construct a query string
	v := url.Values{}
	if rr.Name != "" {
		v.Set("name", rr.Name)
	}
	if rr.Type != "" {
		v.Set("type", rr.Type)
	}
	if rr.Content != "" {
		v.Set("content", rr.Content)
	}
	var query string
	if len(v) > 0 {
		query = "?" + v.Encode()
	}
	uri := "/zones/" + zoneID + "/dns_records" + query
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return []DNSRecord{}, errors.Wrap(err, errMakeRequestError)
	}
	var r DNSListResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return []DNSRecord{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

/*
Fetches a single DNS record.

API reference:
  https://api.cloudflare.com/#dns-records-for-a-zone-dns-record-details
  GET /zones/:zone_identifier/dns_records/:identifier
*/
func (api *API) DNSRecord(zoneID, recordID string) (DNSRecord, error) {
	uri := "/zones/" + zoneID + "/dns_records/" + recordID
	res, err := api.makeRequest("GET", uri, nil)
	if err != nil {
		return DNSRecord{}, errors.Wrap(err, errMakeRequestError)
	}
	var r DNSRecordResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return DNSRecord{}, errors.Wrap(err, errUnmarshalError)
	}
	return r.Result, nil
}

/*
Change a DNS record.

API reference:
  https://api.cloudflare.com/#dns-records-for-a-zone-update-dns-record
  PUT /zones/:zone_identifier/dns_records/:identifier
*/
func (api *API) UpdateDNSRecord(zoneID, recordID string, rr DNSRecord) error {
	rec, err := api.DNSRecord(zoneID, recordID)
	if err != nil {
		return err
	}
	rr.Name = rec.Name
	rr.Type = rec.Type
	uri := "/zones/" + zoneID + "/dns_records/" + recordID
	res, err := api.makeRequest("PUT", uri, rr)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}
	var r DNSRecordResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return errors.Wrap(err, errUnmarshalError)
	}
	return nil
}

/*
Delete a DNS record.

API reference:
  https://api.cloudflare.com/#dns-records-for-a-zone-delete-dns-record
  DELETE /zones/:zone_identifier/dns_records/:identifier
*/
func (api *API) DeleteDNSRecord(zoneID, recordID string) error {
	uri := "/zones/" + zoneID + "/dns_records/" + recordID
	res, err := api.makeRequest("DELETE", uri, nil)
	if err != nil {
		return errors.Wrap(err, errMakeRequestError)
	}
	var r DNSRecordResponse
	err = json.Unmarshal(res, &r)
	if err != nil {
		return errors.Wrap(err, errUnmarshalError)
	}
	return nil
}

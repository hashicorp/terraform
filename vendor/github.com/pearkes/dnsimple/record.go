package dnsimple

import (
	"fmt"
	"strconv"
)

type RecordResponse struct {
	Record Record `json:"record"`
}

// Record is used to represent a retrieved Record. All properties
// are set as strings.
type Record struct {
	Name       string `json:"name"`
	Content    string `json:"content"`
	DomainId   int64  `json:"domain_id"`
	Id         int64  `json:"id"`
	Prio       int64  `json:"prio"`
	RecordType string `json:"record_type"`
	Ttl        int64  `json:"ttl"`
}

// Returns the domain id
func (r *Record) StringDomainId() string {
	return strconv.FormatInt(r.DomainId, 10)
}

// Returns the id
func (r *Record) StringId() string {
	return strconv.FormatInt(r.Id, 10)
}

// Returns the string for prio
func (r *Record) StringPrio() string {
	return strconv.FormatInt(r.Prio, 10)
}

// Returns the string for Locked
func (r *Record) StringTtl() string {
	return strconv.FormatInt(r.Ttl, 10)
}

// ChangeRecord contains the request parameters to create or update a
// record.
type ChangeRecord struct {
	Name  string // name of the record
	Value string // where the record points
	Type  string // type, i.e a, mx
	Ttl   string // TTL of record
}

// CreateRecord creates a record from the parameters specified and
// returns an error if it fails. If no error and an ID is returned,
// the Record was succesfully created.
func (c *Client) CreateRecord(domain string, opts *ChangeRecord) (string, error) {
	// Make the request parameters
	params := make(map[string]interface{})

	params["name"] = opts.Name
	params["record_type"] = opts.Type
	params["content"] = opts.Value

	if opts.Ttl != "" {
		ttl, err := strconv.ParseInt(opts.Ttl, 0, 0)
		if err != nil {
			return "", nil
		}
		params["ttl"] = ttl
	}

	endpoint := fmt.Sprintf("/domains/%s/records", domain)

	req, err := c.NewRequest(params, "POST", endpoint)
	if err != nil {
		return "", err
	}

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return "", fmt.Errorf("Error creating record: %s", err)
	}

	record := new(RecordResponse)

	err = decodeBody(resp, &record)

	if err != nil {
		return "", fmt.Errorf("Error parsing record response: %s", err)
	}

	// The request was successful
	return record.Record.StringId(), nil
}

// UpdateRecord updated a record from the parameters specified and
// returns an error if it fails.
func (c *Client) UpdateRecord(domain string, id string, opts *ChangeRecord) (string, error) {
	// Make the request parameters
	params := make(map[string]interface{})

	if opts.Name != "" {
		params["name"] = opts.Name
	}

	if opts.Type != "" {
		params["record_type"] = opts.Type
	}

	if opts.Value != "" {
		params["content"] = opts.Value
	}

	if opts.Ttl != "" {
		ttl, err := strconv.ParseInt(opts.Ttl, 0, 0)
		if err != nil {
			return "", nil
		}
		params["ttl"] = ttl
	}

	endpoint := fmt.Sprintf("/domains/%s/records/%s", domain, id)

	req, err := c.NewRequest(params, "PUT", endpoint)
	if err != nil {
		return "", err
	}

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return "", fmt.Errorf("Error updating record: %s", err)
	}

	record := new(RecordResponse)

	err = decodeBody(resp, &record)

	if err != nil {
		return "", fmt.Errorf("Error parsing record response: %s", err)
	}

	// The request was successful
	return record.Record.StringId(), nil
}

// DestroyRecord destroys a record by the ID specified and
// returns an error if it fails. If no error is returned,
// the Record was succesfully destroyed.
func (c *Client) DestroyRecord(domain string, id string) error {
	var body map[string]interface{}
	req, err := c.NewRequest(body, "DELETE", fmt.Sprintf("/domains/%s/records/%s", domain, id))

	if err != nil {
		return err
	}

	_, err = checkResp(c.Http.Do(req))

	if err != nil {
		return fmt.Errorf("Error destroying record: %s", err)
	}

	// The request was successful
	return nil
}

// RetrieveRecord gets  a record by the ID specified and
// returns a Record and an error. An error will be returned for failed
// requests with a nil Record.
func (c *Client) RetrieveRecord(domain string, id string) (*Record, error) {
	var body map[string]interface{}
	req, err := c.NewRequest(body, "GET", fmt.Sprintf("/domains/%s/records/%s", domain, id))
	if err != nil {
		return nil, err
	}

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return nil, fmt.Errorf("Error retrieving record: %s", err)
	}

	recordResp := RecordResponse{}

	err = decodeBody(resp, &recordResp)

	if err != nil {
		return nil, fmt.Errorf("Error decoding record response: %s", err)
	}

	// The request was successful
	return &recordResp.Record, nil
}

// GetRecords retrieves all the records for the given domain.
func (c *Client) GetRecords(domain string) ([]Record, error) {
	req, err := c.NewRequest(nil, "GET", "/domains/"+domain+"/records")
	if err != nil {
		return nil, err
	}
	resp, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	recordResponses := make([]RecordResponse, 10)
	err = decode(resp.Body, &recordResponses)
	if err != nil {
		return nil, err
	}
	records := make([]Record, len(recordResponses))
	for i, rr := range recordResponses {
		records[i] = rr.Record
	}
	return records, nil
}

package cloudflare

import (
	"errors"
	"fmt"
	"strings"
)

type RecordsResponse struct {
	Response struct {
		Recs struct {
			Records []Record `json:"objs"`
		} `json:"recs"`
	} `json:"response"`
	Result  string `json:"result"`
	Message string `json:"msg"`
}

func (r *RecordsResponse) FindRecord(id string) (*Record, error) {
	if r.Result == "error" {
		return nil, fmt.Errorf("API Error: %s", r.Message)
	}

	objs := r.Response.Recs.Records
	notFoundErr := errors.New("Record not found")

	// No objects, return nil
	if len(objs) < 0 {
		return nil, notFoundErr
	}

	for _, v := range objs {
		// We have a match, return that
		if v.Id == id {
			return &v, nil
		}
	}

	return nil, notFoundErr
}

func (r *RecordsResponse) FindRecordByName(name string, wildcard bool) ([]Record, error) {
	if r.Result == "error" {
		return nil, fmt.Errorf("API Error: %s", r.Message)
	}

	objs := r.Response.Recs.Records
	notFoundErr := errors.New("Record not found")

	// No objects, return nil
	if len(objs) < 0 {
		return nil, notFoundErr
	}

	var recs []Record
	suffix := "." + name

	for _, v := range objs {
		if v.Name == name {
			recs = append(recs, v)
		} else if wildcard && strings.HasSuffix(v.Name, suffix) {
			recs = append(recs, v)
		}
	}

	return recs, nil
}

type RecordResponse struct {
	Response struct {
		Rec struct {
			Record Record `json:"obj"`
		} `json:"rec"`
	} `json:"response"`
	Result  string `json:"result"`
	Message string `json:"msg"`
}

func (r *RecordResponse) GetRecord() (*Record, error) {
	if r.Result == "error" {
		return nil, fmt.Errorf("API Error: %s", r.Message)
	}

	return &r.Response.Rec.Record, nil
}

// Record is used to represent a retrieved Record. All properties
// are set as strings.
type Record struct {
	Id       string `json:"rec_id"`
	Domain   string `json:"zone_name"`
	Name     string `json:"display_name"`
	FullName string `json:"name"`
	Value    string `json:"content"`
	Type     string `json:"type"`
	Priority string `json:"prio"`
	Ttl      string `json:"ttl"`
}

// CreateRecord contains the request parameters to create a new
// record.
type CreateRecord struct {
	Type     string
	Name     string
	Content  string
	Ttl      string
	Priority string
}

// CreateRecord creates a record from the parameters specified and
// returns an error if it fails. If no error and the name is returned,
// the Record was succesfully created.
func (c *Client) CreateRecord(domain string, opts *CreateRecord) (*Record, error) {
	// Make the request parameters
	params := make(map[string]string)
	params["z"] = domain

	params["type"] = opts.Type

	if opts.Name != "" {
		params["name"] = opts.Name
	}

	if opts.Content != "" {
		params["content"] = opts.Content
	}

	if opts.Priority != "" {
		params["prio"] = opts.Priority
	}

	if opts.Ttl != "" {
		params["ttl"] = opts.Ttl
	} else {
		params["ttl"] = "1"
	}

	req, err := c.NewRequest(params, "POST", "rec_new")
	if err != nil {
		return nil, err
	}

	resp, err := checkResp(c.Http.Do(req))

	if err != nil {
		return nil, fmt.Errorf("Error creating record: %s", err)
	}

	recordResp := new(RecordResponse)

	err = decodeBody(resp, &recordResp)

	if err != nil {
		return nil, fmt.Errorf("Error parsing record response: %s", err)
	}
	record, err := recordResp.GetRecord()
	if err != nil {
		return nil, err
	}

	// The request was successful
	return record, nil
}

// DestroyRecord destroys a record by the ID specified and
// returns an error if it fails. If no error is returned,
// the Record was succesfully destroyed.
func (c *Client) DestroyRecord(domain string, id string) error {
	params := make(map[string]string)

	params["z"] = domain
	params["id"] = id

	req, err := c.NewRequest(params, "POST", "rec_delete")
	if err != nil {
		return err
	}

	resp, err := checkResp(c.Http.Do(req))

	if err != nil {
		return fmt.Errorf("Error deleting record: %s", err)
	}

	recordResp := new(RecordResponse)

	err = decodeBody(resp, &recordResp)

	if err != nil {
		return fmt.Errorf("Error parsing record response: %s", err)
	}
	_, err = recordResp.GetRecord()
	if err != nil {
		return err
	}

	// The request was successful
	return nil
}

// UpdateRecord contains the request parameters to update a
// record.
type UpdateRecord struct {
	Type     string
	Name     string
	Content  string
	Ttl      string
	Priority string
}

// UpdateRecord destroys a record by the ID specified and
// returns an error if it fails. If no error is returned,
// the Record was succesfully updated.
func (c *Client) UpdateRecord(domain string, id string, opts *UpdateRecord) error {
	params := make(map[string]string)
	params["z"] = domain
	params["id"] = id

	params["type"] = opts.Type

	if opts.Name != "" {
		params["name"] = opts.Name
	}

	if opts.Content != "" {
		params["content"] = opts.Content
	}

	if opts.Priority != "" {
		params["prio"] = opts.Priority
	}

	if opts.Ttl != "" {
		params["ttl"] = opts.Ttl
	} else {
		params["ttl"] = "1"
	}

	req, err := c.NewRequest(params, "POST", "rec_edit")
	if err != nil {
		return err
	}

	resp, err := checkResp(c.Http.Do(req))

	if err != nil {
		return fmt.Errorf("Error updating record: %s", err)
	}

	recordResp := new(RecordResponse)

	err = decodeBody(resp, &recordResp)

	if err != nil {
		return fmt.Errorf("Error parsing record response: %s", err)
	}
	_, err = recordResp.GetRecord()
	if err != nil {
		return err
	}

	// The request was successful
	return nil
}

func (c *Client) RetrieveRecordsByName(domain string, name string, wildcard bool) ([]Record, error) {
	params := make(map[string]string)
	// The zone we want
	params["z"] = domain

	req, err := c.NewRequest(params, "GET", "rec_load_all")

	if err != nil {
		return nil, err
	}

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return nil, fmt.Errorf("Error retrieving record: %s", err)
	}

	records := new(RecordsResponse)

	err = decodeBody(resp, records)

	if err != nil {
		return nil, fmt.Errorf("Error decoding record response: %s", err)
	}

	record, err := records.FindRecordByName(name, wildcard)
	if err != nil {
		return nil, err
	}

	// The request was successful
	return record, nil
}

// RetrieveRecord gets  a record by the ID specified and
// returns a Record and an error. An error will be returned for failed
// requests with a nil Record.
func (c *Client) RetrieveRecord(domain string, id string) (*Record, error) {
	params := make(map[string]string)
	// The zone we want
	params["z"] = domain
	params["id"] = id

	req, err := c.NewRequest(params, "GET", "rec_load_all")

	if err != nil {
		return nil, err
	}

	resp, err := checkResp(c.Http.Do(req))
	if err != nil {
		return nil, fmt.Errorf("Error retrieving record: %s", err)
	}

	records := new(RecordsResponse)

	err = decodeBody(resp, records)

	if err != nil {
		return nil, fmt.Errorf("Error decoding record response: %s", err)
	}

	record, err := records.FindRecord(id)
	if err != nil {
		return nil, err
	}

	// The request was successful
	return record, nil
}

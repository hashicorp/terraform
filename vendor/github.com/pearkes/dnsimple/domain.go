package dnsimple

import (
	"encoding/json"
	"io"
	"time"
)

// GetDomains retrieves all the domains for a given account.
func (c *Client) GetDomains() ([]Domain, error) {
	req, err := c.NewRequest(nil, "GET", "/domains")
	if err != nil {
		return nil, err
	}
	resp, err := c.Http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	domainResponses := []DomainResponse{}
	err = decode(resp.Body, &domainResponses)
	if err != nil {
		return nil, err
	}
	domains := make([]Domain, len(domainResponses))
	for i, dr := range domainResponses {
		domains[i] = dr.Domain
	}
	return domains, nil
}

func decode(reader io.Reader, obj interface{}) error {
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&obj)
	if err != nil {
		return err
	}
	return nil
}

type DomainResponse struct {
	Domain Domain `json:"domain"`
}

type Domain struct {
	Id             int       `json:"id"`
	UserId         int       `json:"user_id"`
	RegistrantId   int       `json:"registrant_id"`
	Name           string    `json:"name"`
	UnicodeName    string    `json:"unicode_name"`
	Token          string    `json:"token"`
	State          string    `json:"state"`
	Language       string    `json:"language"`
	Lockable       bool      `json:"lockable"`
	AutoRenew      bool      `json:"auto_renew"`
	WhoisProtected bool      `json:"whois_protected"`
	RecordCount    int       `json:"record_count"`
	ServiceCount   int       `json:"service_count"`
	ExpiresOn      string    `json:"expires_on"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

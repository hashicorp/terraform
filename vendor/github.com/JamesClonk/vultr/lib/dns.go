package lib

import (
	"fmt"
	"net/url"
)

// DNS Domain
type DnsDomain struct {
	Domain  string `json:"domain"`
	Created string `json:"date_created"`
}

// DNS Record
type DnsRecord struct {
	RecordID int    `json:"RECORDID"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Data     string `json:"data"`
	Priority int    `json:"priority"`
	TTL      int    `json:"ttl"`
}

func (c *Client) GetDnsDomains() (dnsdomains []DnsDomain, err error) {
	if err := c.get(`dns/list`, &dnsdomains); err != nil {
		return nil, err
	}
	return dnsdomains, nil
}

func (c *Client) CreateDnsDomain(domain, serverip string) error {
	values := url.Values{
		"domain":   {domain},
		"serverip": {serverip},
	}

	if err := c.post(`dns/create_domain`, values, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteDnsDomain(domain string) error {
	values := url.Values{
		"domain": {domain},
	}

	if err := c.post(`dns/delete_domain`, values, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) GetDnsRecords(domain string) (dnsrecords []DnsRecord, err error) {
	if err := c.get(`dns/records?domain=`+domain, &dnsrecords); err != nil {
		return nil, err
	}
	return dnsrecords, nil
}

func (c *Client) CreateDnsRecord(domain, name, rtype, data string, priority, ttl int) error {
	values := url.Values{
		"domain":   {domain},
		"name":     {name},
		"type":     {rtype},
		"data":     {data},
		"priority": {fmt.Sprintf("%v", priority)},
		"ttl":      {fmt.Sprintf("%v", ttl)},
	}

	if err := c.post(`dns/create_record`, values, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) UpdateDnsRecord(domain string, dnsrecord DnsRecord) error {
	values := url.Values{
		"domain":   {domain},
		"RECORDID": {fmt.Sprintf("%v", dnsrecord.RecordID)},
	}

	if dnsrecord.Name != "" {
		values.Add("name", dnsrecord.Name)
	}
	if dnsrecord.Data != "" {
		values.Add("data", dnsrecord.Data)
	}
	if dnsrecord.Priority != 0 {
		values.Add("priority", fmt.Sprintf("%v", dnsrecord.Priority))
	}
	if dnsrecord.TTL != 0 {
		values.Add("ttl", fmt.Sprintf("%v", dnsrecord.TTL))
	}

	if err := c.post(`dns/update_record`, values, nil); err != nil {
		return err
	}
	return nil
}

func (c *Client) DeleteDnsRecord(domain string, recordID int) error {
	values := url.Values{
		"domain":   {domain},
		"RECORDID": {fmt.Sprintf("%v", recordID)},
	}

	if err := c.post(`dns/delete_record`, values, nil); err != nil {
		return err
	}
	return nil
}

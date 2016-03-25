package namecheap

import (
	"bytes"
	"fmt"
	"github.com/hashicorp/terraform/helper/hashcode"
)

func (c *Client) AddRecord(domain string, record *Record) (*Record, error) {
	allRecords, err := c.GetHosts(domain)
	if err != nil {
		return nil, err
	}
	records := RemoveParkingRecords(domain, allRecords)
	records = append(records, *record)
	_, err = c.SetHosts(domain, records)
	if err != nil {
		return nil, err
	}

	return record, nil
}

func (c *Client) ReadRecord(domain string, hashId int) (*Record, error) {
	records, err := c.GetHosts(domain)
	if err != nil {
		return nil, err
	}
	record, err := c.FindRecordByHash(hashId, records)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (c *Client) UpdateRecord(domain string, hashId int, record *Record) error {
	allRecords, err := c.GetHosts(domain)
	if err != nil {
		return err
	}
	records := c.RemoveRecordByHash(hashId, allRecords)
	records = append(records, *record)
	_, err = c.SetHosts(domain, records)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteRecord(domain string, hashId int) error {
	allRecords, err := c.GetHosts(domain)
	if err != nil {
		return err
	}
	records := c.RemoveRecordByHash(hashId, allRecords)
	_, err = c.SetHosts(domain, records)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) CreateHash(record *Record) int {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(record.HostName))
	buf.WriteString(fmt.Sprintf(record.RecordType))
	buf.WriteString(fmt.Sprintf(record.Address))
	return hashcode.String(buf.String())
}

func (c *Client) FindRecordByHash(hashId int, records []Record) (*Record, error) {
	for _, record := range records {
		recordHash := c.CreateHash(&record)
		if recordHash == hashId {
			return &record, nil
		}
	}
	return nil, fmt.Errorf("Could not find the record")
}

func (c *Client) RemoveRecordByHash(hashId int, records []Record) []Record {
	var ret []Record
	for _, record := range records {
		recordHash := c.CreateHash(&record)
		if recordHash == hashId {
			continue
		}
		ret = append(ret, record)
	}
	return ret
}

func RemoveParkingRecords(domain string, records []Record) []Record {
	var ret []Record
	for _, record := range records {
		if record.RecordType == "CNAME" && record.HostName == "www" && record.Address == "parkingpage.namecheap.com." {
			continue
		}
		if record.RecordType == "URL" && record.HostName == "@" && record.Address == "http://www."+domain+"/?from=@" {
			continue
		}
		ret = append(ret, record)
	}
	return ret
}

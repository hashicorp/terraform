package dynect

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ConvenientClient A client with extra helper methods for common actions
type ConvenientClient struct {
	Client
}

// NewConvenientClient Creates a new ConvenientClient
func NewConvenientClient(customerName string) *ConvenientClient {
	return &ConvenientClient{
		Client{
			CustomerName: customerName,
			httpclient:   &http.Client{},
		}}
}

// PublishZone Publish a specific zone and the changes for the current session
func (c *ConvenientClient) PublishZone(zone string) error {
	data := &PublishZoneBlock{
		Publish: true,
	}
	return c.Do("PUT", "Zone/"+zone, data, nil)
}

// GetRecordID finds the dns record ID by fetching all records for a FQDN
func (c *ConvenientClient) GetRecordID(record *Record) error {
	finalID := ""
	url := fmt.Sprintf("AllRecord/%s/%s", record.Zone, record.FQDN)
	var records AllRecordsResponse
	err := c.Do("GET", url, nil, &records)
	if err != nil {
		return fmt.Errorf("Failed to find Dyn record id: %s", err)
	}
	for _, recordURL := range records.Data {
		id := strings.TrimPrefix(recordURL, fmt.Sprintf("/REST/%sRecord/%s/%s/", record.Type, record.Zone, record.FQDN))
		if !strings.Contains(id, "/") && id != "" {
			finalID = id
			log.Printf("[INFO] Found Dyn record ID: %s", id)
		}
	}
	if finalID == "" {
		return fmt.Errorf("Failed to find Dyn record id!")
	}

	record.ID = finalID
	return nil
}

// CreateRecord Method to create a DNS record
func (c *ConvenientClient) CreateRecord(record *Record) error {
	if record.FQDN == "" {
		record.FQDN = fmt.Sprintf("%s.%s", record.Name, record.Zone)
	}
	rdata, err := buildRData(record)
	if err != nil {
		return fmt.Errorf("Failed to create Dyn RData: %s", err)
	}
	url := fmt.Sprintf("%sRecord/%s/%s", record.Type, record.Zone, record.FQDN)
	data := &RecordRequest{
		RData: rdata,
		TTL:   record.TTL,
	}
	return c.Do("POST", url, data, nil)
}

// UpdateRecord Method to update a DNS record
func (c *ConvenientClient) UpdateRecord(record *Record) error {
	if record.FQDN == "" {
		record.FQDN = fmt.Sprintf("%s.%s", record.Name, record.Zone)
	}
	rdata, err := buildRData(record)
	if err != nil {
		return fmt.Errorf("Failed to create Dyn RData: %s", err)
	}
	url := fmt.Sprintf("%sRecord/%s/%s/%s", record.Type, record.Zone, record.FQDN, record.ID)
	data := &RecordRequest{
		RData: rdata,
		TTL:   record.TTL,
	}
	return c.Do("PUT", url, data, nil)
}

// DeleteRecord Method to delete a DNS record
func (c *ConvenientClient) DeleteRecord(record *Record) error {
	if record.FQDN == "" {
		record.FQDN = fmt.Sprintf("%s.%s", record.Name, record.Zone)
	}
	// safety check that we have an ID, otherwise we could accidentally delete everything
	if record.ID == "" {
		return fmt.Errorf("No ID found! We can't continue!")
	}
	url := fmt.Sprintf("%sRecord/%s/%s/%s", record.Type, record.Zone, record.FQDN, record.ID)
	return c.Do("DELETE", url, nil, nil)
}

// GetRecord Method to get record details
func (c *ConvenientClient) GetRecord(record *Record) error {
	url := fmt.Sprintf("%sRecord/%s/%s/%s", record.Type, record.Zone, record.FQDN, record.ID)
	var rec RecordResponse
	err := c.Do("GET", url, nil, &rec)
	if err != nil {
		return err
	}

	record.Zone = rec.Data.Zone
	record.FQDN = rec.Data.FQDN
	record.Name = strings.TrimSuffix(rec.Data.FQDN, "."+rec.Data.Zone)
	record.Type = rec.Data.RecordType
	record.TTL = strconv.Itoa(rec.Data.TTL)

	switch rec.Data.RecordType {
	case "A", "AAAA":
		record.Value = rec.Data.RData.Address
	case "CNAME":
		record.Value = rec.Data.RData.CName
	case "TXT", "SPF":
		record.Value = rec.Data.RData.TxtData
	default:
		return fmt.Errorf("Invalid Dyn record type: %s", rec.Data.RecordType)
	}

	return nil
}

func buildRData(r *Record) (DataBlock, error) {
	var rdata DataBlock

	switch r.Type {
	case "A", "AAAA":
		rdata = DataBlock{
			Address: r.Value,
		}
	case "CNAME":
		rdata = DataBlock{
			CName: r.Value,
		}
	case "TXT", "SPF":
		rdata = DataBlock{
			TxtData: r.Value,
		}
	default:
		return rdata, fmt.Errorf("Invalid Dyn record type: %s", r.Type)
	}

	return rdata, nil
}

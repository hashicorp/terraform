package openstack

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform/helper/resource"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"
)

// RecordSetCreateOpts represents the attributes used when creating a new DNS record set.
type RecordSetCreateOpts struct {
	recordsets.CreateOpts
	ValueSpecs map[string]string `json:"value_specs,omitempty"`
}

// ToRecordSetCreateMap casts a CreateOpts struct to a map.
// It overrides recordsets.ToRecordSetCreateMap to add the ValueSpecs field.
func (opts RecordSetCreateOpts) ToRecordSetCreateMap() (map[string]interface{}, error) {
	b, err := BuildRequest(opts, "")
	if err != nil {
		return nil, err
	}

	if m, ok := b[""].(map[string]interface{}); ok {
		return m, nil
	}

	return nil, fmt.Errorf("Expected map but got %T", b[""])
}

func dnsRecordSetV2RefreshFunc(
	dnsClient *gophercloud.ServiceClient, zoneID, recordsetId string) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		recordset, err := recordsets.Get(dnsClient, zoneID, recordsetId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return recordset, "DELETED", nil
			}

			return nil, "", err
		}

		log.Printf("[DEBUG] openstack_dns_recordset_v2 %s current status: %s", recordset.ID, recordset.Status)
		return recordset, recordset.Status, nil
	}
}

func dnsRecordSetV2ParseID(id string) (string, string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("Unable to determine openstack_dns_recordset_v2 ID from raw ID: %s", id)
	}

	zoneID := idParts[0]
	recordsetID := idParts[1]

	return zoneID, recordsetID, nil
}

func expandDNSRecordSetV2Records(v []interface{}) []string {
	records := make([]string, len(v))

	// Strip out any [ ] characters in the address.
	// This is to format IPv6 records in a way that DNSaaS / Designate wants.
	re := regexp.MustCompile("[][]")
	for i, rawRecord := range v {
		record := rawRecord.(string)
		record = re.ReplaceAllString(record, "")
		records[i] = record
	}

	return records
}

// dnsRecordSetV2RecordsStateFunc will strip brackets from IPv6 addresses.
func dnsRecordSetV2RecordsStateFunc(v interface{}) string {
	if addr, ok := v.(string); ok {
		re := regexp.MustCompile("[][]")
		addr = re.ReplaceAllString(addr, "")

		return addr
	}

	return ""
}

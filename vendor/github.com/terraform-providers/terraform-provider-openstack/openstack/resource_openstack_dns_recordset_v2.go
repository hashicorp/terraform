package openstack

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/dns/v2/recordsets"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDNSRecordSetV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceDNSRecordSetV2Create,
		Read:   resourceDNSRecordSetV2Read,
		Update: resourceDNSRecordSetV2Update,
		Delete: resourceDNSRecordSetV2Delete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"records": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"value_specs": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDNSRecordSetV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	recordsraw := d.Get("records").([]interface{})
	records := make([]string, len(recordsraw))
	for i, recordraw := range recordsraw {
		records[i] = recordraw.(string)
	}

	createOpts := RecordSetCreateOpts{
		recordsets.CreateOpts{
			Name:        d.Get("name").(string),
			Description: d.Get("description").(string),
			Records:     records,
			TTL:         d.Get("ttl").(int),
			Type:        d.Get("type").(string),
		},
		MapValueSpecs(d),
	}

	zoneID := d.Get("zone_id").(string)

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	n, err := recordsets.Create(dnsClient, zoneID, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS record set: %s", err)
	}

	log.Printf("[DEBUG] Waiting for DNS record set (%s) to become available", n.ID)
	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Pending:    []string{"PENDING"},
		Refresh:    waitForDNSRecordSet(dnsClient, zoneID, n.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	id := fmt.Sprintf("%s/%s", zoneID, n.ID)
	d.SetId(id)

	log.Printf("[DEBUG] Created OpenStack DNS record set %s: %#v", n.ID, n)
	return resourceDNSRecordSetV2Read(d, meta)
}

func resourceDNSRecordSetV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	// Obtain relevant info from parsing the ID
	zoneID, recordsetID, err := parseDNSV2RecordSetID(d.Id())
	if err != nil {
		return err
	}

	n, err := recordsets.Get(dnsClient, zoneID, recordsetID).Extract()
	if err != nil {
		return CheckDeleted(d, err, "record_set")
	}

	log.Printf("[DEBUG] Retrieved  record set %s: %#v", recordsetID, n)

	d.Set("name", n.Name)
	d.Set("description", n.Description)
	d.Set("ttl", n.TTL)
	d.Set("type", n.Type)
	d.Set("records", n.Records)
	d.Set("region", GetRegion(d, config))
	d.Set("zone_id", zoneID)

	return nil
}

func resourceDNSRecordSetV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	var updateOpts recordsets.UpdateOpts
	if d.HasChange("ttl") {
		updateOpts.TTL = d.Get("ttl").(int)
	}

	if d.HasChange("records") {
		recordsraw := d.Get("records").([]interface{})
		records := make([]string, len(recordsraw))
		for i, recordraw := range recordsraw {
			records[i] = recordraw.(string)
		}
		updateOpts.Records = records
	}

	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}

	// Obtain relevant info from parsing the ID
	zoneID, recordsetID, err := parseDNSV2RecordSetID(d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Updating  record set %s with options: %#v", recordsetID, updateOpts)

	_, err = recordsets.Update(dnsClient, zoneID, recordsetID, updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack DNS  record set: %s", err)
	}

	log.Printf("[DEBUG] Waiting for DNS record set (%s) to update", recordsetID)
	stateConf := &resource.StateChangeConf{
		Target:     []string{"ACTIVE"},
		Pending:    []string{"PENDING"},
		Refresh:    waitForDNSRecordSet(dnsClient, zoneID, recordsetID),
		Timeout:    d.Timeout(schema.TimeoutUpdate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	return resourceDNSRecordSetV2Read(d, meta)
}

func resourceDNSRecordSetV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	dnsClient, err := config.dnsV2Client(GetRegion(d, config))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack DNS client: %s", err)
	}

	// Obtain relevant info from parsing the ID
	zoneID, recordsetID, err := parseDNSV2RecordSetID(d.Id())
	if err != nil {
		return err
	}

	err = recordsets.Delete(dnsClient, zoneID, recordsetID).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack DNS  record set: %s", err)
	}

	log.Printf("[DEBUG] Waiting for DNS record set (%s) to be deleted", recordsetID)
	stateConf := &resource.StateChangeConf{
		Target:     []string{"DELETED"},
		Pending:    []string{"ACTIVE", "PENDING"},
		Refresh:    waitForDNSRecordSet(dnsClient, zoneID, recordsetID),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()

	d.SetId("")
	return nil
}

func waitForDNSRecordSet(dnsClient *gophercloud.ServiceClient, zoneID, recordsetId string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		recordset, err := recordsets.Get(dnsClient, zoneID, recordsetId).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				return recordset, "DELETED", nil
			}

			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack DNS record set (%s) current status: %s", recordset.ID, recordset.Status)
		return recordset, recordset.Status, nil
	}
}

func parseDNSV2RecordSetID(id string) (string, string, error) {
	idParts := strings.Split(id, "/")
	if len(idParts) != 2 {
		return "", "", fmt.Errorf("Unable to determine DNS record set ID from raw ID: %s", id)
	}

	zoneID := idParts[0]
	recordsetID := idParts[1]

	return zoneID, recordsetID, nil
}

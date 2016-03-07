package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsARecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsARecordCreate,
		Read:   resourceArmDnsARecordRead,
		Update: resourceArmDnsARecordCreate,
		Delete: resourceArmDnsARecordDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"zone_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"records": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsARecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateARecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	recordStrings := d.Get("records").(*schema.Set).List()
	records := make([]dns.ARecord, len(recordStrings))
	for i, v := range recordStrings {
		records[i] = dns.ARecord{
			IPv4Address: v.(string),
		}
	}
	createCommand.ARecords = records

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS A Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS A Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetARecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS A Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS A Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetARecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsARecordRead(d, meta)
}

func resourceArmDnsARecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetARecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS A Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS A Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS A Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetARecordSetResponse)

	d.Set("ttl", resp.TTL)

	if resp.ARecords != nil {
		records := make([]string, 0, len(resp.ARecords))
		for _, record := range resp.ARecords {
			records = append(records, record.IPv4Address)
		}

		if err := d.Set("records", records); err != nil {
			return err
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsARecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "A",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS A Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS A Record: %s", deleteResponse.Error)
	}

	return nil
}

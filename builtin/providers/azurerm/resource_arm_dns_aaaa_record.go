package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsAAAARecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsAAAARecordCreate,
		Read:   resourceArmDnsAAAARecordRead,
		Update: resourceArmDnsAAAARecordCreate,
		Delete: resourceArmDnsAAAARecordDelete,

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

func resourceArmDnsAAAARecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateAAAARecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	recordStrings := d.Get("records").(*schema.Set).List()
	records := make([]dns.AAAARecord, len(recordStrings))
	for i, v := range recordStrings {
		records[i] = dns.AAAARecord{
			IPv6Address: v.(string),
		}
	}
	createCommand.AAAARecords = records

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS AAAA Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS AAAA Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetAAAARecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS AAAA Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS AAAA Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetAAAARecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsAAAARecordRead(d, meta)
}

func resourceArmDnsAAAARecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetAAAARecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS AAAA Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS AAAA Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS AAAA Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetAAAARecordSetResponse)

	d.Set("ttl", resp.TTL)

	if resp.AAAARecords != nil {
		records := make([]string, 0, len(resp.AAAARecords))
		for _, record := range resp.AAAARecords {
			records = append(records, record.IPv6Address)
		}

		if err := d.Set("records", records); err != nil {
			return err
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsAAAARecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "AAAA",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS AAAA Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS AAAA Record: %s", deleteResponse.Error)
	}

	return nil
}

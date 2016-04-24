package azurerm

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsCNameRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsCNameRecordCreate,
		Read:   resourceArmDnsCNameRecordRead,
		Update: resourceArmDnsCNameRecordCreate,
		Delete: resourceArmDnsCNameRecordDelete,

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

func resourceArmDnsCNameRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateCNAMERecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	recordStrings := d.Get("records").(*schema.Set).List()
	records := make([]dns.CNAMERecord, len(recordStrings))
	for i, v := range recordStrings {
		records[i] = dns.CNAMERecord{
			CNAME: v.(string),
		}
	}
	createCommand.CNAMERecords = records

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS CName Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS CName Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetCNAMERecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS CName Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS CName Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetCNAMERecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsCNameRecordRead(d, meta)
}

func resourceArmDnsCNameRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetCNAMERecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS A Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS A Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS A Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetCNAMERecordSetResponse)

	d.Set("ttl", resp.TTL)

	if resp.CNAMERecords != nil {
		records := make([]string, 0, len(resp.CNAMERecords))
		for _, record := range resp.CNAMERecords {
			records = append(records, record.CNAME)
		}

		if err := d.Set("records", records); err != nil {
			return err
		}
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsCNameRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "CNAME",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS CName Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS CName Record: %s", deleteResponse.Error)
	}

	return nil
}

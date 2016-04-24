package azurerm

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsSrvRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsSrvRecordCreate,
		Read:   resourceArmDnsSrvRecordRead,
		Update: resourceArmDnsSrvRecordCreate,
		Delete: resourceArmDnsSrvRecordDelete,

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

			"record": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"priority": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"weight": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Required: true,
						},

						"target": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmDnsSrvRecordHash,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsSrvRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateSRVRecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	srvRecords, recordErr := expandAzureRmDnsSrvRecord(d)
	if recordErr != nil {
		return fmt.Errorf("Error Building Azure RM SRV Record: %s", recordErr)
	}
	createCommand.SRVRecords = srvRecords

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS SRV Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS SRV Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetSRVRecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS SRV Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS SRV Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetSRVRecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsSrvRecordRead(d, meta)
}

func resourceArmDnsSrvRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetSRVRecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS SRV Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS SRV Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS SRV Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetSRVRecordSetResponse)

	d.Set("ttl", resp.TTL)

	if err := d.Set("record", flattenAzureRmDnsSrvRecord(resp.SRVRecords)); err != nil {
		log.Printf("[INFO] Error setting the Azure RM SRV Record State: %s", err)
		return err
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsSrvRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "SRV",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS SRV Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS SRV Record: %s", deleteResponse.Error)
	}

	return nil
}

func expandAzureRmDnsSrvRecord(d *schema.ResourceData) ([]dns.SRVRecord, error) {
	config := d.Get("record").(*schema.Set).List()
	records := make([]dns.SRVRecord, 0, len(config))

	for _, pRaw := range config {
		data := pRaw.(map[string]interface{})

		srvRecord := dns.SRVRecord{
			Priority: data["priority"].(int),
			Weight:   data["weight"].(int),
			Port:     data["port"].(int),
			Target:   data["target"].(string),
		}

		records = append(records, srvRecord)

	}

	return records, nil

}

func flattenAzureRmDnsSrvRecord(records []dns.SRVRecord) []map[string]interface{} {

	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]interface{}{
			"priority": record.Priority,
			"weight":   record.Weight,
			"port":     record.Port,
			"target":   record.Target,
		})
	}
	return result

}

func resourceArmDnsSrvRecordHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%d-", m["priority"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["weight"].(int)))
	buf.WriteString(fmt.Sprintf("%d-", m["port"].(int)))
	buf.WriteString(fmt.Sprintf("%s-", m["target"].(string)))

	return hashcode.String(buf.String())
}

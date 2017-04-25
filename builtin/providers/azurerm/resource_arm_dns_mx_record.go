package azurerm

import (
	"bytes"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/jen20/riviera/dns"
)

func resourceArmDnsMxRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmDnsMxRecordCreate,
		Read:   resourceArmDnsMxRecordRead,
		Update: resourceArmDnsMxRecordCreate,
		Delete: resourceArmDnsMxRecordDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
						"preference": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"exchange": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: resourceArmDnsMxRecordHash,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmDnsMxRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	tags := d.Get("tags").(map[string]interface{})
	expandedTags := expandTags(tags)

	createCommand := &dns.CreateMXRecordSet{
		Name:              d.Get("name").(string),
		Location:          "global",
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
		TTL:               d.Get("ttl").(int),
		Tags:              *expandedTags,
	}

	mxRecords, recordErr := expandAzureRmDnsMxRecord(d)
	if recordErr != nil {
		return fmt.Errorf("Error Building Azure RM MX Record: %s", recordErr)
	}
	createCommand.MXRecords = mxRecords

	createRequest := rivieraClient.NewRequest()
	createRequest.Command = createCommand

	createResponse, err := createRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error creating DNS MX Record: %s", err)
	}
	if !createResponse.IsSuccessful() {
		return fmt.Errorf("Error creating DNS MX Record: %s", createResponse.Error)
	}

	readRequest := rivieraClient.NewRequest()
	readRequest.Command = &dns.GetMXRecordSet{
		Name:              d.Get("name").(string),
		ResourceGroupName: d.Get("resource_group_name").(string),
		ZoneName:          d.Get("zone_name").(string),
	}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS MX Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		return fmt.Errorf("Error reading DNS MX Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetMXRecordSetResponse)
	d.SetId(resp.ID)

	return resourceArmDnsMxRecordRead(d, meta)
}

func resourceArmDnsMxRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}

	readRequest := rivieraClient.NewRequestForURI(d.Id())
	readRequest.Command = &dns.GetMXRecordSet{}

	readResponse, err := readRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error reading DNS MX Record: %s", err)
	}
	if !readResponse.IsSuccessful() {
		log.Printf("[INFO] Error reading DNS MX Record %q - removing from state", d.Id())
		d.SetId("")
		return fmt.Errorf("Error reading DNS MX Record: %s", readResponse.Error)
	}

	resp := readResponse.Parsed.(*dns.GetMXRecordSetResponse)

	d.Set("name", resp.Name)
	d.Set("resource_group_name", id.ResourceGroup)
	d.Set("zone_name", id.Path["dnszones"])
	d.Set("ttl", resp.TTL)

	if err := d.Set("record", flattenAzureRmDnsMxRecord(resp.MXRecords)); err != nil {
		log.Printf("[INFO] Error setting the Azure RM MX Record State: %s", err)
		return err
	}

	flattenAndSetTags(d, &resp.Tags)

	return nil
}

func resourceArmDnsMxRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	rivieraClient := client.rivieraClient

	deleteRequest := rivieraClient.NewRequestForURI(d.Id())
	deleteRequest.Command = &dns.DeleteRecordSet{
		RecordSetType: "MX",
	}

	deleteResponse, err := deleteRequest.Execute()
	if err != nil {
		return fmt.Errorf("Error deleting DNS MX Record: %s", err)
	}
	if !deleteResponse.IsSuccessful() {
		return fmt.Errorf("Error deleting DNS MX Record: %s", deleteResponse.Error)
	}

	return nil
}

func expandAzureRmDnsMxRecord(d *schema.ResourceData) ([]dns.MXRecord, error) {
	config := d.Get("record").(*schema.Set).List()
	records := make([]dns.MXRecord, 0, len(config))

	for _, pRaw := range config {
		data := pRaw.(map[string]interface{})

		mxrecord := dns.MXRecord{
			Preference: data["preference"].(string),
			Exchange:   data["exchange"].(string),
		}

		records = append(records, mxrecord)

	}

	return records, nil

}

func flattenAzureRmDnsMxRecord(records []dns.MXRecord) []map[string]interface{} {

	result := make([]map[string]interface{}, 0, len(records))
	for _, record := range records {
		result = append(result, map[string]interface{}{
			"preference": record.Preference,
			"exchange":   record.Exchange,
		})
	}
	return result

}

func resourceArmDnsMxRecordHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["preference"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["exchange"].(string)))

	return hashcode.String(buf.String())
}

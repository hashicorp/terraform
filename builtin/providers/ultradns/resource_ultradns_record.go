package ultradns

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"udnssdk"
)

func resourceUltraDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceUltraDNSRecordCreate,
		Read:   resourceUltraDNSRecordRead,
		Update: resourceUltraDNSRecordUpdate,
		Delete: resourceUltraDNSRecordDelete,

		Schema: map[string]*schema.Schema{
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"rdata": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},

		},
	}
}

func resourceUltraDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)
	newRecord := &udnssdk.RRSet{
		OwnerName:	d.Get("name").(string),
		RRType: d.Get("type").(string),
		RData: d.Get("rdata").([]string),
		}

	if ttl, ok := d.GetOk("ttl"); ok {
		newRecord.TTL = ttl.(int)
	}

	log.Printf("[DEBUG] UltraDNS RRSet create configuration: %#v", newRecord)

	_, err := client.RRSets.CreateRRSet(d.Get("zone").(string), *newRecord)
	recId := fmt.Sprintf("%s.%s",d.Get("name").(string),d.Get("zone").(string))
	if err != nil {
		return fmt.Errorf("Failed to create UltraDNS RRSet: %s", err)
	}

	d.SetId(recId)
	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	rrsets, _, err := client.RRSets.GetRRSets(d.Get("zone").(string), d.Get("name").(string), d.Get("type").(string))
	if err != nil {
		return fmt.Errorf("Couldn't find UltraDNS RRSet: %s", err)
	}
	rec := rrsets[0]
	d.Set("rdata", rec.RData)
	d.Set("ttl", rec.TTL)

	if rec.OwnerName == "" {
		d.Set("hostname", d.Get("zone").(string))
	} else {
		d.Set("hostname", fmt.Sprintf("%s.%s", rec.OwnerName, d.Get("zone").(string)))
	}

	return nil
}

func resourceUltraDNSRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	updateRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		updateRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		updateRecord.RRType = attr.(string)
	}

	if attr, ok := d.GetOk("rdata"); ok {
		updateRecord.RData = attr.([]string)
	}

	if attr, ok := d.GetOk("ttl"); ok {
		updateRecord.TTL = attr.(int)
	}

	log.Printf("[DEBUG] UltraDNS RRSet update configuration: %#v", updateRecord)

	_, err := client.RRSets.UpdateRRSet(d.Get("zone").(string), *updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update UltraDNS RRSet: %s", err)
	}

	return resourceUltraDNSRecordRead(d, meta)
}

func resourceUltraDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*udnssdk.Client)

	log.Printf("[INFO] Deleting UltraDNS RRSet: %s, %s", d.Get("zone").(string), d.Id())
	deleteRecord := &udnssdk.RRSet{}

	if attr, ok := d.GetOk("name"); ok {
		deleteRecord.OwnerName = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		deleteRecord.RRType = attr.(string)
	}


	_, err := client.RRSets.DeleteRRSet(d.Get("zone").(string), *deleteRecord)

	if err != nil {
		return fmt.Errorf("Error deleting UltraDNS RRSet: %s", err)
	}

	return nil
}

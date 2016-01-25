package powerdns

import (
	"log"

	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePDNSRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourcePDNSRecordCreate,
		Read:   resourcePDNSRecordRead,
		Delete: resourcePDNSRecordDelete,
		Exists: resourcePDNSRecordExists,

		Schema: map[string]*schema.Schema{
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ttl": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"records": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				ForceNew: true,
				Set:      schema.HashString,
			},
		},
	}
}

func resourcePDNSRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	rrSet := ResourceRecordSet{
		Name: d.Get("name").(string),
		Type: d.Get("type").(string),
	}

	zone := d.Get("zone").(string)
	ttl := d.Get("ttl").(int)
	recs := d.Get("records").(*schema.Set).List()

	if len(recs) > 0 {
		records := make([]Record, 0, len(recs))
		for _, recContent := range recs {
			records = append(records, Record{Name: rrSet.Name, Type: rrSet.Type, TTL: ttl, Content: recContent.(string)})
		}
		rrSet.Records = records

		log.Printf("[DEBUG] Creating PowerDNS Record: %#v", rrSet)

		recId, err := client.ReplaceRecordSet(zone, rrSet)
		if err != nil {
			return fmt.Errorf("Failed to create PowerDNS Record: %s", err)
		}

		d.SetId(recId)
		log.Printf("[INFO] Created PowerDNS Record with ID: %s", d.Id())

	} else {
		log.Printf("[DEBUG] Deleting empty PowerDNS Record: %#v", rrSet)
		err := client.DeleteRecordSet(zone, rrSet.Name, rrSet.Type)
		if err != nil {
			return fmt.Errorf("Failed to delete PowerDNS Record: %s", err)
		}

		d.SetId(rrSet.Id())
	}

	return resourcePDNSRecordRead(d, meta)
}

func resourcePDNSRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[DEBUG] Reading PowerDNS Record: %s", d.Id())
	records, err := client.ListRecordsByID(d.Get("zone").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't fetch PowerDNS Record: %s", err)
	}

	recs := make([]string, 0, len(records))
	for _, r := range records {
		recs = append(recs, r.Content)
	}
	d.Set("records", recs)

	if len(records) > 0 {
		d.Set("ttl", records[0].TTL)
	}

	return nil
}

func resourcePDNSRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)

	log.Printf("[INFO] Deleting PowerDNS Record: %s", d.Id())
	err := client.DeleteRecordSetByID(d.Get("zone").(string), d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting PowerDNS Record: %s", err)
	}

	return nil
}

func resourcePDNSRecordExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	zone := d.Get("zone").(string)
	name := d.Get("name").(string)
	tpe := d.Get("type").(string)

	log.Printf("[INFO] Checking existence of PowerDNS Record: %s, %s", name, tpe)

	client := meta.(*Client)
	exists, err := client.RecordExists(zone, name, tpe)

	if err != nil {
		return false, fmt.Errorf("Error checking PowerDNS Record: %s", err)
	} else {
		return exists, nil
	}
}

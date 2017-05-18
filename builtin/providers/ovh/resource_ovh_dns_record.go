package ovh

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
)

type NewRecord struct {
	Target    string `json:"target"`
	Ttl       int    `json:"ttl"`
	FieldType string `json:"fieldType"`
	SubDomain string `json:"subDomain"`
}

type Record struct {
	Id        int    `json:"id"`
	Zone      string `json:"zone"`
	Target    string `json:"target"`
	Ttl       int    `json:"ttl"`
	FieldType string `json:"fieldType"`
	SubDomain string `json:"subDomain"`
}

func resourceOVHDomainZoneRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceOVHRecordCreate,
		Read:   resourceOVHRecordRead,
		Update: resourceOVHRecordUpdate,
		Delete: resourceOVHRecordDelete,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Required: true,
			},
			"target": {
				Type:     schema.TypeString,
				Required: true,
			},
			"ttl": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},
			"fieldType": {
				Type:     schema.TypeString,
				Required: true,
			},
			"subDomain": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceOVHRecordCreate(d *schema.ResourceData, meta interface{}) error {
	provider := meta.(*Client)

	// Create the new record
	newRecord := &NewRecord{
		FieldType: d.Get("fieldType").(string),
		SubDomain: d.Get("subDomain").(string),
		Target:    d.Get("target").(string),
	}

	newRecord.Ttl, _ = strconv.Atoi(d.Get("ttl").(string))

	log.Printf("[DEBUG] OVH Record create configuration: %#v", newRecord)

	resultRecord := Record{}

	err := provider.client.Post(
		fmt.Sprintf("/domain/zone/%s/record", d.Get("zone").(string)),
		newRecord,
		&resultRecord,
	)

	if err != nil {
		return fmt.Errorf("Failed to create OVH Record: %s", err)
	}

	d.SetId(strconv.Itoa(resultRecord.Id))
	d.Set("id", strconv.Itoa(resultRecord.Id))
	log.Printf("[INFO] OVH Record ID: %s", d.Id())

	return resourceOVHRecordRead(d, meta)
}

func resourceOVHRecordRead(d *schema.ResourceData, meta interface{}) error {
	provider := meta.(*Client)

	recordID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Error converting Record ID: %s", err)
	}

	record := Record{}
	err = provider.client.Get(
		fmt.Sprintf("/domain/zone/%s/record/%d", d.Get("zone").(string), recordID),
		&record,
	)

	if err != nil {
		d.SetId("")
		return nil
	}

	d.Set("id", record.Id)
	d.Set("zone", record.Zone)
	d.Set("fieldType", record.FieldType)
	d.Set("subDomain", record.SubDomain)
	d.Set("ttl", strconv.Itoa(record.Ttl))
	d.Set("target", record.Target)

	return nil
}

func resourceOVHRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	provider := meta.(*Client)

	recordID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Error converting Record ID: %s", err)
	}

	record := NewRecord{}

	if attr, ok := d.GetOk("subDomain"); ok {
		record.SubDomain = attr.(string)
	}
	if attr, ok := d.GetOk("fieldType"); ok {
		record.FieldType = attr.(string)
	}
	if attr, ok := d.GetOk("target"); ok {
		record.Target = attr.(string)
	}
	if attr, ok := d.GetOk("ttl"); ok {
		record.Ttl, _ = strconv.Atoi(attr.(string))
	}

	log.Printf("[DEBUG] OVH Record update configuration: %#v", record)

	err = provider.client.Put(
		fmt.Sprintf("/domain/zone/%s/record/%d", d.Get("zone").(string), recordID),
		record,
		nil,
	)
	if err != nil {
		return fmt.Errorf("Failed to update OVH Record: %s", err)
	}

	return resourceOVHRecordRead(d, meta)
}

func resourceOVHRecordDelete(d *schema.ResourceData, meta interface{}) error {
	provider := meta.(*Client)

	log.Printf("[INFO] Deleting OVH Record: %s.%s, %s", d.Get("zone").(string), d.Get("subDomain").(string), d.Id())

	recordID, err := strconv.Atoi(d.Id())
	if err != nil {
		return fmt.Errorf("Error converting Record ID: %s", err)
	}

	err = provider.client.Delete(
		fmt.Sprintf("/domain/zone/%s/record/%d", d.Get("zone").(string), recordID),
		nil,
	)

	if err != nil {
		return fmt.Errorf("Error deleting OVH Record: %s", err)
	}

	return nil
}

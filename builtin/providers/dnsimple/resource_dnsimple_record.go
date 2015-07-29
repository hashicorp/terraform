package dnsimple

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/dnsimple"
)

func resourceDNSimpleRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDNSimpleRecordCreate,
		Read:   resourceDNSimpleRecordRead,
		Update: resourceDNSimpleRecordUpdate,
		Delete: resourceDNSimpleRecordDelete,

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"domain_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
				ForceNew: true,
			},

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "3600",
			},

			"priority": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceDNSimpleRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsimple.Client)

	// Create the new record
	newRecord := &dnsimple.ChangeRecord{
		Name:  d.Get("name").(string),
		Type:  d.Get("type").(string),
		Value: d.Get("value").(string),
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		newRecord.Ttl = ttl.(string)
	}

	log.Printf("[DEBUG] DNSimple Record create configuration: %#v", newRecord)

	recId, err := client.CreateRecord(d.Get("domain").(string), newRecord)

	if err != nil {
		return fmt.Errorf("Failed to create DNSimple Record: %s", err)
	}

	d.SetId(recId)
	log.Printf("[INFO] record ID: %s", d.Id())

	return resourceDNSimpleRecordRead(d, meta)
}

func resourceDNSimpleRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsimple.Client)

	rec, err := client.RetrieveRecord(d.Get("domain").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Couldn't find DNSimple Record: %s", err)
	}

	d.Set("domain_id", rec.StringDomainId())
	d.Set("name", rec.Name)
	d.Set("type", rec.RecordType)
	d.Set("value", rec.Content)
	d.Set("ttl", rec.StringTtl())
	d.Set("priority", rec.StringPrio())

	if rec.Name == "" {
		d.Set("hostname", d.Get("domain").(string))
	} else {
		d.Set("hostname", fmt.Sprintf("%s.%s", rec.Name, d.Get("domain").(string)))
	}

	return nil
}

func resourceDNSimpleRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsimple.Client)

	updateRecord := &dnsimple.ChangeRecord{}

	if attr, ok := d.GetOk("name"); ok {
		updateRecord.Name = attr.(string)
	}

	if attr, ok := d.GetOk("type"); ok {
		updateRecord.Type = attr.(string)
	}

	if attr, ok := d.GetOk("value"); ok {
		updateRecord.Value = attr.(string)
	}

	if attr, ok := d.GetOk("ttl"); ok {
		updateRecord.Ttl = attr.(string)
	}

	log.Printf("[DEBUG] DNSimple Record update configuration: %#v", updateRecord)

	_, err := client.UpdateRecord(d.Get("domain").(string), d.Id(), updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update DNSimple Record: %s", err)
	}

	return resourceDNSimpleRecordRead(d, meta)
}

func resourceDNSimpleRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dnsimple.Client)

	log.Printf("[INFO] Deleting DNSimple Record: %s, %s", d.Get("domain").(string), d.Id())

	err := client.DestroyRecord(d.Get("domain").(string), d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting DNSimple Record: %s", err)
	}

	return nil
}

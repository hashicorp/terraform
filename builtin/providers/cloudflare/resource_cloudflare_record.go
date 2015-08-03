package cloudflare

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/cloudflare"
)

func resourceCloudFlareRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudFlareRecordCreate,
		Read:   resourceCloudFlareRecordRead,
		Update: resourceCloudFlareRecordUpdate,
		Delete: resourceCloudFlareRecordDelete,

		Schema: map[string]*schema.Schema{
			"domain": &schema.Schema{
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

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ttl": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"priority": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceCloudFlareRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	// Create the new record
	newRecord := &cloudflare.CreateRecord{
		Name:    d.Get("name").(string),
		Type:    d.Get("type").(string),
		Content: d.Get("value").(string),
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		newRecord.Ttl = ttl.(string)
	}

	if priority, ok := d.GetOk("priority"); ok {
		newRecord.Priority = priority.(string)
	}

	log.Printf("[DEBUG] CloudFlare Record create configuration: %#v", newRecord)

	rec, err := client.CreateRecord(d.Get("domain").(string), newRecord)

	if err != nil {
		return fmt.Errorf("Failed to create CloudFlare Record: %s", err)
	}

	d.SetId(rec.Id)
	log.Printf("[INFO] CloudFlare Record ID: %s", d.Id())

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	rec, err := client.RetrieveRecord(d.Get("domain").(string), d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf(
			"Couldn't find CloudFlare Record ID (%s) for domain (%s): %s",
			d.Id(), d.Get("domain").(string), err)
	}

	d.Set("name", rec.Name)
	d.Set("hostname", rec.FullName)
	d.Set("type", rec.Type)
	d.Set("value", rec.Value)
	d.Set("ttl", rec.Ttl)
	d.Set("priority", rec.Priority)

	return nil
}

func resourceCloudFlareRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	// CloudFlare requires we send all values for an update request
	updateRecord := &cloudflare.UpdateRecord{
		Name:    d.Get("name").(string),
		Type:    d.Get("type").(string),
		Content: d.Get("value").(string),
	}

	if ttl, ok := d.GetOk("ttl"); ok {
		updateRecord.Ttl = ttl.(string)
	}

	if priority, ok := d.GetOk("priority"); ok {
		updateRecord.Priority = priority.(string)
	}

	log.Printf("[DEBUG] CloudFlare Record update configuration: %#v", updateRecord)

	err := client.UpdateRecord(d.Get("domain").(string), d.Id(), updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update CloudFlare Record: %s", err)
	}

	return resourceCloudFlareRecordRead(d, meta)
}

func resourceCloudFlareRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*cloudflare.Client)

	log.Printf("[INFO] Deleting CloudFlare Record: %s, %s", d.Get("domain").(string), d.Id())

	err := client.DestroyRecord(d.Get("domain").(string), d.Id())

	if err != nil {
		return fmt.Errorf("Error deleting CloudFlare Record: %s", err)
	}

	return nil
}

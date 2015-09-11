package digitalocean

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/digitalocean"
)

func resourceDigitalOceanRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceDigitalOceanRecordCreate,
		Read:   resourceDigitalOceanRecordRead,
		Update: resourceDigitalOceanRecordUpdate,
		Delete: resourceDigitalOceanRecordDelete,

		Schema: map[string]*schema.Schema{
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"domain": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"priority": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"weight": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"value": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceDigitalOceanRecordCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	newRecord := digitalocean.CreateRecord{
		Type:     d.Get("type").(string),
		Name:     d.Get("name").(string),
		Data:     d.Get("value").(string),
		Priority: d.Get("priority").(string),
		Port:     d.Get("port").(string),
		Weight:   d.Get("weight").(string),
	}

	log.Printf("[DEBUG] record create configuration: %#v", newRecord)
	recId, err := client.CreateRecord(d.Get("domain").(string), &newRecord)
	if err != nil {
		return fmt.Errorf("Failed to create record: %s", err)
	}

	d.SetId(recId)
	log.Printf("[INFO] Record ID: %s", d.Id())

	return resourceDigitalOceanRecordRead(d, meta)
}

func resourceDigitalOceanRecordRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)
	domain := d.Get("domain").(string)

	rec, err := client.RetrieveRecord(domain, d.Id())
	if err != nil {
		// If the record is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return err
	}

	// Update response data for records with domain value
	if t := rec.Type; t == "CNAME" || t == "MX" || t == "NS" || t == "SRV" {
		// Append dot to response if resource value is absolute
		if value := d.Get("value").(string); strings.HasSuffix(value, ".") {
			rec.Data += "."
			// If resource value ends with current domain, make response data absolute
			if strings.HasSuffix(value, domain+".") {
				rec.Data += domain + "."
			}
		}
	}

	d.Set("name", rec.Name)
	d.Set("type", rec.Type)
	d.Set("value", rec.Data)
	d.Set("weight", rec.StringWeight())
	d.Set("priority", rec.StringPriority())
	d.Set("port", rec.StringPort())

	return nil
}

func resourceDigitalOceanRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	var updateRecord digitalocean.UpdateRecord
	if v, ok := d.GetOk("name"); ok {
		updateRecord.Name = v.(string)
	}

	log.Printf("[DEBUG] record update configuration: %#v", updateRecord)
	err := client.UpdateRecord(d.Get("domain").(string), d.Id(), &updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update record: %s", err)
	}

	return resourceDigitalOceanRecordRead(d, meta)
}

func resourceDigitalOceanRecordDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*digitalocean.Client)

	log.Printf(
		"[INFO] Deleting record: %s, %s", d.Get("domain").(string), d.Id())
	err := client.DestroyRecord(d.Get("domain").(string), d.Id())
	if err != nil {
		// If the record is somehow already destroyed, mark as
		// successfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			return nil
		}

		return fmt.Errorf("Error deleting record: %s", err)
	}

	return nil
}

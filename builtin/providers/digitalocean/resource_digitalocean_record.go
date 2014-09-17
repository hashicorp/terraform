package digitalocean

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/pearkes/digitalocean"
)

func resourceRecord() *schema.Resource {
	return &schema.Resource{
		Create: resourceRecordCreate,
		Read:   resourceRecordRead,
		Update: resourceRecordUpdate,
		Delete: resourceRecordDelete,

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

func resourceRecordCreate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

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

	return resourceRecordRead(d, meta)
}

func resourceRecordUpdate(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	var updateRecord digitalocean.UpdateRecord
	if v, ok := d.GetOk("name"); ok {
		updateRecord.Name = v.(string)
	}

	log.Printf("[DEBUG] record update configuration: %#v", updateRecord)
	err := client.UpdateRecord(d.Get("domain").(string), d.Id(), &updateRecord)
	if err != nil {
		return fmt.Errorf("Failed to update record: %s", err)
	}

	return resourceRecordRead(d, meta)
}

func resourceRecordDelete(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	log.Printf(
		"[INFO] Deleting record: %s, %s", d.Get("domain").(string), d.Id())
	err := client.DestroyRecord(d.Get("domain").(string), d.Id())
	if err != nil {
		// If the record is somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			return nil
		}

		return fmt.Errorf("Error deleting record: %s", err)
	}

	return nil
}

func resourceRecordRead(d *schema.ResourceData, meta interface{}) error {
	p := meta.(*ResourceProvider)
	client := p.client

	rec, err := client.RetrieveRecord(d.Get("domain").(string), d.Id())
	if err != nil {
		// If the record is somehow already destroyed, mark as
		// succesfully gone
		if strings.Contains(err.Error(), "404 Not Found") {
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", rec.Name)
	d.Set("type", rec.Type)
	d.Set("value", rec.Data)
	d.Set("weight", rec.StringWeight())
	d.Set("priority", rec.StringPriority())
	d.Set("port", rec.StringPort())

	return nil
}
